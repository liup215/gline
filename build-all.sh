#!/usr/bin/env bash
set -euo pipefail

# gline Build Script (CLI + GUI)
# macOS / Linux version
# Usage: ./build-all.sh [-d|--dev] [-s|--skip-bindings] [-o <output>]

DEV_MODE=0
SKIP_BINDINGS=0
OUTPUT="bin/gline"
OUTPUT_GUI="bin/gline-gui"

while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--dev) DEV_MODE=1; shift ;;
    -s|--skip-bindings) SKIP_BINDINGS=1; shift ;;
    -o|--output) OUTPUT="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [-d|--dev] [-s|--skip-bindings] [-o <output>]"
      echo "  -d, --dev            Development mode (skip frontend minification)"
      echo "  -s, --skip-bindings  Skip wails3 bindings generation"
      echo "  -o, --output         CLI output path (default: bin/gline)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "══════════════════════════════════════════"
echo "  gline Build (CLI + GUI)"
echo "══════════════════════════════════════════"

# ───────────────────────────────────────────
# 1. 生成 Wails Bindings
# ───────────────────────────────────────────
echo ""
echo "[1/6] Generating Wails bindings..."
if ! command -v wails3 &> /dev/null; then
  echo "Error: wails3 CLI not found. Install with:"
  echo "  go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
  exit 1
fi

if [[ "$SKIP_BINDINGS" -eq 0 ]]; then
  (
    cd "$SCRIPT_DIR/cmd/gline"
    wails3 generate bindings --ts -d "../../frontend/bindings"
  )
else
  echo "  Skipped (--skip-bindings)"
fi

# ───────────────────────────────────────────
# 2. 构建前端
# ───────────────────────────────────────────
echo ""
echo "[2/6] Building frontend..."
cd "$SCRIPT_DIR/frontend"

if [[ ! -d "node_modules" ]]; then
  echo "  Installing dependencies first..."
  npm install
fi

if [[ "$DEV_MODE" -eq 1 ]]; then
  npm run build:dev
else
  npm run build
fi

# ───────────────────────────────────────────
# 3. 同步到 Go embed 目录 (CLI + GUI)
# ───────────────────────────────────────────
echo ""
echo "[3/6] Syncing frontend to embed paths..."
FRONTEND_DIST="$SCRIPT_DIR/frontend/dist"

for dst_rel in "cmd/gline/frontend/dist" "cmd/gline-gui/frontend/dist"; do
  dst="$SCRIPT_DIR/$dst_rel"
  mkdir -p "$dst"
  rm -rf "$dst/"*
  cp -r "$FRONTEND_DIST/"* "$dst/"
  echo "  -> $dst_rel"
done

# Copy icon for the GUI entry point
mkdir -p "$SCRIPT_DIR/cmd/gline-gui/build"
cp -f "$SCRIPT_DIR/cmd/gline/build/appicon.png" "$SCRIPT_DIR/cmd/gline-gui/build/appicon.png"
echo "  -> cmd/gline-gui/build/appicon.png"

# ───────────────────────────────────────────
# 4. 编译 CLI 应用
# ───────────────────────────────────────────
echo ""
echo "[4/6] Building CLI binary -> $OUTPUT ..."
cd "$SCRIPT_DIR"

version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
build_time=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "  Version: $version ($commit)"

ldflags="-X github.com/liup215/gline/internal/version.Version=$version \
  -X github.com/liup215/gline/internal/version.Commit=$commit \
  -X github.com/liup215/gline/internal/version.BuildTime=$build_time \
  -s -w"

mkdir -p "$(dirname "$OUTPUT")"

# macOS/Linux: no -H=windowsgui needed
go build -ldflags "$ldflags" -o "$OUTPUT" ./cmd/gline

# ───────────────────────────────────────────
# 5. 编译 GUI 应用 (standalone, no console)
# ───────────────────────────────────────────
echo ""
echo "[5/6] Building GUI binary -> $OUTPUT_GUI ..."

go build -tags gui -ldflags "$ldflags" -o "$OUTPUT_GUI" ./cmd/gline-gui

# ───────────────────────────────────────────
# 6. 验证
# ───────────────────────────────────────────
echo ""
echo "[6/6] Verifying binaries..."
if [[ ! -f "$OUTPUT" ]]; then
  echo "Error: CLI binary not found!"
  exit 1
fi
if [[ ! -f "$OUTPUT_GUI" ]]; then
  echo "Error: GUI binary not found!"
  exit 1
fi

cli_size=$(du -sh "$OUTPUT" | cut -f1)
gui_size=$(du -sh "$OUTPUT_GUI" | cut -f1)

echo ""
echo "══════════════════════════════════════════"
echo "  Build Success!"
echo "  CLI  : $OUTPUT  ($cli_size)"
echo "  GUI  : $OUTPUT_GUI  ($gui_size)"
echo "  Version: $version ($commit)"
echo "══════════════════════════════════════════"
