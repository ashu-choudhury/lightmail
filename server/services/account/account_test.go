package account

import (
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/config"
)

func withDomains(t *testing.T, primary string, extra ...string) {
	t.Helper()
	old := config.Get().Clone()
	domains := []config.Domain{{Name: primary}}
	for _, name := range extra {
		domains = append(domains, config.Domain{Name: name})
	}
	config.Set(&config.Config{Domain: primary, Domains: domains})
	t.Cleanup(func() { config.Set(old) })
}

func TestNormalizeCompletesBareLoginWithPrimaryDomain(t *testing.T) {
	withDomains(t, "a.com", "b.com")

	cases := map[string]string{
		"Bob":            "bob@a.com",
		"  bob  ":        "bob@a.com",
		"bob@b.com":      "bob@b.com",
		"BOB@B.COM":      "bob@b.com",
		"bob@unknown.tv": "bob@unknown.tv",
		"":               "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeWithoutConfiguredDomainKeepsInput(t *testing.T) {
	old := config.Get().Clone()
	config.Set(&config.Config{})
	t.Cleanup(func() { config.Set(old) })

	if got := Normalize("admin"); got != "admin" {
		t.Fatalf("Normalize(admin) = %q, want admin", got)
	}
}

func TestCandidatesFallsBackToLegacyBareAccount(t *testing.T) {
	withDomains(t, "a.com", "b.com")

	got := Candidates("bob")
	if strings.Join(got, ",") != "bob@a.com,bob" {
		t.Fatalf("Candidates(bob) = %v, want [bob@a.com bob]", got)
	}

	// A secondary domain must never alias the bare account.
	got = Candidates("bob@b.com")
	if strings.Join(got, ",") != "bob@b.com" {
		t.Fatalf("Candidates(bob@b.com) = %v, want [bob@b.com]", got)
	}
}

func TestLoginPredicateBindsEveryCandidate(t *testing.T) {
	withDomains(t, "a.com", "b.com")

	predicate, args := LoginPredicate("bob")
	if predicate != "(LOWER(account) = ? or LOWER(account) = ?)" {
		t.Fatalf("predicate = %q", predicate)
	}
	if len(args) != 2 || args[0] != "bob@a.com" || args[1] != "bob" {
		t.Fatalf("args = %v, want [bob@a.com bob]", args)
	}

	predicate, args = LoginPredicate("")
	if predicate != "1 = 0" || len(args) != 0 {
		t.Fatalf("empty login produced %q %v", predicate, args)
	}
}

func TestSplitAndBuild(t *testing.T) {
	if local, domain := Split("Bob@A.com"); local != "bob" || domain != "a.com" {
		t.Fatalf("Split = %q, %q", local, domain)
	}
	if local, domain := Split("bare"); local != "bare" || domain != "" {
		t.Fatalf("Split(bare) = %q, %q", local, domain)
	}
	if got := Build("Bob", "a.com"); got != "bob@a.com" {
		t.Fatalf("Build = %q", got)
	}
}

func TestValidateLocalRejectsUnsafeNames(t *testing.T) {
	for _, ok := range []string{"bob", "bo.b", "bo_b", "bo-b", "bo+b", "BoB"} {
		if err := ValidateLocal(ok); err != nil {
			t.Errorf("ValidateLocal(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", " ", "bo b", "bo@b", "böb", strings.Repeat("a", maxLocalLength+1)} {
		if err := ValidateLocal(bad); err == nil {
			t.Errorf("ValidateLocal(%q) = nil, want error", bad)
		}
	}
}
