#!/usr/bin/env bash
# 從已保存的蓋加斯探索狀態轉向；只在入口三格送正常方向鍵，不向北踏入戰鬥格。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd";ST="$HD/inventory/issue22/gargath-movement-start"
test -f "$ST/end.state";test -d "$ST/scratch";test -f "$HD/out/issue22-cok-hd-probe"
for p in "$ROOT/workplace/input/dos/ckrynn" "$ROOT/workplace/font" "$ROOT/text";do test -d "$p";done
for scale in 3 2;do
 for mode in base hd;do
  name="gargath-turns-$mode$scale"
  bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
from pathlib import Path
import shutil,sys
p=Path("/src/workplace/hd/out/issue22-runtime")/sys.argv[1]
if p.exists():shutil.rmtree(p)
p.mkdir(parents=True)
shutil.copytree(Path("/src/workplace/hd/inventory/issue22/gargath-movement-start/scratch"),p/"scratch")
' "$name"
  out="$HD/out/issue22-runtime/$name";args=()
  [[ "$mode" == hd ]] && args=(-hd /hd/theme)
  DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" DOSGOLEM_GO_CMD=/hd/probe \
  DOSGOLEM_EXTRA_MOUNTS="$ST/end.state:/source.state:ro;$out:/out;$HD/theme:/hd/theme:ro;$HD/out/issue22-cok-hd-probe:/hd/probe:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro" \
  bash "$ROOT/tools/issue22-hd-go.sh" -state /source.state -root /orig -scratch /out/scratch -out /out \
   -until 5440000000 -sample 100000 -limit 100 -scale "$scale" -font /font/cok-eten.golemfnt -text-dir /text \
   -key-from 5060000000 -key-every 20000000 \
   -key-names 'M,Right,Right,Right,Right,Right,Right,Up,Right,Right,Right,Right,Up,Right,Right,Right,Right' \
   -save-state /out/end.state "${args[@]}" > "$out/log.txt" 2>&1
  if [[ "$mode$scale" == base3 ]];then
   bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
import sys,json,numpy as np
sys.path.insert(0,"tools")
from hd_issue22_receipts import CellOracle,HD,OUT,is_3d,sha
p=OUT/"issue22-runtime/gargath-turns-base3";r=json.loads((p/"receipt.json").read_text());w=CellOracle(HD/"theme");seen=set()
for s in r["shots"]:
 raw=(p/(s["name"]+".bin")).read_bytes();a=np.frombuffer(raw,np.uint8).reshape(200,320);c=w.matches(raw)
 if is_3d(raw) and sum(v is not None and v["guard"] for v in c)>=3:seen.add(sha(a[8:176,8:176].tobytes()))
print("Gargath different 3D frames:",len(seen));assert len(seen)>=10,"先停止，未達十個不同3D畫格，不繼續A/B湊數"
'
  fi
 done
done
