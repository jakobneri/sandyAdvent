# Plan sandyAdvent

Deadline: Code + Inhalte bis **30.11.2026**. Erstes Türchen öffnet am 01.12.2026 00:00 (Berlin).
Ein Branch pro Meilenstein (`feature/mN`), PR gegen `main`.

## M1 – Grundgerüst (Branch `feature/m1`)
- [x] Go-Server (ein Binary, `embed` für Templates/Static/Fonts), Config + Env-Overrides
- [x] Routing `/d/{token}` mit Token → Tag aus der Config, neutrale 404
- [x] Datumssperre Europe/Berlin (N.12. 00:00), Countdown-Seite mit frechem Spruch
- [x] Dev-Modus: `ADVENT_NOW`, `/dev/preview`, `/dev/door/{tag}` (nur Dev)
- [x] Gemeinsames Layout, Weihnachts-Design, Öffnungsanimation, Banner „Schatzogram – noch X Tage“
- [x] Template **media** (Audio, Video, Slideshow, Text, Link kombinierbar)
- [x] `doors/config.example.json` mit allen 17 Türchen + Dummy-Assets
- [x] `sandyadvent gen-tokens`
- [x] QR-Tool `tools/qrpdf` (A4-PDF, EC H, ≥ 3 cm, Tagesnummer, Base-URL-Parameter)
- [x] `scripts/dev-setup.sh`, Makefile, `.gitignore`, Basis-Tests

## M2 – Spiele-Templates (Branch `feature/m2`)
- [x] quiz (Multiple Choice, Score, Abschlusstext) – Tage 3, 15, 21
- [x] memory (Paaranzahl konfigurierbar) – Tag 6 + Tag 18 (schwer)
- [x] wordle (eigene Wörter, variable Länge, Umlaute) – Tag 9
- [x] puzzle (Modi jigsaw + sliding) – Tag 12 (jigsaw)
- [x] Entscheidung Tag 18: memory (schwer)
- [x] Fix: Pluralisierung in Countdown und Banner („1 Tag“, „1 Stunde“ …)
- [x] Fix: Schatzogram-Banner auf Türchen 23 und 24 ausblenden
- [x] Fix: Start außerhalb Dev-Modus mit EXAMPLE-Tokens verweigern

## M3 – Rätsel & Wrapped (Branch `feature/m3`)
- [ ] crossword-Template (Raster-JSON, Handy-Tastatur) – Tag 16
- [ ] Kreuzworträtsel-Generator (`tools/crossword`)
- [ ] wrapped-Template (animierte Slides aus `stats.json`) – Tag 22
- [ ] WhatsApp-Parser (`tools/waparse`), Android + iOS, deutsches Locale, synthetische Beispiel-Exporte

## M4 – Reveal, Admin, Deployment (Branch `feature/m4`)
- [ ] reveal-Template (Tag 24) mit Button zu `schatzogram.welcome_url` (siehe `docs/interface.md`)
- [ ] Öffnungs-Log JSONL + `/admin` mit Basic Auth
- [ ] `scripts/setup-pi.sh` (Abhängigkeiten, systemd, nginx-Vhost), `make build` auf dem Pi
- [ ] GitHub Action: ARM64-Binary als Release-Artefakt
- [ ] `docs/deploy.md` (Umzug auf den Pi)
- [ ] Tests Datumssperre + Token-Routing (vollständig)

## Inhalte (Jakob liefert)
Wird nach jedem Meilenstein konkretisiert; Stand siehe PR-Beschreibung von M1.
