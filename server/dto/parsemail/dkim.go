package parsemail

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/utils/consts"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/file"
	"github.com/emersion/go-msgauth/dkim"
	log "github.com/sirupsen/logrus"
)

// DkimKeyBits is the RSA key size used for generated DKIM keys. The setup wizard
// used to generate 1024 bit keys, which is no longer considered safe.
const DkimKeyBits = 2048

// Dkim owns the signing material of a single mail domain.
type Dkim struct {
	Domain     string
	Selector   string
	privateKey crypto.Signer
}

type signerSet struct {
	byDomain map[string]*Dkim
	primary  *Dkim
}

// signers is the published DKIM key set. It is swapped atomically so that
// message building never observes a half-reloaded key set.
var signers atomic.Pointer[signerSet]

// Init loads the DKIM keys of every configured domain.
func Init() {
	Reload()
}

// Reload rebuilds the DKIM key set from the current configuration. Domains whose
// key cannot be read are skipped: their mail is sent unsigned, which is strictly
// better than a signature the receiver cannot verify.
func Reload() {
	cfg := config.Get()
	next := &signerSet{byDomain: make(map[string]*Dkim, len(cfg.Domains))}

	for _, d := range cfg.Domains {
		key, err := loadPrivateKey(d.DKIMPrivateKeyPath)
		if err != nil {
			log.Warnf("DKIM disabled for %s: %v (publish a key from the admin panel to enable it)", d.Name, err)
			continue
		}
		signer := &Dkim{Domain: d.Name, Selector: d.DKIMSelector, privateKey: key}
		next.byDomain[d.Name] = signer
		if d.Name == cfg.PrimaryDomain() {
			next.primary = signer
		}
	}

	signers.Store(next)
}

// Sign returns msgData signed with the DKIM key that matches from's domain.
//
// The domain of the envelope sender decides the key: signing everything with the
// primary domain's key breaks DMARC alignment for mail sent from any other
// domain we serve. Mail sent as a domain we do not serve falls back to the
// primary key, and a served domain without a key is left unsigned.
func Sign(msgData string, from *User) []byte {
	set := signers.Load()
	if set == nil || len(set.byDomain) == 0 {
		return []byte(msgData)
	}

	domain := ""
	if from != nil {
		_, domain = from.GetDomainAccount()
	}
	domain = strings.ToLower(strings.TrimSpace(domain))

	signer := set.byDomain[domain]
	if signer == nil {
		if config.Get().HasDomain(domain) {
			log.Warnf("no DKIM key for %s, message sent unsigned", domain)
			return []byte(msgData)
		}
		signer = set.primary
	}
	if signer == nil {
		return []byte(msgData)
	}

	signed, err := signer.sign(msgData)
	if err != nil {
		log.Errorf("DKIM sign failed for %s: %v", signer.Domain, err)
		return []byte(msgData)
	}
	return signed
}

func (p *Dkim) sign(msgData string) ([]byte, error) {
	var b bytes.Buffer

	options := &dkim.SignOptions{
		Domain:   p.Domain,
		Selector: p.Selector,
		Signer:   p.privateKey,
	}

	if err := dkim.Sign(&b, strings.NewReader(msgData), options); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// PublicKeyPath is the convention used to remember a domain's published key:
// the DNS record is cached next to the private key.
func PublicKeyPath(privateKeyPath string) string {
	if strings.HasSuffix(privateKeyPath, ".priv") {
		return strings.TrimSuffix(privateKeyPath, ".priv") + ".public"
	}
	return privateKeyPath + ".public"
}

// EnsureDkimKeyPair creates an RSA key pair for the domain if it has none, and
// returns the value of the TXT record that must be published as
// <selector>._domainkey.<domain>. Existing keys are never overwritten.
func EnsureDkimKeyPair(privateKeyPath string) (string, error) {
	if file.PathExist(privateKeyPath) {
		return DkimPublicRecord(privateKeyPath)
	}

	key, err := rsa.GenerateKey(rand.Reader, DkimKeyBits)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := file.WriteAtomic(privateKeyPath, pemBlock, 0600); err != nil {
		return "", err
	}
	return writeDkimPublicRecord(privateKeyPath, key.Public())
}

// DkimPublicRecord returns the TXT record value for the key at privateKeyPath,
// deriving it from the private key when no cached copy exists.
func DkimPublicRecord(privateKeyPath string) (string, error) {
	if cached, err := os.ReadFile(PublicKeyPath(privateKeyPath)); err == nil && len(bytes.TrimSpace(cached)) > 0 {
		return string(bytes.TrimSpace(cached)), nil
	}

	key, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return "", err
	}
	return writeDkimPublicRecord(privateKeyPath, key.Public())
}

func writeDkimPublicRecord(privateKeyPath string, publicKey crypto.PublicKey) (string, error) {
	keyType, err := dkimKeyType(publicKey)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	params := []string{
		"v=DKIM1",
		"k=" + keyType,
		"p=" + base64.StdEncoding.EncodeToString(der),
	}
	record := strings.Join(params, "; ")

	if err := file.WriteAtomic(PublicKeyPath(privateKeyPath), []byte(record+"\n"), 0644); err != nil {
		return "", err
	}
	return record, nil
}

func dkimKeyType(publicKey crypto.PublicKey) (string, error) {
	switch publicKey.(type) {
	case *rsa.PublicKey:
		// RFC 6376 is inconsistent about whether RSA public keys should
		// be formatted as RSAPublicKey or SubjectPublicKeyInfo.
		// Erratum 3017 (https://www.rfc-editor.org/errata/eid3017)
		// proposes allowing both. We use SubjectPublicKeyInfo for
		// consistency with other implementations including opendkim,
		// Gmail, and Fastmail.
		return "rsa", nil
	case ed25519.PublicKey:
		return "ed25519", nil
	default:
		return "", fmt.Errorf("unsupported DKIM key type %T", publicKey)
	}
}

func loadPrivateKey(path string) (crypto.Signer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("no PEM data found")
	}

	switch strings.ToUpper(block.Type) {
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		signer, ok := k.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("key %T cannot sign", k)
		}
		return signer, nil
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EDDSA PRIVATE KEY":
		if len(block.Bytes) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid Ed25519 private key size")
		}
		return ed25519.PrivateKey(block.Bytes), nil
	default:
		return nil, fmt.Errorf("unknown private key type: '%v'", block.Type)
	}
}

func Check(ctx *context.Context, mail io.Reader) bool {

	verifications, err := dkim.Verify(mail)
	if err != nil {
		log.WithContext(ctx).Warnf("DKIM Error:%v", err)
	}

	if len(verifications) == 0 {
		return false
	}

	for _, v := range verifications {
		if v.Domain == consts.TEST_DOMAIN {
			return true
		}
		if v.Err == nil {
			log.Println("Valid signature for:", v.Domain)
		} else {
			log.Println("Invalid signature for:", v.Domain, v.Err)
			return false
		}
	}
	return true
}

// DomainKeyDir is where per-domain keys are stored.
func DomainKeyDir() string {
	return filepath.Join(config.ROOT_PATH, "config", "dkim")
}
