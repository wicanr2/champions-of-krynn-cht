#!/usr/bin/env bash
# 既有 demo 戰鬥續跑，保留短暫畫格供 COMSPR 完整模板搜尋。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ST="$ROOT/workplace/hd/out/issue13-runtime/battle-source"
for p in "$ST/battle.state" "$ROOT/workplace/hd/out/issue21-cok-hd-probe"; do test -f "$p"; done
for p in "$ST/scratch" "$ROOT/workplace/input/dos/ckrynn"; do test -d "$p"; done
bash "$ROOT/tools/issue21-hd-img.sh" python -B -c '
from pathlib import Path
import shutil
p=Path("/src/workplace/probe/issue21-battle-source");p.mkdir(exist_ok=True)
shutil.copytree("/src/workplace/hd/out/issue13-runtime/battle-source/scratch",p/"scratch",dirs_exist_ok=True)
'
export DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn"
export DOSGOLEM_EXTRA_MOUNTS="$ST:/st:ro;$ROOT/workplace/hd:/hd:ro;$ROOT/workplace/probe/issue21-battle-source:/out"
export DOSGOLEM_GO_CMD=/hd/out/issue21-cok-hd-probe
exec timeout 20m bash "$ROOT/tools/issue21-hd-go.sh" -state /st/battle.state -root /orig \
 -scratch /out/scratch -out /out -until 2200000000 -sample 1 -limit 1600 \
 -capture-transient -indexed-only -save-state /out/end.state
