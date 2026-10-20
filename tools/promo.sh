#!/usr/bin/env bash
# 推廣影片（規格 016，#15）：錄五段、組裝成 workplace/promo/cok-promo.mp4。各段分鏡檢查用 tools/promo-sheet.sh。
# 前置：dosgolem 的 cok-promo（tools/go.sh build -o workplace/bin/cok-promo ./cmd/cok-promo）、cok-ffmpeg:r1
# （tools/docker/ffmpeg.Dockerfile）、HD 主題、各語言字型、配樂 workplace/music/out/、下列狀態檔。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; R="$ROOT/tools/promo-rec.sh"; CR=$'\r'
OPEN=(-keys "$CR$CR$CR$CR$CR$CR$CR" -key-from 2510000000 -key-every 60000000)
"$R" title "$ROOT"/workplace/probe/g2-receipt start-200m.state 1870000000 1995000000 &
"$R" battle "$ROOT"/workplace/hd/out/issue13-runtime/battle-source battle.state 1460000000 1600000000; wait
"$R" open "$ROOT"/workplace/probe/g3-party 5-opening.state 2450000000 2950000000 "${OPEN[@]}" &
"$R" langs "$ROOT"/workplace/probe/g3-party 5-opening.state 2640000000 2720000000 "${OPEN[@]}" \
  -actions "2650000000:lang=zh-CN,2664000000:lang=ja,2678000000:lang=ko,2692000000:lang=en,2706000000:lang=zh-TW"; wait
"$R" throtl "$ROOT"/workplace/probe/throtl1 end.state 19850000000 20150000000 -key-names 'Right,Up,Left,Left,Up,Right' \
  -key-from 19862000000 -key-every 20000000 \
  -actions "20000000000:map,20030000000:map-reveal,20060000000:map-close,20062000000:guide=2:1,20095000000:guide-page,20125000000:guide-close"
"$ROOT/tools/promo-build.sh"
