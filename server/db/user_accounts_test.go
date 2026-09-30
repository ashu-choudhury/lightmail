package db

import (
	"path/filepath"
	"testing"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/models"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func newMigrationEngine(t *testing.T) *xorm.Engine {
	t.Helper()
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "migrate.db"))
	if err != nil {
		t.Fatal(err)
	}
	engine.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = engine.Close() })
	return engine
}

func withPrimaryDomain(t *testing.T, domain string) {
	t.Helper()
	old := config.Get().Clone()
	config.Set(&config.Config{Domain: domain})
	t.Cleanup(func() { config.Set(old) })
}

func TestMigrateUserAccountsAppendsPrimaryDomain(t *testing.T) {
	withPrimaryDomain(t, "a.com")
	engine := newMigrationEngine(t)
	if err := engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}

	// Simulate rows written before accounts were domain scoped.
	for _, account := range []string{"ashu", "BOB", "carol@b.com"} {
		if _, err := engine.Exec("insert into user (account, name, password, disabled, is_admin) values (?, ?, ?, 0, 0)", account, account, "x"); err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateUserAccounts(engine); err != nil {
		t.Fatal(err)
	}

	var users []models.User
	if err := engine.Table(&models.User{}).OrderBy("id asc").Find(&users); err != nil {
		t.Fatal(err)
	}
	got := []string{users[0].Account, users[1].Account, users[2].Account}
	want := []string{"ashu@a.com", "bob@a.com", "carol@b.com"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("account %d = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestMigrateUserAccountsIsIdempotent(t *testing.T) {
	withPrimaryDomain(t, "a.com")
	engine := newMigrationEngine(t)
	if err := engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Insert(&models.User{Account: "ashu@a.com", Name: "ashu", Password: "x"}); err != nil {
		t.Fatal(err)
	}

	if err := migrateUserAccounts(engine); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserAccounts(engine); err != nil {
		t.Fatal(err)
	}

	var users []models.User
	if err := engine.Table(&models.User{}).Find(&users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Account != "ashu@a.com" {
		t.Fatalf("users = %+v, want the canonical row untouched", users)
	}
}

func TestMigrateUserAccountsKeepsCollision(t *testing.T) {
	withPrimaryDomain(t, "a.com")
	engine := newMigrationEngine(t)
	if err := engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}
	// Both rows already exist, so appending the domain to "bob" would collide.
	if _, err := engine.Insert(&models.User{Account: "bob", Name: "legacy", Password: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Insert(&models.User{Account: "bob@a.com", Name: "new", Password: "x"}); err != nil {
		t.Fatal(err)
	}

	if err := migrateUserAccounts(engine); err != nil {
		t.Fatal(err)
	}

	var users []models.User
	if err := engine.Table(&models.User{}).OrderBy("id asc").Find(&users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Account != "bob" || users[1].Account != "bob@a.com" {
		t.Fatalf("users = %+v, want the colliding row left alone", users)
	}
}
