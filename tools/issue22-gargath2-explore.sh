#!/usr/bin/env bash
# 從指定GARGATH2狀態的正常戰後大地圖重入；不注入座標或戰鬥結果。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
run() {
 local scene="$1" st="$2" until="$3" from="$4" keys="$5" scale="$6" mode="$7"
 test -f "$st/end.state";test -d "$st/scratch";test -f "$HD/out/issue22-cok-hd-probe"
 for p in "$ROOT/workplace/input/dos/ckrynn" "$ROOT/workplace/font" "$ROOT/text";do test -d "$p";done
 local name="$scene-$mode$scale"
 bash "$ROOT/tools/issue22-hd-img.sh" python -B -c '
from pathlib import Path
import shutil,sys
p=Path("/src/workplace/hd/out/issue22-runtime")/sys.argv[1]
if p.exists():shutil.rmtree(p)
p.mkdir(parents=True)
shutil.copytree(Path(sys.argv[2])/"scratch",p/"scratch")
' "$name" "/src/${st#"$ROOT/"}"
 local out="$HD/out/issue22-runtime/$name";local args=()
 [[ "$mode" == hd ]] && args=(-hd /hd/theme)
 DOSGOLEM_ORIG="$ROOT/workplace/input/dos/ckrynn" DOSGOLEM_GO_CMD=/hd/probe \
 DOSGOLEM_EXTRA_MOUNTS="$st/end.state:/source.state:ro;$out:/out;$HD/theme:/hd/theme:ro;$HD/out/issue22-cok-hd-probe:/hd/probe:ro;$ROOT/workplace/font:/font:ro;$ROOT/text:/text:ro" \
 bash "$ROOT/tools/issue22-hd-go.sh" -state /source.state -root /orig -scratch /out/scratch -out /out \
  -until "$until" -sample 100000 -limit 100 -scale "$scale" -font /font/cok-eten.golemfnt -text-dir /text \
  -key-from "$from" -key-every 20000000 -key-names "$keys" -save-state /out/end.state \
  "${args[@]}" > "$out/log.txt" 2>&1
}
# 此原始狀態正常重入即進入戰鬥，尚未找到可自由探索的入口；保留診斷，不臆造探索樣本。
run gargath2-entry "$HD/out/issue22-runtime/gargath2-base3" 10020000000 9986000000 'Y,Return' 3 base
