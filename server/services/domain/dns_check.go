package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Jinnrry/pmail/config"
)

// DNSCheckResult holds the live verification status of a domain's email DNS records.
type DNSCheckResult struct {
	Domain      string   `json:"domain"`
	Resolves    bool     `json:"resolves"`
	IPs         []string `json:"ips"`
	MXValid     bool     `json:"mx_valid"`
	MXRecords   []string `json:"mx_records"`
	SPFValid    bool     `json:"spf_valid"`
	SPFRecord   string   `json:"spf_record"`
	DKIMValid   bool     `json:"dkim_valid"`
	DKIMRecord  string   `json:"dkim_record"`
	DMARCValid  bool     `json:"dmarc_valid"`
	DMARCRecord string   `json:"dmarc_record"`
}

// publicResolver queries public resolvers (Cloudflare and Google DNS over IPv6 and IPv4)
// to avoid local/NAT64 negative cache delays when verifying DNS records in real-time.
var publicResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 3 * time.Second}
		servers := []string{
			"[2606:4700:4700::1111]:53", // Cloudflare DNS IPv6 (fastest update)
			"[2001:4860:4860::8888]:53", // Google DNS IPv6
			"1.1.1.1:53",
			"8.8.8.8:53",
		}
		for _, s := range servers {
			conn, err := d.DialContext(ctx, "udp", s)
			if err == nil {
				return conn, nil
			}
		}
		return d.DialContext(ctx, network, address)
	},
}

// CheckDNS performs live DNS queries for MX, SPF, DKIM, and DMARC records.
func CheckDNS(name string) (DNSCheckResult, error) {
	name, err := Normalize(name)
	if err != nil {
		return DNSCheckResult{}, err
	}

	result := DNSCheckResult{
		Domain:    name,
		IPs:       []string{},
		MXRecords: []string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 1. Check A / AAAA resolution
	ips, err := publicResolver.LookupHost(ctx, name)
	if err != nil || len(ips) == 0 {
		ips, _ = net.LookupHost(name)
	}
	if len(ips) > 0 {
		result.Resolves = true
		result.IPs = ips
	}

	// 2. Check MX Records
	mxRecords, err := publicResolver.LookupMX(ctx, name)
	if err != nil || len(mxRecords) == 0 {
		mxRecords, _ = net.LookupMX(name)
	}
	if len(mxRecords) > 0 {
		result.MXValid = true
		for _, mx := range mxRecords {
			result.MXRecords = append(result.MXRecords, fmt.Sprintf("%s (pri %d)", mx.Host, mx.Pref))
		}
	}

	// 3. Check SPF Record (TXT record containing v=spf1)
	txts, err := publicResolver.LookupTXT(ctx, name)
	if err != nil || len(txts) == 0 {
		txts, _ = net.LookupTXT(name)
	}
	for _, txt := range txts {
		if strings.Contains(txt, "v=spf1") {
			result.SPFValid = true
			result.SPFRecord = txt
			break
		}
	}

	// 4. Check DKIM Record
	d, ok := config.Get().FindDomain(name)
	selector := config.DefaultDKIMSelector
	if ok && d.DKIMSelector != "" {
		selector = d.DKIMSelector
	}
	dkimHost := fmt.Sprintf("%s._domainkey.%s", selector, name)
	dkimTxts, err := publicResolver.LookupTXT(ctx, dkimHost)
	if err != nil || len(dkimTxts) == 0 {
		dkimTxts, _ = net.LookupTXT(dkimHost)
	}
	for _, txt := range dkimTxts {
		if strings.Contains(txt, "v=DKIM1") || strings.Contains(txt, "p=") {
			result.DKIMValid = true
			result.DKIMRecord = txt
			break
		}
	}

	// 5. Check DMARC Record
	dmarcHost := fmt.Sprintf("_dmarc.%s", name)
	dmarcTxts, err := publicResolver.LookupTXT(ctx, dmarcHost)
	if err != nil || len(dmarcTxts) == 0 {
		dmarcTxts, _ = net.LookupTXT(dmarcHost)
	}
	for _, txt := range dmarcTxts {
		if strings.Contains(txt, "v=DMARC1") {
			result.DMARCValid = true
			result.DMARCRecord = txt
			break
		}
	}

	return result, nil
}

// CloudflareSyncRequest represents the parameters to sync records via Cloudflare API.
type CloudflareSyncRequest struct {
	Token  string `json:"token"`
	Email  string `json:"email,omitempty"`
	Domain string `json:"domain"`
	ZoneID string `json:"zone_id,omitempty"`
}

type cfErrorDetail struct {
	Code       int             `json:"code"`
	Message    string          `json:"message"`
	ErrorChain []cfErrorDetail `json:"error_chain,omitempty"`
}

type cfZoneResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Errors []cfErrorDetail `json:"errors"`
}

