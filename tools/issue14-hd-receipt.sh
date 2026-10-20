#!/usr/bin/env bash
# 規格010 §7：美術閘門、主題、完整測試與獨立整幀回歸。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
for f in "$HD/requests/b13/manifest.json" "$HD/requests/b14/manifest.json" "$HD/screens/normal/manifest.json";do
 if [[ ! -f "$f" ]];then echo "SKIP：缺少本機HD與原版畫面驗證素材：$f";exit 0;fi
done
for d in "$HD" "$ROOT/tools" "$ROOT/workplace/dosgolem";do test -d "$d";done
img() {
 timeout 300s docker run --rm --label cok.issue=14 --init --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 \
  -v "$ROOT:/project:ro" -v "$HD:/project/workplace/hd" --entrypoint python cok-img:r1 "$@"
}
img /project/tools/hd_wall_art.py calibrate --root /project > "$HD/out/issue14-structure-calibration.log"
img /project/tools/hd_wall_art.py review --root /project > "$HD/out/issue14-art-validation.log"
img /project/tools/hd_wall_theme.py --root /project > "$HD/out/issue14-theme-build.log"
bash "$ROOT/tools/issue14-hd-go.sh" test -p 2 ./apps/cok/ -count=1 -v > "$HD/out/issue14-go-tests.log" 2>&1
DOSGOLEM_GO_CMD=xvfb-run bash "$ROOT/tools/issue14-hd-go.sh" -a go test -p 2 ./cmd/cok-play/ -count=1 -v > "$HD/out/issue14-cok-play-tests.log" 2>&1
bash "$ROOT/tools/issue14-hd-go.sh" build -p 2 -o /hd/out/issue14-cok-hd-frame-check ./cmd/cok-hd-frame-check
img /project/tools/hd_wall_receipts.py frames --root /project > "$HD/out/issue14-frame-build.log"
timeout 300s docker run --rm --label cok.issue=14 --init --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 -v "$HD:/hd" \
  --entrypoint /hd/out/issue14-cok-hd-frame-check cok-img:r1 -hd /hd/theme -cases /hd/out/issue14-frames/cases.json \
  -receipt /hd/out/issue14-frame-receipts.jsonl
img -c 'import json,sys;from pathlib import Path
p=Path("/project/workplace/hd/out/issue14-frame-receipts.jsonl")
rows=[json.loads(s) for s in p.read_text().splitlines()]
assert rows and all(r["ok"] for r in rows)
print("全部整幀通過",len(rows))'
