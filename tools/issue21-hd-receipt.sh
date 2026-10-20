#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
for p in "$HD/theme/exact.json" "$HD/theme/walls.json"; do test -f "$p"; done
export DOSGOLEM_EXTRA_MOUNTS="$HD:/hd"
bash "$ROOT/tools/issue21-hd-go.sh" build -p 2 -o /hd/out/issue21-cok-hd-frame-check ./cmd/cok-hd-frame-check
bash "$ROOT/tools/issue21-hd-img.sh" python -B tools/hd_issue21_receipts.py frames
DOSGOLEM_GO_CMD=/hd/out/issue21-cok-hd-frame-check timeout 15m bash "$ROOT/tools/issue21-hd-go.sh" \
 -hd /hd/theme -cases /hd/out/issue21-frames/cases.json -receipt /hd/out/issue21-frame-receipts.jsonl
