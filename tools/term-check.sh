#!/bin/sh
# 術語一致性：術語表的取代異譯不得出現在遊戲譯文、手札、攻略面板與地圖點；雜誌異寫須登在 text/walkthrough-terms.tsv。不一致時結束碼 1。
set -eu
cd "$(dirname "$0")/.."
mg=""; [ -f text/map-guide.zh-TW.tsv ] && mg="--map-guide /w/text/map-guide.zh-TW.tsv"
zh="/w/text/walkthrough.zh-TW.tsv /w/text/journal.zh-TW.tsv /w/text/ecl-text.zh-TW.tsv /w/text/ui-text.zh-TW.tsv"
[ -f text/map-guide.zh-TW.tsv ] && zh="$zh /w/text/map-guide.zh-TW.tsv"
timeout 300 docker run --rm --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -v "$PWD":/w:ro python:3.13-slim \
  python -I /w/tools/term_check.py --glossary /w/text/glossary.tsv --zh-files $zh \
    --magazine-terms /w/text/walkthrough-terms.tsv $mg "$@"
