#!/usr/bin/env bash
# 多語系收據（規格 015 §2、§8）：cok-play 自動模式冷開機到建立隊伍主選單，比對
#   base     改動前的 cok-play（BASE_DG，預設 workplace/dosgolem-base）＋正式譯文
#   ml       多語系 cok-play（ML_DG，預設 workplace/dosgolem）＋正式譯文：要與 base 畫面、記憶體逐位元組相同
#   tw       ml＋測試譯文目錄（正式譯文加一種假語言 zz）、起始 zh-TW：畫面同 base
#   zz       ml＋測試譯文目錄、起始 zz：畫面與 tw 不同，記憶體相同
#   tw2zz    起始 zh-TW，主選單出現後 F8 換到 zz：畫面要與 zz 相同（就地重譯），記憶體相同
#   back     tw2zz 再按一次 F8 回 zh-TW：畫面要與 tw 相同
# 假語言 zz 只在 workplace，不進 Git：由繁中表兩兩互換幾個常用字，字型沿用倚天（譯文子集），替換後仍有字模。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_DG="${BASE_DG:-$ROOT/workplace/dosgolem-base}"
ML_DG="${ML_DG:-$ROOT/workplace/dosgolem}"
ORIG="$ROOT/workplace/input/dos/ckrynn"
FONT="$ROOT/workplace/font"
OUT="$ROOT/workplace/probe/g9-lang-receipt"
FRAMES=${FRAMES:-1600}
KEYS=${KEYS:-1065:p,1325:p}   # -clock 100 每畫格 192,500 步：205M、255M 步送 p（同 G2），300M 步前後是主選單
SWITCH=${SWITCH:-1560}
IMAGE=dosgolem-go-ebiten:1.24-r1
for f in "$ORIG/START.EXE" "$FONT/cok-eten.golemfnt" "$BASE_DG/tools/go.sh" "$ML_DG/tools/go.sh"; do
  test -f "$f" || { echo "缺 $f" >&2; exit 1; }
done
rm -rf "$OUT"; mkdir -p "$OUT/text" "$OUT/bin-base" "$OUT/bin-ml" "$BASE_DG/workplace/bin" "$ML_DG/workplace/bin"
(cd "$BASE_DG" && DOSGOLEM_TIMEOUT=15m tools/go.sh build -o /src/workplace/bin/cok-play-g9 ./cmd/cok-play) && mv "$BASE_DG/workplace/bin/cok-play-g9" "$OUT/bin-base/cok-play"
(cd "$ML_DG" && DOSGOLEM_TIMEOUT=15m tools/go.sh build -o /src/workplace/bin/cok-play-g9 ./cmd/cok-play) && mv "$ML_DG/workplace/bin/cok-play-g9" "$OUT/bin-ml/cok-play"

# 測試譯文目錄：正式譯文全部複製，再由 zh-TW 表產生 zz
cp -a "$ROOT/text/." "$OUT/text/"
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$OUT/text":/t cok-img:r1 python -I - <<'PY'
import os, re
swap = str.maketrans({'的': '了', '了': '的', '你': '我', '我': '你', '隊': '伍', '伍': '隊'})  # 字型是譯文子集，只互換已有的字
def conv(s): return s.translate(swap)
for name in os.listdir('/t'):
    m = re.fullmatch(r'(.+)\.zh-TW\.tsv', name)
    if not m:
        continue
    src = open(f'/t/{name}', encoding='utf-8').read()
    out = conv(src)
    if m.group(1) == 'host-text':
        out += 'lang.name\t攻略\t\njournal.ref\t %d 則%s\t\njournal.refdot\t。\t\n'
        open(f'/t/{name}', 'a', encoding='utf-8').write('lang.name\t中文\t\n')
    open(f'/t/{m.group(1)}.zz.tsv', 'w', encoding='utf-8').write(out)
for stem in ('composed-templates', 'menu-masters'):
    open(f'/t/{stem}.zz.tsv', 'w', encoding='utf-8').write(conv(open(f'/t/{stem}.tsv', encoding='utf-8').read()))
