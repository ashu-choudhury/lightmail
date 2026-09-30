package session

import (
	"time"

	"github.com/Jinnrry/pmail/db"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

var Instance *scs.SessionManager

func Init() {
	Instance = scs.New()
	Instance.Lifetime = 7 * 24 * time.Hour
	Instance.Store = sqlite3store.New(db.Instance.DB().DB)
}
