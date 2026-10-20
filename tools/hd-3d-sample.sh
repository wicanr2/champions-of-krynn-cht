#!/usr/bin/env bash
# 3D 視窗 HD 抽樣（Issue #22 的獨立驗證）：六個地區續跑走動，cok-promo 每格輸出 3 倍 HD 畫面與原版色號，
# 在 3D 視窗（原生 (24,24) 88×88，11×11 格，規格 010）逐 8×8 格比較 HD 畫面與「原版色號最近鄰放大 3 倍」：不同＝已換 HD，相同＝保留原版（單色格另計）。
# 用法：tools/hd-3d-sample.sh 標籤；PROMO_THEME 可指定主題目錄；ANALYZE_ONLY=1 只重算既有錄影。輸出 workplace/promo/seg/hd3d-<標籤>-<區>/、彙總 workplace/promo/hd3d-<標籤>.txt。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; P="$ROOT/workplace/probe"; TAG="${1:?標籤}"
WALK='Up,Up,Right,Up,Up,Left,Up,Up,Left,Up,Right,Up'
run() { # 區 狀態目錄 狀態檔 起 迄 [送鍵…]
  local area=$1 sd=$2 st=$3 from=$4 until=$5; shift 5
  "$ROOT/tools/promo-rec.sh" "hd3d-$TAG-$area" "$sd" "$st" "$from" "$until" -fps 1 -indexed "$@" >/dev/null
}
if [[ -z "${ANALYZE_ONLY:-}" ]]; then
run demo "$P/demo3d" d450.state 450000000 700000000 &
run throtl1 "$P/throtl1" end.state 19850000000 20090000000 -key-names "$WALK" -key-from 19862000000 -key-every 20000000; wait
run throtl2 "$P/area2/THROTL" end.state 12208000000 12450000000 -key-names "$WALK" -key-from 12218000000 -key-every 20000000 &
run gargath "$P/area1/GARGATH" end.state 5443000000 5685000000 -key-names "$WALK" -key-from 5453000000 -key-every 20000000; wait
run gargath2 "$P/area1/GARGATH2" end.state 9366000000 9608000000 -key-names "$WALK" -key-from 9376000000 -key-every 20000000 &
run sanction "$P/area2/SANCTION" end.state 16982000000 17224000000 -key-names "$WALK" -key-from 16992000000 -key-every 20000000; wait
fi
timeout 600s docker run -i --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 2 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$ROOT/workplace/promo:/p" -e TAG="$TAG" --entrypoint python cok-img:r1 -I - <<'PY' | tee "$ROOT/workplace/promo/hd3d-$TAG.txt"
import os, glob
import numpy as np
from PIL import Image, ImageDraw
tag = os.environ["TAG"]; rows = []
for d in sorted(glob.glob(f"/p/seg/hd3d-{tag}-*")):
    area = d.rsplit("-", 1)[1]; hd = orig = flat = frames3d = 0; thumbs = []
    for f in sorted(glob.glob(d + "/frames/*.idx")):
        idx = np.frombuffer(open(f, "rb").read(), np.uint8).reshape(200, 320)
        pal = np.frombuffer(open(f[:-4] + ".pal", "rb").read(), np.uint8).reshape(256, 3)
        img = np.asarray(Image.open(f[:-4] + ".png").convert("RGB"))
        up = pal[idx].repeat(3, 0).repeat(3, 1)
        win = idx[24:112, 24:112]
        if len(np.unique(win)) < 4:  # 不是 3D 畫面（黑屏、全螢幕圖）
            continue
        frames3d += 1; h = o = fl = 0
        for cy in range(11):
            for cx in range(11):
                y, x = 24 + cy * 8, 24 + cx * 8
                same = np.array_equal(img[y*3:y*3+24, x*3:x*3+24], up[y*3:y*3+24, x*3:x*3+24])
                if not same: h += 1
                elif len(np.unique(idx[y:y+8, x:x+8])) == 1: fl += 1
                else: o += 1
        hd += h; orig += o; flat += fl
        thumbs.append((Image.fromarray(img[72:336, 72:336]).resize((176, 176)), h, o))
    if frames3d:
        tot = hd + orig
        rows.append(f"{area}: 3D 畫格 {frames3d}，HD 格 {hd}，保留原版（非單色）{orig}，單色格 {flat}，HD 覆蓋率 {hd*100/max(tot,1):.1f}%")
        cols = 8; sheet = Image.new("RGB", (cols * 178, ((len(thumbs) + cols - 1) // cols) * 190), "white")
        for k, (t, h, o) in enumerate(thumbs):
            x, y = k % cols * 178, k // cols * 190
            sheet.paste(t, (x, y + 12)); ImageDraw.Draw(sheet).text((x + 2, y), f"HD {h} 原 {o}", fill="black")
        sheet.save(f"/p/hd3d-{tag}-{area}.png")
    else:
        rows.append(f"{area}: 沒有 3D 畫格")
print("\n".join(rows))
PY
