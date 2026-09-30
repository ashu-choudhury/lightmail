package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/dto/parsemail"
)

func TestNormalize(t *testing.T) {
	valid := map[string]string{
		"Example.COM":         "example.com",
		"  mail.example.com. ": "mail.example.com",
		"a-b.example.co.uk":   "a-b.example.co.uk",
	}
	for input, want := range valid {
		got, err := Normalize(input)
		if err != nil {
			t.Fatalf("Normalize(%q) failed: %v", input, err)
		}
		if got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{"", "localhost", "exa mple.com", "-example.com", "example.com-", "example..com", "192.168.0.1", "example.com/path"}
	for _, input := range invalid {
		if _, err := Normalize(input); err == nil {
			t.Fatalf("Normalize(%q) must be rejected", input)
		}
	}
}

func TestDNSRecordsCoverMailRouting(t *testing.T) {
	d := config.Domain{
		Name:               "example.com",
		DKIMSelector:       config.DefaultDKIMSelector,
		DKIMPrivateKeyPath: filepath.Join(t.TempDir(), "example.com.priv"),
	}

	records := DNSRecords(d)
	if len(records) != 3 {
		t.Fatalf("a domain without a key must still get MX, SPF and DMARC, got %+v", records)
	}
	if records[0].Type != "MX" || records[0].Value != "smtp.example.com" {
		t.Fatalf("expected mail to arrive through smtp.example.com, got %+v", records[0])
	}

	if _, err := parsemail.EnsureDkimKeyPair(d.DKIMPrivateKeyPath); err != nil {
		t.Fatal(err)
	}

	records = DNSRecords(d)
	dkim := records[len(records)-1]
	if dkim.Host != "default._domainkey" {
		t.Fatalf("expected the DKIM record below the configured selector, got %+v", dkim)
	}
	if !strings.HasPrefix(dkim.Value, "v=DKIM1; k=rsa; p=") {
		t.Fatalf("unexpected DKIM record value %q", dkim.Value)
	}
}

func TestAddAndDeleteDomainHotApplies(t *testing.T) {
	oldRoot := config.ROOT_PATH
	oldConfig := config.Get()
	t.Cleanup(func() {
		config.ROOT_PATH = oldRoot
		config.Set(oldConfig)
	})

	config.ROOT_PATH = filepath.Join(t.TempDir()) + string(filepath.Separator)
	if err := config.Reload(); err != nil {
		t.Fatal(err)
	}
	config.Set(&config.Config{Domain: "primary.example.com"})

	added, err := Add("Second.Example.COM")
	if err != nil {
		t.Fatal(err)
	}
	if added.Name != "second.example.com" {
		t.Fatalf("expected a canonical domain name, got %q", added.Name)
	}
	if added.Primary {
		t.Fatal("an added domain must not become the primary domain")
	}
	if !added.DKIMReady {
		t.Fatal("adding a domain must generate its DKIM key")
	}
	if len(added.Records) != 4 {
		t.Fatalf("expected MX, SPF, DMARC and DKIM records, got %+v", added.Records)
	}

	// The change is live without restarting any listener.
	if !config.Get().HasDomain("second.example.com") {
		t.Fatal("the domain must be hot applied to the running configuration")
	}
	if _, err := os.Stat(filepath.Join(config.ROOT_PATH, "config", "dkim", "second.example.com.priv")); err != nil {
		t.Fatalf("the DKIM key must exist on disk: %v", err)
	}

	// And it survives a restart.
	reloaded, err := config.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.HasDomain("second.example.com") {
		t.Fatal("the domain must be persisted to config.json")
	}

	if _, err := Add("second.example.com"); err == nil {
		t.Fatal("adding an already served domain must fail")
	}
	if err := Delete("primary.example.com"); err == nil {
		t.Fatal("the primary domain must not be removable")
	}
	if err := Delete("missing.example.com"); err == nil {
		t.Fatal("deleting an unknown domain must fail")
	}

	if err := Delete("second.example.com"); err != nil {
		t.Fatal(err)
	}
	if config.Get().HasDomain("second.example.com") {
		t.Fatal("the domain must be removed from the running configuration")
	}
	reloaded, err = config.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.HasDomain("second.example.com") {
		t.Fatal("the removal must be persisted to config.json")
	}
}

func TestGenerateDKIMRotatesKeys(t *testing.T) {
	oldRoot := config.ROOT_PATH
	oldConfig := config.Get()
	t.Cleanup(func() {
		config.ROOT_PATH = oldRoot
		config.Set(oldConfig)
	})

	config.ROOT_PATH = filepath.Join(t.TempDir()) + string(filepath.Separator)
	config.Set(&config.Config{Domain: "example.com"})

	first, err := GenerateDKIM("example.com", false)
	if err != nil {
		t.Fatal(err)
	}

	same, err := GenerateDKIM("example.com", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Records[len(first.Records)-1].Value != same.Records[len(same.Records)-1].Value {
		t.Fatal("generating without rotate must keep the published key")
	}

	rotated, err := GenerateDKIM("example.com", true)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Records[len(rotated.Records)-1].Value == first.Records[len(first.Records)-1].Value {
		t.Fatal("rotate must publish a new key")
	}

	if _, err := GenerateDKIM("missing.example.com", false); err == nil {
		t.Fatal("generating a key for an unknown domain must fail")
	}
}
