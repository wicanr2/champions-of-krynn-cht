#!/usr/bin/env bash
# 規格 017 收據：cok-play 冷開機跑 demo 戰鬥（不送鍵，-clock 100，約 1.7e9 步），無敵關與開各一次。
# 要求：開啟時有還原、隊員 HP 與狀態不低於開戰前；關閉時不印無敵報告；兩次都沒有缺字。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/workplace/dosgolem/workplace/bin"; ORIG="$ROOT/workplace/input/dos/ckrynn"; FONT="$ROOT/workplace/font"
OUT="$ROOT/workplace/probe/g10-cheat"; FRAMES=${FRAMES:-8840}; IMAGE=dosgolem-go-ebiten:1.24-r1
for f in "$BIN/cok-play" "$FONT/cok-eten.golemfnt" "$ORIG/START.EXE"; do test -f "$f" || { echo "缺 $f" >&2; exit 1; }; done
rm -rf "$OUT"; mkdir -p "$OUT"
run() { # 名稱 旗標…
  local name=$1; shift; mkdir -p "$OUT/$name"
  timeout 40m docker run --rm --init --network none --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp -e COK_NO_DIALOG=1 \
    -v "$BIN":/app:ro -v "$ORIG":/orig:ro -v "$ROOT/text":/text:ro -v "$FONT":/font:ro -v "$OUT/$name":/out \
    --entrypoint xvfb-run "$IMAGE" -a /app/cok-play -original /orig -save /out/save -text-dir /text \
    -font /font/cok-eten.golemfnt -audio=false -clock 100 -frames "$FRAMES" -shot /out/shot.rgba "$@" > "$OUT/$name/log.txt" 2>&1
}
run off & run on -cheat; wait
grep -h "無敵\|memory_sha256" "$OUT"/*/log.txt
if grep -l "缺字" "$OUT"/*/log.txt; then echo "有缺字" >&2; exit 1; fi
grep -q "無敵" "$OUT/off/log.txt" && { echo "關閉時不該有無敵報告" >&2; exit 1; }
line=$(grep "無敵" "$OUT/on/log.txt") || { echo "開啟時沒有無敵報告" >&2; exit 1; }
n=$(sed -E 's/.*還原 ([0-9]+) 次.*/\1/' <<< "$line"); [[ "$n" -gt 0 ]] || { echo "開啟時沒有任何還原" >&2; exit 1; }
grep -q "/0；" <<< "$line" && ! grep -qE "/[1-9][0-9]*；" <<< "$line" || { echo "有隊員狀態異常" >&2; exit 1; }
# 第二段：預建六人隊在大地圖遇山巨人與卡帕克（workplace/probe/explore2 結束狀態，重播 explore3 的送鍵紀錄，
# 戰鬥中按 Q 交給電腦）。原版有人被麻痺、倒下；無敵開啟時要有還原與效果移除，結束時全員狀態 0、沒有有害效果。
KA=$(grep -E "^#[0-9]+ 送 " "$ROOT/workplace/probe/explore3/explore.log" | sed -E 's/^#([0-9]+) 送 (.*)$/\1:\2/' | awk -F: '$1<5100000000' | paste -sd, -)
EX="$ROOT/workplace/probe/explore2"; test -f "$EX/end.state"
"$ROOT/tools/promo-rec.sh" cheat-ex3-off "$EX" end.state 5000000000 5000000001 -fps 1 -keys-at "$KA" -dump-mem-at 5000000000 >/dev/null &
"$ROOT/tools/promo-rec.sh" cheat-ex3-on "$EX" end.state 5000000000 5000000001 -fps 1 -keys-at "$KA" -cheat -dump-mem-at 5000000000 >/dev/null
wait
docker run -i --rm --network none -u "$(id -u):$(id -g)" --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$ROOT/workplace/promo/seg:/d:ro" python:3.13-slim python -I - <<'PY'
import json, struct, sys
HARM = {0x1F, 0x21, 0x2A, 0x33, 0x34, 0x35}
def far(m, a):
    o, s = struct.unpack_from("<HH", m, a); return s * 16 + o
def party(name):
    m = open(f"/d/{name}/mem-5000000000.bin", "rb").read(); a = far(m, 0xCD60 + 0x609A); out = []
    while a and len(out) < 12 and m[a + 0x18A] == 0:
        t, q = [], far(m, a + 0xE3)
        while q and len(t) < 64: t.append(m[q]); q = far(m, q + 5)
        out.append((m[a+1:a+1+m[a]].decode(), m[a+0x197], m[a+0x188], t)); a = far(m, a + 0x17F)
    return out
ok = True
for name in ("cheat-ex3-off", "cheat-ex3-on"):
    p = party(name); r = json.load(open(f"/d/{name}/receipt.json"))
    bad = [x for x in p if x[2] != 0 or HARM & set(x[3])]
    print(name, "還原", r["cheat_restored"], "移除", r["cheat_removed_affects"], "；", "、".join(f"{n} HP{h} 狀態{s}" for n, h, s, _ in p))
    if name.endswith("on"):
        ok &= not bad and r["cheat_restored"] > 0 and r["cheat_removed_affects"] > 0
    else:
        ok &= bool(bad) or any(h < 20 for _, h, _, _ in p)  # 原版要真的有傷亡或異常，對照才有意義
sys.exit(0 if ok else 1)
PY
echo "通過"
