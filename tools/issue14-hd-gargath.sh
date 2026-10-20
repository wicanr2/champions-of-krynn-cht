#!/usr/bin/env bash
# 規格010 §7：蓋加斯城門，沿用docs/re/009的大地圖座標診斷法。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd";ST="$ROOT/workplace/probe/ovmap"
if [[ ! -d "$ROOT/workplace/input/dos/ckrynn" ]];then
 echo 'SKIP：缺少玩家原版資料，略過蓋加斯城門探針。';exit 0
fi
test -f "$ST/ov.state";test -d "$ST/scratch";test -d "$HD";test -d "$ROOT/workplace/input/dos/ckrynn"
bash "$ROOT/tools/issue14-hd-go.sh" build -p 2 -o /hd/out/issue14-cok-hd-probe ./cmd/cok-hd-probe
timeout 40s docker run --rm --label cok.issue=14 --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$ST:/st:ro" -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import shutil
p=Path("/hd/out/issue14-runtime/gargath-entry/scratch")
p.parent.mkdir(parents=True,exist_ok=True)
shutil.copytree("/st/scratch",p,dirs_exist_ok=True)'
timeout 180s docker run --rm --label cok.issue=14 --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
  -v "$ROOT/workplace/input/dos/ckrynn:/orig:ro" -v "$ST:/st:ro" -v "$HD:/hd" \
  --entrypoint /hd/out/issue14-cok-hd-probe cok-img:r1 -root /orig -state /st/ov.state \
  -scratch /hd/out/issue14-runtime/gargath-entry/scratch -out /hd/out/issue14-runtime/gargath-entry \
  -initial-poke '41B96=0d00;41B98=0300' -key-names 'Return,M,Right,Y' -key-from 4455000000 -key-every 15000000 \
  -until 4515000000 -sample 100000 -limit 100
