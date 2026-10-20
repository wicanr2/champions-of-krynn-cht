#!/usr/bin/env bash
# 相同 state 的原版／HD 續跑，只改呈現，含人物與非格點 COMSPR。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
for p in "$HD" "$ROOT/workplace/input/dos/ckrynn" "$ROOT/workplace/font" "$ROOT/text"; do test -d "$p"; done
export DOSGOLEM_EXTRA_MOUNTS="$HD:/hd"
bash "$ROOT/tools/issue21-hd-go.sh" build -p 2 -o /hd/out/issue21-cok-hd-probe ./cmd/cok-hd-probe
run() {
 local scene="$1" st="$2" scratch="$3" until="$4" scale="$5" mode="$6"
 test -f "$st";test -d "$scratch"
 local name="$scene-$mode$scale" out="$HD/out/issue21-runtime/$scene-$mode$scale"
 bash "$ROOT/tools/issue21-hd-img.sh" python -B -c '
import shutil,sys
from pathlib import Path
p=Path("/src/workplace/hd/out/issue21-runtime")/sys.argv[1];p.mkdir(parents=True,exist_ok=True)
shutil.rmtree(p);p.mkdir()
shutil.copytree(sys.argv[2],p/"scratch")
' "$name" "/src/${scratch#"$ROOT/"}"
 local args=()
 [[ "$mode" == hd ]] && args=(-hd /hd/theme)
 DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" \
 DOSGOLEM_EXTRA_MOUNTS="$st:/source.state:ro;$out:/out;$HD/theme:/hd/theme:ro;$HD/out/issue21-cok-hd-probe:/hd/probe:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro" \
 DOSGOLEM_GO_CMD=/hd/probe timeout 20m bash "$ROOT/tools/issue21-hd-go.sh" \
 -state /source.state -root /orig -scratch /out/scratch -out /out -until "$until" \
 -sample 1 -limit 600 -capture-transient -scale "$scale" \
 -font /font/cok-eten.golemfnt -text-dir /text "${args[@]}" > "$out/log.txt" 2>&1
}
for scale in 3 2; do
 for mode in base hd; do
  run portrait "$ROOT/workplace/probe/demo3d/d450.state" "$ROOT/workplace/probe/cok-demo/scratch" 510000000 "$scale" "$mode"
  run battle "$HD/out/issue13-runtime/battle-source/battle.state" "$HD/out/issue13-runtime/battle-source/scratch" 1580000000 "$scale" "$mode"
 done
done
bash "$ROOT/tools/issue21-hd-img.sh" python -B tools/hd_issue21_receipts.py runtime
