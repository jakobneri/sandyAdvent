# sandyAdvent

Digitaler Adventskalender – ein einzelnes Go-Binary mit eingebetteten Templates, CSS, JS und Fonts.
Kontext und verbindliche Entscheidungen: [CLAUDE.md](CLAUDE.md), Meilensteine: [docs/plan.md](docs/plan.md),
Schnittstelle zu Schatzogram: [docs/interface.md](docs/interface.md).

## Quickstart (Dev)

```bash
scripts/dev-setup.sh          # Go, ffmpeg, vips, rclone, sqlite3 … (idempotent)
make dev                      # http://localhost:8081/dev/preview
make test
```

`make dev` nutzt `config.json`, falls vorhanden, sonst `config.example.json`
(Dummy-Tokens `EXAMPLE-01` … und Dummy-Assets aus `doors/assets-example/`).

Zeitreise im Dev-Modus:

```bash
ADVENT_NOW=2026-12-07T23:59 make dev   # Europe/Berlin; die Uhr läuft ab da weiter
```

Dev-Routen (nur mit `ADVENT_DEV=1` bzw. `"dev": true`, sonst neutrale 404):

| Route | Zweck |
|---|---|
| `/dev/preview` | Übersicht aller 24 Tage: Gutschein-Tage, Sperrstatus, fehlende Dateien, Links |
| `/dev/door/{tag}` | Türchen ohne Datumssperre (`?intro=1` spielt die Öffnungsanimation erneut) |
| `/dev/countdown/{tag}` | Countdown-Seite des Tages (ohne Auto-Reload) |

## Befehle

| Befehl | |
|---|---|
| `make dev` / `make run` | Server mit / ohne Dev-Modus |
| `make build` | `bin/sandyadvent` (CGO frei; `make build GOARCH=arm64` für den Pi) |
| `make test`, `make vet`, `make fmt` | Go-Tests + JS-Logik-Tests (`node --test tests/js/*.test.mjs`, ohne Node übersprungen), `go vet`, `gofmt` |
| `make test-js` | nur die JS-Logik-Tests (Wordle-Wertung, Schiebepuzzle) |
| `make check` | Config + Türchen-Config prüfen, fehlende Dateien melden (die Vorlage `config.example.json` im Dev-Modus) |
| `make tokens` | `config.json` anlegen (falls nötig) und Tokens für die 17 QR-Tage eintragen |
| `make qr BASE_URL=http://<Pi-IP>` | QR-PDF nach `out/qr-codes.pdf` |
| `make sync-brand` | Kopien der geteilten Brand-Dateien aktualisieren |
| `make example-assets` | Dummy-Assets mit ffmpeg (SVG über librsvg, sonst vips) neu erzeugen |

Binary direkt: `sandyadvent [serve|check|gen-tokens] [-config pfad]`.
`gen-tokens` ohne Flags druckt eine neue Token-Map; `-write` trägt fehlende Tokens in die Config ein
(vorhandene bleiben, `EXAMPLE-…` werden ersetzt, `-force` erzeugt alle neu – alte QR-Codes werden ungültig).

## Konfiguration

`config.json` (gitignored, Vorlage `config.example.json`). Relative Pfade gelten relativ zur Config-Datei.

| Feld | Bedeutung |
|---|---|
| `listen` | Adresse, Standard `localhost:8081` (hinter nginx) |
| `base_url` | z. B. `http://<Pi-IP>` – für das QR-PDF |
| `year` | Kalenderjahr (2026) |
| `doors_config`, `assets_dir` | Türchen-Registry und Medienordner |
| `tokens` | Token → Tag; genau 10 Zeichen `[A-Za-z0-9_-]`, nur die 17 QR-Tage, ein Token pro Tag |
| `schatzogram.welcome_url`, `schatzogram.reveal_date` | siehe `docs/interface.md` |
| `countdown_lines` | eigene freche Sprüche für die Countdown-Seite (leer = eingebaute) |
| `admin`, `open_log` | reserviert für M4 |
| `dev` | Dev-Modus |

