#!/usr/bin/env bash
# 切換列收據（幽靈騎士規格 011 §3 第 3 項的本作版）：cok-play 自動模式冷開機跑到標題畫面，以腳本動作 bar 讓切換列顯示，
# 與不顯示的同一畫格比對：切換列（前 12 邏輯列）以外逐像素相同，切換列內有畫。本作尚無 HD 主題，只驗 2 倍、3 倍無 HD（顯示「無HD素材」）。
# docker run 帶 --init：xvfb-run 當 PID 1 時等不到 Xvfb 的就緒訊號會一直卡住。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
ORIG="$ROOT/workplace/input/dos/ckrynn"
OUT="$ROOT/workplace/probe/g7-toolbar-receipt"
FRAMES=${FRAMES:-700}
IMAGE=dosgolem-go-ebiten:1.24-r1
for f in "$ORIG/START.EXE" "$ROOT/workplace/font/cok-eten.golemfnt"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
rm -rf "$OUT"; mkdir -p "$OUT"
(cd "$DG" && DOSGOLEM_TIMEOUT=10m tools/go.sh build -o /src/workplace/bin/cok-play ./cmd/cok-play)
run() { # 名稱 腳本 旗標…
  local name=$1 script=$2; shift 2
  mkdir -p "$OUT/$name"
  timeout 10m docker run --rm --init --network none --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp -e COK_NO_DIALOG=1 \
    -v "$DG/workplace/bin":/app:ro -v "$ORIG":/orig:ro -v "$ROOT/text":/text:ro \
    -v "$ROOT/workplace/font":/font:ro -v "$OUT/$name":/out \
    --entrypoint xvfb-run "$IMAGE" -a /app/cok-play -original /orig -save /out/save -text-dir /text \
    -font /font/cok-eten.golemfnt -audio=false -frames "$FRAMES" ${script:+-script "$script"} -shot /out/shot.rgba "$@" \
    > "$OUT/$name/log.txt" 2>&1
}
run nohd-2-bar 1:bar -scale 2 &
run nohd-2 "" -scale 2 &
run nohd-3-bar 1:bar -scale 3 &
run nohd-3 "" -scale 3 &
wait
if grep -l "缺字" "$OUT"/*/log.txt; then echo "有缺字" >&2; exit 1; fi
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$OUT":/r cok-img:r1 python -I - <<'PY'
import sys
import numpy as np
from PIL import Image
ok = True
for name, sc in (('nohd-2', 2), ('nohd-3', 3)):
    a = np.fromfile(f'/r/{name}-bar/shot.rgba', np.uint8).reshape(200 * sc, 320 * sc, 4)
    b = np.fromfile(f'/r/{name}/shot.rgba', np.uint8).reshape(200 * sc, 320 * sc, 4)
    below = int((a[12 * sc:] != b[12 * sc:]).any(axis=2).sum())
    inside = int((a[:12 * sc] != b[:12 * sc]).any(axis=2).sum())
    good = below == 0 and inside > 0
    ok &= good
    print(f'{name}：切換列外差異 {below} 像素，切換列內改變 {inside} 像素', '通過' if good else '不通過')
    Image.fromarray(a[..., :3]).save(f'/r/{name}-bar.png')
sys.exit(0 if ok else 1)
PY
