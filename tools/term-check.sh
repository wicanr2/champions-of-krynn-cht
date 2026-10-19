#!/bin/sh
# 術語一致性檢查（tools/term_check.py）：術語表 ↔ 手冊 ↔ 提示書 ↔ 攻略面板 ↔ 地圖攻略點
set -eu
cd "$(dirname "$0")/.."
timeout 300 docker run --rm --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -v "$PWD":/w:ro python:3.13-slim \
  python -I /w/tools/term_check.py --glossary /w/text/glossary.tsv \
    --manual-dir /w/reference/manual-zh --hint-dir /w/reference/hintbook-zh \
    --zh-files /w/text/walkthrough.zh-TW.tsv /w/text/map-guide.zh-TW.tsv \
    --magazine-terms /w/text/walkthrough-terms.tsv --map-guide /w/text/map-guide.zh-TW.tsv "$@"
