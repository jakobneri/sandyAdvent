# Design-System sandyAdvent

Verbindlich für alle Templates und Features ab M2 (quiz, memory, wordle, puzzle, crossword, wrapped,
reveal). Umsetzung: `web/static/css/app.css` (Tokens in `:root`, Komponenten als Klassen).
Entscheidungen aus `CLAUDE.md` und `docs/interface.md` gehen immer vor.

## Idee

Jede Türchen-Seite ist **das, was hinter dem Papiertürchen liegt**: ein Abreißkalender-Blatt mit großer
goldgeprägter Tageszahl und „Dezember“ daneben. Darunter liegen die Sachen des Tages direkt in der
Nacht, ohne Karten. Nur echte Dinge bekommen ein Material: Foto (Polaroid), Anhänger, Siegel, Tür.

1. **Die Zahl ist das Gesicht.** Die Gold-Folie gibt es nur für die Tageszahl (Hero und Intro-Tür).
2. **Wörter liegen in der Nacht, Dinge sind aus Papier.** Keine generische Karte.
3. **Eine Bewegung pro Tag.** Tür öffnen plus ein Glanz auf der Zahl. Alles andere antwortet auf einen Tipp.
4. **Bastelkiste, nicht Clipart.** Keine Schneeflocken, Bäume, Glocken. Ornament nur, wenn es etwas
   bedeutet: Perforation = Tür, Schleife = noch verpackt, gestapelte Bögen = mehrere Fotos,
   gestrichelt = gedruckt.
5. **Schatzogram bleibt ein Fremdkörper.** `--sg-*`, Ember und Fraunces nur in Banner, Teaser (23) und
   Reveal (24).

## Farben

| Token | Hex | Aufgabe |
|---|---|---|
| `--tannennacht` | `#0F2B25` | Grund (html, body, `theme-color`) |
| `--tannenzweig` | `#1F4A3F` | leise Flächen in der Nacht: Progress-Track, gesperrte Kacheln, leere Spielfelder |
| `--kerzengold` | `#E8B652` | Licht und Folie: Fortschritt, Fokusring, aktiver Punkt, Siegelring. Nie Fließtext auf Papier |
| `--preiselbeere` | `#A8283E` | alles, was man drückt oder öffnet: Siegel, Tür, Ziffern auf dem Anhänger |
| `--zuckerguss` | `#FBF4E6` | Papier und Text in der Nacht |
| `--lebkuchen` | `#3A2418` | Tinte auf Papier und Fotos |

Abgeleitet (keine neuen Töne): `--gold-hell` (Überschriften in der Nacht), `--gold-tief` (Prägung),
`--preiselbeere-hell`/`-tief` (Verlauf, gedrückt), `--zuckerguss-tief` (hinterer Papierbogen),
`--fotoweiss` (Polaroidrahmen), `--text-leise` (Zweittext in der Nacht).
Spielzustände: `--ok` `#2F7A5F`, `--near` = Kerzengold, `--wrong` = Preiselbeere, `--miss` = Tannenzweig.

Kontrast (WCAG, gerechnet): Zuckerguss/Tannennacht 13,8 · text-leise/Tannennacht 7,8 ·
Gold-hell/Tannennacht 11,5 · Preiselbeere/Zuckerguss 6,3 · Lebkuchen/Zuckerguss 13,3 ·
Zuckerguss/`--ok` 4,7 · Lebkuchen/Kerzengold 7,8 · Gold-hell/Tannenzweig 7,6.
**Preiselbeere als Text in der Nacht ist verboten** (2,2:1, auf Tannenzweig 1,4:1); dort Gold-hell nehmen.

Licht: genau eine Lichtquelle, der warme Schein hinter der Tageszahl (`.hero::before`). Keine weiteren
Verläufe als Deko. Die Countdown-Seite bleibt dunkel (die Tür ist noch zu).

## Schrift

