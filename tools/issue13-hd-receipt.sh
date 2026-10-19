#!/usr/bin/env bash
# 規格010 §6：驗收、來源／保存畫面獨立整幀回歸及 Go 測試。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
for d in "$HD" "$ROOT/tools" "$ROOT/workplace/dosgolem"; do test -d "$d"; done
timeout 80s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 -c '
from pathlib import Path
import subprocess,sys
sys.path.insert(0,"/project/tools")
from hd_battle_art import theme
for batch in ("b07","b08","b09","b10","b11"):
    out=Path("/project/workplace/hd/out")/batch
    with (out/"validation.log").open("w") as f:
        subprocess.run(["python","/project/tools/hd_validate.py","--req","/project/workplace/hd/requests/"+batch,"--hd",str(out)],stdout=f,check=True)
theme(Path("/project"))'
bash "$ROOT/tools/issue13-hd-go.sh" test -p 2 ./apps/cok/ -count=1 -v > "$HD/out/issue13-go-tests.log" 2>&1
DOSGOLEM_GO_CMD=xvfb-run bash "$ROOT/tools/issue13-hd-go.sh" -a go test -p 2 ./cmd/cok-play/ -count=1 -v > "$HD/out/issue13-cok-play-tests.log" 2>&1
bash "$ROOT/tools/issue13-hd-go.sh" build -p 2 -o /hd/out/cok-hd-frame-check ./cmd/cok-hd-frame-check
timeout 300s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 \
  /project/tools/hd_battle_receipts.py --root /project > "$HD/out/issue13-frame-build.log" 2>&1
timeout 90s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 \
  /project/tools/hd_battle_finalize.py controls --root /project > "$HD/out/issue13-controls.log" 2>&1
DOSGOLEM_COK_HD_MATRIX=/hd bash "$ROOT/tools/issue13-hd-go.sh" test -p 2 ./apps/cok/ \
  -run TestHDIssue13CharacterMatrix -count=1 -v > "$HD/out/issue13-character-matrix-tests.log" 2>&1
timeout 240s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 -v "$HD:/hd" \
  --entrypoint /hd/out/cok-hd-frame-check cok-img:r1 -hd /hd/theme \
  -cases /hd/out/issue13-frames/cases.json -receipt /hd/out/issue13-frame-receipts.jsonl
timeout 20s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$HD:/hd:ro" --entrypoint python cok-img:r1 -c '
import json,collections
from pathlib import Path
rows=[json.loads(s) for s in Path("/hd/out/issue13-frame-receipts.jsonl").read_text().splitlines()]
assert rows and all(r["ok"] for r in rows)
print("全部通過",len(rows),dict(collections.Counter(r["group"] for r in rows)))'
