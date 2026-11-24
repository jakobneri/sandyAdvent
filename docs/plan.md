# Plan sandyAdvent

Deadline: Code + Inhalte bis **30.11.2026**. Erstes Türchen öffnet am 01.12.2026 00:00 (Berlin).
Ein Branch pro Meilenstein (`feature/mN`), PR gegen `main`.

## M1 – Grundgerüst (Branch `feature/m1`)
- [ ] Go-Server (ein Binary, `embed` für Templates/Static/Fonts), Config + Env-Overrides
- [ ] Routing `/d/{token}` mit Token → Tag aus der Config, neutrale 404
- [ ] Datumssperre Europe/Berlin (N.12. 00:00), Countdown-Seite mit frechem Spruch
- [ ] Dev-Modus: `ADVENT_NOW`, `/dev/preview`, `/dev/door/{tag}` (nur Dev)
- [ ] Gemeinsames Layout, Weihnachts-Design, Öffnungsanimation, Banner „Schatzogram – noch X Tage“
- [ ] Template **media** (Audio, Video, Slideshow, Text, Link kombinierbar)
- [ ] `doors/config.example.json` mit allen 17 Türchen + Dummy-Assets
- [ ] `sandyadvent gen-tokens`
- [ ] QR-Tool `tools/qrpdf` (A4-PDF, EC H, ≥ 3 cm, Tagesnummer, Base-URL-Parameter)
- [ ] `scripts/dev-setup.sh`, Makefile, `.gitignore`, Basis-Tests

## M2 – Spiele-Templates (Branch `feature/m2`)
- [ ] quiz (Multiple Choice, Score, Abschlusstext) – Tage 3, 15, 21
- [ ] memory (Paaranzahl konfigurierbar) – Tag 6 (+ ggf. 18)
- [ ] wordle (eigene Wörter, variable Länge, Umlaute) – Tag 9
- [ ] puzzle (Schiebe- oder Jigsaw aus Foto) – Tag 12 (+ ggf. 18)
- [ ] Entscheidung Tag 18: memory oder puzzle (Jakob)

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
