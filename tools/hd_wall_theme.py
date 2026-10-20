#!/usr/bin/env python3
"""規格010 §7：由已驗收來源編譯8×8完整組合格，不從截圖取基準。"""
import argparse
from collections import Counter
import json
from pathlib import Path
import shutil
import numpy as np
from PIL import Image
from dax_pics import EGA
from hd_battle_inventory import sha,write

def compile_theme(root,preview=False):
    hd=root/'workplace/hd';model=json.loads((hd/'inventory/issue14/model.json').read_text())
    assets={};specs=[];checks=[]
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;out=hd/'out'/batch;m=json.loads((req/'manifest.json').read_text())
        validation=json.loads((out/'validation.json').read_text())
        if not preview:
            visual=json.loads((out/'visual-review.json').read_text())
            assert validation['all_pass'] and validation['request_sha256']==sha((req/'manifest.json').read_bytes())
            assert visual['request_sha256']==sha((req/'manifest.json').read_bytes())
            vr={r['key']:r for r in visual['items']};nr={r['key']:r for r in validation['items']}
        for it in m['items']:
            raw=(req/it['raw_indexed']).read_bytes();assert sha(raw)==it['raw_indexed_sha256']
            p=out/(it['key']+'.png');assert p.is_file(),p
            if not preview:
                assert vr[it['key']]['approved'] and vr[it['key']]['png_sha256']==sha(p.read_bytes())
                assert nr[it['key']]['target_met'] and nr[it['key']]['png_sha256']==sha(p.read_bytes())
            w,h=it['size'];idx=np.frombuffer(raw,np.uint8).reshape(h,w);pix=np.array(Image.open(p).convert('RGBA'));assert pix.shape==(h*3,w*3,4)
            assets[it['key']]=dict(ref=idx,pix=pix,wall=batch=='b13')
            specs.append(dict(key=it['key'],kind='wall' if batch=='b13' else 'sky',size=[w,h],
                indexed='walls/'+it['key']+'.idx',indexed_sha256=sha(raw),png='walls/'+it['key']+'.png',png_sha256=sha(p.read_bytes())))
        checks.append(dict(batch=batch,request_sha256=sha((req/'manifest.json').read_bytes()),validation_sha256=sha((out/'validation.json').read_bytes()),
            visual_sha256=None if preview else sha((out/'visual-review.json').read_bytes())))
    def compose(base,layers):
        ref=np.full((8,8),base,np.uint8);pix=np.empty((24,24,4),np.uint8);pix[:]=[*EGA[base],255];owner=np.zeros((8,8),bool)
        for layer in layers:
            a=assets[layer['key']];dx,dy=layer['at'];src=a['ref'];h,w=src.shape
            L,R=max(0,dx),min(8,dx+w);T,B=max(0,dy),min(8,dy+h)
            if L>=R or T>=B:raise ValueError('不相交來源')
            q=src[T-dy:B-dy,L-dx:R-dx];mask=q!=13;ref[T:B,L:R][mask]=q[mask];owner[T:B,L:R][mask]=a['wall']
            big=np.repeat(np.repeat(mask,3,0),3,1);part=a['pix'][(T-dy)*3:(B-dy)*3,(L-dx)*3:(R-dx)*3]
            pix[T*3:B*3,L*3:R*3][big]=part[big]
        return ref,pix,bool(owner.sum()>=16 and len(set(map(int,ref[owner])))>=2)
    globals={};local={};ambiguities=[];cells=[]
    for key,a in assets.items():
        if not a['wall'] or (a['ref']==13).any():continue
        ref,pix,guard=compose(0,[dict(key=key,at=[0,0])]);digest=sha(ref.tobytes());assert digest not in globals
        spec=dict(key=key,positions=[],base_color=0,layers=[dict(key=key,at=[0,0])],indexed_sha256=digest,paint=True)
        globals[digest]=(spec,ref,pix,guard);cells.append(spec)
    def add(pos,base,layers,label):
        ref,pix,guard=compose(base,layers);digest=sha(ref.tobytes())
        if digest in globals:return
        k=(pos,digest)
        if k in local:
            old=local[k]
            if not np.array_equal(pix,old[2]):
                old[0]['paint']=False;ambiguities.append(dict(position=pos,indexed_sha256=digest,existing=old[0]['key'],candidate=label))
            return
        spec=dict(key=label,positions=[pos],base_color=base,layers=layers,indexed_sha256=digest,paint=True)
        local[k]=(spec,ref,pix,guard);cells.append(spec)
    def bg_layers(bg,pos):
        x,y=pos%11*8,pos//11*8;layers=[]
        for bid,dx,dy in bg['layers']:
            key=f'sky-{bid:03d}-00';h,w=assets[key]['ref'].shape;dx-=x;dy-=y
            if dx<8 and dy<8 and dx+w>0 and dy+h>0:layers.append(dict(key=key,at=[dx,dy]))
        return layers
    # 僅使用本輪模型的可重建背景與三種已觀測純底色。未觀測時段仍不推測。
    bgs={}
    for r in model['results']:
        bg=r['matches'][0]['bg'];bgs.setdefault(json.dumps(bg,sort_keys=True),bg)
    for color in (0,11,15):
        bg=dict(color=color,mountain=False,sun=None,layers=[[252,0,40]]);bgs.setdefault(json.dumps(bg,sort_keys=True),bg)
    for n,bg in enumerate(bgs.values()):
        for pos in range(121):add(pos,bg['color'],bg_layers(bg,pos),f'background-{n:02d}-{pos:03d}')
    for ri,r in enumerate(model['results']):
        match=r['matches'][0];bg=match['bg']
        for pos,part in enumerate(match['plan']):
            layers=bg_layers(bg,pos)
            if isinstance(part,str):layers.append(dict(key=part,at=[0,0]))
            elif isinstance(part,list):layers.extend(dict(key=key,at=[0,0]) for key in reversed(part))
            add(pos,bg['color'],layers,f'composed-{ri:03d}-{pos:03d}')
    manifest=dict(version=1,scale=3,kind='wall-grid',viewport=[24,24,11,11,8],assets=specs,cells=cells,
        provenance=dict(model_sha256=sha((hd/'inventory/issue14/model.json').read_bytes()),checks=checks,
            description='完整色號從DAX來源重建。背景及已量測組合限格位，未知格全視窗回退。'))
    dest=hd/'out/issue14-theme-preview' if preview else hd/'theme';dest.mkdir(exist_ok=True);data=dest/'walls';data.mkdir(exist_ok=True)
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;out=hd/'out'/batch;m=json.loads((req/'manifest.json').read_text())
        for it in m['items']:
            shutil.copyfile(req/it['raw_indexed'],data/(it['key']+'.idx'));shutil.copyfile(out/(it['key']+'.png'),data/(it['key']+'.png'))
    if preview:write(dest/'theme.json',dict(scale=3,items=[]))
    write(dest/'walls.json',manifest)
    # 供獨立Pillow預期使用，不把組合格重算成新增美術數量。
    write(hd/'out/issue14-theme-receipt.json',dict(assets=len(specs),global_cells=len(globals),position_cells=len(local),ambiguous_cells=sum(not v[0]['paint'] for v in local.values()),
        ambiguity_events=ambiguities,manifest_sha256=sha((dest/'walls.json').read_bytes()),checks=checks,preview=preview))
    print('assets',len(specs),'global',len(globals),'local',len(local),'ambiguous',sum(not v[0]['paint'] for v in local.values()))
    return manifest,assets,globals,local

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);ap.add_argument('--preview',action='store_true');a=ap.parse_args();compile_theme(a.root,a.preview)
