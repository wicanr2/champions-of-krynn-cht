#!/usr/bin/env bash
# 規格 010 §5：固定前端二進位，從正常冷啟動取得 3 倍及 2 倍 A/B。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HD="$ROOT/workplace/hd"
DG="$ROOT/workplace/dosgolem"
ORIG="$ROOT/workplace/input/dos/ckrynn"
FONT="$ROOT/workplace/font"
TEXT="$ROOT/text"
for d in "$HD/out" "$HD/theme" "$DG/workplace/bin" "$ORIG" "$FONT" "$TEXT"; do test -d "$d"; done
test -f "$DG/workplace/bin/cok-play"
test -f "$HD/theme/theme.json"
timeout 30s docker run -i --rm --network none -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$HD:/hd" cok-img:r1 python - <<'PY'
from pathlib import Path
for n in ('title-base3','title-hd3','dragon-base3','dragon-hd3','dragon-base2','dragon-hd2'):
    (Path('/hd/out/runtime')/n).mkdir(parents=True,exist_ok=True)
PY
run() {
  local name="$1" frames="$2" scale="$3"; shift 3
  local output="$HD/out/runtime/$name"
  test -d "$output"
  timeout 60s docker run --rm --init --network none -u "$(id -u):$(id -g)" \
    --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
    -e HOME=/tmp -e COK_NO_DIALOG=1 -e GOMAXPROCS=2 \
    -v "$DG/workplace/bin:/app:ro" -v "$ORIG:/orig:ro" -v "$FONT:/font:ro" \
    -v "$TEXT:/text:ro" -v "$HD/theme:/hd:ro" -v "$output:/out" \
    --entrypoint xvfb-run dosgolem-go-ebiten:1.24-r1 -a /app/cok-play \
    -original /orig -save /out/save -text-dir /text -font /font/cok-eten.golemfnt \
    -audio=false -frames "$frames" -scale "$scale" -shot /out/shot.rgba "$@" \
    > "$output/log.txt" 2>&1
}
run title-base3 550 3
run title-hd3 550 3 -hd /hd
run dragon-base3 750 3
run dragon-hd3 750 3 -hd /hd
run dragon-base2 750 2
run dragon-hd2 750 2 -hd /hd
timeout 30s docker run -i --rm --network none -u "$(id -u):$(id -g)" \
  --memory 2g --cpus 2 --pids-limit 64 --log-opt max-size=10m --log-opt max-file=3 \
  -v "$HD:/hd" -v "$DG/workplace/bin:/app:ro" cok-img:r1 python - <<'PY'
import json, re, hashlib
from pathlib import Path
import numpy as np
from PIL import Image
r=Path('/hd/out/runtime');rows=[]
for prefix,s in (('title',3),('dragon',3),('dragon',2)):
    a=r/f'{prefix}-base{s}';b=r/f'{prefix}-hd{s}'
    sa=a.joinpath('shot.rgba').read_bytes();sb=b.joinpath('shot.rgba').read_bytes()
    pa=np.frombuffer(sa,np.uint8).reshape(200*s,320*s,4);pb=np.frombuffer(sb,np.uint8).reshape(200*s,320*s,4)
    diff=int(np.any(pa!=pb,axis=2).sum())
    la=a.joinpath('log.txt').read_text();lb=b.joinpath('log.txt').read_text()
    state=lambda log:dict(re.findall(r'(steps|memory_sha256|indexed_sha256)=([a-f0-9]+)',log))
    aa,bb=state(la),state(lb)
    assert set(aa)=={'steps','memory_sha256','indexed_sha256'},la
    ok=aa==bb and (diff>0 if s==3 else diff==0)
    row=dict(scene=prefix,scale=s,state_without_hd=aa,state_with_hd=bb,display_difference=diff,ok=ok)
    rows.append(row)
    Image.fromarray(pa).save(a/'shot.png');Image.fromarray(pb).save(b/'shot.png')
    print(prefix,s,'倍，差異',diff,'原版記憶體與色號相同',aa==bb)
receipt=dict(evidence_class='dosgolem cold startup frontend A/B',tool='cok-play',
    binary_sha256=hashlib.sha256(Path('/app/cok-play').read_bytes()).hexdigest(),
    theme_manifest_sha256=hashlib.sha256(Path('/hd/theme/theme.json').read_bytes()).hexdigest(),
    randomness='固定冷啟動與相同畫格數；標題無亂數比較點，未控制 seed',
    limitations='未量測存檔或逐項暫存器；未進入新 PIC/BIGPIC 的正常玩家路徑。',
    items=rows,all_pass=all(x['ok'] for x in rows))
(r/'receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n')
assert receipt['all_pass']
PY
