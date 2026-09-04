#!/usr/bin/env bash
# Build SSN688.app — launches without a Terminal window on macOS, with Dock icon.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP_NAME="SSN688"
APP_DIR="$ROOT/dist/${APP_NAME}.app"
BIN="$APP_DIR/Contents/MacOS/${APP_NAME}"
ICON_SRC="$ROOT/assets/app_icon/icon.png"
RES_DIR="$APP_DIR/Contents/Resources"
ICONSET="$ROOT/dist/${APP_NAME}.iconset"
VER="$(tr -d '[:space:]' < "$ROOT/VERSION")"

mkdir -p "$APP_DIR/Contents/MacOS" "$RES_DIR"

echo "Sync VERSION ..."
cp "$ROOT/VERSION" "$ROOT/internal/version/VERSION"

echo "Building $BIN ..."
(cd "$ROOT" && go build -o "$BIN" .)
chmod +x "$BIN"

echo "Building icon.icns ..."
rm -rf "$ICONSET"
mkdir -p "$ICONSET"
# iconutil expects specific names
for px in 16 32 128 256 512; do
  sips -z "$px" "$px" "$ICON_SRC" --out "$ICONSET/icon_${px}x${px}.png" >/dev/null
  sips -z $((px * 2)) $((px * 2)) "$ICON_SRC" --out "$ICONSET/icon_${px}x${px}@2x.png" >/dev/null
done
# 1024 for App Store-style completeness
sips -z 512 512 "$ICON_SRC" --out "$ICONSET/icon_512x512.png" >/dev/null
sips -z 1024 1024 "$ICON_SRC" --out "$ICONSET/icon_512x512@2x.png" >/dev/null
iconutil -c icns "$ICONSET" -o "$RES_DIR/AppIcon.icns"
rm -rf "$ICONSET"

cat > "$APP_DIR/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>SSN688</string>
	<key>CFBundleIdentifier</key>
	<string>com.ssn688.sim</string>
	<key>CFBundleName</key>
	<string>SSN688</string>
	<key>CFBundleDisplayName</key>
	<string>SSN-688 Modern Submarine Combat Simulator</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>CFBundleShortVersionString</key>
	<string>${VER}</string>
	<key>CFBundleVersion</key>
	<string>${VER}</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
</dict>
</plist>
PLIST

echo "Done: $APP_DIR"
echo "Launch: open \"$APP_DIR\""