- **Young Serif 400** (`--font-display`): Zahlen, Titel, Bildunterschriften, Anhänger. Hat nur einen
  Schnitt: jede Regel setzt `font-weight: 400` und `font-synthesis: none` (sonst Faux-Bold).
- **Nunito 400/700** (`--font-ui`): Fließtext, Buttons, Hinweise.
- **Fraunces 700 italic**: nur die Schatzogram-Wortmarke (`--sg-font-logo`).
- Alle Fonts lokal (OFL, Lizenzen in `web/static/fonts/`), kein CDN.

Skala 12 · 14 · 16 · 18 · 21 · 24 · 36 · 48 · 144. Jedes Token ist genau ein Schritt:

| Token | px | Verwendung |
|---|---|---|
| `--fs-numeral` | 144 | Tageszahl (Folie); Intro-Tür `× .75` |
| `--fs-display` | 48 | „Morgen.“ (Teaser) |
| `--fs-h1` | 36 (48 ab 600px) | Türchentitel |
| `--fs-clock` | 36 | tickende/ausgerichtete Ziffern: Uhr, Medaille, Kachelzahlen, Ergebnisse |
| `--fs-h2` | 24 | Blocküberschrift, frecher Spruch, „Heute vor X Jahren“ |
| `--fs-lead` | 21 | „Dezember“, Anhänger-Titel, Untertitel |
| `--fs-body` | 18 | Fließtext (Zeilenhöhe 1,6, max. 34em) |
| `--fs-caption` / `--fs-ui` | 16 | Bildunterschrift (Young Serif) / Aktionen (Nunito 700) |
| `--fs-small` | 14 | Hinweise, Player-Zeit, Notizen |
| `--fs-micro` | 12 | nur Dev-Vorschau |

Regeln:
- Zwei Anzeigezeilen direkt untereinander unterscheiden sich um mindestens einen Schritt.
- Ziffern, die ticken oder fluchten müssen: `font-variant-numeric: lining-nums tabular-nums`.
  Mediävalziffern bleiben nur im Lauftext („vor 3 Jahren“, „21. Dezember“).
- Keine Versalien, kein Letter-Spacing auf Labels, keine Mittelpunkt-Meta-Strings („A · B“),
  kein „Wort – Fragment“, kein „→/↗“ hinter Linktexten. Satzschreibung überall.

## Layout

- Eine Spalte, linksbündig (liest sich wie ein Brief), `.wrap` max. 600px (Vorschau 980px).
  Zentriert nur die „Objekt“-Seiten: Countdown und Platzhalter.
- Seitenrand `max(20px, safe-area)`, unter 375px 16px. Oben/unten Safe-Area + 14/48px.
- 8er-Raster: `--space-1…8` = 4/8/12/16/24/32/40/56. Hero → erster Block 32px, Block-Abstand 40px
  (48px ab 600px). Abstand zwischen Blöcken kommt **nur** von `.blocks { gap }`, Blöcke haben keine
  Außenabstände.
- Radien folgen dem Material: Foto 3px, Papier 6px, Tür 8px, Zeile 12px, Siegel 50%, Chip 999px.
- Kein horizontales Scrollen: `overflow-x: clip` (Fallback `hidden`) auf html und body ist nur das
  Netz; gedrehte Dinge müssen mit ihrer Drehung in die Spalte passen
  (z. B. `max-width: min(320px, calc(100% - 24px))`). Prüfen: bei 360px mit `overflow-x: visible`
  muss `scrollWidth <= innerWidth` gelten.
- Höhe: `min-height: 100vh; min-height: 100dvh`. Hintergrund-Ebenen als `position: fixed`
  (iOS ignoriert `background-attachment: fixed`).

## Komponenten

Eine Klasse pro Teil (BEM: `block__el`, `block--mod`), keine Regeln auf nackte Kindelemente
(Ausnahme `.prose > p`). Texte und Icons auf Blockebene dürfen keinen Inhalt annehmen: der
Audio-Block kann Sprachnachricht oder Lied sein, der Link-Block irgendein Ziel.

