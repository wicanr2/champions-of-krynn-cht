#!/usr/bin/env bash
# Windows zip 的 Wine smoke（規格 012 §4 第 3 項；做法取自拯救地球 tools/release/verify-windows.sh）：
# 解開 zip、原版放在 exe 旁，自動模式跑到標題畫面截圖；再驗缺原版時以錯誤結束並留下 cok-error.log。
# ⚠ Wine 不是 Windows。原版、HD 兩包各跑一次。
#
#   tools/release/smoke-windows.sh <版本>
# 輸出：dist-all/<版本>/smoke/windows-<版別>/。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VER="${1:?版本}"
ORIG="$ROOT/workplace/input/dos/ckrynn"
IMAGE=psgolem-wine-verify:noble-r2  # 只使用，不刪除、不改標籤
FRAMES=${FRAMES:-700}
for ed in base hd; do
  sfx=""; [[ $ed == hd ]] && sfx=-hd
  zip="$ROOT/dist-all/$VER/patch/COKCHT-$VER$sfx-win64.zip"
  if ! test -f "$zip"; then [[ $ed == hd ]] && { echo "略過 hd（沒有 HD 包）"; continue; }; echo "缺 $zip" >&2; exit 1; fi
  out="$ROOT/dist-all/$VER/smoke/windows-$ed"
  rm -rf "$out"; mkdir -p "$out/win"
  timeout 120 docker run --rm --network none --memory 512m --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" -v "$zip:/z.zip:ro" -v "$out:/v" python:3.13-slim \
    python -I -c "import zipfile; zipfile.ZipFile('/z.zip').extractall('/v/win')"
  mkdir -p "$out/win/COKCHT/original"; cp -r "$ORIG"/. "$out/win/COKCHT/original/"
  timeout 30m docker run --rm --init --network none --memory 4g --cpus 2 --pids-limit 512 --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" -e HOME=/wine -e WINEDEBUG=-all -e WINEDLLOVERRIDES="mscoree,mshtml=" -e COK_NO_DIALOG=1 \
    -e F="$FRAMES" -v "$out:/v" --tmpfs "/wine:exec,uid=$(id -u),gid=$(id -g),size=2g" --entrypoint bash "$IMAGE" -c '
export WINEPREFIX=/wine/prefix
mkdir -p /tmp/.X11-unix; Xvfb :99 -screen 0 1280x800x24 -nolisten tcp -ac >/dev/null 2>&1 &
i=0; while [ ! -S /tmp/.X11-unix/X99 ] && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done; export DISPLAY=:99
wine --version; wine wineboot -i >/dev/null 2>&1
cd /v/win/COKCHT
wine ./COKCHT.exe -audio=false -scale 3 -frames $F -shot "Z:\\v\\title.rgba" > /v/run.log 2>&1; echo "run exit=$?"
D=$(ls -d /wine/prefix/drive_c/users/*/AppData/Roaming/COKCHT); ls "$D"
rm -rf original "$D/game"
wine ./COKCHT.exe -audio=false -frames 5 > /v/noorig.log 2>&1; echo "noorig exit=$?"
cp "$D/cok-error.log" /v/ 2>/dev/null; ls "$D"' > "$out/smoke.txt" 2>&1
  cat "$out/smoke.txt"
  rm -rf "$out/win"
done
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$ROOT/dist-all/$VER/smoke":/r -v "$ROOT/tools":/tools:ro cok-img:r1 python -I - <<'PY'
import os
import numpy as np
from PIL import Image
for ed in ('base', 'hd'):
    if not os.path.exists(f'/r/windows-{ed}/title.rgba'):
        continue
    a = np.fromfile(f'/r/windows-{ed}/title.rgba', np.uint8).reshape(600, 960, 4)
    Image.fromarray(a[..., :3]).save(f'/r/windows-{ed}/title.png')
    l = np.fromfile(f'/r/linux-{ed}/title.rgba', np.uint8).reshape(600, 960, 4)
    print(ed, 'Windows 標題截圖與 Linux 相同' if (a == l).all() else f'與 Linux 不同 {(a != l).any(axis=2).sum()} 像素')
PY
