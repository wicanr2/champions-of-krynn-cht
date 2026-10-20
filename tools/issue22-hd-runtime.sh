#!/usr/bin/env bash
# Issue22：由指定狀態續跑；不注入座標，不改存檔或正式規則。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"; ORIG="$ROOT/workplace/input/dos/ckrynn"
for p in "$HD" "$ORIG" "$ROOT/workplace/font" "$ROOT/text"; do test -d "$p"; done
export DOSGOLEM_EXTRA_MOUNTS="$HD:/hd"
if [[ "${1:-}" == build ]]; then
 exec bash "$ROOT/tools/issue22-hd-go.sh" build -p 2 -o /hd/out/issue22-cok-hd-probe ./cmd/cok-hd-probe
fi
scene="${1:?scene}"; mode="${2:-initial}"; scale="${3:-3}"; until="${4:-1}"; from="${5:-0}"; keys="${6:-}"; every="${7:-20000000}"
poke=""
case "$scene" in
 demo) state="$ROOT/workplace/probe/demo3d/d450.state"; scratch="$ROOT/workplace/probe/demo3d/scratch";;
 throtl) state="$ROOT/workplace/probe/throtl1/end.state"; scratch="$ROOT/workplace/probe/throtl1/scratch";;
 throtl2) state="$ROOT/workplace/probe/area2/THROTL/end.state"; scratch="$ROOT/workplace/probe/area2/THROTL/scratch";;
 gargath) state="$ROOT/workplace/probe/area1/GARGATH/end.state"; scratch="$ROOT/workplace/probe/area1/GARGATH/scratch";;
 gargath2) state="$ROOT/workplace/probe/area1/GARGATH2/end.state"; scratch="$ROOT/workplace/probe/area1/GARGATH2/scratch";;
 sanction) state="$ROOT/workplace/probe/area2/SANCTION/end.state"; scratch="$ROOT/workplace/probe/area2/SANCTION/scratch";;
 gargath-entry) state="$ROOT/workplace/probe/ovmap/ov.state"; scratch="$ROOT/workplace/probe/ovmap/scratch";poke='41B96=0d00;41B98=0300';;
 gargath3d-a|gargath3d-b) state="$HD/out/issue22-runtime/gargath-entry-base3/end.state";scratch="$HD/out/issue22-runtime/gargath-entry-base3/scratch";;
 gargath2-reenter) state="$HD/out/issue22-runtime/gargath2-base3/end.state";scratch="$HD/out/issue22-runtime/gargath2-base3/scratch";;
 *) echo '未知場景' >&2;exit 1;;
esac
test -f "$state";test -d "$scratch";test -f "$HD/out/issue22-cok-hd-probe"
name="$scene-$mode$scale"
bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
import shutil,sys
from pathlib import Path
p=Path("/src/workplace/hd/out/issue22-runtime")/sys.argv[1]
if p.exists():shutil.rmtree(p)
p.mkdir(parents=True)
shutil.copytree(sys.argv[2],p/"scratch")
' "$name" "/src/${scratch#"$ROOT/"}"
out="$HD/out/issue22-runtime/$name";args=()
[[ "$mode" == hd ]] && args=(-hd /hd/theme)
DOSGOLEM_ORIG="$ORIG" DOSGOLEM_GO_CMD=/hd/probe \
DOSGOLEM_EXTRA_MOUNTS="$state:/source.state:ro;$out:/out;$HD/theme:/hd/theme:ro;$HD/out/issue22-cok-hd-probe:/hd/probe:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro" \
bash "$ROOT/tools/issue22-hd-go.sh" -state /source.state -root /orig -scratch /out/scratch -out /out \
 -until "$until" -sample 100000 -limit 160 -scale "$scale" \
 -font /font/cok-eten.golemfnt -text-dir /text -key-from "$from" -key-every "$every" -key-names "$keys" \
 -initial-poke "$poke" -save-state /out/end.state \
 "${args[@]}" > "$out/log.txt" 2>&1
