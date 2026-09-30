package auth

import (
	"strings"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/services/account"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
)

// HasAuth 检查当前用户是否有某个邮件的auth
func HasAuth(ctx *context.Context, email *models.Email) bool {
	if ctx.IsAdmin {
		return true
	}
	var ue []models.UserEmail
	err := db.Instance.Table(&models.UserEmail{}).Where("email_id = ? and user_id = ?", email.Id, ctx.UserID).Find(&ue)
	if err != nil {
		log.Errorf("Error while checking user: %v", err)
		return false
	}

	return len(ue) != 0
}

// CanSendAs reports whether the authenticated session may send mail with the
// given From address. A mailbox always may send as itself. To keep the
// multi-domain setup working, the same local part on another served domain is
// also allowed - but only while no mailbox actually owns that address, so a
// user can never impersonate another domain-scoped account.
func CanSendAs(ctx *context.Context, address string) bool {
	if ctx == nil || address == "" {
		return false
	}
	if ctx.IsAdmin {
		return true
	}

	local, domain := account.Split(address)
	if local == "" || domain == "" || !config.Get().HasDomain(domain) {
		return false
	}
	if strings.EqualFold(address, ctx.UserAccount) {
		return true
	}
	if local != account.LocalPart(ctx.UserAccount) {
		return false
	}

	exists, err := db.Instance.Table(&models.User{}).Where("LOWER(account)=?", strings.ToLower(address)).Exist()
	if err != nil {
		log.WithContext(ctx).Errorf("Error while checking sender address: %v", err)
		return false
	}
	return !exists
}
