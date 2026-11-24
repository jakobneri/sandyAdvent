# sandyAdvent – siehe README.md
GO       ?= go
BIN      ?= bin/sandyadvent
GOOS     ?= $(shell $(GO) env GOOS)
GOARCH   ?= $(shell $(GO) env GOARCH)
# config.json (gitignored) falls vorhanden, sonst die Vorlage mit Dummy-Tokens/-Assets.
CONFIG   ?= $(if $(wildcard config.json),config.json,config.example.json)
BASE_URL ?=
QR_OUT   ?= out/qr-codes.pdf
SIZE_MM  ?= 40
QR_TEST  ?=

.PHONY: dev run build test qr tokens check fmt vet sync-brand example-assets clean

dev: ## Server im Dev-Modus (ADVENT_NOW, /dev/preview) auf http://localhost:8081
	ADVENT_DEV=1 ADVENT_CONFIG=$(CONFIG) $(GO) run ./cmd/sandyadvent serve

run: ## Server wie in Produktion (config.json)
	ADVENT_CONFIG=$(CONFIG) $(GO) run ./cmd/sandyadvent serve

build: ## statisches Binary nach bin/ (GOARCH=arm64 make build für den Pi)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/sandyadvent

test:
	$(GO) test ./...

check: ## config + Türchen-Config prüfen
	$(GO) run ./cmd/sandyadvent check -config $(CONFIG)

qr: ## QR-PDF: make qr BASE_URL=http://<Pi-IP>  (Testdruck mit localhost/Beispiel-Tokens: QR_TEST=1)
	$(GO) run ./tools/qrpdf -config $(CONFIG) $(if $(BASE_URL),-base-url $(BASE_URL)) $(if $(QR_TEST),-test) -size-mm $(SIZE_MM) -out $(QR_OUT)

tokens: ## Tokens für die 17 QR-Tage in config.json schreiben (vorhandene bleiben)
	@test -f config.json || { cp config.example.json config.json && \
	  echo "config.json aus der Vorlage erstellt – base_url, doors_config, assets_dir und schatzogram.welcome_url für den Pi anpassen!"; }
	$(GO) run ./cmd/sandyadvent gen-tokens -write -config config.json

fmt:
	gofmt -w cmd internal tools web

vet:
	$(GO) vet ./...

sync-brand: ## Kopien der geteilten Brand-Dateien aktualisieren (Test prüft Gleichheit)
	cp shared/schatzogram-brand/tokens.css web/static/brand/tokens.css
	cp shared/schatzogram-brand/logo-mark.svg web/brand/logo-mark.svg

example-assets: ## Dummy-Assets in doors/assets-example/ neu erzeugen (ffmpeg)
	scripts/gen-example-assets.sh

clean:
	rm -rf bin out
