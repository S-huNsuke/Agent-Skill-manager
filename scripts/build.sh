#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="${PROJECT_DIR}/build/bin"
BIN_NAME="agent-skills-manager"
APP_BUNDLE="${BUILD_DIR}/agent-skills-manager.app"
APP_MACOS="${APP_BUNDLE}/Contents/MacOS"
APP_RESOURCES="${APP_BUNDLE}/Contents/Resources"
APP_BINARY="${APP_MACOS}/${BIN_NAME}"

export GOCACHE="${PROJECT_DIR}/.gocache"
export GOMODCACHE="/private/tmp/go-mod-cache"
export TMPDIR="/private/tmp/agent-skills-manager"
export CI=true
export CGO_ENABLED=1
export CGO_CFLAGS="-mmacosx-version-min=10.13"
export CGO_LDFLAGS="-framework UniformTypeIdentifiers -mmacosx-version-min=10.13"

GO_BIN="$(command -v go || true)"
if [ -z "${GO_BIN}" ] && [ -x "${HOME}/.local/share/mise/installs/go/latest/bin/go" ]; then
  GO_BIN="${HOME}/.local/share/mise/installs/go/latest/bin/go"
fi
if [ -z "${GO_BIN}" ]; then
  echo "go binary not found; install Go or activate mise first" >&2
  exit 127
fi

mkdir -p "${GOCACHE}"
mkdir -p "${TMPDIR}"

echo "=== Building Agent Skills Manager ==="
echo "[1/4] Building frontend..."
pnpm --dir "${PROJECT_DIR}/frontend" install
pnpm --dir "${PROJECT_DIR}/frontend" build

echo "[2/4] Compiling Go binary..."
mkdir -p "${BUILD_DIR}"
"${GO_BIN}" build -buildvcs=false \
  -tags "desktop,wv2runtime.download,production" \
  -ldflags "-w -s" \
  -o "${BUILD_DIR}/${BIN_NAME}" \
  ./cmd/agent-skills-manager/

if [ ! -x "${BUILD_DIR}/${BIN_NAME}" ]; then
  echo "missing app binary: ${BUILD_DIR}/${BIN_NAME}" >&2
  exit 1
fi

echo "[3/4] Packaging .app bundle..."
rm -rf "${APP_BUNDLE}"
mkdir -p "${APP_MACOS}"
mkdir -p "${APP_RESOURCES}"
cp "${BUILD_DIR}/${BIN_NAME}" "${APP_BINARY}"
chmod +x "${APP_BINARY}"
mkdir -p "${APP_RESOURCES}/python"
cp -R "${PROJECT_DIR}/python/worker" "${APP_RESOURCES}/python/worker"
find "${APP_RESOURCES}/python" -type d -name "__pycache__" -prune -exec rm -rf {} +
find "${APP_RESOURCES}/python" -type f \( -name "*.pyc" -o -name "*.pyo" \) -delete

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
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright 2026. All rights reserved.</string>
</dict>
</plist>
PLIST

if [ -f "${PROJECT_DIR}/build/appicon.icns" ]; then
  cp "${PROJECT_DIR}/build/appicon.icns" "${APP_RESOURCES}/iconfile.icns"
fi

xattr -cr "${APP_BUNDLE}" 2>/dev/null || true

echo "[4/4] Signing .app bundle..."
codesign --force --deep --sign - "${APP_BUNDLE}" 2>/dev/null || echo "  Warning: codesign failed (non-fatal for development)"

echo ""
echo "=== Build complete ==="
echo "  Bundle:  ${APP_BUNDLE}"
echo "  Open:    /usr/bin/open -n ${APP_BUNDLE}"
