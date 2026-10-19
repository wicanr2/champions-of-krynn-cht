#!/usr/bin/env bash
# 開《克萊恩英豪》繁中覆繪視窗（dosgolem cmd/cok-play）。
# 編譯與執行都在 Docker 內；視窗透過主機 X11（DISPLAY、XAUTHORITY）顯示。
# 存檔留在 workplace/play/save/（第一次由原版複製，之後沿用）；F10 截圖在 workplace/play/screenshots。
# HD 圖（規格 010）：有 workplace/hd/theme/theme.json（tools/hd_theme.py 產生）就自動載入，只在 3× 顯示。
# 按鍵：F2 2×／3×、F12 中文／原版切換、F11 全螢幕、F10 截圖、F9 查驗題答案、F5 手札翻閱、F6 攻略、F7 地圖（V 全圖）。
# 滑鼠移到視窗頂端出現切換列（規格 011），點項目等同按對應的鍵。容器內沒有音訊裝置，預設無聲。
# 其餘參數轉給 cok-play。印字路徑盤點：tools/play.sh -inventory /data/inventory.tsv，
# 關窗時寫到 workplace/play/inventory.tsv（含原文，不進 Git）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
ORIG="$ROOT/workplace/input/dos/ckrynn"
FONT="$ROOT/workplace/font/cok-eten.golemfnt"
DATA="$ROOT/workplace/play"
THEME="$ROOT/workplace/hd/theme"
IMAGE=dosgolem-go-ebiten:1.24-r1
test -f "$ORIG/START.EXE" || { echo "缺原版（先依 docs/re/001 解壓到 $ORIG）" >&2; exit 1; }
test -f "$FONT" || { echo "缺字型（見 font/README.md）" >&2; exit 1; }
: "${DISPLAY:?需要主機 X11 顯示（DISPLAY）}"
XAUTH="${XAUTHORITY:-$HOME/.Xauthority}"
test -f "$XAUTH" || { echo "找不到 X 授權檔 $XAUTH" >&2; exit 1; }
test -d /tmp/.X11-unix || { echo "找不到 /tmp/.X11-unix" >&2; exit 1; }
mkdir -p "$DATA"
HD=()
if test -f "$THEME/theme.json"; then HD=(-v "$THEME":/hd:ro); HDFLAG=(-hd /hd); else HDFLAG=(); fi
(cd "$DG" && DOSGOLEM_TIMEOUT=10m tools/go.sh build -o /src/workplace/bin/cok-play ./cmd/cok-play)
test -x "$DG/workplace/bin/cok-play"
exec timeout 12h docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e DISPLAY="$DISPLAY" -e XAUTHORITY=/tmp/xauth -e HOME=/tmp \
  -v /tmp/.X11-unix:/tmp/.X11-unix:ro -v "$XAUTH":/tmp/xauth:ro \
  -v "$DG/workplace/bin":/app:ro -v "$ORIG":/orig:ro -v "$ROOT/text":/text:ro \
  -v "$ROOT/workplace/font":/font:ro "${HD[@]}" -v "$DATA":/data \
  --entrypoint /app/cok-play "$IMAGE" \
  -original /orig -save /data/save -text-dir /text -font /font/cok-eten.golemfnt -audio=false "${HDFLAG[@]}" "$@"
