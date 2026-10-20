#!/usr/bin/env bash
# 規格010 §7：三地區既有狀態續跑，無本輪記憶體注入。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
if [[ ! -d "$ROOT/workplace/input/dos/ckrynn" ]];then
 echo 'SKIP：缺少玩家原版資料，略過3D來源擷取。';exit 0
fi
for d in "$HD" "$ROOT/workplace/input/dos/ckrynn" "$ROOT/workplace/probe/throtl1" "$ROOT/workplace/probe/area1/GARGATH" "$ROOT/workplace/probe/area2/SANCTION"; do test -d "$d"; done
bash "$ROOT/tools/issue14-hd-go.sh" build -p 2 -o /hd/out/issue14-cok-hd-probe ./cmd/cok-hd-probe
run() {
  local name="$1" state="$2" steps="$3"
  test -f "$state/end.state"; test -d "$state/scratch"
  timeout 50s docker run --rm --label cok.issue=14 --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -v "$ROOT:/project:ro" -v "$HD:/hd" --entrypoint python cok-img:r1 -c '
import shutil,sys
from pathlib import Path
dest=Path("/hd/out/issue14-runtime")/sys.argv[1]/"scratch"
dest.parent.mkdir(parents=True,exist_ok=True)
shutil.copytree(sys.argv[2],dest,dirs_exist_ok=True)
' "$name" "/project/${state#"$ROOT/"}/scratch"
  timeout 240s docker run --rm --label cok.issue=14 --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
    -v "$ROOT/workplace/input/dos/ckrynn:/orig:ro" -v "$state:/st:ro" -v "$HD:/hd" \
    --entrypoint /hd/out/issue14-cok-hd-probe cok-img:r1 -root /orig -state /st/end.state \
    -scratch "/hd/out/issue14-runtime/$name/scratch" -out "/hd/out/issue14-runtime/$name" \
    -until "$((steps+150000000))" -sample 1000000 -limit 40 \
    -key-from "$((steps+20000000))" -key-every 20000000 -key-names 'Right,Up,Left,Left,Up,Right'
}
run throtl-source "$ROOT/workplace/probe/throtl1" 19842000000
run gargath-source "$ROOT/workplace/probe/area1/GARGATH" 5443000000
run sanction-source "$ROOT/workplace/probe/area2/SANCTION" 16982000000
