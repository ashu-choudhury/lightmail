// Package account resolves the mailbox addresses that users log in with and
// that messages are stored against. A mailbox is always identified by its full
// address (local@domain), so bob@a.com and bob@b.com are different mailboxes.
package account

import (
	"fmt"
	"strings"

	"github.com/Jinnrry/pmail/config"
)

// MaxAddressLength bounds the stored address and matches the user.account column.
const MaxAddressLength = 190

// maxLocalLength is the RFC 5321 limit for the part before the @.
const maxLocalLength = 64

// primaryDomain returns the canonical primary domain, or "" when the server has
// not been configured with a domain yet.
func primaryDomain() string {
	return strings.ToLower(strings.TrimSpace(config.Get().PrimaryDomain()))
}

// Normalize turns a submitted login name into the canonical mailbox address. A
// bare local part is completed with the primary domain, which keeps logins such
// as "admin" working on a single-domain server and for legacy accounts.
func Normalize(login string) string {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return ""
	}
	if strings.Contains(login, "@") {
		return login
	}
	domain := primaryDomain()
	if domain == "" {
		return login
	}
	return login + "@" + domain
}

// Split returns the local part and the domain of an address. A missing half
// comes back empty instead of repeating the input.
func Split(address string) (local, domain string) {
	address = strings.ToLower(strings.TrimSpace(address))
	at := strings.LastIndex(address, "@")
	if at < 0 {
		return address, ""
	}
	return strings.TrimSpace(address[:at]), strings.TrimSpace(address[at+1:])
}

// LocalPart returns the part of an address before the @.
func LocalPart(address string) string {
	local, _ := Split(address)
	return local
}

// DomainPart returns the part of an address after the @.
func DomainPart(address string) string {
	_, domain := Split(address)
	return domain
}

// Build combines a local part and a domain into a canonical mailbox address.
func Build(local, domain string) string {
	return strings.ToLower(strings.TrimSpace(local)) + "@" + strings.ToLower(strings.TrimSpace(domain))
}

// Candidates lists the stored account values a login name may match, most
// specific first. Accounts created before mailboxes carried a domain only hold
// the bare local part, so a login for the primary domain still finds them.
func Candidates(login string) []string {
	canonical := Normalize(login)
	if canonical == "" {
		return nil
	}
	ret := []string{canonical}
	if local, domain := Split(canonical); domain != "" && domain == primaryDomain() {
		ret = append(ret, local)
	}
	return ret
}

// LoginPredicate builds the SQL predicate matching every candidate for a login
// name. Every candidate is lowercased, so the comparison only folds stored case.
func LoginPredicate(login string) (predicate string, args []any) {
	candidates := Candidates(login)
	switch len(candidates) {
	case 0:
		// An empty login must never match an account.
		return "1 = 0", nil
	case 1:
		return "LOWER(account) = ?", []any{candidates[0]}
	default:
		parts := make([]string, 0, len(candidates))
		args = make([]any, 0, len(candidates))
		for _, candidate := range candidates {
			parts = append(parts, "LOWER(account) = ?")
			args = append(args, candidate)
		}
		return "(" + strings.Join(parts, " or ") + ")", args
	}
}

// ValidateLocal reports whether local is usable as the part before the @.
func ValidateLocal(local string) error {
	local = strings.ToLower(strings.TrimSpace(local))
	if local == "" {
		return fmt.Errorf("the account name must not be empty")
	}
	if len(local) > maxLocalLength {
		return fmt.Errorf("the account name must be at most %d characters", maxLocalLength)
	}
	for _, r := range local {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit && r != '.' && r != '_' && r != '-' && r != '+' {
			return fmt.Errorf("the account name may only contain letters, digits, dot, underscore, hyphen and plus")
		}
	}
	return nil
}
