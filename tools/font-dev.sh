#!/usr/bin/env bash
# 由全部正式譯文重建開發用倚天字型 workplace/font/cok-eten.golemfnt（font/README.md）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ETEN=/home/anr2/cht/etan_font/ET353S/FILES
test -d "$ETEN" || { echo "缺倚天字型目錄 $ETEN" >&2; exit 1; }
mkdir -p "$ROOT/workplace/font"
cats=(); for f in ecl-text.zh-TW ui-text.zh-TW name.zh-TW composed-templates host-text.zh-TW journal.zh-TW menu-masters translit-chars.zh-TW walkthrough.zh-TW map-guide.zh-TW map-geo; do
  [[ -f "$ROOT/text/$f.tsv" ]] && cats+=("/text/$f.tsv"); done
docker run --rm --network none -u "$(id -u):$(id -g)" --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$ROOT/tools":/tools:ro -v "$ROOT/text":/text:ro -v "$ROOT/workplace":/w -v "$ETEN":/eten:ro python:3.13-slim \
  python -I /tools/eten_font.py "${cats[@]}" --asc /eten/ASCFONT.15 --spc /eten/SPCFONT.15 --std /eten/STDFONT.15 --out /w/font/cok-eten.golemfnt
