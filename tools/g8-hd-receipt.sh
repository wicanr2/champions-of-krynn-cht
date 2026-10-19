#!/usr/bin/env bash
# HD 接入收據（G8，#11）：屯墾區（卡爾爵士）、營火、商隊大圖、開場最後家園旅店大圖，3 倍掛 HD 主題截圖，
# 另跑同一輸入的 2 倍確認 HD 不畫（2 倍顯示原版）。輸出中文合成圖供目視；HD 疊圖記在 stamp_list（hd: 開頭）。
# 前置：workplace/hd/theme（tools/hd_theme.py）、workplace/probe/leave 與 load 的狀態檔（docs/re/008）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"; OUT="$ROOT/workplace/probe/g8-hd-receipt"; THEME="$ROOT/workplace/hd/theme"
test -f "$THEME/theme.json" || { echo "缺 $THEME/theme.json" >&2; exit 1; }
rm -rf "$OUT"; mkdir -p "$OUT"
shot() { # 名稱 狀態目錄 狀態檔 倍率 其餘參數…
  local name="$1" sd="$2" st="$3" sc="$4"; shift 4
  mkdir -p "$OUT/$name"; cp -r "$sd/scratch" "$OUT/$name/scratch"
  (cd "$DG" && DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" \
    DOSGOLEM_EXTRA_MOUNTS="$sd:/st:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro;$THEME:/hd:ro;$OUT/$name:/out" DOSGOLEM_TIMEOUT=30m \
    tools/go.sh run ./cmd/cok-overlay-shot -state "/st/$st" -root /orig -scratch /out/scratch -font /font/cok-eten.golemfnt -text-dir /text \
      -hd /hd -scale "$sc" -out /out "$@") > "$OUT/$name/result.json" 2> "$OUT/$name/err.log"
}
L="$ROOT/workplace/probe/leave"; Q="$ROOT/workplace/probe/load"; G3="$ROOT/workplace/probe/g3-party"
for sc in 3 2; do
  shot "karl-$sc" "$G3" 5-opening.state $sc -bios-keys $'\r\r\r\r' -bios-key-from 2510000000 -bios-key-every 60000000 -shots 2795000000:karl -until 2795000100
  shot "camp-$sc" "$L" town.state $sc -keys-at "1801000000:Return,1901000000:I,2001000000:Return" -shots 2190000000:camp -until 2190000100
  shot "caravan-$sc" "$L" ov1.state $sc -keys-at "2901000000:M,2931000000:Right,2961000000:Right,2991000000:Right,3021000000:Right,3051000000:Right,3081000000:Right,3111000000:Right,3141000000:Right,3171000000:Right,3201000000:Right" -shots 3399000000:caravan -until 3399000100
  shot "solace-$sc" "$Q" quiz.state $sc -bios-keys $'TO\r' -bios-key-from 1000000000 -bios-key-every 5000000 -shots 1100000000:solace -until 1100000100
done
docker run -i --rm --network none --memory 512m --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$OUT":/r:ro python:3.13-slim python -I - <<'PY'
import json, sys
ok = True
# HD 圖面不在疊字清單裡：以「3 倍中文合成與原版的疊字範圍外差異 > 0、2 倍 = 0」判定 HD 有畫、2 倍不畫
for key in ('karl', 'camp', 'caravan', 'solace'):
    for sc in (3, 2):
        s = json.load(open(f'/r/{key}-{sc}/result.json'))['shots'][0]
        d = s['diff_outside_stamps']
        good = d > 0 if sc == 3 else d == 0
        ok &= good
        print(f'{key} {sc}×：疊字外差異 {d}（{"HD 圖面" if sc == 3 else "應為 0"}）', '通過' if good else '不通過')
sys.exit(0 if ok else 1)
PY
