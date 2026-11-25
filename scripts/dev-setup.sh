#!/usr/bin/env bash
# sandyAdvent – Systemabhängigkeiten für die Entwicklung (Debian/Ubuntu, amd64/arm64).
# Idempotent: kann beliebig oft laufen. Gleicher Aufbau wie scripts/dev-setup.sh in schatzogram.
#
#   scripts/dev-setup.sh              # apt-Pakete + Go
#   WITH_NODE=1 scripts/dev-setup.sh  # zusätzlich Node.js (nur zum Aktualisieren der Fonts nötig)
#   GO_VERSION=1.24.7 scripts/dev-setup.sh
set -euo pipefail

GO_VERSION="${GO_VERSION:-1.24.7}"   # wird installiert, falls Go fehlt oder älter als GO_MIN ist
GO_MIN="${GO_MIN:-1.24}"
NODE_MAJOR="${NODE_MAJOR:-22}"
WITH_NODE="${WITH_NODE:-0}"
APT_PACKAGES=(ffmpeg libvips-tools libheif-plugin-libde265 rclone sqlite3 poppler-utils ca-certificates curl make git)

log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  command -v sudo >/dev/null || die "Bitte als root ausführen oder sudo installieren."
  SUDO="sudo"
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "Nicht unterstützte Architektur: $(uname -m)" ;;
esac

# version_ge A B  -> true, wenn A >= B (Punktnotation)
version_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]; }

install_apt() {
  command -v apt-get >/dev/null || { warn "apt-get nicht gefunden – Pakete bitte manuell installieren: ${APT_PACKAGES[*]}"; return; }
  local missing=()
  for p in "${APT_PACKAGES[@]}"; do
    dpkg -s "$p" >/dev/null 2>&1 || missing+=("$p")
  done
  if [ ${#missing[@]} -eq 0 ]; then
    log "apt-Pakete vorhanden"
    return
  fi
  log "Installiere: ${missing[*]}"
  $SUDO apt-get update -qq
  # libheif-plugin-libde265 gibt es erst ab Debian 13 / Ubuntu 24.04 – fehlt es, nicht abbrechen.
  if ! $SUDO env DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends "${missing[@]}"; then
    warn "Sammelinstallation fehlgeschlagen – versuche Pakete einzeln"
    for p in "${missing[@]}"; do
      $SUDO env DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends "$p" || warn "Paket $p nicht installierbar (optional?)"
    done
  fi
}

go_version() { go version 2>/dev/null | sed -n 's/.*go\([0-9][0-9.]*\).*/\1/p'; }

install_go() {
  export PATH="/usr/local/go/bin:$PATH"
  local have; have="$(go_version || true)"
  if [ -n "$have" ] && version_ge "$have" "$GO_MIN"; then
    log "Go $have vorhanden (>= $GO_MIN)"
    return
  fi
  log "Installiere Go $GO_VERSION ($ARCH)${have:+ (vorhanden: $have)}"
  local tmp; tmp="$(mktemp -d)"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" -o "$tmp/go.tgz"
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf "$tmp/go.tgz"
  rm -rf "$tmp"
  if ! grep -qs '/usr/local/go/bin' /etc/profile.d/go.sh; then
    echo 'export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"' | $SUDO tee /etc/profile.d/go.sh >/dev/null
  fi
  log "Go $(go_version) installiert – neue Shell öffnen oder: export PATH=/usr/local/go/bin:\$PATH"
}

install_node() {
  if [ "$WITH_NODE" != "1" ]; then
    log "Node.js übersprungen (für sandyAdvent nicht nötig; WITH_NODE=1 zum Installieren)"
    return
  fi
  if command -v node >/dev/null && version_ge "$(node -v | tr -d v)" "$NODE_MAJOR"; then
    log "Node $(node -v) vorhanden"
    return
  fi
  log "Installiere Node.js $NODE_MAJOR (NodeSource)"
  curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | $SUDO -E bash -
  $SUDO apt-get install -y -qq nodejs
}

check() {
  local ok=1
  for c in go ffmpeg vips vipsthumbnail rclone sqlite3; do
    if command -v "$c" >/dev/null; then printf '   %-14s %s\n' "$c" "ok"; else printf '   %-14s %s\n' "$c" "FEHLT"; ok=0; fi
  done
  if command -v vips >/dev/null && ! grep -qi heifload <<<"$(vips -l 2>/dev/null)"; then
    warn "vips ohne HEIF-Unterstützung (libheif-plugin-libde265 fehlt?)"
  fi
  [ $ok -eq 1 ] && log "Fertig. Weiter mit: make dev" || warn "Nicht alles installiert – siehe oben."
}

install_apt
install_go
install_node
check
