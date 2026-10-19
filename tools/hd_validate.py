#!/usr/bin/env python3
"""規格 010 §20：批次數值閘門，補 hd_compare 的格式檢查與人工判讀。

Docker 入口：dkk-pyimg:r1，--req 請求目錄 --hd 輸出目錄。
"""
import argparse
import hashlib
import json
from pathlib import Path
import numpy as np
from PIL import Image, ImageFilter
from hd_compare import read_png, align_score, hue_score, structural_score


def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--req',required=True,type=Path)
    ap.add_argument('--hd',required=True,type=Path)
    a=ap.parse_args()
    request=json.loads((a.req/'manifest.json').read_text())
    rows=[]
    for item in request['items']:
        key=item['key']; p=a.hd/f'{key}.png'
        if item.get('disposition'):
            rows.append(dict(key=key,required=False,target_met=None,status=item['disposition']))
            continue
        if not p.exists():
            rows.append(dict(key=key,target_met=False,error='缺輸出'))
            continue
        ct,hd=read_png(p)
        idx=(a.req/item['indexed']).read_bytes()
        assert hashlib.sha256(idx).hexdigest()==item['indexed_sha256'],key
        w,h=item['size']; W,H=item['target_size']
        assert len(idx)==w*h and max(idx)<16,key
        orig=np.array(request['palette']['rgb'],np.uint8)[np.frombuffer(idx,np.uint8).reshape(h,w)]
        good=ct==6 and hd.shape==(H,W,4) and bool(np.all(hd[...,3]==255))
        edge,structure,hue,off=0.,0.,360.,None
        if hd.shape[:2]==(H,W):
            edge=float(align_score(orig,hd)[0]); hue=float(hue_score(orig,hd)[0])
            structure=structural_score(orig,hd)
            anim=item.get('composition') or item.get('animation') or {}
            if anim.get('mask'):
                m=np.asarray(Image.open(a.req/anim['mask']).convert('L').filter(ImageFilter.MaxFilter(13)))>0
                bp=a.hd/f'{anim["base"]}.png'
                if not bp.exists():
                    off=-1
                else:
                    base=read_png(bp)[1]
                    off=int(np.any(base!=hd,axis=2)[~m].sum()) if base.shape==hd.shape else -1
        constant = item.get('rendering')=='native_exact_constant' and len(set(idx))==1
        constant_exact = constant and hd.shape==(H,W,4) and bool(np.all(hd[...,:3]==orig[0,0]))
        preserved_difference=0
        if hd.shape==(H,W,4):
            native=np.repeat(np.repeat(np.frombuffer(idx,np.uint8).reshape(h,w),H//h,0),W//w,1)
            for color in item.get('preserve_native_colors',[]):
                preserved_difference+=int(np.any(hd[...,:3]!=request['palette']['rgb'][color],axis=2)[native==color].sum())
        good &= (constant_exact or structure>=.4) and hue<30 and off in (None,0) and preserved_difference==0
        rows.append(dict(key=key,edge_overlap=edge,median_hue_difference_degrees=hue,
                         structural_edge_overlap=structure,
                         constant_source_exact=bool(constant_exact),
                         preserved_color_difference=preserved_difference,
                         outside_mask_difference=off,format_ok=ct==6 and hd.shape==(H,W,4),
                         alpha_ok=bool(np.all(hd[...,3]==255)),
                         target_met=bool(good),png_sha256=hashlib.sha256(p.read_bytes()).hexdigest()))
        print(key,f'raw={edge:.3f} structure={structure:.3f} hue={hue:.1f} off={off}',bool(good))
    result=dict(batch=request['batch'],spec=request['spec'],
                request_manifest_sha256=hashlib.sha256((a.req/'manifest.json').read_bytes()).hexdigest(),
                all_pass=all(r['target_met'] for r in rows if r.get('required',True)),items=rows)
    (a.hd/'validation.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    raise SystemExit(0 if result['all_pass'] else 1)


if __name__=='__main__': main()
