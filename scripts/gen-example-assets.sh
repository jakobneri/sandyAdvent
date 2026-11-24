#!/usr/bin/env bash
# Erzeugt die kleinen Dummy-Assets in doors/assets-example/ (werden committet).
# Echte Inhalte gehören NIE hierher, sondern nach doors/assets/ (gitignored).
set -euo pipefail
cd "$(dirname "$0")/.."
OUT=doors/assets-example
mkdir -p "$OUT"
command -v ffmpeg >/dev/null || { echo "ffmpeg fehlt – scripts/dev-setup.sh ausführen" >&2; exit 1; }

FONT=""
for f in /usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf /usr/share/fonts/truetype/freefont/FreeSerifBold.ttf; do
  [ -f "$f" ] && FONT="$f" && break
done
text() { # text <label> <size>
  if [ -n "$FONT" ]; then
    echo ",drawtext=fontfile=$FONT:text='$1':fontcolor=0xFFF8EC:fontsize=$2:x=(w-text_w)/2:y=(h-text_h)/2:shadowcolor=0x00000088:shadowx=3:shadowy=3"
  fi
}
FF=(ffmpeg -hide_banner -loglevel error -y)

photo() { # photo <n> <WxH> <farbe>
  "${FF[@]}" -f lavfi -i "color=c=$3:s=$2:d=1" \
    -vf "vignette=PI/4$(text "Foto $1" 120)" -frames:v 1 -q:v 5 "$OUT/foto-$1.jpg"
}
photo 1 1200x900 0x7f1628
photo 2 1200x900 0x1e4d40
photo 3 900x1200 0xc98f2b
photo 4 1200x900 0x3b1f4a
photo 5 1200x900 0x2a6553

# ~5 s Ton (Akkord) als AAC in .m4a – iOS-sicher
"${FF[@]}" -f lavfi -i "sine=frequency=440:duration=5" -f lavfi -i "sine=frequency=554.37:duration=5" \
  -f lavfi -i "sine=frequency=659.25:duration=5" \
  -filter_complex "amix=inputs=3,afade=t=in:d=0.3,afade=t=out:st=4.3:d=0.7,volume=0.8" \
  -c:a aac -b:a 64k -movflags +faststart "$OUT/ton.m4a"

# ~5 s Video, H.264 (High, yuv420p) + AAC, faststart – iOS-sicher
"${FF[@]}" -f lavfi -i "testsrc2=s=640x360:r=25:d=5" -f lavfi -i "sine=frequency=523.25:duration=5" \
  -vf "format=yuv420p$(text "Video" 72)" -c:v libx264 -profile:v high -preset veryfast -crf 32 \
  -c:a aac -b:a 64k -shortest -movflags +faststart "$OUT/video.mp4"
"${FF[@]}" -ss 1 -i "$OUT/video.mp4" -frames:v 1 -q:v 5 "$OUT/video-poster.jpg"

