package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/services/account"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/password"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"io"
	"math"
	"net/http"
	"strings"
)

type userCreateRequest struct {
	Id       int    `json:"id"`
	Account  string `json:"account"`
	Domain   string `json:"domain"`
	Username string `json:"username"`
	Password string `json:"password"`
	Disabled int    `json:"disabled"`
	IsAdmin  int    `json:"is_admin"`
	Gender   string `json:"gender"`
}

func CreateUser(ctx *context.Context, w http.ResponseWriter, req *http.Request) {

	if !ctx.IsAdmin {
		response.NewErrorResponse(response.NoAccessPrivileges, "No Access Privileges", "").FPrint(w)
		return
	}

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Errorf("%+v", err)
	}
	var reqData userCreateRequest
	err = json.Unmarshal(reqBytes, &reqData)
	if err != nil {
		log.Errorf("%+v", err)
	}

	if reqData.Username == "" || reqData.Password == "" || reqData.Account == "" {
		response.NewErrorResponse(response.ParamsError, "Params Error", "").FPrint(w)
		return
	}

	address, err := resolveNewAddress(reqData.Account, reqData.Domain)
	if err != nil {
		response.NewErrorResponse(response.ParamsError, err.Error(), err.Error()).FPrint(w)
		return
	}

	exists, err := db.Instance.Table(&models.User{}).Where("LOWER(account)=?", address).Exist()
	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}
	if exists {
		response.NewErrorResponse(response.ParamsError, "Account already exists", "Account already exists").FPrint(w)
		return
	}

	var user models.User
	user.Name = reqData.Username
	user.Password = password.Encode(reqData.Password)
	user.Account = address
	user.IsAdmin = reqData.IsAdmin
	user.Gender = reqData.Gender

	_, err = db.Instance.Insert(&user)
	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}

	response.NewSuccessResponse(user).FPrint(w)
}

// resolveNewAddress builds the mailbox address for a new account. account may be
// a full address or a bare local part; a bare one is completed with domain,
// which defaults to the primary domain.
func resolveNewAddress(accountName, domain string) (string, error) {
	local, embedded := account.Split(accountName)
	if embedded != "" {
		domain = embedded
	}
	if domain == "" {
		domain = config.Get().PrimaryDomain()
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !config.Get().HasDomain(domain) {
		return "", fmt.Errorf("%s is not a domain this server hosts", domain)
	}
	if err := account.ValidateLocal(local); err != nil {
		return "", err
	}
	return account.Build(local, domain), nil
}

type userListRequest struct {
	CurrentPage int `json:"current_page"`
	PageSize    int `json:"page_size"`
}

func UserList(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !ctx.IsAdmin {
		response.NewErrorResponse(response.NoAccessPrivileges, "No Access Privileges", "").FPrint(w)
		return
	}

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Errorf("%+v", err)
	}
	var reqData userListRequest
	err = json.Unmarshal(reqBytes, &reqData)
	if err != nil {
		log.Errorf("%+v", err)
	}

	offset := 0
	if reqData.CurrentPage >= 1 {
		offset = (reqData.CurrentPage - 1) * reqData.PageSize
	}

	if reqData.PageSize == 0 {
		reqData.PageSize = 15
	}

	var users []models.User

	totalNum, err := db.Instance.Table(&models.User{}).Limit(reqData.PageSize, offset).FindAndCount(&users)
	if err != nil {
		log.Errorf("%+v", err)
	}

	response.NewSuccessResponse(map[string]any{
		"current_page": reqData.CurrentPage,
		"total_page":   cast.ToInt(math.Ceil(cast.ToFloat64(totalNum) / cast.ToFloat64(reqData.PageSize))),
		"list":         users,
	}).FPrint(w)

}

func Info(ctx *context.Context, w http.ResponseWriter, req *http.Request) {

	// DomainNames 保证主域名排在最前面
	domains := config.Get().DomainNames()

	response.NewSuccessResponse(map[string]any{
		"account":  ctx.UserAccount,
		"domain":   account.DomainPart(ctx.UserAccount),
		"name":     ctx.UserName,
		"is_admin": ctx.IsAdmin,
		"domains":  domains,
	}).FPrint(w)
}

func EditUser(ctx *context.Context, w http.ResponseWriter, req *http.Request) {

	if !ctx.IsAdmin {
		response.NewErrorResponse(response.NoAccessPrivileges, "No Access Privileges", "").FPrint(w)
		return
	}

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Errorf("%+v", err)
	}
	var reqData userCreateRequest
	err = json.Unmarshal(reqBytes, &reqData)
	if err != nil {
		log.Errorf("%+v", err)
	}

	if reqData.Id == 0 && reqData.Account == "" {
		response.NewErrorResponse(response.ParamsError, "Params Error", "").FPrint(w)
		return
	}
	var user models.User
	if reqData.Id != 0 {
		_, err = db.Instance.Where("id=?", reqData.Id).Get(&user)
		if err != nil {
			log.Errorf("SQL Error: %+v", err)
		}
	} else {
		accountFilter, accountArgs := account.LoginPredicate(reqData.Account)
		_, err = db.Instance.Where(accountFilter, accountArgs...).Get(&user)
		if err != nil {
			log.Errorf("SQL Error: %+v", err)
		}
	}
	if user.ID == 0 {
		response.NewErrorResponse(response.ParamsError, "User not found", "").FPrint(w)
		return
	}
	if reqData.Username != "" && reqData.Username != user.Name {
		user.Name = reqData.Username
	}

	if reqData.Disabled != user.Disabled {
		user.Disabled = reqData.Disabled
	}
	if reqData.IsAdmin != user.IsAdmin {
		user.IsAdmin = reqData.IsAdmin
	}
	if reqData.Gender != "" && reqData.Gender != user.Gender {
		user.Gender = reqData.Gender
	}
	if reqData.Password != "" {
		user.Password = password.Encode(reqData.Password)
	}

	num, err := db.Instance.ID(user.ID).Cols("name", "password", "disabled", "is_admin", "gender").Update(&user)

	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}
	if num == 0 {
		response.NewErrorResponse(response.ServerError, "No Data Update", "").FPrint(w)
		return
	}

	response.NewSuccessResponse(user).FPrint(w)
}
