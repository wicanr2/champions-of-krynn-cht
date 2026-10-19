#!/usr/bin/env bash
# 規格010 §6：從未改記憶體的 demo 戰鬥保存狀態重播，含地形、人物及特效。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"; ORIG="$ROOT/workplace/input/dos/ckrynn"; ST="$ROOT/workplace/probe/demo3d"
BATTLE="$HD/out/issue13-runtime/battle-source"
for d in "$HD" "$ORIG" "$ST" "$ROOT/workplace/probe/cok-demo/scratch" "$ROOT/workplace/font" "$ROOT/text"; do test -d "$d"; done
bash "$ROOT/tools/issue13-hd-go.sh" build -p 2 -o /hd/out/cok-hd-probe ./cmd/cok-hd-probe
timeout 20s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$ROOT:/project:ro" -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import shutil
out=Path("/hd/out/issue13-runtime/battle-source");out.mkdir(exist_ok=True)
if not (out/"battle.state").exists():shutil.copytree("/project/workplace/probe/cok-demo/scratch",out/"scratch",dirs_exist_ok=True)'
if [[ ! -f "$BATTLE/battle.state" ]]; then
  timeout 300s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
    -v "$ORIG:/orig:ro" -v "$ST:/st:ro" -v "$HD:/hd" --entrypoint /hd/out/cok-hd-probe cok-img:r1 \
    -state /st/d450.state -root /orig -scratch /hd/out/issue13-runtime/battle-source/scratch \
    -out /hd/out/issue13-runtime/battle-source -until 1450000000 -sample 5000000 -limit 100 \
    -save-state /hd/out/issue13-runtime/battle-source/battle.state > "$BATTLE/log.txt" 2>&1
fi
test -f "$BATTLE/battle.state"
timeout 20s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import shutil
base=Path("/hd/out/issue13-runtime")
for name in ("battle-base3","battle-hd3","battle-base2","battle-hd2"):
    out=base/name;out.mkdir(exist_ok=True);shutil.copytree(base/"battle-source/scratch",out/"scratch",dirs_exist_ok=True)'
run() {
  local name="$1" scale="$2"; shift 2
  local out="$HD/out/issue13-runtime/$name"; test -d "$out"
  timeout 180s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
    -v "$ORIG:/orig:ro" -v "$HD:/hd" -v "$BATTLE:/hd/out/issue13-runtime/battle-source:ro" \
    -v "$ROOT/workplace/font:/font:ro" -v "$ROOT/text:/text:ro" \
    --entrypoint /hd/out/cok-hd-probe cok-img:r1 -state /hd/out/issue13-runtime/battle-source/battle.state -root /orig \
    -scratch "/hd/out/issue13-runtime/$name/scratch" -out "/hd/out/issue13-runtime/$name" \
    -until 1600000000 -sample 1000000 -limit 100 -scale "$scale" \
    -font /font/cok-eten.golemfnt -text-dir /text "$@" > "$out/log.txt" 2>&1
}
run battle-base3 3
run battle-hd3 3 -hd /hd/theme
run battle-base2 2
run battle-hd2 2 -hd /hd/theme
timeout 90s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 \
  /project/tools/hd_battle_finalize.py battle_runtime --root /project