# --- Spiele (M2): 12 Motive für Memory/Quiz + ein strukturiertes Puzzle-Bild ---
# SVG → JPG über ffmpeg (librsvg); fehlt das, über vips.
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
svg2jpg() { # svg2jpg <in.svg> <out.jpg>
  if ffmpeg -hide_banner -loglevel error -y -i "$1" -frames:v 1 -q:v 4 "$2" 2>/dev/null; then return; fi
  command -v vips >/dev/null || { echo "weder ffmpeg mit librsvg noch vips verfügbar" >&2; exit 1; }
  vips copy "$1" "$2[Q=82,strip]"
}
STAR='<path d="M240 70l45 92 101 15-73 71 17 101-90-48-90 48 17-101-73-71 101-15z"/>'
HEART='<path d="M240 380C120 300 80 240 80 180c0-50 40-90 88-90 32 0 56 16 72 42 16-26 40-42 72-42 48 0 88 40 88 90 0 60-40 120-160 200z"/>'
TREE='<path d="M240 60l110 150h-50l80 110H100l80-110h-50z"/><rect x="215" y="320" width="50" height="60"/>'
MOON='<mask id="crescent"><rect width="480" height="480" fill="#fff"/><circle cx="295" cy="200" r="112" fill="#000"/></mask><circle cx="232" cy="248" r="138" mask="url(#crescent)"/>'
GIFT='<rect x="100" y="190" width="280" height="190" rx="14"/><rect x="85" y="140" width="310" height="60" rx="12"/><rect x="225" y="140" width="30" height="240" fill="#00000033"/>'
BALL='<circle cx="240" cy="255" r="130"/><rect x="215" y="100" width="50" height="40" rx="8"/><path d="M118 230h244M112 280h256" stroke="#00000030" stroke-width="18" fill="none"/>'
FLAKE='<g stroke-width="26" stroke-linecap="round" fill="none" stroke="currentColor"><path d="M240 80v320M101 160l278 160M101 320l278-160"/><path d="M200 100l40 40 40-40M200 380l40-40 40 40"/></g>'
BELL='<path d="M240 80c-70 0-110 55-110 130v80l-40 50h300l-40-50v-80c0-75-40-130-110-130z"/><circle cx="240" cy="380" r="30"/>'
CANDLE='<rect x="190" y="190" width="100" height="200" rx="10"/><path d="M240 80c30 40 40 70 0 100-40-30-30-60 0-100z" fill="#f0c66e"/>'
SNOWMAN='<circle cx="240" cy="310" r="95"/><circle cx="240" cy="160" r="65"/><circle cx="218" cy="150" r="8" fill="#2b1e17"/><circle cx="262" cy="150" r="8" fill="#2b1e17"/>'
CANE='<path d="M290 390V170a60 60 0 0 0-120 0" fill="none" stroke="currentColor" stroke-width="44" stroke-linecap="round"/>'
CUP='<path d="M120 170h210v110a100 100 0 0 1-100 100h-10a100 100 0 0 1-100-100z"/><path d="M330 200h20a40 40 0 0 1 0 80h-20" fill="none" stroke="currentColor" stroke-width="22"/>'
motif() { # motif <nr> <hintergrund> <vordergrund> <form>
  local f="$TMP/m$1.svg"
  cat >"$f" <<SVG
<svg xmlns="http://www.w3.org/2000/svg" width="480" height="480" viewBox="0 0 480 480">
<defs><radialGradient id="g" cx="50%" cy="35%" r="75%"><stop offset="0" stop-color="#ffffff" stop-opacity=".28"/><stop offset="1" stop-color="#000000" stop-opacity=".25"/></radialGradient></defs>
<rect width="480" height="480" fill="$2"/><rect width="480" height="480" fill="url(#g)"/>
<g fill="$3" color="$3">$4</g>
<circle cx="404" cy="404" r="52" fill="#fffaf1"/>
<text x="404" y="424" font-family="DejaVu Serif, serif" font-weight="bold" font-size="56" fill="#2b1e17" text-anchor="middle">$1</text>
</svg>
SVG
  svg2jpg "$f" "$OUT/motiv-$(printf %02d "$1").jpg"
}
motif 1  "#a11f36" "#f9e3b0" "$STAR"
motif 2  "#1e4d40" "#ff8a8a" "$HEART"
motif 3  "#0f2c24" "#7fd1a8" "$TREE"
motif 4  "#3b1f4a" "#ffd36e" "$MOON"
motif 5  "#c98f2b" "#7f1628" "$GIFT"
motif 6  "#14506e" "#ff6b6b" "$BALL"
motif 7  "#5b6f8a" "#ffffff" "$FLAKE"
motif 8  "#6b2f1a" "#ffcc4d" "$BELL"
motif 9  "#2a6553" "#fff1d6" "$CANDLE"
motif 10 "#7a1f5c" "#ffffff" "$SNOWMAN"
motif 11 "#f2e3c6" "#c23447" "$CANE"
motif 12 "#24161f" "#ff9a5c" "$CUP"

