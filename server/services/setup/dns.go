package setup

import (
	"strings"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/services/domain"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/ip"
)

type DNSItem struct {
	Type  string `json:"type"`
	Host  string `json:"host"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Tips  string `json:"tips"`
}

func GetDNSSettings(ctx *context.Context) (map[string][]*DNSItem, error) {
	configData, err := config.ReadConfig()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	ret := make(map[string][]*DNSItem)
	ipValue := ip.GetIp()
	ipTips := i18n.GetText(ctx.Lang, "ip_taps")
	host := strings.ReplaceAll(configData.WebDomain, "."+configData.Domain, "")

	for _, d := range configData.Domains {
		items := []*DNSItem{
			{Type: "A", Host: host, Value: ipValue, TTL: 3600, Tips: ipTips},
			{Type: "A", Host: "smtp", Value: ipValue, TTL: 3600, Tips: ipTips},
			{Type: "A", Host: "imap", Value: ipValue, TTL: 3600, Tips: ipTips},
			{Type: "A", Host: "pop", Value: ipValue, TTL: 3600, Tips: ipTips},
			{Type: "A", Host: "@", Value: ipValue, TTL: 3600, Tips: ipTips},
		}
		// 收信、SPF、DKIM、DMARC 记录按域名生成，新增域名时设置向导与后台保持一致
		for _, record := range domain.DNSRecords(d) {
			items = append(items, &DNSItem{Type: record.Type, Host: record.Host, Value: record.Value, TTL: record.TTL})
		}
		ret[d.Name] = items
	}

	return ret, nil
}
