# sandyAdvent – Kontext & Entscheidungen

Digitaler Adventskalender für Jakobs Freundin („Sandy“/„Schatzo“). Gehört zusammen mit dem Repo
**schatzogram** (privates Instagram für die beiden), das am 24.12.2026 über Türchen 24 enthüllt wird.
Gemeinsame Schnittstelle: `docs/interface.md` und `shared/schatzogram-brand/` (in beiden Repos identisch).

**Regel:** Von den Entscheidungen in dieser Datei nicht abweichen, ohne vorher Jakob zu fragen.

Deadline: Code **und** alle Inhalte bis **30.11.2026** fertig. Meilensteine: `docs/plan.md`.

## Arbeitsweise
- Ein Feature-Branch pro Meilenstein (`feature/m1`, `feature/m2`, …), Push + PR gegen `main`.
- Private Inhalte (Fotos, Audio, Video, Chat-Exporte, echte Configs, Tokens) **nie committen**.
  `.gitignore` + `*.example*`-Dateien mit Dummy-Assets. Echte Inhalte kommen erst auf dem Pi dazu.
- Keine hart codierten IPs/Domains – alles über Config/Env.

## Infrastruktur
- Go-Backend als **ein einzelnes Binary** (`cmd/sandyadvent`), Templates/CSS/JS/Fonts/Libraries per
  `embed` im Binary. Vanilla HTML/CSS/JS, **kein Framework, keine CDNs**, Libraries lokal gebündelt.
- Zeitzone fest **Europe/Berlin**, `time/tzdata` wird eingebettet (Pi braucht kein tzdata).
- Dev (Cloud): `make dev` → http://localhost:8081. Systemabhängigkeiten via `scripts/dev-setup.sh`.
- Pi 5 (ARM64, Linux, nginx vorhanden): Server auf **Port 8081** hinter nginx, erreichbar über
  `http://<Pi-IP>/`, **kein HTTPS**, nur Heim-WLAN, kein Tunnel. Base-URL aus der Config.
- Zielgerät: Android + Chrome; iOS Safari muss voll funktionieren. **Mobile-first.**
- Deployment (M4): `scripts/setup-pi.sh` (Abhängigkeiten, systemd-Unit, nginx-Vhost), `make build`
  baut direkt auf dem Pi, zusätzlich GitHub Action mit ARM64-Binary als Release-Artefakt,
  `docs/deploy.md` als Schritt-für-Schritt-Anleitung.

## Pi „archimedes“ (Stand 25.11.2026, von Jakob)
- Raspberry Pi 5, 8 GB, NVMe-SSD, Raspberry Pi OS, headless, LAN-IP **192.168.188.22**
  (IP nur in `config.json`/Doku-Beispielen, nie im Code). SSH als `jakob`.
- Node.js **20.x** (NodeSource) – zu alt für Angular 22 (braucht ≥ 22.22 bzw. ≥ 24.15).
- Belegte Ports (vor jedem neuen Dienst `ss -tlnp` prüfen): 53 (AdGuard DNS), 3000 (Cockpit),
  3002 (Homepage), 5678 (n8n), 8080 (Traefik aus n8n-Compose), 8181 (Kiwix), 8888 (AdGuard UI),
  9119 (Hermes), 11443 (UniFi OS), 11444 (nginx unifi-proxy, Container). Geplant: 8081 sandyAdvent,
  8082 schatzogram. Wer 80/443 hält, ist noch zu klären.
- Podman + Docker vorhanden; unsere Dienste laufen trotzdem nativ als systemd-Units.
- DNS im LAN: **AdGuard Home** auf dem Pi (Router EX7 verteilt 192.168.188.22 als DNS).
- cloudflared-Tunnel existiert für andere Dienste (nerifeige.com) – sandyAdvent/schatzogram
  bekommen **keinen** Tunnel-Eintrag (nur Heim-WLAN).

## Konfiguration
- `config.json` (gitignored; Vorlage `config.example.json`): `listen`, `base_url`, `year`,
  `doors_config`, `assets_dir`, `tokens` (Token → Tag), `schatzogram.welcome_url`,
  `schatzogram.reveal_date`, `admin` (Basic Auth, M4), `open_log` (JSONL, M4), `dev`.
- Env-Overrides: `ADVENT_CONFIG` (Pfad), `ADVENT_LISTEN`, `ADVENT_BASE_URL`, `ADVENT_DEV=1`,
  `ADVENT_NOW=2026-12-05T08:00` (nur im Dev-Modus wirksam, interpretiert in Europe/Berlin).
- `doors/config.json` (gitignored; Vorlage `doors/config.example.json`): Tag → Template, Titel,
  Parameter, Assets. Assets liegen in `doors/assets/` (gitignored), Dummies in `doors/assets-example/`.
- Tokens erzeugen: `sandyadvent gen-tokens` (10 Zeichen, URL-safe, crypto/rand) für die 17 QR-Tage.

## Logik
- Route `/d/{token}`: Token → Tag aus der Config. Unbekannter Token → **neutrale 404-Seite**
  (verrät nichts über den Kalender).
