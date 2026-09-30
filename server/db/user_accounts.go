package db

import (
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/services/account"
	log "github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

// MigrateUserAccounts upgrades mailboxes created before accounts belonged to a
// domain. Such rows hold only the local part, which would now alias every served
// domain, so the primary domain is appended once. The migration is idempotent:
// addresses that are already canonical are left untouched.
func MigrateUserAccounts() error {
	return migrateUserAccounts(Instance)
}

func migrateUserAccounts(engine *xorm.Engine) error {
	if engine == nil {
		return nil
	}

	var users []models.User
	if err := engine.Table(&models.User{}).OrderBy("id asc").Find(&users); err != nil {
		return err
	}

	owner := make(map[string]int, len(users))
	for _, u := range users {
		owner[u.Account] = u.ID
	}

	for _, u := range users {
		canonical := account.Normalize(u.Account)
		if canonical == "" || canonical == u.Account {
			continue
		}
		if otherID, taken := owner[canonical]; taken && otherID != u.ID {
			// Two rows would collapse into the same mailbox. Keep both and let an
			// administrator resolve it rather than silently dropping messages.
			log.Errorf("account migration skipped user %d: %q is already used by user %d", u.ID, canonical, otherID)
			continue
		}
		if _, err := engine.ID(u.ID).Cols("account").Update(&models.User{Account: canonical}); err != nil {
			return err
		}
		delete(owner, u.Account)
		owner[canonical] = u.ID
		log.Infof("account migration: user %d %q -> %q", u.ID, u.Account, canonical)
	}
	return nil
}
