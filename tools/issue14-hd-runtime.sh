#!/usr/bin/env bash
# 規格010 §7：正式Live的三地區3倍／2倍A/B。所有工作負載在Docker。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd";ORIG="$ROOT/workplace/input/dos/ckrynn"
if [[ ! -d "$ORIG" ]];then
 echo 'SKIP：缺少玩家原版資料，略過3D探索A/B。';exit 0
fi
for d in "$HD" "$ORIG" "$ROOT/workplace/font" "$ROOT/text";do test -d "$d";done
test -f "$HD/theme/walls.json";test -f "$ROOT/workplace/font/cok-eten.golemfnt"
bash "$ROOT/tools/issue14-hd-go.sh" build -p 2 -o /hd/out/issue14-cok-hd-probe ./cmd/cok-hd-probe
run() {
  local area="$1" state="$2" filename="$3" start="$4" until="$5" scale="$6" mode="$7"
  shift 7
  local name="$area-$mode$scale";test -f "$state/$filename";test -d "$state/scratch"
  timeout 40s docker run --rm --label cok.issue=14 --init --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -v "$state:/st:ro" -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import shutil,sys
p=Path("/hd/out/issue14-runtime")/sys.argv[1]/"scratch"
p.parent.mkdir(parents=True,exist_ok=True)
# A/B每次都從同一乾淨暫存層開始，只移除這個腳本自己的輸出。
if p.exists():shutil.rmtree(p)
shutil.copytree("/st/scratch",p)' "$name"
  local art=();if [[ "$mode" == hd ]];then art=(-hd /hd/theme);fi
  timeout 300s docker run --rm --label cok.issue=14 --init --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
    -v "$ORIG:/orig:ro" -v "$state:/st:ro" -v "$HD:/hd" \
    -v "$ROOT/workplace/font:/font:ro" -v "$ROOT/text:/text:ro" \
    --entrypoint /hd/out/issue14-cok-hd-probe cok-img:r1 -root /orig -state "/st/$filename" \
    -scratch "/hd/out/issue14-runtime/$name/scratch" -out "/hd/out/issue14-runtime/$name" \
    -until "$until" -sample 100000 -limit 100 -scale "$scale" \
    -font /font/cok-eten.golemfnt -text-dir /text -key-from "$start" -key-every 20000000 \
    "${art[@]}" "$@" > "$HD/out/issue14-runtime/$name/log.txt" 2>&1
}
pair() {
  local common=("${@:1:6}") pid status
  shift 6
  run "${common[@]}" base "$@" & pid=$!
  if run "${common[@]}" hd "$@";then
    wait "$pid"
  else
    status=$?;wait "$pid" || true;return "$status"
  fi
}
for scale in 3 2;do
  pair throtl "$ROOT/workplace/probe/throtl1" end.state 19862000000 19992000000 "$scale" -key-names 'Right,Up,Left,Left,Up,Right'
  pair gargath "$ROOT/workplace/probe/ovmap" ov.state 4455000000 4515000000 "$scale" \
    -initial-poke '41B96=0d00;41B98=0300' -key-every 15000000 -key-names 'Return,M,Right,Y'
  pair sanction "$ROOT/workplace/probe/area2/SANCTION" end.state 17002000000 17132000000 "$scale" -key-names 'Right,Up,Left,Left,Up,Right'
done
timeout 240s docker run --rm --label cok.issue=14 --init --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 \
  /project/tools/hd_wall_receipts.py runtime --root /project
