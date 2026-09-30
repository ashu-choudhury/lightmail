build: build_fe build_server package

clean:
	rm -rf output

build_fe:
	cd fe-svelte && bun run build
	rm -rf server/listen/http_server/dist
	mkdir -p server/listen/http_server/dist
	cp -rf fe-svelte/dist/* server/listen/http_server/dist/

build_server:
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o lightmail_linux_amd64 main.go
	cd server && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o lightmail_windows_amd64.exe main.go
	cd server && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o lightmail_mac_amd64 main.go
	cd server && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o lightmail_mac_arm64 main.go

package: clean
	mkdir -p output
	mv server/lightmail* output/
	mkdir -p output/config
	cp -r server/config/dkim output/config/ 2>/dev/null || true
	cp -r server/config/ssl output/config/ 2>/dev/null || true
	cp -r server/config/config.json output/config/ 2>/dev/null || true
	cp README.md output/ 2>/dev/null || true

test:
	export setup_port=17888 && cd server && export PMail_ROOT=$(CURDIR)/server/ && go test -v -p 1 ./...