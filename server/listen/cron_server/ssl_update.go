package cron_server

import (
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/services/setup/ssl"
	"github.com/Jinnrry/pmail/signal"
	log "github.com/sirupsen/logrus"
)

var expiredTime time.Time

func Start() {
	for !config.IsInit {
		time.Sleep(10 * time.Second)
	}
	go sslCheck()
}

// sslCheck periodically checks if certificates on disk have been renewed (e.g. by reverse proxy),
// and reloads mail listeners cleanly when updated.
func sslCheck() {
	_, expiredTime, _, _ = ssl.CheckSSLCrtInfo()

	for {
		time.Sleep(12 * time.Hour)
		_, newExpTime, _, err := ssl.CheckSSLCrtInfo()
		if err != nil {
			log.Debugf("Certificate check: %v", err)
			continue
		}
		if !newExpTime.IsZero() && !expiredTime.IsZero() && newExpTime != expiredTime {
			expiredTime = newExpTime
			log.Infoln("SSL certificates updated on disk, reloading listeners...")
			signal.RestartChan <- true
		}
	}
}
