package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/signal"
	"github.com/Jinnrry/pmail/utils/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// EnsureSelfSignedCert creates a self-signed fallback ECDSA certificate if no cert exists on disk.
func EnsureSelfSignedCert(certPath, keyPath, domain string) error {
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return err
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Lightmail Self-Signed"},
			CommonName:   domain,
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain, "mail." + domain, "smtp." + domain, "imap." + domain, "pop." + domain},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return err
	}

	log.Infof("Generated fallback self-signed TLS certificate for %s", domain)
	return nil
}

func GetSSL() string {
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}
	if cfg.SSLType == "" {
		return config.SSLTypeUser
	}
	return cfg.SSLType
}

func SetSSL(sslType, priKey, crtKey string) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	cfg.SSLType = sslType
	cfg.SSLPrivateKeyPath = priKey
	cfg.SSLPublicKeyPath = crtKey

	if err := config.WriteConfig(cfg); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func GenSSL(update bool) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	domain := cfg.PrimaryDomain()
	if domain == "" {
		domain = "localhost"
	}

	return EnsureSelfSignedCert(cfg.SSLPublicKeyPath, cfg.SSLPrivateKeyPath, domain)
}

// CheckSSLCrtInfo returns the remaining days of certificate validity.
func CheckSSLCrtInfo() (int, time.Time, bool, error) {
	cfg, err := config.ReadConfig()
	if err != nil {
		return -1, time.Now(), true, err
	}

	tlsCert, err := tls.LoadX509KeyPair(cfg.SSLPublicKeyPath, cfg.SSLPrivateKeyPath)
	if err != nil {
		return -1, time.Now(), true, errors.Wrap(err)
	}

	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return -1, time.Now(), true, errors.Wrap(err)
	}

	nameMatchFail := true
	for _, name := range cert.DNSNames {
		if strings.Contains(name, "imap") || strings.Contains(name, cfg.PrimaryDomain()) {
			nameMatchFail = false
			break
		}
	}

	hours := cert.NotAfter.Sub(time.Now()).Hours()
	if hours <= 0 {
		return -1, time.Now(), nameMatchFail, errors.New("Certificate has expired")
	}

	return cast.ToInt(hours / 24), cert.NotAfter, nameMatchFail, nil
}

// Update verifies certificate validity and reloads if files have been replaced on disk.
func Update(needRestart bool) {
	days, _, _, err := CheckSSLCrtInfo()
	if err != nil {
		log.Warnf("SSL certificate check: %v", err)
		return
	}
	if days < 15 {
		log.Warnf("SSL certificate expires in %d days. Please update certificate files.", days)
	}
	if needRestart {
		signal.RestartChan <- true
	}
}
