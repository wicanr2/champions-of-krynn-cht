#!/usr/bin/env bash
# 推廣影片分鏡檢查：tools/promo-sheet.sh 段名 [每幾格取一張]，縮成 320×200 拼成 workplace/promo/seg/<段名>/sheet.png。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; SEG="$ROOT/workplace/promo/seg/$1"; EVERY="${2:-30}"
test -d "$SEG/frames"
timeout 300s docker run --rm --network none -u "$(id -u):$(id -g)" --memory 1g --cpus 1 --pids-limit 32 \
  --log-opt max-size=10m --log-opt max-file=3 -v "$SEG:/s" --entrypoint python cok-img:r1 -I -c '
import sys, os
from PIL import Image, ImageDraw
ev = int(sys.argv[1]); fs = sorted(os.listdir("/s/frames"))[::ev]
cols = 6; rows = (len(fs) + cols - 1) // cols
m = Image.new("RGB", (cols * 320, rows * 212), "white")
for k, f in enumerate(fs):
    im = Image.open("/s/frames/" + f).convert("RGB").resize((320, 200))
    x, y = k % cols * 320, k // cols * 212
    m.paste(im, (x, y + 12)); ImageDraw.Draw(m).text((x + 2, y), f, fill="black")
m.save("/s/sheet.png")' "$EVERY"
echo "$SEG/sheet.png"
