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

du -ch "$OUT"/* | tail -1
