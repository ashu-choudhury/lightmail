package setup

import (
	"strings"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/errors"
)

func GetDomainSettings() (string, string, []string, error) {
	configData, err := config.ReadConfig()
	if err != nil {
		return "", "", []string{}, errors.Wrap(err)
	}

	return configData.Domain, configData.WebDomain, array.Difference(configData.DomainNames(), []string{configData.Domain}), nil
}

func SetDomainSettings(smtpDomain, webDomain, multiDomains string) error {
	configData, err := config.ReadConfig()
	if err != nil {
		return errors.Wrap(err)
	}

	if smtpDomain == "" {
		return errors.New("domain must not empty!")
	}

	if webDomain == "" {
		return errors.New("web domain must not empty!")
	}

	var domains []config.Domain
	for _, name := range strings.Split(multiDomains, ",") {
		if name = strings.TrimSpace(name); name != "" {
			domains = append(domains, config.NewDomain(name))
		}
	}
	// 主域名必须排在第一位
	domains = append(domains, config.NewDomain(smtpDomain))

	configData.Domain = strings.ToLower(strings.TrimSpace(smtpDomain))
	configData.WebDomain = webDomain
	configData.Domains = domains

	// 检查域名是否指向本机 todo

	err = config.WriteConfig(configData)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}