- QR-Codes kleben hinter **17 Türchen**. Gutschein-Tage **2, 5, 8, 11, 14, 17, 20** sind gedruckt:
  keine Route, kein QR-Code, kein Token.
- **Datumssperre** (Europe/Berlin): Türchen N öffnet ab **N.12. 00:00**. Vorher Countdown mit frechem
  Spruch (Sprüche konfigurierbar, Defaults im Code).
- **Dev-Modus**: `ADVENT_NOW` überschreibt „jetzt“; Vorschau-Routen `/dev/preview` (Übersicht) und
  `/dev/door/{tag}` (ohne Sperre) – existieren **nur** im Dev-Modus (sonst 404).
- Öffnungs-Log als JSONL (Tag, Zeitstempel, User-Agent) + `/admin` mit Basic Auth (M4).
- Auf jedem Türchen dezenter Banner **„Schatzogram – noch X Tage“** (bis 24.12.2026 00:00 Berlin),
  **ohne** zu erklären, was das ist. Optik aus den gemeinsamen Brand-Tokens.
  **Nicht** auf Türchen 23 und 24 (dort übernimmt Teaser bzw. Reveal).
- Alle Zeitangaben korrekt pluralisiert („1 Tag“, „2 Tage“, „1 Stunde“, „1 Minute“, „1 Sekunde“) –
  Countdown und Banner.
- Außerhalb des Dev-Modus **verweigert der Server den Start**, solange `EXAMPLE-`-Tokens in der
  Config stehen.

## Türchen (doors/config.json)
| Tag | Template | Inhalt |
|---|---|---|
| 1 | media | Sprachnachricht mit Erklärung |
| 3 | quiz | Quiz über uns |
| 4 | media | Foto-Slideshow „Unser Jahr“ |
| 6 | memory | Memory mit unseren Fotos |
| 7 | media | Video |
| 9 | wordle | Wordle mit Insider-Wörtern |
| 10 | media | Playlist-Link |
| 12 | puzzle | Foto-Puzzle (Modus jigsaw) |
| 13 | media | „Heute vor X Jahren“ |
| 15 | quiz | „Wer hat's gesagt?“ |
| 16 | crossword | Kreuzworträtsel mit Insider-Begriffen |
| 18 | memory | schwerere Memory-Variante (mehr Paare) – entschieden 24.11. |
| 19 | media | Sprachnachricht |
| 21 | quiz | Quiz Teil 2 |
| 22 | wrapped | „Schatzo Wrapped“ im Spotify-Wrapped-Stil |
| 23 | media | Teaser: verschwommenes Schatzogram-Logo + „Morgen.“ |
| 24 | reveal | große Reveal-Seite, Button zu https://sg.n3ri.com/welcome |

## Templates
- **media**: Audio, Video, Slideshow, Text, Link – frei kombinierbar (Liste von Blöcken).
- **quiz**: Multiple Choice, Score, Abschlusstext.
- **memory**: Paaranzahl konfigurierbar.
- **wordle**: eigene Wörter, variable Länge, Umlaute. Verlorene Runde: Wort wird aufgedeckt, weiter zur
  nächsten Runde (kein zweiter Versuch); Tipp nur per „Tipp anzeigen“ (bestätigt 25.11.).
- **puzzle**: Schiebe- oder Jigsaw-Puzzle aus einem Foto.
- **crossword**: rendert Raster-JSON, Eingabe per Handy-Tastatur.
- **wrapped**: animierte Slides aus `stats.json`.
- **reveal**: große Reveal-Seite für Tag 24.
- Design: gemütlich-weihnachtlich, verspielt, hochwertig, **keine generische Bootstrap-Optik**,
  einheitliche Optik, kleine Öffnungsanimationen. Fonts lokal (OFL), keine Google-Fonts-CDN.

## Tools (`tools/`)
- **QR-Generator** (`tools/qrpdf`): A4-PDF mit den 17 Codes, Error-Correction **H**, mind. **3 cm**,
  Tagesnummer darunter, Base-URL als Parameter, Tokens aus `config.json`.
- **Kreuzworträtsel-Generator** (M3): Wörter + Hinweise → Raster-JSON.
- **WhatsApp-Parser** (M3): Export (.txt, ohne Medien, deutsches Locale), Android- und iOS-Format →
  `stats.json` (Tage zusammen, Nachrichten pro Person, Top-Emojis, wer öfter zuerst „gute Nacht“
  schreibt, aktivste Stunde/Wochentag, längste Serie, Top-Wörter ohne Stoppwörter). Rohdaten bleiben
  lokal. Synthetischer Beispiel-Export in beiden Formaten zum Testen.

## Befehle
```bash
scripts/dev-setup.sh      # Systemabhängigkeiten (Go, ffmpeg …) für Dev
make dev                  # Server mit Dev-Modus auf http://localhost:8081
make test                 # go test ./...
make build                # Binary nach bin/
make qr BASE_URL=http://192.168.178.50   # QR-PDF nach out/qr-codes.pdf
```
