#!/usr/bin/env bash
# 規格 017 收據：cok-play 冷開機跑 demo 戰鬥（不送鍵，-clock 100，約 1.7e9 步），無敵關與開各一次。
# 要求：開啟時有還原、隊員 HP 與狀態不低於開戰前；關閉時不印無敵報告；兩次都沒有缺字。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/workplace/dosgolem/workplace/bin"; ORIG="$ROOT/workplace/input/dos/ckrynn"; FONT="$ROOT/workplace/font"
OUT="$ROOT/workplace/probe/g10-cheat"; FRAMES=${FRAMES:-8840}; IMAGE=dosgolem-go-ebiten:1.24-r1
for f in "$BIN/cok-play" "$FONT/cok-eten.golemfnt" "$ORIG/START.EXE"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
rm -rf "$OUT"; mkdir -p "$OUT"
run() { # 名稱 旗標…
  local name=$1; shift; mkdir -p "$OUT/$name"
  timeout 40m docker run --rm --init --network none --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp -e COK_NO_DIALOG=1 \
    -v "$BIN":/app:ro -v "$ORIG":/orig:ro -v "$ROOT/text":/text:ro -v "$FONT":/font:ro -v "$OUT/$name":/out \
    --entrypoint xvfb-run "$IMAGE" -a /app/cok-play -original /orig -save /out/save -text-dir /text \
    -font /font/cok-eten.golemfnt -audio=false -clock 100 -frames "$FRAMES" -shot /out/shot.rgba "$@" > "$OUT/$name/log.txt" 2>&1
}
run off & run on -cheat; wait
grep -h "無敵\|memory_sha256" "$OUT"/*/log.txt
if grep -l "缺字" "$OUT"/*/log.txt; then echo "有缺字" >&2; exit 1; fi
grep -q "無敵" "$OUT/off/log.txt" && { echo "關閉時不該有無敵報告" >&2; exit 1; }
line=$(grep "無敵" "$OUT/on/log.txt") || { echo "開啟時沒有無敵報告" >&2; exit 1; }
n=$(sed -E 's/.*還原 ([0-9]+) 次.*/\1/' <<< "$line"); [[ "$n" -gt 0 ]] || { echo "開啟時沒有任何還原" >&2; exit 1; }
grep -q "/0；" <<< "$line" && ! grep -qE "/[1-9][0-9]*；" <<< "$line" || { echo "有隊員狀態異常" >&2; exit 1; }
echo "通過"
