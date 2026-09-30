package http_server

import (
	"net/http"
	"testing"
)

func TestAdminDomainRoutesAreRegistered(t *testing.T) {
	mux := http.NewServeMux()
	router(mux)

	for _, path := range []string{"/api/domain/list", "/api/domain/add", "/api/domain/del", "/api/domain/dkim"} {
		req, err := http.NewRequest(http.MethodPost, path, nil)
		if err != nil {
			t.Fatal(err)
		}

		_, pattern := mux.Handler(req)
		if pattern != path {
			t.Fatalf("%s is not routed (matched %q)", path, pattern)
		}
	}
}
