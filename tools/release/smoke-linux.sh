#!/usr/bin/env bash
# Linux AppImage smoke（規格 012 §4 第 3 項）：原版放在 AppImage 旁，冷啟動自動模式跑到標題畫面截圖；
# 再驗缺原版時以錯誤結束並留下 cok-error.log。原版、HD 兩包各跑一次。
#
#   tools/release/smoke-linux.sh <版本>
# 輸出：dist-all/<版本>/smoke/linux-<版別>/（截圖 PNG、log、錯誤紀錄）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VER="${1:?版本}"
ORIG="$ROOT/workplace/input/dos/ckrynn"
IMAGE=dosgolem-go-ebiten:1.24-r1
FRAMES=${FRAMES:-700}
for ed in base hd; do
  sfx=""; [[ $ed == hd ]] && sfx=-hd
  app="$ROOT/dist-all/$VER/patch/COKCHT-$VER$sfx-x86_64.AppImage"
  if ! test -f "$app"; then [[ $ed == hd ]] && { echo "略過 hd（沒有 HD 包）"; continue; }; echo "缺 $app" >&2; exit 1; fi
  out="$ROOT/dist-all/$VER/smoke/linux-$ed"
  rm -rf "$out"; mkdir -p "$out/pkg/original" "$out/home"
  cp "$app" "$out/pkg/COKCHT.AppImage"
  cp -r "$ORIG"/. "$out/pkg/original/"
  # 第 1 次：有原版；-scale 3 讓 HD 版畫出 HD。APPIMAGE_EXTRACT_AND_RUN：容器內沒有 FUSE。
  # 第 2 次：拿掉原版與已匯入的 game/，應以錯誤結束。
  timeout 10m docker run --rm --init --network none --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/s/home -e APPIMAGE_EXTRACT_AND_RUN=1 \
    -e COK_NO_DIALOG=1 -e FRAMES="$FRAMES" -v "$out:/s" -w /s/pkg --entrypoint sh "$IMAGE" -c '
      xvfb-run -a ./COKCHT.AppImage -audio=false -scale 3 -frames $FRAMES -shot /s/title.rgba > /s/run.log 2>&1; echo "run exit=$?"
      xvfb-run -a ./COKCHT.AppImage -version  # ebiten 在 init 就開 GLFW，沒有顯示會 panic
      ls /s/home/.local/share/cok-cht
      rm -rf original /s/home/.local/share/cok-cht/game
      xvfb-run -a ./COKCHT.AppImage -audio=false -frames 5 > /s/noorig.log 2>&1; echo "noorig exit=$?"
      ls /s/home/.local/share/cok-cht' > "$out/smoke.txt" 2>&1
  cat "$out/smoke.txt"
  rm -rf "$out/pkg"
done
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$ROOT/dist-all/$VER/smoke":/r -v "$ROOT/tools":/tools:ro cok-img:r1 python -I - <<'PY'
import os
import numpy as np
from PIL import Image
for ed in ('base', 'hd'):
    if not os.path.exists(f'/r/linux-{ed}/title.rgba'):
        continue
    a = np.fromfile(f'/r/linux-{ed}/title.rgba', np.uint8).reshape(600, 960, 4)
    Image.fromarray(a[..., :3]).save(f'/r/linux-{ed}/title.png')
    print(ed, '標題截圖', a[..., :3].reshape(-1, 3).std(axis=0).round(1))
PY
