package auth

import (
	"path/filepath"
	"testing"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func withSendAsFixture(t *testing.T) *xorm.Engine {
	t.Helper()

	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "send_as.db"))
	if err != nil {
		t.Fatal(err)
	}
	engine.SetMaxOpenConns(1)
	if err = engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}

	oldDB := db.Instance
	oldConfig := config.Get().Clone()
	db.Instance = engine
	config.Set(&config.Config{
		Domain:  "a.com",
		Domains: []config.Domain{{Name: "a.com"}, {Name: "b.com"}},
	})
	t.Cleanup(func() {
		db.Instance = oldDB
		config.Set(oldConfig)
		_ = engine.Close()
	})
	return engine
}

func TestCanSendAsOwnMailbox(t *testing.T) {
	withSendAsFixture(t)

	ctx := &context.Context{UserID: 1, UserAccount: "bob@a.com"}
	if !CanSendAs(ctx, "BOB@a.com") {
		t.Fatal("a mailbox must be able to send as itself regardless of case")
	}
	if CanSendAs(ctx, "bob@unserved.com") {
		t.Fatal("a domain this server does not host must be rejected")
	}
	if CanSendAs(ctx, "alice@a.com") {
		t.Fatal("a different local part must not be accepted")
	}
}

func TestCanSendAsUnclaimedAliasDomain(t *testing.T) {
	withSendAsFixture(t)

	ctx := &context.Context{UserID: 1, UserAccount: "bob@a.com"}
	if !CanSendAs(ctx, "bob@b.com") {
		t.Fatal("an unclaimed alias on a served domain keeps the multi-domain setup working")
	}
}

func TestCanSendAsRejectsAnotherMailbox(t *testing.T) {
	engine := withSendAsFixture(t)
	if _, err := engine.Insert(&models.User{Account: "bob@b.com", Name: "other", Password: "x"}); err != nil {
		t.Fatal(err)
	}

	ctx := &context.Context{UserID: 1, UserAccount: "bob@a.com"}
	if CanSendAs(ctx, "bob@b.com") {
		t.Fatal("a mailbox owned by another account must never be impersonated")
	}
}

func TestCanSendAsAdminOverride(t *testing.T) {
	engine := withSendAsFixture(t)
	if _, err := engine.Insert(&models.User{Account: "alice@a.com", Name: "alice", Password: "x"}); err != nil {
		t.Fatal(err)
	}

	ctx := &context.Context{UserID: 1, UserAccount: "admin@a.com", IsAdmin: true}
	if !CanSendAs(ctx, "alice@a.com") {
		t.Fatal("an administrator may send as any served mailbox")
	}
}

func TestCanSendAsEmptyInput(t *testing.T) {
	withSendAsFixture(t)

	if CanSendAs(nil, "bob@a.com") {
		t.Fatal("a missing session must not be able to send")
	}
	if CanSendAs(&context.Context{UserAccount: "bob@a.com"}, "") {
		t.Fatal("an empty sender must not be accepted")
	}
}
