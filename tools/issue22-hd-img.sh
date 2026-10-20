#!/usr/bin/env bash
# Issue22：影像與分析只在既有 Docker 映像內執行。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for p in "$ROOT/workplace/hd" "$ROOT/workplace/probe" "$ROOT/tools"; do test -d "$p"; done
exec timeout 10m docker run --rm --label cok.issue=22 --memory 2g --cpus 2 --pids-limit 128 \
 --log-opt max-size=10m --log-opt max-file=3 --network none -u "$(id -u):$(id -g)" \
 -e PYTHONDONTWRITEBYTECODE=1 -e OPENBLAS_NUM_THREADS=1 -v "$ROOT:/src:ro" \
 -v "$ROOT/workplace/hd:/src/workplace/hd" -w /src cok-img:r1 "$@"
