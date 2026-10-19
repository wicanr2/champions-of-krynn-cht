#!/usr/bin/env bash
# 規格 010 §5：只讀原版保存畫面，收據全部寫入 workplace/hd。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
DG="$ROOT/workplace/dosgolem"
for d in "$ROOT/tools" "$HD" "$DG"; do test -d "$d"; done
test -f "$HD/theme/theme.json"
timeout 60s docker run --rm --network none -e PYTHONDONTWRITEBYTECODE=1 -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$ROOT:/project:ro" -v "$HD:/hd" -w /project cok-img:r1 \
  python tools/hd_source_fixtures.py --root /project --out /hd/screens/normal
(cd "$DG"
  # 既有 go.sh 固定寫 256；只在本次 Docker 啟動參數收緊為本 Issue 的 64。
  # 保留 go.sh 的映像、module cache 與唯讀素材掛載，不修改其檔案。
  docker() {
    local args=("$@") i
    if [[ "${args[0]:-}" == run ]]; then
      for ((i=0;i<${#args[@]};i++)); do
        if [[ "${args[i]}" == --pids-limit ]]; then args[i+1]=64; fi
      done
    fi
    command docker "${args[@]}"
  }
  export -f docker
  DOSGOLEM_MEM=2g DOSGOLEM_CPUS=2 DOSGOLEM_TIMEOUT=5m \
    DOSGOLEM_EXTRA_MOUNTS="$HD:/hd" DOSGOLEM_COK_HD_RECEIPT=/hd \
    DOSGOLEM_GO_CMD=env tools/go.sh GOMAXPROCS=2 go test -p 2 ./apps/cok/ -count=1 -v
) > "$HD/out/issue12-go-tests.log" 2>&1
timeout 30s docker run --rm --network none -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$HD:/hd:ro" cok-img:r1 python -c \
  'import json; r=json.load(open("/hd/out/issue12-receipts.json")); assert r["all_pass"]; print("主題",r["theme_items"],"項；原版保存畫面",r["normal_frames"],"張；全部通過")'