# Puzzle-Bild 800×600: Winterlandschaft mit klar unterscheidbaren Bereichen
cat >"$TMP/puzzle.svg" <<'SVG'
<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600">
<defs>
<linearGradient id="sky" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#0b1f3a"/><stop offset=".55" stop-color="#3b1f4a"/><stop offset="1" stop-color="#a11f36"/></linearGradient>
<radialGradient id="moon" cx="50%" cy="50%" r="50%"><stop offset="0" stop-color="#fff8dc"/><stop offset=".7" stop-color="#f9e3b0"/><stop offset="1" stop-color="#f9e3b0" stop-opacity="0"/></radialGradient>
<linearGradient id="snow" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ffffff"/><stop offset="1" stop-color="#c9d8e8"/></linearGradient>
</defs>
<rect width="800" height="600" fill="url(#sky)"/>
<circle cx="640" cy="120" r="90" fill="url(#moon)"/><circle cx="640" cy="120" r="52" fill="#fff6d0"/>
<g fill="#ffffff"><circle cx="80" cy="60" r="3"/><circle cx="190" cy="120" r="2"/><circle cx="300" cy="50" r="3"/><circle cx="420" cy="140" r="2"/><circle cx="520" cy="40" r="3"/><circle cx="740" cy="260" r="2"/><circle cx="120" cy="200" r="2"/><circle cx="360" cy="220" r="2"/></g>
<path d="M0 330 Q200 250 400 320 T800 300 V600 H0z" fill="#2a6553"/>
<path d="M0 400 Q220 330 430 400 T800 380 V600 H0z" fill="url(#snow)"/>
<g fill="#0f2c24"><path d="M90 400l50-130 50 130z"/><path d="M60 430l60-160 60 160z"/><path d="M680 390l45-120 45 120z"/><path d="M720 420l40-100 40 100z"/></g>
<g transform="translate(300 250)">
<rect x="0" y="80" width="200" height="150" fill="#7f1628"/><path d="M-20 90L100 0l120 90z" fill="#5e0f1d"/><path d="M-20 90L100 0l120 90" fill="none" stroke="#ffffff" stroke-width="14" stroke-linejoin="round"/>
<rect x="30" y="120" width="44" height="44" fill="#f0c66e"/><rect x="126" y="120" width="44" height="44" fill="#f0c66e"/><rect x="80" y="170" width="40" height="60" fill="#3b1f0f"/>
<path d="M52 120v44M30 142h44M148 120v44M126 142h44" stroke="#7f1628" stroke-width="4"/>
</g>
<g transform="translate(560 430)"><circle cx="0" cy="40" r="46" fill="#ffffff"/><circle cx="0" cy="-20" r="30" fill="#ffffff"/><circle cx="-10" cy="-25" r="4" fill="#24161f"/><circle cx="10" cy="-25" r="4" fill="#24161f"/><path d="M0-17l22 6-22 4z" fill="#ff8a3d"/><rect x="-30" y="-58" width="60" height="10" fill="#24161f"/><rect x="-20" y="-90" width="40" height="34" fill="#24161f"/></g>
<text x="400" y="560" font-family="DejaVu Serif, serif" font-weight="bold" font-style="italic" font-size="58" fill="#a11f36" text-anchor="middle">Frohe Weihnachten</text>
<g fill="#f0c66e"><path d="M40 520l8 16 18 3-13 12 3 18-16-9-16 9 3-18-13-12 18-3z"/><path d="M760 520l8 16 18 3-13 12 3 18-16-9-16 9 3-18-13-12 18-3z"/></g>
</svg>
SVG
svg2jpg "$TMP/puzzle.svg" "$OUT/puzzle.jpg"

du -ch "$OUT"/* | tail -1
