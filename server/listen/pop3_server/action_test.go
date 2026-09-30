package pop3_server

import (
	"bytes"
	"fmt"
	"github.com/Jinnrry/gopop"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/emersion/go-message/mail"
	"io"
	"path/filepath"
	"testing"
)

func Test_action_Retr(t *testing.T) {
	config.Init()
	cfg := config.Get().Clone()
	cfg.DbType = config.DBTypeSQLite
	cfg.DbDSN = filepath.Join(config.ROOT_PATH, "config", "pmail_temp.db")
	config.Set(cfg)
	db.Init("")

	a := action{}
	session := &gopop.Session{
		Ctx: &context.Context{
			UserID: 1,
		},
	}
	got, got1, err := a.Retr(session, 301)

	_, _, _ = got, got1, err
}

func Test_email(t *testing.T) {
	var b bytes.Buffer

	// Create our mail header
	var h mail.Header

	// Create a new mail writer
	mw, _ := mail.CreateWriter(&b, h)

	// Create a text part
	tw, _ := mw.CreateInline()

	var html mail.InlineHeader

	html.Header.Set("Content-Transfer-Encoding", "base64")
	w, _ := tw.CreatePart(html)

	io.WriteString(w, "=")

	w.Close()

	tw.Close()

	fmt.Printf("%s", b.String())

}
