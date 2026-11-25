# Gemeinsame Schnittstelle sandyAdvent ↔ schatzogram

Diese Datei und `shared/schatzogram-brand/` sind in **beiden** Repos identisch.
Quelle der Wahrheit: Repo `schatzogram`. Änderungen immer in beiden Repos gleichzeitig.

## 1. Reveal-Link (Türchen 24)
| | |
|---|---|
| Ziel-URL (Prod) | `https://sg.n3ri.com/welcome` |
| Config sandyAdvent | `schatzogram.welcome_url` (Default = Ziel-URL; Dev z. B. `http://localhost:4200/welcome`) |
| Query-Parameter | `?from=advent` wird angehängt (Schatzogram darf darauf mit einer Extra-Begrüßung reagieren, muss aber nicht) |
| Auth | `/welcome` ist **ohne Login** erreichbar (Begrüßung + Login + Install-Anleitung) |
| Verfügbarkeit | Schatzogram muss ab **24.12.2026 00:00 Europe/Berlin** unter der URL laufen |

Netz: Beide laufen nur im Heim-WLAN. sandyAdvent per `http://<Pi-IP>/`, Schatzogram per
`https://sg.n3ri.com` (Cloudflare-DNS → LAN-IP des Pi, Let's-Encrypt-Zertifikat per DNS-01).
Der Wechsel http → https ist ein normaler Link (neuer Tab nicht nötig).

## 2. Countdown „Schatzogram – noch X Tage“
- Zieldatum: `schatzogram.reveal_date` = `2026-12-24` (00:00 Europe/Berlin).
- X = Anzahl Kalendertage bis zum Zieldatum (Berlin). Texte:
  X ≥ 2 → „Schatzogram – noch X Tage“, X = 1 → „Schatzogram – noch 1 Tag“ (korrekt pluralisiert).
- Auf Türchen **23 und 24 kein Banner** (Teaser bzw. Reveal übernehmen). Wird ein früheres Türchen
  am 24. oder später geöffnet: „Schatzogram – heute“.
- Dezent, ohne Erklärung, kein Link.

## 3. Teaser (Türchen 23)
- Verschwommenes Schatzogram-Logo (`shared/schatzogram-brand/logo-mark.svg`, CSS `filter: blur(...)`)
  + Text „Morgen.“

## 4. Design-Tokens
`shared/schatzogram-brand/tokens.css` (CSS Custom Properties, Präfix `--sg-`) und `tokens.json`
(dieselben Werte maschinenlesbar). sandyAdvent nutzt sie **nur** für Banner, Teaser (23) und Reveal (24);
der Rest des Kalenders hat seine eigene Weihnachts-Optik. Schatzogram nutzt sie als Basis seines Themes.

| Token | Wert | Verwendung |
|---|---|---|
| `--sg-ember` | `#FF5A36` | Primärfarbe („Glut“, passt zu den 🔥-Flammen) |
| `--sg-gold` | `#FFB627` | Sekundärfarbe |
| `--sg-plum` | `#3B1F4A` | Dunkle Akzentfarbe, Text auf hellem Grund für Logo |
| `--sg-blush` | `#FFE3D8` | Flächen, Chips |
| `--sg-cream` | `#FFF8F2` | Hintergrund hell |
| `--sg-night` | `#1B1220` | Hintergrund dunkel |
| `--sg-ink` | `#24161F` | Text hell-Theme |
| `--sg-gradient` | `linear-gradient(135deg, #FF5A36 0%, #FF8A3D 45%, #FFB627 100%)` | Logo, Buttons, Reveal |
| `--sg-radius` | `18px` | Karten/Buttons |
| `--sg-font-logo` | `"Fraunces", Georgia, serif` (Italic, 700) | Wortmarke |
| `--sg-font-ui` | `system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` | UI |

Logo: `logo-mark.svg` (Bildmarke: abgerundetes Quadrat im Verlauf, darin ein Herz mit Schlüsselloch)
und `logo-wordmark.svg` (Bildmarke + „Schatzogram“). Bewusst **keine** Ähnlichkeit zum
Instagram-Logo (keine Kamera-Silhouette, kein Instagram-Verlauf).
Font-Dateien werden in jedem Repo lokal gebündelt (OFL), nie per CDN geladen.
