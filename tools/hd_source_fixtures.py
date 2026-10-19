#!/usr/bin/env python3
"""Issue #12 保存畫面正規化；來源衍生正負對照由 Go 收據另行生成。

入口：docs/spec/010-hd-theme.md §5.3。只接受 EGA 最近鄰整數倍。
"""
import argparse
import json
from pathlib import Path
import numpy as np
from PIL import Image
from dax_pics import EGA
from hd_coverage import sha


def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--root',type=Path,required=True)
    ap.add_argument('--out',type=Path,required=True)
    a=ap.parse_args();a.out.mkdir(parents=True,exist_ok=True)
    paths=set((a.root/'workplace/probe').rglob('*-base.png'))
    paths.update(p for p in (a.root/'workplace/hd/screens').glob('*.bin') if p.stat().st_size==64000)
    rows=[]
    for n,p in enumerate(sorted(paths)):
        if p.suffix=='.bin':raw=bytes(x&15 for x in p.read_bytes())
        else:
            rgb=np.asarray(Image.open(p).convert('RGB'));h,w=rgb.shape[:2]
            s=w//320;assert s>0 and (w,h)==(320*s,200*s),p
            small=rgb[::s,::s]
            assert np.array_equal(rgb,np.repeat(np.repeat(small,s,0),s,1)),p
            idx=np.full((200,320),255,np.uint8)
            for k,color in enumerate(EGA):idx[np.all(small==color,axis=2)]=k
            assert idx.max()<16,p
            raw=idx.tobytes()
        name=f'{n:04d}.bin';(a.out/name).write_bytes(raw)
        rows.append(dict(file=name,source=str(p.relative_to(a.root)),source_sha256=sha(p.read_bytes()),indexed_sha256=sha(raw)))
    (a.out/'manifest.json').write_text(json.dumps(dict(evidence_class='saved original screen normalization',items=rows),ensure_ascii=False,indent=2)+'\n')
    print(len(rows),'份原版側保存畫面，未生成遊戲狀態')


if __name__=='__main__':main()