type cfDNSRecordPayload struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Proxied  bool   `json:"proxied"`
}

type cfDNSListResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		Content  string `json:"content"`
		Priority int    `json:"priority"`
		Proxied  bool   `json:"proxied"`
	} `json:"result"`
	Errors []cfErrorDetail `json:"errors"`
}

type cfDNSMutationResponse struct {
	Success bool `json:"success"`
	Result  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Errors []cfErrorDetail `json:"errors"`
}

type cfAuth struct {
	IsToken bool
	Token   string
	Email   string
	Key     string
}

func parseCloudflareCredentials(token, email string) (cfAuth, error) {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, "\"'`")
	email = strings.TrimSpace(email)

	// Remove leading "Bearer " or "bearer " if user pasted with prefix
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	// Check if user entered "email:key" in the token field
	if strings.Contains(token, ":") {
		parts := strings.SplitN(token, ":", 2)
		pEmail := strings.TrimSpace(parts[0])
		pKey := strings.TrimSpace(parts[1])
		if strings.Contains(pEmail, "@") {
			email = pEmail
			token = pKey
		}
	}

	// If email is provided, authenticate with Global API Key
	if email != "" {
		if token == "" {
			return cfAuth{}, fmt.Errorf("Cloudflare API Key is required when Email is specified")
		}
		return cfAuth{
			IsToken: false,
			Email:   email,
			Key:     token,
		}, nil
	}

	if token == "" {
		return cfAuth{}, fmt.Errorf("Cloudflare API Token is required")
	}

	// Check if user entered a 37-char hex string (which is a Global API Key) without email
	if len(token) == 37 && isHexString(token) {
		return cfAuth{}, fmt.Errorf("the credential entered is a Cloudflare Global API Key (37 hex characters). Global API Keys require your Cloudflare Account Email. Please provide your Email in the field, OR create an API Token at dash.cloudflare.com/profile/api-tokens (Create Token -> Edit zone DNS)")
	}

	return cfAuth{
		IsToken: true,
		Token:   token,
	}, nil
}

