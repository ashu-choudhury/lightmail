// Package domain manages the mail domains this server serves and the DNS records
// they need. It is the only place that adds or removes domains from the config
// file, and it hot-applies every change to the running server.
package domain

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/dto/parsemail"
)

// Record is one DNS record a domain needs in order to send and receive mail.
type Record struct {
	Type  string `json:"type"`
	Host  string `json:"host"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// Domain is the admin facing view of a served domain.
type Domain struct {
	Name         string   `json:"name"`
	Primary      bool     `json:"primary"`
	DKIMSelector string   `json:"dkim_selector"`
	DKIMReady    bool     `json:"dkim_ready"`
	Records      []Record `json:"records"`
}

// mutex serializes config mutations so two concurrent admin requests cannot lose
// each other's change to a read-modify-write cycle.
var mutex sync.Mutex

// List returns every served domain together with the DNS records it still needs.
func List() []Domain {
	cfg := config.Get()
	ret := make([]Domain, 0, len(cfg.Domains))
	for _, d := range cfg.Domains {
		ret = append(ret, describe(cfg, d))
	}
	return ret
}

// Add starts serving name: it generates a DKIM key pair and publishes the new
// configuration to the running server without restarting any listener.
func Add(name string) (Domain, error) {
	name, err := Normalize(name)
	if err != nil {
		return Domain{}, err
	}
	if config.Get().HasDomain(name) {
		return Domain{}, fmt.Errorf("%s is already served by this server", name)
	}

	// Key generation is deliberately done before taking the lock: it is the slow
	// part and it must not happen while holding a lock other requests need.
	keyPath := filepath.Join(parsemail.DomainKeyDir(), name+".priv")
	if _, err := parsemail.EnsureDkimKeyPair(keyPath); err != nil {
		return Domain{}, fmt.Errorf("generating a DKIM key for %s failed: %w", name, err)
	}

	err = mutate(func(cfg *config.Config) error {
		// Re-check under the lock: the pre-check above only exists to fail fast
		// before generating a key.
		if cfg.HasDomain(name) {
			return fmt.Errorf("%s is already served by this server", name)
		}
		cfg.Domains = append(cfg.Domains, config.Domain{
			Name:               name,
			DKIMSelector:       config.DefaultDKIMSelector,
			DKIMPrivateKeyPath: keyPath,
		})
		return nil
	})
	if err != nil {
		return Domain{}, err
	}

	d, _ := config.Get().FindDomain(name)
	return describe(config.Get(), d), nil
}

// Delete stops serving name. The DKIM key is kept on disk so that republishing
// the same domain does not invalidate records already in DNS.
func Delete(name string) error {
	name, err := Normalize(name)
	if err != nil {
		return err
	}

	return mutate(func(cfg *config.Config) error {
		if cfg.PrimaryDomain() == name {
			return fmt.Errorf("%s is the primary domain and cannot be removed", name)
		}

		domains := make([]config.Domain, 0, len(cfg.Domains))
		found := false
		for _, d := range cfg.Domains {
			if d.Name == name {
				found = true
				continue
			}
			domains = append(domains, d)
		}
		if !found {
			return fmt.Errorf("%s is not served by this server", name)
		}

		cfg.Domains = domains
		return nil
	})
}

// GenerateDKIM makes sure an existing domain has a DKIM key. With rotate set the
// current key is discarded, which requires publishing the new record in DNS.
func GenerateDKIM(name string, rotate bool) (Domain, error) {
	name, err := Normalize(name)
	if err != nil {
		return Domain{}, err
	}

	d, ok := config.Get().FindDomain(name)
	if !ok {
		return Domain{}, fmt.Errorf("%s is not served by this server", name)
	}

	if rotate {
		for _, path := range []string{d.DKIMPrivateKeyPath, parsemail.PublicKeyPath(d.DKIMPrivateKeyPath)} {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return Domain{}, err
			}
		}
	}

	if _, err := parsemail.EnsureDkimKeyPair(d.DKIMPrivateKeyPath); err != nil {
		return Domain{}, fmt.Errorf("generating a DKIM key for %s failed: %w", name, err)
	}
	parsemail.Reload()

	return describe(config.Get(), d), nil
}

// DetectServerIPs inspects network interfaces to find active non-loopback public IPs.
func DetectServerIPs() (ipv4 string, ipv6 string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				if ipv4 == "" && !ipNet.IP.IsPrivate() {
					ipv4 = ipNet.IP.String()
				}
			} else {
				if ipv6 == "" && ipNet.IP.IsGlobalUnicast() {
					ipv6 = ipNet.IP.String()
				}
			}
		}
	}
	return
}

// DNSRecords describes the records a domain needs to receive mail and to pass
// SPF, DKIM and DMARC.
func DNSRecords(d config.Domain) []Record {
	cfg := config.Get()
	primary := cfg.PrimaryDomain()

	// Mail host: if domain is secondary, prefer primary server smtp host if defined
	mailHost := "smtp." + d.Name
	if primary != "" && primary != d.Name {
		mailHost = "smtp." + primary
	}

	// Dynamic SPF with server IPs
	ipv4, ipv6 := DetectServerIPs()
	var spfParts []string
	spfParts = append(spfParts, "v=spf1")
	if ipv6 != "" {
		spfParts = append(spfParts, "ip6:"+ipv6)
	}
	if ipv4 != "" {
		spfParts = append(spfParts, "ip4:"+ipv4)
	}
	spfParts = append(spfParts, "a", "mx", "~all")
	spfValue := strings.Join(spfParts, " ")

	records := []Record{
		{Type: "MX", Host: "@", Value: mailHost, TTL: 3600},
		{Type: "TXT", Host: "@", Value: spfValue, TTL: 3600},
		{Type: "TXT", Host: "_dmarc", Value: "v=DMARC1; p=none; rua=mailto:postmaster@" + d.Name, TTL: 3600},
	}

	// If primary domain and we have an IPv6 address, also publish the smtp AAAA record
	if d.Name == primary && ipv6 != "" {
		records = append(records, Record{Type: "AAAA", Host: "smtp", Value: ipv6, TTL: 3600})
	} else if d.Name != primary && primary != "" {
		// Secondary domain gets CNAME smtp -> smtp.primary for client configuration
		records = append(records, Record{Type: "CNAME", Host: "smtp", Value: "smtp." + primary, TTL: 3600})
	}

	if record, err := parsemail.DkimPublicRecord(d.DKIMPrivateKeyPath); err == nil {
		selector := d.DKIMSelector
		if selector == "" {
			selector = config.DefaultDKIMSelector
		}
		records = append(records, Record{Type: "TXT", Host: selector + "._domainkey", Value: record, TTL: 3600})
	}
	return records
}

func describe(cfg *config.Config, d config.Domain) Domain {
	selector := d.DKIMSelector
	if selector == "" {
		selector = config.DefaultDKIMSelector
	}
	return Domain{
		Name:         d.Name,
		Primary:      cfg.PrimaryDomain() == d.Name,
		DKIMSelector: selector,
		DKIMReady:    hasDKIMKey(d.DKIMPrivateKeyPath),
		Records:      DNSRecords(d),
	}
}

func hasDKIMKey(privateKeyPath string) bool {
	if _, err := os.Stat(privateKeyPath); err != nil {
		return false
	}
	_, err := parsemail.DkimPublicRecord(privateKeyPath)
	return err == nil
}

// mutate applies fn to a clone of the configuration, persists it to config.json
// and publishes it. Nothing is published when the write fails, so the running
// server never diverges from what is on disk.
func mutate(fn func(cfg *config.Config) error) error {
	mutex.Lock()
	defer mutex.Unlock()

	cfg := config.Get().Clone()
	if err := fn(cfg); err != nil {
		return err
	}
	if err := config.WriteConfig(cfg); err != nil {
		return err
	}

	config.Set(cfg)
	parsemail.Reload()
	return nil
}

// Normalize validates a mail domain name and returns it in canonical form.
func Normalize(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".")

	if name == "" {
		return "", fmt.Errorf("the domain must not be empty")
	}
	if len(name) > 253 {
		return "", fmt.Errorf("%s is not a valid domain name", name)
	}

	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return "", fmt.Errorf("%s is not a valid domain name, use the fully qualified form such as example.com", name)
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return "", fmt.Errorf("%s is not a valid domain name", name)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", fmt.Errorf("%s is not a valid domain name", name)
		}
		for _, r := range label {
			isLetter := r >= 'a' && r <= 'z'
			isDigit := r >= '0' && r <= '9'
			if !isLetter && !isDigit && r != '-' {
				return "", fmt.Errorf("%s is not a valid domain name", name)
			}
		}
	}
	if !strings.ContainsAny(labels[len(labels)-1], "abcdefghijklmnopqrstuvwxyz") {
		return "", fmt.Errorf("%s is not a valid domain name", name)
	}
	return name, nil
}
