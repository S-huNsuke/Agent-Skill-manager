#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_NAME="agent-skills-manager"
BUILD_DIR="${PROJECT_DIR}/build/bin"
APP_BUNDLE="${BUILD_DIR}/${BIN_NAME}.app"

echo "=== Building Agent Skills Manager ==="

echo "[1/4] Building frontend..."
pnpm --dir "${PROJECT_DIR}/frontend" install
pnpm --dir "${PROJECT_DIR}/frontend" build

echo "[2/4] Compiling Go binary..."
mkdir -p "${BUILD_DIR}"
CGO_ENABLED=1 \
  CGO_CFLAGS="-mmacosx-version-min=10.13" \
  CGO_LDFLAGS="-framework UniformTypeIdentifiers -mmacosx-version-min=10.13" \
  GOOS=darwin GOARCH=arm64 \
  go build -buildvcs=false \
    -tags "desktop,wv2runtime.download,production" \
    -ldflags "-w -s" \
    -o "${BUILD_DIR}/${BIN_NAME}" \
    ./cmd/agent-skills-manager/

chmod +x "${BUILD_DIR}/${BIN_NAME}"
echo "  Binary: $(file "${BUILD_DIR}/${BIN_NAME}")"

echo "[3/4] Packaging .app bundle..."
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${APP_BUNDLE}/Contents/Resources"

cp "${BUILD_DIR}/${BIN_NAME}" "${APP_BUNDLE}/Contents/MacOS/${BIN_NAME}"

cat > "${APP_BUNDLE}/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>agent-skills-manager</string>
    <key>CFBundleIdentifier</key>
    <string>com.wails.agent-skills-manager</string>
    <key>CFBundleName</key>
    <string>Agent Skills Manager</string>
    <key>CFBundleDisplayName</key>
    <string>Agent Skills Manager</string>
    <key>CFBundleVersion</key>
    <string>0.1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>0.1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleIconFile</key>
    <string>iconfile</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright 2026. All rights reserved.</string>
</dict>
</plist>
PLIST

if [ -f "${PROJECT_DIR}/build/appicon.icns" ]; then
  cp "${PROJECT_DIR}/build/appicon.icns" "${APP_BUNDLE}/Contents/Resources/iconfile.icns"
fi

echo "[4/4] Signing .app bundle..."
xattr -cr "${APP_BUNDLE}" 2>/dev/null || true
codesign --force --deep --sign - "${APP_BUNDLE}" 2>/dev/null || echo "  Warning: codesign failed (non-fatal for development)"

echo ""
echo "=== Build complete ==="
echo "  Binary:  ${BUILD_DIR}/${BIN_NAME}"
echo "  Bundle:  ${APP_BUNDLE}"
echo ""
echo "Run with:"
echo "  ${BUILD_DIR}/${BIN_NAME}"
echo "  open ${APP_BUNDLE}"
