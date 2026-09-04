#!/usr/bin/env bash
# Regenerate Windows .syso resources (exe icon + manifest) from assets/app_icon/icon.png.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
V="$(tr -d '[:space:]' < VERSION)"
ICON="$ROOT/assets/app_icon/icon.png"
mkdir -p winres
for px in 256 128 64 48 32 16; do
  sips -z "$px" "$px" "$ICON" --out "winres/icon_${px}.png" >/dev/null
done
go run github.com/tc-hib/go-winres@v0.3.3 make \
  --in winres/winres.json \
  --arch amd64,386 \
  --product-version "$V" \
  --file-version "$V"
ls -la rsrc_windows_*.syso
