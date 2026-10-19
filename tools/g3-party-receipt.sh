#!/usr/bin/env bash
# 正常玩家路徑：冷啟動狀態 → 建立角色 ANNA → 加入隊伍 → 開始冒險 → 手冊查驗題 → 開場劇情。分段存狀態，FROM=<段名> 從該段續跑。
# 前置：tools/g2-startup-receipt.sh 的 start-200m.state。scratch 每次清空（殘留存檔會改變流程），只放 AdLib 設定。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"; ST="$ROOT/workplace/probe/g2-receipt"; OUT="$ROOT/workplace/probe/g3-party"
test -f "$ST/start-200m.state" || { echo "缺 start-200m.state（先跑 tools/g2-startup-receipt.sh）" >&2; exit 1; }
FROM="${FROM:-menu}"
if [[ "$FROM" == menu ]]; then rm -rf "$OUT"; mkdir -p "$OUT/scratch"; "$ROOT/tools/cfg-adlib.sh" "$ROOT/workplace/input/dos/ckrynn/KRYNN.CFG" "$OUT/scratch/KRYNN.CFG"; fi
CR=$'\r'
run() {
  (cd "$DG" && DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" DOSGOLEM_EXTRA_MOUNTS="$ST:/st:ro;$OUT:/out" DOSGOLEM_TIMEOUT=20m \
    tools/go.sh run ./cmd/probe -root /orig -scratch-dir /out/scratch "$@")
}
stage() { [[ "$FROM" == "$1" ]] && FROM=go; [[ "$FROM" == go ]]; }
if stage menu; then  # 1：PLAY DEMO 選單送 p
  run -load-state /st/start-200m.state -steps 300001000 -bios-keys p -bios-key-from 205000000 -bios-key-every 50000000 \
    -save-state 300000000:/out/1-menu.state > "$OUT/1.log" 2>&1
fi
if stage create; then  # 2：主選單、種族、性別、職業、神祇、陣營各 Enter；屬性畫面 REROLL STATS? 按 N；名字 ANNA；圖示編輯器 E E E、圖示 Y、存檔 Y
  run -load-state /out/1-menu.state -steps 880001000 -bios-keys "$CR$CR$CR$CR$CR${CR}N" -bios-key-from 301000000 -bios-key-every 80000000 \
    -save-state 880000000:/out/2a-name.state -dump-at "879000000:/out/2a-name.png" > "$OUT/2a.log" 2>&1
  run -load-state /out/2a-name.state -steps 1600001000 -bios-keys "ANNA${CR}EEEYY" -bios-key-from 881000000 -bios-key-every 60000000 \
    -save-state 1600000000:/out/2-created.state -dump-at "1599000000:/out/2-created.png" > "$OUT/2.log" 2>&1
fi
if stage add; then  # 3：主選單 Down、Enter 進加入畫面，Enter 標記 ANNA，E 離開到隊伍選單（游標在 DROP CHARACTER）
  run -load-state /out/2-created.state -steps 1900001000 -bios-key-names "Down,Return,Return" -bios-key-from 1601000000 -bios-key-every 60000000 \
    -save-state 1800000000:/out/3a.state > "$OUT/3a.log" 2>&1
  run -load-state /out/3a.state -steps 1900001000 -bios-keys E -bios-key-from 1801000000 -bios-key-every 60000000 \
    -save-state 1900000000:/out/3-party.state -dump-at "1899000000:/out/3-party.png" > "$OUT/3.log" 2>&1
fi
if stage begin; then  # 4：Down×6 到 BEGIN ADVENTURING、Enter；讀查驗題字串（路徑 A 0631:0353）
  run -load-state /out/3-party.state -steps 2200001000 -bios-key-names "Down,Down,Down,Down,Down,Down,Return" -bios-key-from 1901000000 -bios-key-every 10000000 \
    -call-args "0631:0353:6:1900000000:2200000000" -call-length-string \
    -save-state 2200000000:/out/4-quiz.state -dump-at "2199000000:/out/4-quiz.png" > "$OUT/4.log" 2>&1
fi
if stage answer; then  # 5：依 text/quiz-answers.tsv 作答，進入開場劇情
  q=$(grep -o 'string="[^"]*"' "$OUT/4.log" | tr -d '"' | sed 's/string=//' | tr '\n' ' ')
  w=$(sed -n 's/.*what is word # *\([0-9]*\).*/\1/p' <<<"$q"); e=$(sed -n 's/.*journal entry # *\([0-9]*\).*/\1/p' <<<"$q"); p=$(sed -n 's/.*journal, page *\([0-9]*\).*/\1/p' <<<"$q")
  a=$(awk -F'\t' -v p="$p" -v e="$e" -v w="$w" '$1==p && $2==e && $3==w {print $4}' "$ROOT/text/quiz-answers.tsv")
  [[ -n "$a" ]] || { echo "查驗題 p$p e$e w$w 不在答案表" >&2; exit 1; }
  echo "查驗題：第 $p 頁第 $e 則第 $w 字 → $a"
  run -load-state /out/4-quiz.state -steps 2500001000 -bios-keys "$a$CR" -bios-key-from 2201000000 -bios-key-every 5000000 \
    -save-state 2500000000:/out/5-opening.state -dump-at "2499000000:/out/5-opening.png" > "$OUT/5.log" 2>&1
fi
sha256sum "$OUT"/*.state
