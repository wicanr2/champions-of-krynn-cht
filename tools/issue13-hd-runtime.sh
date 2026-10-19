#!/usr/bin/env bash
# 規格010 §6：同一 demo 狀態續跑，3倍與2倍各做 HD／無HD A/B。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"; ORIG="$ROOT/workplace/input/dos/ckrynn"; ST="$ROOT/workplace/probe/demo3d"
for d in "$HD" "$ORIG" "$ST" "$ROOT/workplace/probe/cok-demo/scratch" "$ROOT/workplace/font" "$ROOT/text"; do test -d "$d"; done
test -f "$ST/d450.state"; test -f "$ROOT/workplace/font/cok-eten.golemfnt"
timeout 20s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$ROOT:/project:ro" -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import shutil
hd=Path("/hd")
old=hd/"out/issue13-runtime/demo-source/cok-hd-probe-original"
if not old.exists():shutil.copyfile(hd/"out/cok-hd-probe",old)
for name in ("base3","hd3","base2","hd2"):
    out=hd/"out/issue13-runtime"/name
    out.mkdir(parents=True,exist_ok=True)
    shutil.copytree("/project/workplace/probe/cok-demo/scratch",out/"scratch",dirs_exist_ok=True)'
bash "$ROOT/tools/issue13-hd-go.sh" build -p 2 -o /hd/out/cok-hd-probe ./cmd/cok-hd-probe
run() {
  local name="$1" scale="$2"; shift 2
  local out="$HD/out/issue13-runtime/$name"; test -d "$out"
  timeout 240s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
    -v "$ORIG:/orig:ro" -v "$ST:/st:ro" -v "$HD:/hd" \
    -v "$ROOT/workplace/font:/font:ro" -v "$ROOT/text:/text:ro" \
    --entrypoint /hd/out/cok-hd-probe cok-img:r1 -state /st/d450.state -root /orig \
    -scratch "/hd/out/issue13-runtime/$name/scratch" -out "/hd/out/issue13-runtime/$name" \
    -until 1000000000 -sample 1000000 -limit 100 -scale "$scale" \
    -font /font/cok-eten.golemfnt -text-dir /text "$@" > "$out/log.txt" 2>&1
}
run base3 3
run hd3 3 -hd /hd/theme
run base2 2
run hd2 2 -hd /hd/theme
timeout 80s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 \
  /project/tools/hd_battle_finalize.py runtime --root /project
