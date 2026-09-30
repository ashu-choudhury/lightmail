package main

import (
	"context"
	"net"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/listen/cron_server"
	"github.com/Jinnrry/pmail/res_init"
	log "github.com/sirupsen/logrus"
)

var (
	gitHash   string
	buildTime string
	goVersion string
	version   string
)

func initDNSResolver() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			conn, err := d.DialContext(ctx, network, address)
			if err == nil {
				return conn, nil
			}
			// Fallback list of reliable IPv6 DNS resolvers (DNS64 & Cloudflare/Google IPv6)
			fallbacks := []string{
				"[2a00:1098:2c::1]:53",      // NAT64.net DNS64
				"[2606:4700:4700::1111]:53", // Cloudflare DNS
				"[2001:4860:4860::8888]:53", // Google DNS
				"[2a01:4f8:c2c:123f::1]:53", // Hetzner DNS64
			}
			for _, fb := range fallbacks {
				if fbConn, fbErr := d.DialContext(ctx, "udp", fb); fbErr == nil {
					return fbConn, nil
				}
			}
			return nil, err
		},
	}
}

func main() {
	initDNSResolver()

	config.Init()

	if version == "" {
		version = "TestVersion"
	}

	log.Infoln("===================================================================")
	log.Infof("    Lightmail Mail Server Started")
	log.Infof("    Version:     %s", version)
	log.Infof("    Commit:      %s", gitHash)
	log.Infof("    Build Date:  %s", buildTime)
	log.Infof("    Go Version:  %s", goVersion)
	log.Infoln("===================================================================")

	// 定时任务启动
	go cron_server.Start()

	// 核心服务启动
	res_init.Init(version)

	log.Warnf("Server Stoped \n")

}
