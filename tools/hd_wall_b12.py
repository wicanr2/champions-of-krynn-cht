#!/usr/bin/env python3
"""規格010 §7：b12來源與3D視窗的精確位置反查。未命中不推定原版未用。"""
import argparse
from pathlib import Path
from collections import defaultdict
import numpy as np
from hd_wall_inventory import load_sources,saved_screens
from hd_battle_inventory import sha,write

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',required=True,type=Path);a=ap.parse_args()
    tiles,defs,pictures,files=load_sources(a.root);rows=[p for p in pictures if p['key'].startswith(('head','body','tiles'))]
    lookups=defaultdict(list);parts=[]
    for p in rows:
        if p['key'].startswith('tiles'):lookups[sha(p['raw'])].append(p['key'])
        else:parts.append(p)
    results=[];screens=saved_screens(a.root)
    for name,raw in screens:
        arr=np.frombuffer(raw,np.uint8).reshape(200,320)
        for p in parts:
            w,h=p['size'];y=64 if p['key'].startswith('body') else 24
            q=arr[y:y+h,24:24+w]
            if q.tobytes()==p['raw']:results.append(dict(screen=name,kind='portrait-part',keys=[p['key']],position=[24,y]))
        for y in range(24,89,8):
            for x in range(24,89,8):
                keys=lookups.get(sha(arr[y:y+24,x:x+24].tobytes()))
                if keys:results.append(dict(screen=name,kind='3d-window-24x24-exact',keys=keys,position=[x,y]))
    write(a.root/'workplace/hd/inventory/issue14/b12-review.json',dict(issue=14,source_files={k:v for k,v in files.items() if k.startswith(('HEAD','BODY','TILES'))},
        inputs=[dict(key=p['key'],size=p['size'],source=p['source'],raw_sha256=sha(p['raw'])) for p in rows],
        saved_screens_tested=len(screens),matches=results,
        method='HEAD在(24,24)、BODY在(24,64)完整原始色號；TILES在3D視窗8像素格點逐24×24精確匹配。未套調色或透明假說。',
        decision='維持b12 DRAFT，保留原版；本輪未證實其3D輸出位置與遮擋。',
        limits='未命中不代表原版沒有使用。未查證原版loader／callsite；沒有以圖片內容猜測呈現家族。'))
    print('b12 parts',len(rows),'screens',len(screens),'matches',len(results))

if __name__=='__main__':main()
