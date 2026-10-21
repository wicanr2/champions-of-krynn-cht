#!/usr/bin/env bash
# 推廣影片單段錄製（規格 016）：tools/promo-rec.sh 段名 狀態目錄 狀態檔 開始步數 結束步數 [cok-promo 其餘參數…]
# 狀態目錄要有 scratch/；每次從乾淨的 scratch 副本開始。輸出 workplace/promo/seg/<段名>/。
# PROMO_NOHD=1 不載入 HD 主題（原版對照段，給 HD 切換示範用）；PROMO_FONT 換字型。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
W="$ROOT/workplace"; BIN="$W/dosgolem/workplace/bin/cok-promo"
name="$1" sd="$(cd "$2" && pwd)" st="$3" from="$4" until="$5"; shift 5
OUT="$W/promo/seg/$name"; THEME="${PROMO_THEME:-$ROOT/hd/theme}"
FONT="${PROMO_FONT:-cok-rounded.golemfnt}"
for f in "$BIN" "$sd/$st" "$W/font/$FONT" "$W/music/out/music.tsv"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
for d in "$sd/scratch" "$W/input/dos/ckrynn" "$THEME" "$ROOT/text" "$W/font"; do test -d "$d" || { echo "缺 $d" >&2; exit 1; }; done
rm -rf "$OUT"; mkdir -p "$OUT"; cp -r "$sd/scratch" "$OUT/scratch"
LF="zh-CN=/font/cok-zh-CN.golemfnt,ja=/font/cok-ja.golemfnt,ko=/font/cok-ko.golemfnt,en=/font/cok-en.golemfnt"
timeout "${PROMO_TIMEOUT:-40m}" docker run --rm --init --network none -u "$(id -u):$(id -g)" --memory 3g --cpus 2 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -e GOMAXPROCS=2 \
  -v "$W/input/dos/ckrynn:/orig:ro" -v "$sd:/st:ro" -v "$THEME:/hd:ro" -v "$W/font:/font:ro" -v "$ROOT/text:/text:ro" \
  -v "$W/music/out:/music:ro" -v "$BIN:/app/cok-promo:ro" -v "$OUT:/out" \
  --entrypoint /app/cok-promo cok-img:r1 -state "/st/$st" -root /orig -scratch /out/scratch -out /out \
  -font "/font/$FONT" -text-dir /text ${PROMO_NOHD:+-hd-off} -hd /hd -music-dir /music -lang-fonts "$LF" \
  -from "$from" -until "$until" "$@" > "$OUT/log.txt" 2>&1 || { tail -5 "$OUT/log.txt" >&2; exit 1; }
rm -rf "$OUT/scratch"
tail -1 "$OUT/log.txt"
