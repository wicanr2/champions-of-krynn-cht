#!/usr/bin/env bash
# G5 原七場景命令、步數與輸入；只將輸出搬到本輪目錄。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd";G2="$ROOT/workplace/probe/g2-receipt";G3="$ROOT/workplace/probe/g3-party"
for p in "$HD" "$G2" "$G3" "$ROOT/workplace/font" "$ROOT/text" "$ROOT/workplace/input/dos/ckrynn";do test -d "$p";done
shot() {
 local name="$1" sd="$2" st="$3";shift 3
 test -f "$sd/$st";test -d "$sd/scratch"
 bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
from pathlib import Path
import shutil,sys
p=Path("/src/workplace/hd/out/issue22-g5")/sys.argv[1]
if p.exists():shutil.rmtree(p)
p.mkdir(parents=True)
shutil.copytree(Path(sys.argv[2])/"scratch",p/"scratch")
' "$name" "/src/${sd#"$ROOT/"}"
 local out="$HD/out/issue22-g5/$name"
 DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" \
 DOSGOLEM_EXTRA_MOUNTS="$sd:/st:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro;$out:/out" \
 bash "$ROOT/tools/issue22-hd-go.sh" run ./cmd/cok-overlay-shot -state "/st/$st" -root /orig -scratch /out/scratch \
 -font /font/cok-eten.golemfnt -text-dir /text -out /out "$@" > "$out/result.json" 2> "$out/err.log"
}
if [[ "${1:-}" != --verify-only ]];then
CR=$'\r'
shot menu "$G2" start-200m.state -bios-keys p -bios-key-from 205000000 -bios-key-every 50000000 -shots 299000000:menu -until 299000100
shot demo "$G2" start-200m.state -shots 450000000:demo -until 450000100
shot quiz "$G3" 3-party.state -bios-key-names 'Down,Down,Down,Down,Down,Down,Return' -bios-key-from 1901000000 -bios-key-every 10000000 -shots 2199000000:quiz -until 2199000100
shot journal "$G3" 5-opening.state -bios-keys "$CR$CR$CR$CR$CR" -bios-key-from 2510000000 -bios-key-every 60000000 -shots 2800000000:j51 -until 2800000100
shot town "$G3" 5-opening.state -bios-keys "$CR$CR$CR$CR$CR$CR$CR" -bios-key-from 2510000000 -bios-key-every 60000000 -shots 2925000000:town -until 2925000100
shot guide "$G2" start-200m.state -guide-preview 2:1 -shots 450000000:guide -until 450000100
shot map "$G2" start-200m.state -map-preview 全圖 -shots 450000000:map -until 450000100
fi
bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
import json
import sys
from pathlib import Path
sys.path.insert(0,"tools")
from hd_issue22 import write
p=Path("workplace/hd/out/issue22-g5");old=Path("workplace/probe/issue21-g5-receipt");rows=[]
for name in ("demo","guide","journal","map","menu","quiz","town"):
 a=json.loads((p/name/"result.json").read_text());b=json.loads((old/name/"result.json").read_text());s=a["shots"][0];expected=b["shots"][0]["overlay_sha256"]
 good=s["overlay_sha256"]==expected and s["diff_outside_stamps"]==0 and s["stamps"]==s["stamps_shown"] and s["stamps"]>0 and not s.get("missing_glyphs")
 if name=="journal":good=good and bool(s.get("journal_panel"))
 if name=="town":good=good and bool(a.get("menu_words")) and all(w["Drawn"] for w in a["menu_words"])
 rows.append(dict(scene=name,expected_sha256=expected,actual_sha256=s["overlay_sha256"],pass_=good))
write(p/"receipt.json",dict(all_pass=all(r["pass_"] for r in rows),items=rows));print(rows);assert all(r["pass_"] for r in rows)
'
