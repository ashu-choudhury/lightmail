package http_server

import (
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/controllers"
	"github.com/Jinnrry/pmail/controllers/email"
	"github.com/Jinnrry/pmail/session"
	log "github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// 这个服务是为了拦截http请求转发到https
var httpServer *http.Server

func HttpStop() {
	if httpServer != nil {
		httpServer.Close()
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) WriteHeader(status int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		ext := strings.ToLower(r.URL.Path)
		if strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") || strings.HasSuffix(ext, ".gif") || strings.HasSuffix(ext, ".zip") || strings.HasSuffix(ext, ".gz") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		next.ServeHTTP(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

func router(mux *http.ServeMux) {
	fe, err := fs.Sub(local, "dist")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(fe)))
	// 挑战请求类似这样 /.well-known/acme-challenge/QPyMAyaWw9s5JvV1oruyqWHG7OqkHMJEHPoUz2046KM
	mux.HandleFunc("/.well-known/", controllers.AcmeChallenge)
	mux.HandleFunc("/api/ping", controllers.Ping)
	mux.HandleFunc("/api/login", contextIterceptor(controllers.Login))
	mux.HandleFunc("/api/logout", contextIterceptor(controllers.Logout))
	mux.HandleFunc("/api/group", contextIterceptor(controllers.GetUserGroup))
	mux.HandleFunc("/api/group/list", contextIterceptor(controllers.GetUserGroupList))
	mux.HandleFunc("/api/group/add", contextIterceptor(controllers.AddGroup))
	mux.HandleFunc("/api/group/del", contextIterceptor(controllers.DelGroup))
	mux.HandleFunc("/api/email/list", contextIterceptor(email.EmailList))
	mux.HandleFunc("/api/email/del", contextIterceptor(email.EmailDelete))
	mux.HandleFunc("/api/email/read", contextIterceptor(email.MarkRead))
	mux.HandleFunc("/api/email/detail", contextIterceptor(email.EmailDetail))
	mux.HandleFunc("/api/email/move", contextIterceptor(email.Move))
	mux.HandleFunc("/api/email/send", contextIterceptor(email.Send))
	mux.HandleFunc("/api/settings/modify_password", contextIterceptor(controllers.ModifyPassword))
	mux.HandleFunc("/api/rule/get", contextIterceptor(controllers.GetRule))
	mux.HandleFunc("/api/rule/add", contextIterceptor(controllers.UpsertRule))
	mux.HandleFunc("/api/rule/update", contextIterceptor(controllers.UpsertRule))
	mux.HandleFunc("/api/rule/del", contextIterceptor(controllers.DelRule))
	mux.HandleFunc("/attachments/", contextIterceptor(controllers.GetAttachments))
	mux.HandleFunc("/attachments/download/", contextIterceptor(controllers.Download))
	mux.HandleFunc("/api/user/create", contextIterceptor(controllers.CreateUser))
	mux.HandleFunc("/api/user/edit", contextIterceptor(controllers.EditUser))
	mux.HandleFunc("/api/user/info", contextIterceptor(controllers.Info))
	mux.HandleFunc("/api/user/list", contextIterceptor(controllers.UserList))
	mux.HandleFunc("/api/domain/list", contextIterceptor(controllers.DomainList))
	mux.HandleFunc("/api/domain/add", contextIterceptor(controllers.DomainAdd))
	mux.HandleFunc("/api/domain/del", contextIterceptor(controllers.DomainDelete))
	mux.HandleFunc("/api/domain/dkim", contextIterceptor(controllers.DomainDkim))
	mux.HandleFunc("/api/domain/check", contextIterceptor(controllers.DomainCheck))
	mux.HandleFunc("/api/domain/cloudflare", contextIterceptor(controllers.DomainCloudflare))
	mux.HandleFunc("/api/settings/cloudflare/get", contextIterceptor(controllers.CloudflareGetSettings))
	mux.HandleFunc("/api/settings/cloudflare/save", contextIterceptor(controllers.CloudflareSaveSettings))
	mux.HandleFunc("/api/plugin/settings/", contextIterceptor(controllers.SettingsHtml))
	mux.HandleFunc("/api/plugin/list", contextIterceptor(controllers.GetPluginList))
}

func HttpStart() {
	mux := http.NewServeMux()

	HttpPort := 80
	if config.Get().HttpPort > 0 {
		HttpPort = config.Get().HttpPort
	}

	if config.Get().HttpsEnabled != 2 {
		// 在重定向模式下，也必须显式处理 ACME 挑战，避免跳转导致验证失败
		mux.HandleFunc("/.well-known/", controllers.AcmeChallenge)
		mux.HandleFunc("/api/ping", controllers.Ping)
		mux.HandleFunc("/", controllers.Interceptor)
		httpServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", HttpPort),
			Handler:      gzipMiddleware(mux),
			ReadTimeout:  time.Second * 90,
			WriteTimeout: time.Second * 90,
		}
	} else {

		router(mux)

		fe, err := fs.Sub(local, "dist")
		if err != nil {
			panic(err)
		}
		fileServer := http.FileServer(http.FS(fe))

		sessionHandler := session.Instance.LoadAndSave(mux)

		mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path
			if strings.HasPrefix(p, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				fileServer.ServeHTTP(w, r)
				return
			}
			if p == "/" || p == "/index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				fileServer.ServeHTTP(w, r)
				return
			}
			if p == "/vite.svg" || p == "/favicon.ico" {
				w.Header().Set("Cache-Control", "public, max-age=86400")
				fileServer.ServeHTTP(w, r)
				return
			}
			sessionHandler.ServeHTTP(w, r)
		})

		log.Infof("HttpServer Start On Port :%d", HttpPort)
		httpServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", HttpPort),
			Handler:      gzipMiddleware(mainHandler),
			ReadTimeout:  time.Second * 90,
			WriteTimeout: time.Second * 90,
		}
	}

	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
