package parsemail

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/config"
)

func TestCheck(t *testing.T) {

	res := Check(nil, strings.NewReader(`Received: from jdl.ac.cn ([159.226.42.8])

xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
`))
	if res != false {
		t.Errorf("DKIM Error")
	}

}

// TestSignUsesTheSenderDomainKey is the regression test for DMARC alignment:
// mail sent from a second domain used to be signed with the primary domain's key.
func TestSignUsesTheSenderDomainKey(t *testing.T) {
	dir := t.TempDir()
	oldConfig := config.Get()
	oldSigners := signers.Load()
	t.Cleanup(func() {
		config.Set(oldConfig)
		signers.Store(oldSigners)
	})

	firstKey := filepath.Join(dir, "first.priv")
	secondKey := filepath.Join(dir, "second.priv")
	for _, key := range []string{firstKey, secondKey} {
		if _, err := EnsureDkimKeyPair(key); err != nil {
			t.Fatal(err)
		}
	}

	config.Set(&config.Config{
		Domain: "first.example.com",
		Domains: []config.Domain{
			{Name: "first.example.com", DKIMSelector: config.DefaultDKIMSelector, DKIMPrivateKeyPath: firstKey},
			{Name: "second.example.com", DKIMSelector: config.DefaultDKIMSelector, DKIMPrivateKeyPath: secondKey},
		},
	})
	Reload()

	unsigned := "From: someone@example.com\r\nSubject: hi\r\n\r\nbody\r\n"

	for _, domain := range []string{"first.example.com", "second.example.com"} {
		signed := string(Sign(unsigned, &User{EmailAddress: "someone@" + domain}))
		if !strings.Contains(signed, "d="+domain+";") {
			t.Fatalf("expected %s to be signed with its own key, got %q", domain, signed)
		}
	}

	// Mail sent as an address we do not own still falls back to the primary key.
	foreign := string(Sign(unsigned, &User{EmailAddress: "someone@elsewhere.example"}))
	if !strings.Contains(foreign, "d=first.example.com;") {
		t.Fatalf("expected the primary key to sign foreign senders, got %q", foreign)
	}

	// A served domain whose key is missing must stay unsigned rather than be
	// signed with a key the receiver cannot look up.
	config.Set(&config.Config{
		Domain:  "first.example.com",
		Domains: []config.Domain{{Name: "first.example.com", DKIMPrivateKeyPath: filepath.Join(dir, "missing.priv")}},
	})
	Reload()

	if got := string(Sign(unsigned, &User{EmailAddress: "someone@first.example.com"})); got != unsigned {
		t.Fatalf("expected an unsigned message, got %q", got)
	}
	if got := string(Sign(unsigned, nil)); got != unsigned {
		t.Fatalf("expected a senderless message to stay unsigned, got %q", got)
	}
}
