#!/usr/bin/env bash
# 推廣影片（規格 016，#15）：錄各段、組裝成 workplace/promo/cok-promo.mp4。各段分鏡檢查用 tools/promo-sheet.sh。
# throtl-nohd、battle-nohd 是同參數不載入 HD 的原版對照段（HD 切換示範）；f12 是中文與英文原版切換示範。
# 前置：dosgolem 的 cok-promo（tools/go.sh build -o workplace/bin/cok-promo ./cmd/cok-promo）、cok-ffmpeg:r1
# （tools/docker/ffmpeg.Dockerfile）、HD 主題、各語言字型、配樂 workplace/music/out/、下列狀態檔。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; R="$ROOT/tools/promo-rec.sh"; CR=$'\r'
OPEN=(-keys "$CR$CR$CR$CR$CR$CR$CR" -key-from 2510000000 -key-every 60000000)
"$R" title "$ROOT"/workplace/probe/g2-receipt start-200m.state 1870000000 1995000000 &
"$R" battle "$ROOT"/workplace/probe/demo3d d450.state 1460000000 1600000000; wait
PROMO_NOHD=1 "$R" battle-nohd "$ROOT"/workplace/probe/demo3d d450.state 1460000000 1600000000 &
"$R" f12 "$ROOT"/workplace/probe/g3-party 5-opening.state 2640000000 2710000000 "${OPEN[@]}" \
  -actions "2672000000:original,2695000000:chinese"; wait
"$R" open "$ROOT"/workplace/probe/g3-party 5-opening.state 2450000000 2950000000 "${OPEN[@]}" &
"$R" langs "$ROOT"/workplace/probe/g3-party 5-opening.state 2640000000 2720000000 "${OPEN[@]}" \
  -actions "2650000000:lang=zh-CN,2664000000:lang=ja,2678000000:lang=ko,2692000000:lang=en,2706000000:lang=zh-TW"; wait
# 狀態檔載入前印的隊伍欄沒有覆繪：先紮營再離開讓遊戲重印（第一個 E 被吃掉，第二個開營、第三個離開），M 回移動模式。
"$R" throtl "$ROOT"/workplace/probe/throtl1 end.state 19850000000 20230000000 \
  -keys-at "19860000000:E,19870000000:E,19920000000:E,19940000000:M" -key-names 'Right,Up,Left,Left,Up,Right' \
  -key-from 19960000000 -key-every 20000000 \
  -actions "20090000000:map,20120000000:map-reveal,20150000000:map-close,20152000000:guide=2:1,20185000000:guide-page,20215000000:guide-close"
PROMO_NOHD=1 "$R" throtl-nohd "$ROOT"/workplace/probe/throtl1 end.state 19850000000 20230000000 \
  -keys-at "19860000000:E,19870000000:E,19920000000:E,19940000000:M" -key-names 'Right,Up,Left,Left,Up,Right' \
  -key-from 19960000000 -key-every 20000000
"$ROOT/tools/promo-build.sh"
