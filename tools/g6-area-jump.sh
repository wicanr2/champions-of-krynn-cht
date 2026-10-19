#!/usr/bin/env bash
# 跨區跳躍（比照幽靈騎士規格 010 §14，本作逐項重驗，docs/re/009）：改存檔 B 的欄位，主選單讀檔、開始冒險、
# 依 text/quiz-answers.tsv 作答，記錄 ECL／GEO 讀取並存區域狀態。
# 存檔 savgamB.dat 讀取分段（probe 檔案存取紀錄）：byte 0 → DS:5782；0x001–0x800 → 41A1:0000；0x801–0x1000 → 4221:0000；
# 0x1001–0x1400 → 42A1:0000；0x1401 起 5 bytes → DS:6D85（x、y、方向…）。
# 用法：tools/g6-area-jump.sh <輸出目錄> 改法…；改法寫成 `偏移=值/偏移=值`（十六進位），- 表示不改（對照）。
# 每種改法輸出 <目錄>/<名>/area.state（作答後 END 步）、r1.png、r2.png、log。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
LD="$ROOT/workplace/probe/load"       # menu-load.state：主選單反白 LOAD SAVED GAME（第 400M 步）
SV="$ROOT/workplace/probe/saveload"   # scratch/savgamB.dat：大地圖紮營存檔（docs/re/008）
OUT=$(realpath -m "$1"); shift
END=${END:-2300000000}
for f in "$LD/menu-load.state" "$SV/scratch/savgamB.dat"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
mkdir -p "$OUT"
probe() { # 名 其餘參數…
  local name=$1; shift
  (cd "$DG" && DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" DOSGOLEM_EXTRA_MOUNTS="$LD:/st:ro;$OUT/$name:/out" DOSGOLEM_TIMEOUT=15m \
    timeout 1000 tools/go.sh run ./cmd/probe -scratch-dir /out/scratch "$@")
}
run() { # 改法
  local spec=$1 name="a${1//[\/=]/_}" kv q w e p a
  mkdir -p "$OUT/$name/scratch"; cp "$LD"/scratch/* "$OUT/$name/scratch/"; cp "$SV"/scratch/savgamB.dat "$SV"/scratch/CHRDATB* "$OUT/$name/scratch/"
  if [[ "$spec" != - ]]; then
    IFS=/ read -ra kv <<<"$spec"
    for p in "${kv[@]}"; do
      printf "$(printf '\\x%s' "${p#*=}")" | dd of="$OUT/$name/scratch/savgamB.dat" bs=1 seek=$((0x${p%=*})) conv=notrunc status=none
    done
  fi
  # 讀檔 B → 隊伍選單；E 切到功能選單（游標在第 3 項），Down×5 到 BEGIN ADVENTURING；讀查驗題字串（路徑 A 0631:0353）
  probe "$name" -load-state /st/menu-load.state -steps 1400001000 -bios-keys $'\rB\r' -bios-key-from 401000000 -bios-key-every 60000000 \
    -save-state 1400000000:/out/loaded.state > "$OUT/$name/load.log" 2>&1 || { echo "$name 讀檔失敗" >&2; return 1; }
  probe "$name" -load-state /out/loaded.state -steps 1700001000 -bios-keys E -bios-key-names Down,Down,Down,Down,Down,Return \
    -bios-key-from 1401000000 -bios-key-every 10000000 -call-args "0631:0353:6:1400000000:1700000000" -call-length-string \
    -save-state 1700000000:/out/quiz.state > "$OUT/$name/quiz.log" 2>&1 || { echo "$name 讀檔失敗" >&2; return 1; }
  q=$(grep -o 'string="[^"]*"' "$OUT/$name/quiz.log" | tr -d '"' | sed 's/string=//' | tr '\n' ' ')
  w=$(sed -n 's/.*what is word # *\([0-9]*\).*/\1/p' <<<"$q"); e=$(sed -n 's/.*journal entry # *\([0-9]*\).*/\1/p' <<<"$q"); p=$(sed -n 's/.*journal, page *\([0-9]*\).*/\1/p' <<<"$q")
  a=$(awk -F'\t' -v p="$p" -v e="$e" -v w="$w" '$1==p && $2==e && $3==w {print $4}' "$ROOT/text/quiz-answers.tsv")
  [[ -n "$a" ]] || { echo "$name 查驗題 p$p e$e w$w 不在答案表" >&2; return 1; }
  probe "$name" -load-state /out/quiz.state -steps $((END + 1000)) -bios-keys "$a"$'\r' -bios-key-from 1701000000 -bios-key-every 5000000 \
    -dump-at "2000000000:/out/r1.png;$((END - 1000000)):/out/r2.png" -save-state "$END:/out/area.state" > "$OUT/$name/area.log" 2>&1 \
    || { echo "$name 作答後失敗" >&2; return 1; }
  echo "$name 查驗 p$p e$e w$w=$a $(grep -o '停止原因.*' "$OUT/$name/area.log") $(grep -o '開過的檔（[0-9]*）：.*' "$OUT/$name/area.log" | tr ' ' '\n' | grep -i 'geo\|ecl' | sort -u | tr '\n' ' ')"
}
n=0
for t in "$@"; do
  run "$t" &
  n=$((n + 1)); (( n % 4 == 0 )) && wait
done
wait
