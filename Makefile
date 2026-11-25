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

.PHONY: dev run build test test-js qr tokens check fmt vet sync-brand example-assets crossword wrapped-stats waparse-testdata clean

dev: ## Server im Dev-Modus (ADVENT_NOW, /dev/preview) auf http://localhost:8081
	ADVENT_DEV=1 ADVENT_CONFIG=$(CONFIG) $(GO) run ./cmd/sandyadvent serve

run: ## Server wie in Produktion (config.json)
	ADVENT_CONFIG=$(CONFIG) $(GO) run ./cmd/sandyadvent serve

build: ## statisches Binary nach bin/ (GOARCH=arm64 make build für den Pi)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/sandyadvent

test: ## Go-Tests + JS-Logik-Tests (node --test, übersprungen ohne Node)
	$(GO) test ./...
	@if command -v node >/dev/null 2>&1; then node --test tests/js/*.test.mjs; \
	else echo "node nicht gefunden – JS-Tests (tests/js) übersprungen"; fi

test-js: ## nur die JS-Logik-Tests (Wordle, Puzzle, Kreuzworträtsel, Wrapped)
	node --test tests/js/*.test.mjs

check: ## config + Türchen-Config prüfen (die Vorlage mit EXAMPLE-Tokens im Dev-Modus)
	$(if $(filter config.example.json,$(CONFIG)),ADVENT_DEV=1 )$(GO) run ./cmd/sandyadvent check -config $(CONFIG)

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

# Kreuzworträtsel (Türchen 16): Wortliste "WORT;Hinweis" → Raster-JSON (siehe README)
SEED     ?=
ATTEMPTS ?=
MAX_SIZE ?=
crossword: ## make crossword IN=private/woerter.txt OUT=doors/assets/kreuzwort.json [SEED=42] [MAX_SIZE=13] [ATTEMPTS=1000]
	@test -n "$(IN)" || { echo "Aufruf: make crossword IN=<private/woerter.txt> [OUT=doors/assets/kreuzwort.json] [SEED=…]"; exit 2; }
	$(GO) run ./tools/crossword -in "$(IN)" -out "$(or $(OUT),doors/assets/kreuzwort.json)" -preview \
	  $(if $(SEED),-seed $(SEED)) $(if $(ATTEMPTS),-attempts $(ATTEMPTS)) $(if $(MAX_SIZE),-max-size $(MAX_SIZE))

# Schatzo Wrapped (Türchen 22): WhatsApp-Export(e) → stats.json. Mehrere Dateien/Aliase mit ';' trennen.
# Der Roh-Export bleibt lokal (gitignored) – nur stats.json kommt nach doors/assets/.
SINCE ?=
UNTIL ?=
ALIAS ?=
wrapped-stats: ## make wrapped-stats IN='private/WhatsApp-Chat mit Sandy.txt' ALIAS='Sandy 🌸=Sandy;Jakob N.=Jakob' SINCE=JJJJ-MM-TT [OUT=doors/assets/stats.json]
	@test -n "$(IN)" || { echo "Aufruf: make wrapped-stats IN=<private/export.txt> [OUT=doors/assets/stats.json] [ALIAS='Export-Name=Sandy;…'] [SINCE=JJJJ-MM-TT]"; exit 2; }
	@set -e; set --; old_ifs=$$IFS; IFS=';'; \
	for f in $$WA_IN; do [ -n "$$f" ] && set -- "$$@" -in "$$f"; done; \
	for a in $$WA_ALIAS; do [ -n "$$a" ] && set -- "$$@" -alias "$$a"; done; \
	IFS=$$old_ifs; \
	$(GO) run ./tools/waparse "$$@" -out "$(or $(OUT),doors/assets/stats.json)" $(if $(SINCE),-since $(SINCE)) $(if $(UNTIL),-until $(UNTIL))
wrapped-stats: export WA_IN = $(IN)
wrapped-stats: export WA_ALIAS = $(ALIAS)

waparse-testdata: ## synthetische Beispiel-Exporte + Beispiel-stats.json neu erzeugen
	$(GO) run ./tools/waparse/gen
	$(GO) run ./tools/waparse -in tools/waparse/testdata/android.txt -alias 'Max Mustermann=Jakob' -alias 'Erika 🌸=Sandy' \
	  -since 2024-02-14 -asof 2026-12-22 -out doors/assets-example/stats.json

sync-brand: ## Kopien der geteilten Brand-Dateien aktualisieren (Test prüft Gleichheit)
	cp shared/schatzogram-brand/tokens.css web/static/brand/tokens.css
	cp shared/schatzogram-brand/logo-mark.svg web/brand/logo-mark.svg

example-assets: ## Dummy-Assets in doors/assets-example/ neu erzeugen (ffmpeg)
	scripts/gen-example-assets.sh

clean:
	rm -rf bin out