func isHexString(s string) bool {
	for _, c := range strings.ToLower(s) {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (a cfAuth) Apply(req *http.Request) {
	if a.IsToken {
		req.Header.Set("Authorization", "Bearer "+a.Token)
	} else {
		req.Header.Set("X-Auth-Email", a.Email)
		req.Header.Set("X-Auth-Key", a.Key)
	}
	req.Header.Set("Accept", "application/json")
}

func formatCfErrors(errors []cfErrorDetail) string {
	var msgs []string
	for _, e := range errors {
		msg := e.Message
		if len(e.ErrorChain) > 0 {
			var chainMsgs []string
			for _, sub := range e.ErrorChain {
				chainMsgs = append(chainMsgs, sub.Message)
			}
			msg = fmt.Sprintf("%s: %s", msg, strings.Join(chainMsgs, ", "))
		}
		if e.Code == 6003 {
			msg += " (Tip: make sure you use an API Token with 'Bearer', or provide your Cloudflare account email if using a Global API Key)"
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 {
		return "unknown Cloudflare API error"
	}
	return strings.Join(msgs, "; ")
}

func findCloudflareZoneID(client *http.Client, auth cfAuth, domainName string) (string, error) {
	// 1. Try direct apex or parent match
	lookupDomain := domainName
	parts := strings.Split(domainName, ".")
	if len(parts) > 2 {
		lookupDomain = strings.Join(parts[len(parts)-2:], ".")
	}

	zoneURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", lookupDomain)
	httpReq, _ := http.NewRequest("GET", zoneURL, nil)
	auth.Apply(httpReq)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to query Cloudflare zones: %w", err)
	}
	defer resp.Body.Close()

	var zoneResp cfZoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&zoneResp); err != nil {
		return "", fmt.Errorf("failed to parse Cloudflare zone response: %w", err)
	}

	if !zoneResp.Success {
		return "", fmt.Errorf("Cloudflare API error: %s", formatCfErrors(zoneResp.Errors))
	}

	if len(zoneResp.Result) > 0 {
		return zoneResp.Result[0].ID, nil
	}

	// 2. Fallback: list accessible zones and match by domain suffix
	listURL := "https://api.cloudflare.com/client/v4/zones?per_page=50"
	listReq, _ := http.NewRequest("GET", listURL, nil)
	auth.Apply(listReq)

	listResp, err := client.Do(listReq)
	if err == nil {
		defer listResp.Body.Close()
		var allZones cfZoneResponse
		if err := json.NewDecoder(listResp.Body).Decode(&allZones); err == nil && allZones.Success {
			for _, z := range allZones.Result {
				if domainName == z.Name || strings.HasSuffix(domainName, "."+z.Name) {
					return z.ID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find a Cloudflare zone matching '%s'. Verify your token has Zone.DNS:Edit permissions", domainName)
}

// ApplyCloudflare provisions and upserts required DNS records directly to Cloudflare.
func ApplyCloudflare(req CloudflareSyncRequest) ([]string, error) {
	auth, err := parseCloudflareCredentials(req.Token, req.Email)
	if err != nil {
		return nil, err
	}

	domainName, err := Normalize(req.Domain)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// Find Zone ID if not provided
	zoneID := strings.TrimSpace(req.ZoneID)
	if zoneID == "" {
		var err error
		zoneID, err = findCloudflareZoneID(client, auth, domainName)
		if err != nil {
			return nil, err
		}
	}

	d, ok := config.Get().FindDomain(domainName)
	if !ok {
		return nil, fmt.Errorf("domain %s is not configured on this mail server", domainName)
	}

	records := DNSRecords(d)
	var synced []string

	for _, rec := range records {
		var recordName string
		if rec.Host == "@" {
			recordName = domainName
		} else {
			recordName = fmt.Sprintf("%s.%s", rec.Host, domainName)
		}

		payload := cfDNSRecordPayload{
			Type:     rec.Type,
			Name:     recordName,
			Content:  rec.Value,
			TTL:      1, // Auto TTL in Cloudflare
			Proxied:  false,
		}
		if rec.Type == "MX" {
			payload.Priority = 10
		}

		// 1. Check if record already exists in Cloudflare (GET request without Content-Type)
		queryURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s", zoneID, rec.Type, recordName)
		qReq, _ := http.NewRequest("GET", queryURL, nil)
		auth.Apply(qReq)

		qResp, err := client.Do(qReq)
		if err != nil {
			return nil, fmt.Errorf("failed checking Cloudflare record %s: %w", recordName, err)
		}
		var listResp cfDNSListResponse
		_ = json.NewDecoder(qResp.Body).Decode(&listResp)
		qResp.Body.Close()

		if listResp.Success && len(listResp.Result) > 0 {
			existing := listResp.Result[0]
			// Check if already in sync
			contentMatches := strings.Trim(existing.Content, "\"") == strings.Trim(rec.Value, "\"")
			priorityMatches := rec.Type != "MX" || existing.Priority == payload.Priority
			if contentMatches && priorityMatches {
				synced = append(synced, fmt.Sprintf("%s %s (synced)", rec.Type, recordName))
				continue
			}

			// Value differs -> PUT to update (with Content-Type: application/json)
			bodyBytes, _ := json.Marshal(payload)
			putURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, existing.ID)
			putReq, _ := http.NewRequest("PUT", putURL, bytes.NewReader(bodyBytes))
			auth.Apply(putReq)
			putReq.Header.Set("Content-Type", "application/json")

			pResp, err := client.Do(putReq)
			if err != nil {
				return nil, fmt.Errorf("failed updating Cloudflare record %s: %w", recordName, err)
			}
			var mutResp cfDNSMutationResponse
			_ = json.NewDecoder(pResp.Body).Decode(&mutResp)
			pResp.Body.Close()

			if mutResp.Success {
				synced = append(synced, fmt.Sprintf("%s %s (updated)", rec.Type, recordName))
			} else {
				return nil, fmt.Errorf("failed updating %s %s: %s", rec.Type, recordName, formatCfErrors(mutResp.Errors))
			}
			continue
		}

		// Not found -> POST to create (with Content-Type: application/json)
		bodyBytes, _ := json.Marshal(payload)
		cfURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
		cfReq, _ := http.NewRequest("POST", cfURL, bytes.NewReader(bodyBytes))
		auth.Apply(cfReq)
		cfReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(cfReq)
		if err != nil {
			return nil, fmt.Errorf("failed creating Cloudflare record %s: %w", recordName, err)
		}
		var mutResp cfDNSMutationResponse
		_ = json.NewDecoder(resp.Body).Decode(&mutResp)
		resp.Body.Close()

		if mutResp.Success {
			synced = append(synced, fmt.Sprintf("%s %s (created)", rec.Type, recordName))
		} else {
			return nil, fmt.Errorf("failed creating %s %s: %s", rec.Type, recordName, formatCfErrors(mutResp.Errors))
		}
	}

	return synced, nil
}