Env-Overrides: `ADVENT_CONFIG`, `ADVENT_LISTEN`, `ADVENT_BASE_URL`, `ADVENT_DEV=1`,
`ADVENT_NOW=2026-12-05T08:00` (nur im Dev-Modus; sonst ignoriert und im Log gewarnt).
Der Server startet nicht bei ungültigen Tokens (Gutschein-Tag, falsche Länge, doppelt) oder ungültiger
Türchen-Config – und außerhalb des Dev-Modus auch nicht, solange noch `EXAMPLE-`-Tokens aus der Vorlage in der
Config stehen (Fehler, Exit-Code 1; `check` meldet dasselbe). Abhilfe: `make tokens`.

## Türchen (`doors/config.json`)

Vorlage: `doors/config.example.json` (alle 17 Türchen). Pro Tag `template`, `title`, `params`, `assets`.
`assets` listet die Dateien (oder Ordner mit `/` am Ende) relativ zu `assets_dir`, die dieses Türchen
ausliefern darf – alles andere ist 404. Dateien werden erst ab der Öffnung ausgeliefert.

Template **media** – `params.blocks` ist eine frei kombinierbare Liste:

| `type` | Felder |
|---|---|
| `text` | `text` (Leerzeile = Absatz, Zeilenumbruch bleibt), `heading?` |
| `audio` | `src`, `caption?`, `heading?` – eigener Player (iOS-tauglich) |
| `video` | `src`, `poster?`, `caption?` – `playsinline`, native Controls |
| `slideshow` | `images: [{src, caption?, alt?}]`, `interval_ms?` (0 = aus, sonst ≥ 1500) |
| `link` | `url` (http/https), `label`, `note?` – öffnet in neuem Tab, ohne Referrer |
| `image` | `src`, `caption?`, `alt?` |
| `yearsago` | `date: YYYY-MM-DD`, `image?`, `text?` – „Heute vor X Jahren“ relativ zum Türchen-Datum |
| `teaser` | `text?` (Standard „Morgen.“) – verschwommenes Schatzogram-Logo (Tür 23) |

### Spiele (M2): `quiz`, `memory`, `wordle`, `puzzle`

Gemeinsam: optional `intro` (Text über dem Spiel, Leerzeile = Absatz), am Ende ein Gewinn-Panel mit Abschlusstext,
Konfetti/Schnee (bei „Bewegung reduzieren“ nur ein Leuchten) und „Nochmal spielen“. Der Fortschritt wird pro
Türchen in `localStorage` gemerkt (Schlüssel aus Tag + Hash der Parameter – ändert Jakob die Parameter, startet das
Spiel neu; ohne Speicher läuft es trotzdem). Bilder sind Pfade relativ zu `assets_dir` und müssen in `assets` stehen.
Unbekannte Felder sind ein Fehler (Tippfehler fallen bei `make check` / beim Start auf).

**quiz** (Tage 3, 15, 21) – eine Frage nach der anderen, sofortiges Feedback, Punktestand, Auswertung

