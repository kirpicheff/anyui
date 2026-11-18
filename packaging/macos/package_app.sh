#!/usr/bin/env bash
set -euo pipefail

# Сборка universal binary (arm64 + amd64) и упаковка AnyUI.app для macOS.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build"
APP_DIR="${BUILD_DIR}/AnyUI.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
ZIP_PATH="${BUILD_DIR}/AnyUI-macos-universal.zip"

rm -rf "${APP_DIR}" "${ZIP_PATH}" "${BUILD_DIR}/anyui-demo-arm64" "${BUILD_DIR}/anyui-demo-amd64"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"

echo "Сборка ./cmd/demo для darwin/arm64 (CGO_ENABLED=1)..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "${BUILD_DIR}/anyui-demo-arm64" "${ROOT_DIR}/cmd/demo"

echo "Сборка ./cmd/demo для darwin/amd64 (CGO_ENABLED=1)..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o "${BUILD_DIR}/anyui-demo-amd64" "${ROOT_DIR}/cmd/demo"

echo "Объединение бинарников через lipo..."
lipo -create \
  -output "${MACOS_DIR}/AnyUI" \
  "${BUILD_DIR}/anyui-demo-arm64" \
  "${BUILD_DIR}/anyui-demo-amd64"

chmod +x "${MACOS_DIR}/AnyUI"
lipo -info "${MACOS_DIR}/AnyUI"

cp "${ROOT_DIR}/packaging/macos/Info.plist" "${CONTENTS_DIR}/Info.plist"
printf 'APPL????' > "${CONTENTS_DIR}/PkgInfo"

echo "Ad-hoc подпись AnyUI.app..."
codesign --force --deep --sign - "${APP_DIR}"

echo "Архивация в ${ZIP_PATH}..."
ditto -c -k --sequesterRsrc --keepParent "${APP_DIR}" "${ZIP_PATH}"
ls -lh "${ZIP_PATH}"
