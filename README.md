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
| `make test-js` | nur die JS-Logik-Tests (Wordle-Wertung, Schiebepuzzle, Kreuzworträtsel-Eingabe, Wrapped-Slides/-Abspieler) |
| `make check` | Config + Türchen-Config prüfen, fehlende Dateien melden (die Vorlage `config.example.json` im Dev-Modus) |
| `make tokens` | `config.json` anlegen (falls nötig) und Tokens für die 17 QR-Tage eintragen |
| `make qr BASE_URL=http://<Pi-IP>` | QR-PDF nach `out/qr-codes.pdf` |
| `make sync-brand` | Kopien der geteilten Brand-Dateien aktualisieren |
| `make example-assets` | Dummy-Assets mit ffmpeg (SVG über librsvg, sonst vips) neu erzeugen |
| `make crossword IN=private/woerter.txt [OUT=…] [SEED=42]` | Kreuzworträtsel-Raster für Tür 16 erzeugen (Standard-Ausgabe `doors/assets/kreuzwort.json`), siehe [Kreuzworträtsel-Generator](#kreuzworträtsel-generator-toolscrossword) |
| `make wrapped-stats IN=private/chat.txt ALIAS='…=Sandy;…=Jakob' SINCE=JJJJ-MM-TT` | `stats.json` für Tür 22 aus dem WhatsApp-Export (Standard-Ausgabe `doors/assets/stats.json`), siehe [WhatsApp-Parser](#whatsapp-parser-toolswaparse) |
| `make waparse-testdata` | synthetische Beispiel-Exporte (`tools/waparse/testdata/`) und `doors/assets-example/stats.json` neu erzeugen |

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

### Rätsel & Rückblick (M3): `crossword`, `wrapped`

Beide lesen eine Datendatei aus `assets_dir` (muss in `assets` stehen). Sie wird beim Start bzw. bei `make check`
geprüft (ungültig = Fehler mit Tag und Grund; fehlt sie, nur eine Warnung) und bei jedem Aufruf neu gelesen –
auf dem Pi kann Jakob sie also ohne Neustart austauschen. Fehlt sie beim Öffnen, zeigt das Türchen einen freundlichen
Hinweis statt des Spiels.

**crossword** (Tag 16) – Raster mit Nummern, Hinweislisten „Waagerecht“/„Senkrecht“, Hinweisleiste, die über der
Handy-Tastatur klebt. Kästchen antippen wählt das Wort, dasselbe Kästchen nochmal wechselt die Richtung, Hinweis
antippen springt hin. Getippt wird in ein echtes (unsichtbares) Eingabefeld über dem aktiven Kästchen, damit iOS und
Android die Tastatur öffnen; Umlaute und ß direkt, Löschen geht zurück, Pfeiltasten/Tab/Enter auf echten Tastaturen.
Fortschritt bleibt gespeichert (neue Raster-Datei = neuer Fortschritt).

| Feld | |
|---|---|
| `grid` | Pfad zur Raster-JSON-Datei (z. B. `"kreuzwort.json"`, erzeugt mit `make crossword`) **oder** das Raster direkt als Objekt |
| `allow_check` | Knopf „Prüfen“ markiert falsche Buchstaben rot (Standard `true`) |
| `allow_reveal_letter` | Knopf „Buchstabe aufdecken“ für das aktive Kästchen (Standard `false`) |
| `closing_text` | Abschlusstext im Gewinn-Panel |
| `intro` | Text über dem Rätsel |

Bis 13 Spalten passt das Brett mit ≥ 24 px pro Kästchen auf 360 px; breitere Raster scrollen innerhalb des Bretts.

**wrapped** (Tag 22) – Startkarte im Türchen, dann eine bildschirmfüllende Story („Schatzo Wrapped“, eigene Optik im
Weihnachts-Design): Fortschrittsbalken oben, rechts/links tippen = weiter/zurück, gedrückt halten = Pause, wischen,
Auto-Weiter nach ~6 s (pausiert, wenn der Tab im Hintergrund ist), Pfeiltasten/Leertaste/Esc, Zurück-Taste schließt.
„Bewegung reduzieren“ zeigt die Endwerte ohne Animation.

| Feld | |
|---|---|
| `stats` | Pfad zur `stats.json` (erzeugt mit `make wrapped-stats`) |
| `title` | Titel der Story, Standard „Schatzo Wrapped <Jahr>“ |
| `outro_text` | Text auf der letzten Slide (Leerzeile = Absatz) |
| `slides` | optional: Reihenfolge/Auswahl und Texte, z. B. `[{"id": "days", "title": "So lange schon"}, {"id": "emojis", "text": "…"}]`; `intro` und `outro` kommen immer dazu |
| `intro` | Text über der Startkarte |

Slide-IDs (Standard-Reihenfolge): `intro`, `days` (Tage zusammen, zählt hoch), `messages` (gesamt + Balken pro Person),
`emojis`, `goodnight` (wer öfter zuerst „Gute Nacht“ sagt), `hours` (24-h-Diagramm + Wochentag), `streak` (längste
Serie), `busiest` (Rekordtag), `words` (Wortwolke), `first` (erste Nachricht), `outro`. Zusätzlich verfügbar, aber nicht
im Standard: `response` (Ø Antwortzeit pro Person). Slides ohne Daten fallen weg. „Tage zusammen“ rechnet das Türchen
beim Anzeigen neu (ab `since` bis heute, mindestens bis zum Türchen-Datum).

Die übrigen Templates (`reveal`) zeigen bis M4 „Dieses Türchen wird noch gebastelt“.

Empfohlene Formate (iOS Safari): Audio AAC `.m4a` (oder `.mp3`), Video H.264/AAC `.mp4` mit `faststart`,
Fotos JPG ≤ 2000 px lange Kante. Range-Requests werden unterstützt.

## Kreuzworträtsel-Generator (`tools/crossword`)

```bash
make crossword IN=private/woerter.txt OUT=doors/assets/kreuzwort.json            # zufälliger Seed (wird ausgegeben)
make crossword IN=private/woerter.txt OUT=doors/assets/kreuzwort.json SEED=42     # reproduzierbar
go run ./tools/crossword -help   # -in -out -max-size (13) -seed -attempts (1000) -preview -strict -strip-spaces
```

Die echte Wortliste enthält die Lösungen – sie gehört nach `private/` (gitignored, ebenso jede `*.txt` direkt im
Repo-Wurzelordner), nie ins Repo.

Eingabe (UTF-8): Zeilen `WORT;Hinweis` (`#` = Kommentar, Leerzeilen egal) oder JSON `[{"word": "…", "clue": "…"}]`.
Beispiel: `tools/crossword/beispiel-woerter.txt`. Wörter werden großgeschrieben (NFC), Ä/Ö/Ü/ß bleiben **ein**
Kästchen (ẞ → ß). Leerzeichen, Ziffern und Bindestriche werden mit deutscher Fehlermeldung abgelehnt
(`-strip-spaces` schreibt Mehrwort-Antworten zusammen). Doppelte Wörter und fehlende Hinweise sind Fehler.

Der Generator legt das längste Wort zuerst und setzt dann Wort für Wort über Kreuzungen an (keine parallel
anliegenden Wörter, keine Verlängerung anderer Wörter), probiert viele zufällige Varianten (`-attempts`, mit
`-seed` deterministisch) und nimmt die mit den meisten platzierten Wörtern, dann die kompakteste (Fläche, dann
Breite – das Handy ist hochkant). Nicht platzierte Wörter stehen auf stderr; mit `-strict` ist der Exit-Code dann 2.
Abhilfe: anderen Seed, mehr Versuche, `-max-size` größer (über 13 scrollt das Brett auf 360 px) oder ein Wort tauschen.

### Raster-JSON (Version 1)

```json
{
  "version": 1, "width": 5, "height": 4,
  "cells": [["K","A","T","Z","E"], ["U",null,null,null,null], ["S",null,null,null,null], ["S",null,null,null,null]],
  "entries": [
    {"number": 1, "direction": "across", "row": 0, "col": 0, "answer": "KATZE", "clue": "Miau"},
    {"number": 1, "direction": "down",   "row": 0, "col": 0, "answer": "KUSS",  "clue": "Schmatz"}
  ]
}
```

- `cells`: `height` Zeilen mit je `width` Feldern; ein Großbuchstabe (A–Z, Ä, Ö, Ü, ß) oder `null` = schwarzes Feld.
- `entries`: `row`/`col` 0-basiert ab oben links, `direction` `across` (waagerecht) oder `down` (senkrecht).
- Nummerierung: Felder zeilenweise durchgehen; jedes Feld, an dem ein waagerechtes oder senkrechtes Wort (≥ 2 Buchstaben)
  beginnt, bekommt die nächste Nummer.
- Geprüft wird beim Laden: Maße (1–30), Buchstaben, jede Antwort passt zu den Feldern und ist ein ganzes Wort, Nummern
  stimmen, jedes Wort im Raster hat genau einen Eintrag mit Hinweis (sonst berühren sich Buchstaben unerlaubt).
  Hand-Änderungen sind also erlaubt, solange `make check` grün bleibt.

## WhatsApp-Parser (`tools/waparse`)

Macht aus dem Chat-Export die `stats.json` für Tür 22. **Der Roh-Export bleibt lokal** (Laptop oder Pi, nie ins Repo –
Export am besten nach `private/` legen – `.gitignore` blockt `/private/`, alle `*.txt`/`*.zip` im Wurzelordner,
`WhatsApp*.txt`, `_chat.txt`, `*.chat.txt`, `/chat/`, `/exports/` und eine `stats.json` im Wurzelordner); ins `assets_dir` kommt nur
die aggregierte `stats.json` (enthält Namen, Zahlen, Top-Emojis/-Wörter und den Anfang der ersten Nachricht).

**Export (ohne Medien):**
- Android: Chat öffnen → ⋮ → Mehr → Chat exportieren → **Ohne Medien** → Datei speichern/teilen (`WhatsApp Chat mit … .txt`).
- iPhone: Chat öffnen → Name oben antippen → Chat exportieren → **Ohne Medien** → in „Dateien sichern“ (ZIP mit `_chat.txt`, entpacken).
- Handy-Sprache Deutsch (deutsches Datumsformat). Mehrere Exporte (z. B. von beiden Handys) dürfen zusammen rein,
  doppelte Nachrichten werden erkannt.

```bash
make wrapped-stats IN='private/WhatsApp Chat mit Sandy.txt' ALIAS='Sandy 🌸=Sandy;Jakob=Jakob' SINCE=2023-02-14
# direkt: go run ./tools/waparse -in a.txt [-in b.txt] -out stats.json -alias 'Name im Export=Anzeigename' …
```

| Flag | |
|---|---|
| `-in` | Export-Datei(en), mehrfach (Android `24.12.23, 18:05 - Name: Text`, iOS `[24.12.23, 18:05:12] Name: Text`, 2- oder 4-stellige Jahre) |
| `-alias` | `Name im Export=Anzeigename`, mehrfach. Das Tool listet die gefundenen Namen auf (Emojis im Namen gehören dazu!). **Jeden** Export-Namen angeben, auch unveränderte (`Jakob=Jakob`): Mit `-alias` beginnen nur diese Namen eine Nachricht, andere Zeilen der Form „Datum - Name: Text“ (z. B. eingefügte Chat-Schnipsel) bleiben Text der Nachricht davor. Ergibt der Export mehr als zwei Personen (oder mit `-alias` weniger als zwei), bricht das Tool mit Erklärung ab |
| `-since` | Beginn der Beziehung für „Tage zusammen“ (Standard: erste Nachricht) |
| `-until` | Nachrichten nach diesem Tag ignorieren |
| `-asof` | „Tage zusammen“ bis zu diesem Tag (Standard heute; das Türchen rechnet beim Anzeigen ohnehin neu) |
| `-tz` | Zeitzone, Standard `Europe/Berlin` |
| `-goodnight-pattern` | eigener regulärer Ausdruck (Go/RE2). Standard erkennt u. a. „gute Nacht“, „gn8“, „gn“, „Nachti“, „n8“, „Nächtle“, „schlaf gut/schön“, „träum süß“ als ganze Wörter |
| `-day-start-hour` | Gute-Nacht-Nachrichten vor dieser Uhrzeit zählen zur Nacht davor (Standard 5) |
| `-top` | Länge der Top-Listen (Standard 10) |
| `-keep-laughter` | „haha“, „hihi“, „lol“ … in den Top-Wörtern behalten |

Bereinigt wird datengetrieben (Listen in `internal/waparse/skip.go`): Systemzeilen (Verschlüsselungshinweis,
Sicherheitsnummer, …) fallen ganz weg; Platzhalter wie `<Medien ausgelassen>`, „Bild/Video/Audio/Sticker weggelassen“,
„Diese Nachricht wurde gelöscht.“, `<Anhang: …>` zählen als Nachricht ihres Absenders (Aktivität), aber ohne Text;
„<Diese Nachricht wurde bearbeitet>“ wird abgeschnitten; unsichtbare Richtungszeichen (U+200E …), schmale/geschützte
Leerzeichen, CRLF und BOM sind egal; Zeilen ohne Zeitstempel gehören zur Nachricht davor (mehrzeilig). Zeiten werden
minutengenau ausgewertet (Android hat keine Sekunden – so liefern beide Formate dieselben Zahlen); Tag und Stunde so,
wie sie im Export stehen (Zeitumstellung egal).

### `stats.json` (Version 1)

| Feld | |
|---|---|
| `version`, `generatedAt` | `1`, Zeitpunkt der Erzeugung (RFC 3339) |
| `range` | `{from, to}` erster/letzter Tag mit Nachrichten (`JJJJ-MM-TT`) |
| `since`, `asOf`, `daysTogether` | Beziehungsbeginn, Stichtag, Tage dazwischen |
| `totalMessages`, `activeDays` | Nachrichten gesamt, Tage mit mindestens einer Nachricht |
| `persons[]` | `{name, messages, words, share (0–1), media, avgResponseMinutes?, topEmoji?}`, meiste Nachrichten zuerst; Antwortzeit = Ø Minuten bis zur Antwort nach einem Sprecherwechsel (Pausen > 12 h zählen nicht) |
| `topEmojis[]` | `{emoji, count}` – ganze Graphem-Cluster (ZWJ-Sequenzen 👩‍❤️‍👨, Flaggen, Hauttöne); ❤️ und ❤ zählen gleich |
| `goodNightFirst[]`, `goodNightNights` | `{name, days}`: wer pro Nacht zuerst „Gute Nacht“ sagt; Anzahl Nächte mit Gute-Nacht |
| `mostActiveHour` | `{hour, count, histogram[24]}` |
| `mostActiveWeekday` | `{weekday ("Freitag"), index (0 = Montag), count, histogram[7] (Mo–So)}` |
| `longestStreak` | `{days, from, to}` – aufeinanderfolgende Tage mit Nachrichten |
| `topWords[]` | `{word, count}` ohne Stoppwörter (`internal/waparse/stopwords_de.txt`), Namen/Aliase, Wörter < 3 Buchstaben, URLs, Zahlen und Lachen |
| `firstMessage` | `{date, time, name, text}` (Text auf 140 Zeichen gekürzt) |
| `busiestDay` | `{date, count}` |

Synthetische Testdaten: `tools/waparse/testdata/android.txt` und `ios.txt` (dasselbe erfundene Gespräch zweier
Fake-Personen über ~6 Monate, inkl. Emojis, Mehrzeilern, Systemzeilen, Zeitumstellung), erzeugt von
`go run ./tools/waparse/gen`. Die Tests prüfen, dass beide Formate identische Statistiken ergeben.

## Inhalte für Tür 16 und 22 (Jakob)

- **Kreuzworträtsel:** 10–15 Insider-Wörter mit Hinweis als `WORT;Hinweis`, je 4–10 Buchstaben (höchstens 13), keine
  Leerzeichen/Ziffern/Bindestriche, Umlaute/ß direkt. Mischung aus kurzen und langen Wörtern mit häufigen Buchstaben
  (E, N, R, S …) lässt sich am besten verschränken. `make crossword …`, Vorschau prüfen, Datei nach
  `doors/assets/kreuzwort.json`, in `doors/config.json` bei Tag 16 `grid` + `assets` setzen.
- **Wrapped:** WhatsApp-Export ohne Medien (s. o.), die Namen, wie sie im Export stehen (für `-alias`), Datum des
  Beziehungsbeginns (`SINCE`), optional `outro_text` und Slide-Texte. Parser lokal oder auf dem Pi laufen lassen,
  nur `stats.json` nach `doors/assets/` kopieren, den Export danach löschen.

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
