package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/domain"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
)

type domainRequest struct {
	Domain string `json:"domain"`
	Rotate bool   `json:"rotate"`
}

// DomainList 返回所有已配置的收信域名及其所需的DNS记录
func DomainList(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	response.NewSuccessResponse(domain.List()).FPrint(w)
}

// DomainAdd 新增收信域名，自动生成DKIM密钥并热加载配置
func DomainAdd(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	reqData, ok := parseDomainRequest(ctx, w, req)
	if !ok {
		return
	}

	added, err := domain.Add(reqData.Domain)
	if err != nil {
		log.WithContext(ctx).Warnf("添加域名失败 %s: %v", reqData.Domain, err)
		response.NewErrorResponse(response.ParamsError, err.Error(), err.Error()).FPrint(w)
		return
	}

	response.NewSuccessResponse(added).FPrint(w)
}

// DomainDelete 删除收信域名
func DomainDelete(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	reqData, ok := parseDomainRequest(ctx, w, req)
	if !ok {
		return
	}

	if err := domain.Delete(reqData.Domain); err != nil {
		log.WithContext(ctx).Warnf("删除域名失败 %s: %v", reqData.Domain, err)
		response.NewErrorResponse(response.ParamsError, err.Error(), err.Error()).FPrint(w)
		return
	}

	response.NewSuccessResponse(domain.List()).FPrint(w)
}

// DomainDkim 为域名生成DKIM密钥，rotate为true时轮换已有密钥
func DomainDkim(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	reqData, ok := parseDomainRequest(ctx, w, req)
	if !ok {
		return
	}

	updated, err := domain.GenerateDKIM(reqData.Domain, reqData.Rotate)
	if err != nil {
		log.WithContext(ctx).Warnf("Generate DKIM failed %s: %v", reqData.Domain, err)
		response.NewErrorResponse(response.ServerError, err.Error(), err.Error()).FPrint(w)
		return
	}

	response.NewSuccessResponse(updated).FPrint(w)
}

// DomainCheck performs live DNS lookup verification for a domain's email records
func DomainCheck(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	domainName := req.URL.Query().Get("domain")
	if domainName == "" {
		reqData, ok := parseDomainRequest(ctx, w, req)
		if ok {
			domainName = reqData.Domain
		}
	}

	if domainName == "" {
		response.NewErrorResponse(response.ParamsError, "Domain is required", "").FPrint(w)
		return
	}

	checkResult, err := domain.CheckDNS(domainName)
	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}

	response.NewSuccessResponse(checkResult).FPrint(w)
}

type cloudflareSettingsResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

type cloudflareSettingsRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// CloudflareGetSettings returns the saved Cloudflare credentials from SQLite.
func CloudflareGetSettings(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}
	response.NewSuccessResponse(cloudflareSettingsResponse{
		Token: db.GetSetting("cloudflare_token"),
		Email: db.GetSetting("cloudflare_email"),
	}).FPrint(w)
}

// CloudflareSaveSettings saves Cloudflare credentials to SQLite permanently.
func CloudflareSaveSettings(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}
	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		response.NewErrorResponse(response.ParamsError, "Invalid body", "").FPrint(w)
		return
	}
	var setReq cloudflareSettingsRequest
	if err := json.Unmarshal(reqBytes, &setReq); err != nil {
		response.NewErrorResponse(response.ParamsError, "Invalid JSON", "").FPrint(w)
		return
	}
	if err := db.SetSetting("cloudflare_token", strings.TrimSpace(setReq.Token)); err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}
	if err := db.SetSetting("cloudflare_email", strings.TrimSpace(setReq.Email)); err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}
	response.NewSuccessResponse("saved").FPrint(w)
}

// DomainCloudflare provisions DNS records via Cloudflare API v4
func DomainCloudflare(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if !requireAdmin(ctx, w) {
		return
	}

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		response.NewErrorResponse(response.ParamsError, "Invalid body", "").FPrint(w)
		return
	}

	var syncReq domain.CloudflareSyncRequest
	if err := json.Unmarshal(reqBytes, &syncReq); err != nil {
		response.NewErrorResponse(response.ParamsError, "Invalid JSON", "").FPrint(w)
		return
	}

	// Auto-fill from SQLite if not passed in this request
	if strings.TrimSpace(syncReq.Token) == "" {
		syncReq.Token = db.GetSetting("cloudflare_token")
	}
	if strings.TrimSpace(syncReq.Email) == "" {
		syncReq.Email = db.GetSetting("cloudflare_email")
	}

	// Auto-save any newly entered credentials permanently to SQLite
	if strings.TrimSpace(syncReq.Token) != "" {
		_ = db.SetSetting("cloudflare_token", strings.TrimSpace(syncReq.Token))
	}
	if strings.TrimSpace(syncReq.Email) != "" {
		_ = db.SetSetting("cloudflare_email", strings.TrimSpace(syncReq.Email))
	}

	created, err := domain.ApplyCloudflare(syncReq)
	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}

	response.NewSuccessResponse(map[string]any{
		"created": created,
		"count":   len(created),
	}).FPrint(w)
}

func parseDomainRequest(ctx *context.Context, w http.ResponseWriter, req *http.Request) (domainRequest, bool) {
	var reqData domainRequest

	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.WithContext(ctx).Errorf("%+v", err)
	}
	if err := json.Unmarshal(reqBytes, &reqData); err != nil {
		log.WithContext(ctx).Errorf("%+v", err)
	}

	if reqData.Domain == "" {
		response.NewErrorResponse(response.ParamsError, "Params Error", "Params Error").FPrint(w)
		return reqData, false
	}
	return reqData, true
}

func requireAdmin(ctx *context.Context, w http.ResponseWriter) bool {
	if !ctx.IsAdmin {
		response.NewErrorResponse(response.NoAccessPrivileges, "No Access Privileges", "").FPrint(w)
		return false
	}
	return true
}
