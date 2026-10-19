#!/usr/bin/env bash
# G2 冷啟動收據：原版 START.EXE 由 dosgolem 冷啟動到 PLAY／DEMO 選單存狀態，
# 再送 p 進建立隊伍主選單，重播兩次比對畫面色號雜湊。
# 音效設定：原版附的 KRYNN.CFG 選 Roland LAPC（第 2 行 R），dosgolem 沒有 MPU-401，
# 程式關中斷輪詢埠 331h 不會結束；scratch 層放一份改選 AdLib（A）的設定，等同玩家在安裝選單選 [3] AdLib。
# 注意：-save-state／-dump-at 的步數須小於 -steps，等於上限時不會觸發。
# 前置：workplace/input/dos/ckrynn（G1 解壓）、workplace/dosgolem（固定 commit 的隔離 clone）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORIG="$ROOT/workplace/input/dos/ckrynn"
DG="$ROOT/workplace/dosgolem"
OUT="$ROOT/workplace/probe/g2-receipt"
test -f "$ORIG/START.EXE" || { echo "缺 $ORIG/START.EXE（先做 G1 解壓）" >&2; exit 1; }
test -x "$DG/tools/go.sh" || { echo "缺 $DG（dosgolem 隔離 clone）" >&2; exit 1; }
mkdir -p "$OUT/scratch"
"$ROOT/tools/cfg-adlib.sh" "$ORIG/KRYNN.CFG" "$OUT/scratch/KRYNN.CFG"
run() {
  DOSGOLEM_ORIG="$ORIG" DOSGOLEM_EXTRA_MOUNT="$OUT:/out" DOSGOLEM_TIMEOUT=15m \
    "$DG/tools/go.sh" run ./cmd/probe "$@"
}
cd "$DG"
run -exe /orig/START.EXE -root /orig -scratch-dir /out/scratch -steps 200001000 \
  -save-state 200000000:/out/start-200m.state > "$OUT/cold.log" 2>&1
for tag in a b; do
  run -load-state /out/start-200m.state -root /orig -scratch-dir /out/scratch -steps 300001000 \
    -bios-keys 'p' -bios-key-from 205000000 -bios-key-every 50000000 \
    -dump-at "299000000:/out/menu-$tag.png" > "$OUT/menu-$tag.log" 2>&1
done
sha256sum "$OUT/start-200m.state" "$OUT/menu-a.bin" "$OUT/menu-b.bin"
a=$(sha256sum < "$OUT/menu-a.bin"); b=$(sha256sum < "$OUT/menu-b.bin")
[[ "$a" == "$b" ]] && echo "重播一致" || { echo "重播不一致" >&2; exit 1; }
