#!/usr/bin/env python3
"""規格 010 §6：SPRIT 候選位置與一對一色號映射量測，不授權呈現。"""
import argparse
import json
from pathlib import Path
import numpy as np
from PIL import Image
from hd_battle_inventory import sha,write
from dax_pics import EGA

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);a=ap.parse_args()
    root=a.root;hd=root/'workplace/hd'; inv=hd/'inventory/issue13'
    rows=json.loads((inv/'inventory.json').read_text())['items']
    sources=[]
    for r in rows:
        if r['kind']!='encounter':continue
        w,h=r['size'];p=np.frombuffer((inv/r['indexed']).read_bytes(),np.uint8).reshape(h,w)
        mask=p!=0; colors=np.unique(p[mask]); groups=[np.where(p==c) for c in colors]
        anchors=[(yy[0],xx[0],yy[len(yy)//2],xx[len(xx)//2],yy[-1],xx[-1]) for yy,xx in groups]
        sources.append((r,p,mask,colors,groups,anchors))
    paths=set(p for p in (root/'workplace/probe').rglob('*.bin') if p.stat().st_size==64000)
    paths.update(p for p in (hd/'out/issue13-runtime').rglob('*.bin') if p.stat().st_size==64000)
    # Normal manifest 已核對 EGA；同一畫格只量測一次。
    paths.update((hd/'screens/normal').glob('*.bin'));seen=set();matches=[];count=0
    for path in sorted(paths):
        raw=path.read_bytes();sig=sha(raw)
        if sig in seen:continue
        seen.add(sig);count+=1
        frame=np.frombuffer(raw,np.uint8).reshape(200,320)
        for r,p,mask,colors,groups,anchors in sources:
            h,w=p.shape
            for y in range(8,129-h,8):
                for x in range(8,129-w,8):
                    q=frame[y:y+h,x:x+w];mapped=[];ok=True
                    for ay,ax,by,bx,cy,cx in anchors:
                        v=q[ay,ax]
                        if v!=q[by,bx] or v!=q[cy,cx]:ok=False;break
                        mapped.append(int(v))
                    if not ok or len(set(mapped))!=len(mapped) or len(mapped)<3:continue
                    if not all(np.all(q[g]==v) for g,v in zip(groups,mapped)):continue
                    matches.append(dict(key=r['key'],screen=str(path.relative_to(root)),screen_sha256=sig,
                        x=x,y=y,size=r['size'],source_foreground=int(mask.sum()),
                        mapping={int(c):v for c,v in zip(colors,mapped)},
                        background_colors=list(map(int,np.unique(q[~mask]))),method='來源非0前景逐點一致；一對一映射由本畫面推得，非預設'))
    write(inv/'encounter-probe.json',dict(screens=count,matches=matches,
        limits='候選映射須另經跨畫面與獨立來源檢查；未命中不代表原版未使用。'))
    print('screens',count,'matches',len(matches));print(json.dumps(matches[:5],ensure_ascii=False))

if __name__=='__main__':main()
