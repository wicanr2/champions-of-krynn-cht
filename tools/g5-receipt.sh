#!/usr/bin/env bash
# G5 覆繪收據：同一狀態、同一輸入跑 cok-overlay-shot，每張截圖要求「疊字全部顯示、疊字範圍外與原版逐像素相同、沒有缺字」。
# 場景：主選單（路徑 A）、demo 3D 城市（A＋B＋座標列＋音譯）、查驗題（規格 006）、開場手札第 51 則（B＋規格 007）、
#       城鎮底列選單（C）、攻略面板（規格 008）、地圖疊層（規格 009）。
# 前置：tools/g2-startup-receipt.sh、tools/g3-party-receipt.sh 的狀態檔；tools/font-dev.sh 的字型。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="${DG:-$ROOT/workplace/dosgolem}"; G2="$ROOT/workplace/probe/g2-receipt"; G3="$ROOT/workplace/probe/g3-party"; OUT="$ROOT/workplace/probe/g5-receipt"
# 規格 015：G5_LANG=代碼 G5_LANG_FONT=字型檔名（workplace/font 底下）逐語言跑同一組場景，輸出到 g5-receipt-<代碼>；
# 該語言沒有攻略表就略過攻略場景，沒有手札表就不要求手札面板。
LANGARGS=()
if [[ -n "${G5_LANG:-}" ]]; then
  OUT="$OUT-$G5_LANG"; LANGARGS=(-lang "$G5_LANG" -lang-font "/font/${G5_LANG_FONT:?G5_LANG 需要 G5_LANG_FONT}")
fi
HAS_GUIDE=1; HAS_JOURNAL=1
if [[ -n "${G5_LANG:-}" ]]; then
  [[ -f "$ROOT/text/walkthrough.$G5_LANG.tsv" ]] || HAS_GUIDE=0
  [[ -f "$ROOT/text/journal.$G5_LANG.tsv" ]] || HAS_JOURNAL=0
fi
for f in "$G2/start-200m.state" "$G3/3-party.state" "$G3/5-opening.state" "$ROOT/workplace/font/cok-eten.golemfnt"; do
  test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
rm -rf "$OUT"; mkdir -p "$OUT"
shot() { # 名稱 狀態目錄 狀態檔 scratch來源 其餘參數…
  local name="$1" sd="$2" st="$3" sc="$4"; shift 4
  mkdir -p "$OUT/$name"; cp -r "$sc" "$OUT/$name/scratch"
  (cd "$DG" && DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" \
    DOSGOLEM_EXTRA_MOUNTS="$sd:/st:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro;$OUT/$name:/out" DOSGOLEM_TIMEOUT=30m \
    tools/go.sh run ./cmd/cok-overlay-shot -state "/st/$st" -root /orig -scratch /out/scratch -font /font/cok-eten.golemfnt -text-dir /text \
      ${LANGARGS[@]+"${LANGARGS[@]}"} -out /out "$@") > "$OUT/$name/result.json" 2> "$OUT/$name/err.log"
}
CR=$'\r'
shot menu "$G2" start-200m.state "$G2/scratch" -bios-keys p -bios-key-from 205000000 -bios-key-every 50000000 -shots 299000000:menu -until 299000100
shot demo "$G2" start-200m.state "$G2/scratch" -shots 450000000:demo -until 450000100
shot quiz "$G3" 3-party.state "$G3/scratch" -bios-key-names "Down,Down,Down,Down,Down,Down,Return" -bios-key-from 1901000000 -bios-key-every 10000000 -shots 2199000000:quiz -until 2199000100
shot journal "$G3" 5-opening.state "$G3/scratch" -bios-keys "$CR$CR$CR$CR$CR" -bios-key-from 2510000000 -bios-key-every 60000000 -shots 2800000000:j51 -until 2800000100
shot town "$G3" 5-opening.state "$G3/scratch" -bios-keys "$CR$CR$CR$CR$CR$CR$CR" -bios-key-from 2510000000 -bios-key-every 60000000 -shots 2925000000:town -until 2925000100
[[ $HAS_GUIDE == 1 ]] && shot guide "$G2" start-200m.state "$G2/scratch" -guide-preview 2:1 -shots 450000000:guide -until 450000100
shot map "$G2" start-200m.state "$G2/scratch" -map-preview 全圖 -shots 450000000:map -until 450000100
docker run -i --rm --network none --memory 512m --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e HAS_JOURNAL="$HAS_JOURNAL" -v "$OUT":/r:ro python:3.13-slim python -I - <<'PY'
import json, os, sys
ok = True
want = {'journal': lambda s: s.get('journal_panel'), 'town': None}
for name in sorted(os.listdir('/r')):
    r = json.load(open(f'/r/{name}/result.json'))
    for s in r['shots']:
        good = s['diff_outside_stamps'] == 0 and s['stamps'] == s['stamps_shown'] and s['stamps'] > 0 and not s.get('missing_glyphs')
        if name == 'journal' and os.environ.get('HAS_JOURNAL') == '1':
            good &= bool(s.get('journal_panel'))
        if name == 'town':
            words = r.get('menu_words') or []
            good &= len(words) > 0 and all(w['Drawn'] for w in words)
        ok &= good
        print(f"{name}/{s['name']}: 步 {s['step']} 疊字 {s['stamps_shown']}/{s['stamps']} 範圍外差異 {s['diff_outside_stamps']} "
              f"原版 {s['indexed_sha256'][:12]} 中文 {s['overlay_sha256'][:12]} {'通過' if good else '失敗'}")
sys.exit(0 if ok else 1)
PY
