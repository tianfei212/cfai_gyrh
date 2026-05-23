#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
RELEASE_ROOT="$ROOT_DIR/release"
VERSION="$(date +%Y%m%d%H%M)"
GIT_SHA="$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo local)"
PACKAGE_NAME="gyrh-kiosk-client-${VERSION}-${GIT_SHA}-windows-amd64"
STAGE_DIR="$RELEASE_ROOT/$PACKAGE_NAME"
ARCHIVE_PATH="$RELEASE_ROOT/${PACKAGE_NAME}.zip"

info() { printf '\033[0;32m[INFO]\033[0m %s\n' "$*"; }
fail() { printf '\033[0;31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

require_cmd go
require_cmd zip

rm -rf "$STAGE_DIR" "$ARCHIVE_PATH"
mkdir -p "$STAGE_DIR" "$STAGE_DIR/runtime"

info "构建 Windows kiosk client"
(
  cd "$BACKEND_DIR"
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o "$STAGE_DIR/gyrh-kiosk-client.exe" ./cmd/kiosk-client
)

cp "$ROOT_DIR/configs/kiosk-client.yaml" "$STAGE_DIR/kiosk-client.yaml"

cat > "$STAGE_DIR/README_KIOSK_CLIENT.md" <<'READMEEOF'
# GYRH Windows Kiosk Client

This package launches the Google Chrome installed on Windows in full-screen kiosk mode.

## Files

- `gyrh-kiosk-client.exe`: kiosk launcher.
- `kiosk-client.yaml`: URL and Chrome settings.
- `runtime/`: runtime directory for the isolated Chrome profile.

## Usage

1. Edit `kiosk-client.yaml` and set `url`.
2. Keep `chrome_path` empty to auto-detect Google Chrome.
3. Run `gyrh-kiosk-client.exe`.

The launcher uses Chrome `--kiosk`, so the page opens full-screen without address bar, tabs, or bookmarks.
READMEEOF

info "打包 $ARCHIVE_PATH"
(
  cd "$RELEASE_ROOT"
  zip -qr "$ARCHIVE_PATH" "$PACKAGE_NAME"
)

info "完成: $ARCHIVE_PATH"

