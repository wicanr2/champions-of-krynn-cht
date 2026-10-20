#!/usr/bin/env bash
# Issue 21：無網路的既有影像工具，原始資料維持唯讀。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for p in "$ROOT/workplace/hd" "$ROOT/workplace/probe" "$ROOT/tools"; do test -d "$p"; done
EXTRA=()
if [[ -n "${COK_HD_GENERATED:-}" ]]; then test -d "$COK_HD_GENERATED"; EXTRA=(-v "$COK_HD_GENERATED:/generated:ro"); fi
# 只開放本 Issue 的探針子目錄，不讓容器寫到其他研究狀態。
for name in issue21-battle-source issue21-g5-runner issue21-g5-receipt; do
 mkdir -p "$ROOT/workplace/probe/$name"
 EXTRA+=(-v "$ROOT/workplace/probe/$name:/src/workplace/probe/$name")
done
exec timeout 10m docker run --rm --label cok.issue=21 --memory 2g --cpus 2 --pids-limit 128 \
 --log-opt max-size=10m --log-opt max-file=3 --network none -u "$(id -u):$(id -g)" \
 -e PYTHONDONTWRITEBYTECODE=1 -v "$ROOT:/src:ro" \
 -v "$ROOT/workplace/hd:/src/workplace/hd" \
 "${EXTRA[@]}" -w /src cok-img:r1 "$@"
