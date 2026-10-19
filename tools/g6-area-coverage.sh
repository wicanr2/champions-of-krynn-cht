#!/usr/bin/env bash
# 跨區中文化覆蓋盤點（Issue #6，docs/re/008、009）：從商隊戰後的大地圖狀態出發，把隊伍生命值改成 250、
# 大地圖座標改到據點左側一格，按 Right 觸發 `YOU ARE NEAR <地名>. DO YOU ENTER?` 答 Y，由遊戲正規載入該區，
# 之後交給 cok-explore 自動探索（戰鬥交電腦、選單優先 Attack），記下每次印字與是否覆繪。
# 用法：tools/g6-area-coverage.sh <輸出目錄> 名稱=x,y…   （例 THROTL=2,1 GARGATH=14,3）
# 環境變數：ACTIONS（每區送鍵上限，預設 1200）、PAR（同時跑幾區，預設 3）。
# 起點 workplace/probe/ovmap/ov.state：explore2/end.state（商隊戰後）送 Return×3、L、N 回到大地圖，第 4440M 步。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
ST="$ROOT/workplace/probe/ovmap"
OUT=$(realpath -m "$1"); shift
ACTIONS=${ACTIONS:-1200} PAR=${PAR:-3}
for f in "$ST/ov.state" "$ROOT/workplace/font/cok-eten.golemfnt"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
test -d "$ST/scratch" || { echo "缺 $ST/scratch" >&2; exit 1; }
T=4440001000
# 角色紀錄（線性位址，ov.state 的記憶體搜尋）：+0x62 生命值上限、+0x197 目前生命值（docs/re/009）
HP=""; for a in 52190 52330 52510 527F0 53B80 53E20; do HP="$HP;$(printf %X $((0x$a + 0x62)))@$T=FA;$(printf %X $((0x$a + 0x197)))@$T=FA"; done
mkdir -p "$OUT"
run() { # 名稱=x,y
  local name=${1%%=*} xy=${1#*=} x y d
  x=${xy%,*} y=${xy#*,} d="$OUT/$name"
  rm -rf "$d"; mkdir -p "$d"; cp -r "$ST/scratch" "$d/scratch"
  (cd "$DG" && DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" \
    DOSGOLEM_EXTRA_MOUNTS="$ST:/st:ro;$d:/out;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro" DOSGOLEM_TIMEOUT=180m \
    timeout 10900 tools/go.sh run ./cmd/cok-explore -state /st/ov.state -root /orig -scratch /out/scratch -font /font/cok-eten.golemfnt -text-dir /text \
      -poke "41B96@$T=$(printf %02X $((x - 1)));41B98@$T=$(printf %02X "$y")$HP" -yes "$name" -pick Attack -until NEVERMATCH \
      -overland Right -fight -max-actions "$ACTIONS" -every 15000000 -out /out) > "$d/run.log" 2>&1 || echo "$name 失敗" >&2
  echo "$name $(tail -1 "$d/run.log")"
}
n=0
for t in "$@"; do
  run "$t" &
  n=$((n + 1)); (( n % PAR == 0 )) && wait
done
wait