| Klasse | Was | Für M2/M3 |
|---|---|---|
| `.foil` + `--hero` / `--flat`, `data-num`, `.foil__face` | Goldprägung. `--hero`: Verlauf per `background-clip:text` + einmaliger Glanz (`html.is-opened`). `--flat`: Vollton für bewegte/3D-Kontexte (Intro-Tür). | **nur** Tageszahl. Nicht für Quiz-Score oder Wrapped. |
| `.seal-btn` (`--sm` 44, Standard 64, `--lg` 80, `--line` für Strich-Icons) | Wachssiegel, die Hauptaktion. Läuft-Zustand: gestrichelter Goldring dreht sich. | Quiz „Weiter“, Memory „Nochmal“, Wordle-Absenden |
| `.action` (`__text`, `__label`, `__note`) | Zeile aus Siegel + Text; die ganze Zeile ist das Ziel. | jede „X öffnen“- oder „Weiter“-Zeile |
| `.prose` | Text in der Nacht, 18/1,6, max. 34em | Intro-/Outro-Texte, Gewinntexte |
| `.blocks`, `.block`, `.block__title`, `.block__caption` | Blockhülle, h2 24 Gold-hell, Unterschrift 16 text-leise | jedes Template |
| `.photo` (`__img` quadratisch, `__pic` natürlich, `__cap`, `--tilt`, `--tilt-r`) | Polaroid, Unterschrift auf dem Rand | Memory-Kartenvorderseite, Puzzle-Vorschau |
| `.stack` | zwei Papierbögen dahinter = „mehr als eins“. Bögen sind Pseudo-Elemente des Wrappers, **nie** in einem Scroll-Container. | Memory-Stapel |
| `.slip` (+ `--ok/--near/--miss/--wrong`, `__mark`) | schlichtes Papier | Quiz-Antworten, Wordle-Kacheln, Memory-Karten |
| `.cell` | Eingabefeld/Rasterzelle, Papier, Young Serif ≥ 16px (sonst zoomt iOS) | Kreuzworträtsel, Wordle |
| `.tag-wrap` > `.tag` (`__title`, `__figure`, `__note`) | Geschenkanhänger mit Öse, leicht gedreht | Ergebnis-/Gewinnmeldungen, Quiz-Score in `.tag__figure` |
| `.door` (+ `--intro`, `--wrapped`, `__medal`, `__ribbon`, `__bow`) | Papiertür mit Perforation | Memory-Rückseite = Mini-Tür |
| `.chip` (+ `--ok`, `--wrong`, `--gold`) | kleine Statusmarke, Satzschreibung | Hinweise, „Richtig“/„Leider nein“ |
| `.progress` (auf `input[type=range]`, `--p` per JS) | Track Tannenzweig, Füllung Kerzengold | Quiz-Fortschritt, Slide-Timer |

Markup-Muster:
- **Zustände nie nur über Farbe** (Rot/Grün sind fast gleich hell):
  `<button class="slip slip--ok"><span class="slip__mark" aria-hidden="true">✓</span> Beim Italiener
  <span class="sr-only">(richtig)</span></button>`, `<span class="chip chip--wrong">✗ Leider nein</span>`.
  Wordle: `near` und `miss` bekommen zusätzlich ein Zeichen in `.slip__mark` (z. B. `•` / `–`).
- `.cell`: `<input class="cell" maxlength="1" autocapitalize="characters" autocomplete="off"
  autocorrect="off" spellcheck="false" aria-label="Zeile 2, Spalte 5">`. Umlaute/ß in JS normalisieren,
  kein `text-transform: uppercase` (ß → SS sprengt die Zelle).
- Quiz- und Spielergebnisse: auf `.tag`, Zahl in `.tag__figure` (Preiselbeere), Text in Lebkuchen.
- **Wrapped (Tag 22)** bekommt eine eigene Behandlung, die in M3 entschieden wird (nicht `.foil`).
- Reveal (Tag 24) gehört zur Brand-Zone und nutzt `--sg-*`.

