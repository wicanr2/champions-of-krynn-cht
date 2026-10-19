#!/usr/bin/env bash
# 由繁中正式譯文產生簡中表（規格 015 §4）。CHECK=1 只檢查 text/ 裡的簡中表是否最新。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE=cok-opencc:r1
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "缺映像 $IMAGE：docker build -t $IMAGE -f tools/docker/opencc.Dockerfile tools/docker" >&2; exit 1; }
MODE=(); [[ "${CHECK:-}" == 1 ]] && MODE=(--check)
mkdir -p "$ROOT/workplace/zhcn"
timeout 10m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$ROOT/tools:/tools:ro" -v "$ROOT/text:/text" -v "$ROOT/workplace/zhcn:/out" "$IMAGE" \
  python -I /tools/zhcn_convert.py --text /text --longer /out/longer.tsv "${MODE[@]}"