| Feld | |
|---|---|
| `questions` | Liste, mind. 1: `{question, quote?, image?, options, answer, explain?}` |
| `questions[].question` | Fragetext; darf fehlen, wenn `quote` gesetzt ist (dann „Wer hat's gesagt?“) |
| `questions[].quote` | Zitat, wird als Zitat-Karte gezeigt (Tür 15) |
| `questions[].options` | 2–6 Antworten (nicht doppelt) |
| `questions[].answer` | Index der richtigen Antwort, **0 = erste** |
| `questions[].explain` | Satz nach dem Antworten (richtig oder falsch) |
| `shuffle_questions`, `shuffle_options` | mischen (Standard aus; bei Jakob/Sandy/Beide besser aus lassen) |
| `results` | `[{min_score, text}]` – es gilt der höchste `min_score` ≤ Punkte; braucht einen Eintrag mit `min_score: 0` |
| `closing_text` | zusätzlicher Schlusssatz unter dem Ergebnis |

**memory** (Tag 6, Tag 18 schwer) – 3D-Flip, Zugzähler, optional Timer

| Feld | |
|---|---|
| `pairs` | 2–18 Paare (bis 12 Paare passt das Raster ohne Scrollen auf 360 px) |
| `images` | mind. `pairs` verschiedene Bilder; sind es mehr, wird pro Spiel zufällig ausgewählt |
| `card_back` | optionales Bild für die Kartenrückseite (sonst festliches Muster) |
| `peek_ms` | 0–10000: zu Beginn alle Karten so lange zeigen (Standard 0); startet erst nach der Öffnungsanimation, sobald das Spielfeld zu sehen ist |
| `show_moves` | Zugzähler (Standard `true`), `show_timer` Stoppuhr (Standard `false`) |
| `win_text` | Abschlusstext |

**wordle** (Tag 9) – mehrere Runden, eigene Wörter, Bildschirm-Tastatur QWERTZ + Ä Ö Ü ß, echte Tastatur geht auch

| Feld | |
|---|---|
| `words` | `[{word, hint?}]`, 1–10 Runden in dieser Reihenfolge; je 3–12 Buchstaben, nur A–Z, Ä, Ö, Ü, ß |
| `max_guesses` | 1–10, Standard 6 |
| `allow_any_guess` | nur `true` (Standard): kein Wörterbuch, jeder Versuch mit passender Länge zählt |
| `win_text` | Abschlusstext |

Wörter werden normalisiert (Groß-/Kleinschreibung egal, NFC, ß bleibt **ein** Buchstabe, ẞ → ß). Doppelte Buchstaben
werden wie im Original gewertet (erst grün, dann gelb nach verbleibender Anzahl). Der `hint` erscheint über „Tipp anzeigen“.

**puzzle** (Tag 12 = jigsaw)

| Feld | |
|---|---|
| `mode` | `jigsaw` (Teile mit Nasen ziehen, rasten nahe der richtigen Stelle ein) oder `sliding` (Schiebepuzzle mit Lücke, tippen oder wischen) |
| `image` | das Foto (Seitenverhältnis bleibt erhalten) |
| `grid` | 3–6 (N×N); Standard jigsaw 4, sliding 3 |
| `show_preview` | „Vorschau“-Knopf mit dem ganzen Bild (Standard `true`) |
| `win_text` | Abschlusstext |

Das Schiebepuzzle wird immer lösbar gemischt (Inversionen), nie schon gelöst; „Zahlen“ blendet Kachelnummern ein.

Die übrigen Templates (`crossword`, `wrapped`, `reveal`) zeigen bis M3/M4 „Dieses Türchen wird noch gebastelt“.

Empfohlene Formate (iOS Safari): Audio AAC `.m4a` (oder `.mp3`), Video H.264/AAC `.mp4` mit `faststart`,
Fotos JPG ≤ 2000 px lange Kante. Range-Requests werden unterstützt.

## QR-Codes

```bash
make tokens                                   # einmalig, danach Tokens nie mehr ändern
make qr BASE_URL=http://<Pi-IP>               # out/qr-codes.pdf, 40 mm, Fehlerkorrektur H
go run ./tools/qrpdf -help                    # -size-mm (≥ 30), -cut-marks, -out, -config, -test
```

Codiert wird `<base-url>/d/<token>`. Das Tool bricht ab, wenn nicht genau die 17 QR-Tage Tokens haben, wenn die
Base-URL auf `localhost`/`127.0.0.1`/`::1` zeigt (vom Handy nicht erreichbar) oder wenn noch die `EXAMPLE-`-Tokens
aus der Vorlage drinstehen. Nur für einen Testdruck: `make qr QR_TEST=1` (Kopfzeile „TESTDRUCK“).

## Geteilte Brand-Dateien

`shared/schatzogram-brand/` ist die (aus `schatzogram` gespiegelte) Quelle. Ins Binary eingebettet werden Kopien:
`web/static/brand/tokens.css` (Banner/Teaser) und `web/brand/logo-mark.svg` (inline im Teaser, nicht öffentlich
abrufbar). Nach Änderungen an `shared/` `make sync-brand` ausführen – `go test ./web` schlägt fehl, solange die
Kopien abweichen.

## Fonts

Fraunces und Nunito (SIL OFL 1.1, via `npm pack @fontsource/fraunces @fontsource/nunito` v5.3.0),
nur die Latin-Subsets als woff2 in `web/static/fonts/` inkl. Lizenzen. Keine CDN-Zugriffe.