Fokus: `:focus-visible { outline: 2px solid transparent; outline-offset: 2px; box-shadow: 0 0 0 3px
var(--tannennacht), 0 0 0 5px var(--kerzengold); }` (transparente Outline = sichtbar im
Forced-Colors-Modus). Die Regel steht spät im CSS, damit sie Komponenten-Schatten überschreibt.
Komponenten mit eigenem Goldring (Siegel, Range-Thumb) bekommen außen einen Zuckerguss-Ring. Der
Radius folgt der Komponente. Neue Komponenten mit `box-shadow` und höherer Spezifität als (0,1,0)
brauchen eine eigene `:focus-visible`-Regel.

## Bewegung

- **Der eine Moment:** Intro-Tür (nur erster Besuch, Tipp überspringt, ca. 2,5s): Ruck → Tür schwingt
  auf → Licht flutet → Seite steigt 12px auf → ein Glanz über die Tageszahl.
- **Antworten auf Aktionen:** Siegel `scale(.94)`, 150ms; Slideshow-Punkt wird zur Pille; Countdown:
  Tür wackelt, Anhänger schwingt, Spruch blendet über, `vibrate(40)`.
- **Nur zur Aufmerksamkeit:** Countdown-Tür stupst alle 7s kurz an; Teaser-Logo atmet (es ist der Inhalt).
- Kein Dauerschnee, keine Einblend-Animation pro Abschnitt, keine Hover-Effekte auf allem.
  Hover nur in `@media (hover: hover)`.
- `prefers-reduced-motion`: globaler Kill-Switch plus `animation: none` für jede benannte
  Endlos-Animation (gleiche Selektor-Spezifität wiederholen). Kein Intro, kein Glanz, kein Stupsen.
- WebKit: kein 3D-`transform`, `translateZ` oder `will-change` auf einem Vorfahren von `.foil--hero`;
  der Aufstieg von `.wrap` bleibt ein 2D-`translateY`.

## Technik (iOS Safari 16)

- Präfixe: `-webkit-background-clip:text`, `-webkit-text-fill-color`, `-webkit-clip-path`,
  `-webkit-backface-visibility`.
- Nicht verwenden: CSS-Nesting, `color-mix()`, `@property`, Container Queries.
- `aspect-ratio` mit `@supports not`-Fallback, `dvh` mit `vh`-Fallback, Safe-Area per `env()`.
- Tap-Ziele ≥ 44px (Punkte 24×24 nach WCAG 2.2), Eingaben ≥ 16px Schrift.
- Keine `style=`-Attribute (CSP); dynamische Werte per `style.setProperty` (z. B. `--p`).
- Die Slideshow-Rechnung in `door.js` braucht: Track ohne `padding`/`gap`, `.slide { flex: 0 0 100% }`;
  Rahmen, Schatten und Luft stecken im Innenabstand der Folie.

## Do / Don't

| Do | Don't |
|---|---|
| Tokens aus `:root` nutzen | Hex-Werte in Templates oder neue Farbtöne |
| Wörter direkt auf die Nacht setzen | jeden Block in eine Karte packen |
| Material nur für echte Dinge (Foto, Anhänger, Siegel, Tür, Papier) | gleiche Karte mit gleichem Schatten überall |
| Young Serif 400 für Zahlen und Titel | Faux-Bold, Fraunces außerhalb der Brand-Zone |
| Zustände mit Zeichen oder Text | Rot/Grün als einziges Signal |
| Fehlertexte mit Anleitung („… Lade die Seite neu …“) | vage Fehler oder Entschuldigungen |
| Gold-Folie nur für die Tageszahl | große Zahl + kleines Label + Verlauf als Standard-Deko |
| `--sg-*` nur in Banner, Teaser, Reveal | Schatzogram-Farben im übrigen Kalender |
| Banner-Text exakt nach `docs/interface.md`, ohne Erklärung, ohne Link | Banner verlinken oder erklären |
| 404 neutral lassen (`neutral.css`) | Kalender-Optik auf der 404 |
