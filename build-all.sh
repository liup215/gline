#!/usr/bin/env bash
set -euo pipefail

# gline 统一构建脚本 (CLI + GUI)
# macOS / Linux 版本
# Usage: ./build-all.sh [-d|--dev] [-s|--skip-bindings] [-o <output>]

DEV_MODE=0
SKIP_BINDINGS=0
OUTPUT="bin/gline"

while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--dev) DEV_MODE=1; shift ;;
    -s|--skip-bindings) SKIP_BINDINGS=1; shift ;;
    -o|--output) OUTPUT="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [-d|--dev] [-s|--skip-bindings] [-o <output>]"
      echo "  -d, --dev            Development mode (skip frontend minification)"
      echo "  -s, --skip-bindings  Skip wails3 bindings generation"
      echo "  -o, --output         Output binary path (default: bin/gline)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "═══════════════════════════════════════════"
echo "  gline Unified Build (CLI + GUI)"
echo "═══════════════════════════════════════════"

# ───────────────────────────────────────────
# 1. 生成 Wails Bindings
# ───────────────────────────────────────────
echo ""
echo "[1/5] Generating Wails bindings..."
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
echo "[2/5] Building frontend..."
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
# 3. 同步到 Go embed 目录
# ───────────────────────────────────────────
echo ""
echo "[3/5] Syncing frontend to embed path..."
mkdir -p "$SCRIPT_DIR/cmd/gline/frontend/dist"
rm -rf "$SCRIPT_DIR/cmd/gline/frontend/dist/"*
cp -r "$SCRIPT_DIR/frontend/dist/"* "$SCRIPT_DIR/cmd/gline/frontend/dist/"
echo "  Copied to cmd/gline/frontend/dist"

# ───────────────────────────────────────────
# 4. 编译 Go 应用
# ───────────────────────────────────────────
echo ""
echo "[4/5] Building Go binary -> $OUTPUT ..."
cd "$SCRIPT_DIR"

version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
build_time=$(date -u '+%Y-%m-%d_%H:%M:%S')

ldflags="-X github.com/liup215/gline/internal/version.Version=$version \
  -X github.com/liup215/gline/internal/version.Commit=$commit \
  -X github.com/liup215/gline/internal/version.BuildTime=$build_time \
  -s -w"

mkdir -p "$(dirname "$OUTPUT")"

# macOS/Linux: no -H=windowsgui needed
# macOS requires CGO for Wails WebKit → use default CGO_ENABLED
go build -ldflags "$ldflags" -o "$OUTPUT" ./cmd/gline

# ───────────────────────────────────────────
# 5. 验证
# ───────────────────────────────────────────
echo ""
echo "[5/5] Verifying binary..."
if [[ ! -f "$OUTPUT" ]]; then
  echo "Error: Binary not found!"
  exit 1
fi

size=$(du -sh "$OUTPUT" | cut -f1)

echo ""
echo "═══════════════════════════════════════════"
echo "  Build Success!"
echo "  Binary : $OUTPUT"
echo "  Size   : $size"
echo "  Version: $version ($commit)"
echo "═══════════════════════════════════════════"