rows = [l.split('\t') for l in open('/t/map-geo.tsv', encoding='utf-8').read().splitlines()]
with open('/t/map-titles.zz.tsv', 'w', encoding='utf-8') as f:
    f.write('map\ttitle\n')
    for r in rows[1:]:
        f.write(f'{r[0]}\t{conv(r[2])}\n')
PY

run() { # 名稱 bin 譯文目錄 腳本 旗標…
  local name=$1 bin=$2 text=$3 script=$4; shift 4
  mkdir -p "$OUT/$name"
  timeout 15m docker run --rm --init --network none --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp -e COK_NO_DIALOG=1 \
    -v "$OUT/$bin":/app:ro -v "$ORIG":/orig:ro -v "$text":/text:ro -v "$FONT":/font:ro -v "$OUT/$name":/out \
    --entrypoint xvfb-run "$IMAGE" -a /app/cok-play -original /orig -save /out/save -text-dir /text \
    -font /font/cok-eten.golemfnt -audio=false -clock 100 -frames "$FRAMES" -script "$script" -shot /out/shot.rgba "$@" \
    > "$OUT/$name/log.txt" 2>&1
}
ZZFONT=(-lang-fonts zz=/font/cok-eten.golemfnt)
run base bin-base "$ROOT/text" "$KEYS" &
run ml bin-ml "$ROOT/text" "$KEYS" &
run tw bin-ml "$OUT/text" "$KEYS" -lang zh-TW "${ZZFONT[@]}" &
wait
run zz bin-ml "$OUT/text" "$KEYS" -lang zz "${ZZFONT[@]}" &
run tw2zz bin-ml "$OUT/text" "$KEYS,$SWITCH:lang" -lang zh-TW "${ZZFONT[@]}" &
run back bin-ml "$OUT/text" "$KEYS,$SWITCH:lang,$((SWITCH + 10)):lang" -lang zh-TW "${ZZFONT[@]}" &
wait
if grep -l "缺字" "$OUT"/*/log.txt; then echo "有缺字" >&2; exit 1; fi
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$OUT":/r cok-img:r1 python -I - <<'PY'
import re
import numpy as np
from PIL import Image
def load(n):
    log = open(f'/r/{n}/log.txt', encoding='utf-8').read()
    m = re.search(r'memory_sha256=([0-9a-f]+) indexed_sha256=([0-9a-f]+)', log)
    return np.fromfile(f'/r/{n}/shot.rgba', np.uint8), m.group(1) if m else None, m.group(2) if m else None
r = {n: load(n) for n in ('base', 'ml', 'tw', 'zz', 'tw2zz', 'back')}
for n, (img, mem, idx) in r.items():
    Image.fromarray(img.reshape(400, 640, 4)[..., :3]).save(f'/r/{n}.png')
    print(f'{n}: memory {mem[:12] if mem else None} indexed {idx[:12] if idx else None}')
ok = True
def check(label, cond):
    global ok
    ok &= bool(cond)
    print(label, '通過' if cond else '不通過')
same = lambda a, b: np.array_equal(r[a][0], r[b][0])
diff = lambda a, b: int((r[a][0].reshape(-1, 4) != r[b][0].reshape(-1, 4)).any(axis=1).sum())
check('記憶體與原版畫面六組相同', len({v[1] for v in r.values()}) == 1 and None not in {v[1] for v in r.values()} and len({v[2] for v in r.values()}) == 1)
check('ml 與 base 畫面相同（繁中回歸）', same('ml', 'base'))
check('tw 與 base 畫面相同（多一種語言不影響繁中）', same('tw', 'base'))
check(f'zz 與 tw 畫面不同（{diff("zz", "tw")} 像素）', diff('zz', 'tw') > 0)
check(f'中途換到 zz 與一開始就是 zz 相同（差 {diff("tw2zz", "zz")} 像素）', same('tw2zz', 'zz'))
check(f'換回 zh-TW 與 tw 相同（差 {diff("back", "tw")} 像素）', same('back', 'tw'))
raise SystemExit(0 if ok else 1)
PY
