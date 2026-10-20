#!/usr/bin/env python3
"""Issue 21 的來源盤點、逐色號搜尋與交付索引。只在 Docker 執行。"""
import argparse
from collections import Counter
import json
from pathlib import Path
import platform
import numpy as np
from PIL import Image, ImageDraw
import PIL
from dax_pics import EGA, picture
from hd_battle_inventory import source_records, sha, write
from hd_wall_inventory import saved_screens

ROOT = Path('/src')
HD = ROOT/'workplace/hd'
DEST = HD/'inventory/issue21'

def exact_positions(screen, pattern, transparent=None):
    h,w = pattern.shape
    sh,sw = screen.shape
    hh,ww = sh-h+1,sw-w+1
    if hh<=0 or ww<=0: return []
    mask = np.ones(pattern.shape,bool) if transparent is None else pattern!=transparent
    yy,xx=np.nonzero(mask)
    if len(xx)<16: return []
    # 從稀有色開始篩選，最後仍逐點核對整個矩形或全部前景。
    counts=np.bincount(screen.ravel(),minlength=16)
    order=np.argsort(counts[pattern[yy,xx]],kind='stable')
    select=np.unique(np.r_[order[:8],np.linspace(0,len(xx)-1,24,dtype=int)])
    possible=np.ones((hh,ww),bool)
    for n in select:
        x,y=int(xx[n]),int(yy[n]); possible &= screen[y:y+hh,x:x+ww]==pattern[y,x]
        if not possible.any(): return []
    out=[]
    for y,x in zip(*np.nonzero(possible)):
        q=screen[y:y+h,x:x+w]
        if np.array_equal(q[mask],pattern[mask]):out.append([int(x),int(y)])
    return out

def inventory():
    DEST.mkdir(exist_ok=True)
    original=ROOT/'workplace/input/dos/ckrynn'
    rows={}; files={}
    for name in ('HEAD2','HEAD3','BODY2','BODY3','TILES','CURSOR','COMSPR','RANDCOM'):
        filename=name+'.DAX';files[filename]=sha((original/filename).read_bytes())
        for bid,raw,src in source_records(original,filename):
            w,h,count,x,y,px=picture(raw)
            for n in range(count):
                idx=px[n*w*h:(n+1)*w*h]
                family='portrait-head' if name.startswith('HEAD') else 'portrait-body' if name.startswith('BODY') else name.lower()
                sig=(family,w,h,sha(idx))
                r=rows.setdefault(sig,dict(key=f'{name.lower()}-{bid:03d}-{n:02d}',family=family,size=[w,h],indexed_sha256=sha(idx),sources=[],colors=dict(Counter(idx))))
                r['sources'].append(dict(src,item=n,header_xy=[x,y]));r['raw']=idx
    for r in rows.values():
        raw=r.pop('raw');r['indexed']=r['key']+'.idx';(DEST/r['indexed']).write_bytes(raw)
        im=Image.fromarray(np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(r['size'][1],r['size'][0])])
        im.resize((r['size'][0]*3,r['size'][1]*3),Image.Resampling.NEAREST).save(DEST/(r['key']+'.png'))
    data=dict(issue=21,date='2026-10-20',address_space='DAX 檔案 offset；dosgolem 原版 320×200 色號',files=files,
        tools=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__),items=list(rows.values()))
    write(DEST/'sources.json',data)
    print('sources',Counter(r['family'] for r in rows.values()),'aliases',sum(len(r['sources']) for r in rows.values()))
    screenrows=saved_screens(ROOT)
    for base in ('out/issue13-runtime/demo-source','out/issue13-runtime/battle-source'):
        screenrows += [(str(p.relative_to(ROOT)),p.read_bytes()) for p in sorted((HD/base).glob('*.bin'))]
    # 新增304張戰鬥畫格由battle_search獨立處理，維持此批輸入集合可重播。
    # 相同畫面只搜尋一次，保留所有出處。
    screens={}
    for name,px in screenrows:
        if len(px)!=64000:continue
        r=screens.setdefault(sha(px),dict(raw=px,paths=[]));r['paths'].append(name)
    matches=[];portrait=[];variants=[]
    for r in data['items']:
        raw=(DEST/r['indexed']).read_bytes();a=np.frombuffer(raw,np.uint8).reshape(r['size'][1],r['size'][0])
        if r['family']=='comspr': a=np.where(a==0,8,np.where(a==8,0,a)).astype(np.uint8)
        variants.append((r,'full-native' if r['family']!='comspr' else 'full-combat-0-8-swap',a,None))
        if r['family'].startswith('portrait'):
            # 前作 masked-picture 假說只供研究，不直接成為呈現契約。
            b=np.where(a==13,0,a).astype(np.uint8)
            variants.append((r,'hypothesis-foreground-0-mask-13-to-0',b,0))
    for si,(digest,s) in enumerate(screens.items()):
        screen=np.frombuffer(s['raw'],np.uint8).reshape(200,320)
        for r,method,a,transparent in variants:
            positions=exact_positions(screen,a,transparent)
            if positions:matches.append(dict(key=r['key'],family=r['family'],method=method,positions=positions,screen_sha256=digest,screens=s['paths']))
        if si%50==0:print('searched',si,'/',len(screens),flush=True)
    write(DEST/'search.json',dict(issue=21,screen_count=len(screens),input_count=len(screenrows),
        screens=[dict(sha256=k,paths=v['paths']) for k,v in screens.items()],matches=matches,
        scope='全 320×200、每個像素位置；完整矩形包含背景。另列 HEAD/BODY 前作透明假說搜尋，不能當作已驗證契約。'))
    print('matches',[(m['key'],m['method'],m['positions']) for m in matches])
    for family in ('portrait','tiles','cursor','comspr','randcom'):
        group=[r for r in data['items'] if r['family'].startswith(family)]
        canvas=Image.new('RGB',(1024,((len(group)+3)//4)*180),(32,32,40));d=ImageDraw.Draw(canvas)
        for n,r in enumerate(group):
            im=Image.open(DEST/(r['key']+'.png'));im.thumbnail((240,145))
            x,y=n%4*256,n//4*180;canvas.paste(im,(x+(256-im.width)//2,y+25));d.text((x+4,y+4),r['key'],fill='white')
        canvas.save(DEST/(family+'.png'))

def main():
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['inventory','battle_search']);a=ap.parse_args();globals()[a.action]()

def battle_search():
    rows=json.loads((DEST/'sources.json').read_text())['items'];patterns=[]
    for r in rows:
        if r['family']!='comspr':continue
        raw=np.frombuffer((DEST/r['indexed']).read_bytes(),np.uint8).reshape(24,24)
        for mapping,p in [('raw',raw),('combat-0-8-swap',np.where(raw==0,8,np.where(raw==8,0,raw)).astype(np.uint8))]:
            for flip in (False,True):patterns.append((r['key'],mapping,flip,p[:,::-1] if flip else p))
    paths=sorted((ROOT/'workplace/probe/issue21-battle-source').glob('*.bin'));matches=[]
    for n,path in enumerate(paths):
        idx=path.read_bytes();screen=np.frombuffer(idx,np.uint8).reshape(200,320)
        for key,mapping,flip,p in patterns:
            positions=exact_positions(screen,p)
            if positions:matches.append(dict(key=key,mapping=mapping,flip=flip,positions=positions,screen=str(path.relative_to(ROOT)),indexed_sha256=sha(idx)))
        if n%80==0:print('battle search',n,'/',len(paths),flush=True)
    write(DEST/'comspr-search.json',dict(issue=21,frames=len(paths),patterns=len(patterns),matches=matches,
        source_receipt='workplace/probe/issue21-battle-source/receipt.json',
        scope='既有 demo 戰鬥 1450M→2200M 無注入、無輸入續跑；全部畫面像素位置，原色號與戰鬥0/8互換、兩方向，含背景完整24×24。過渡畫格不自動稱為穩定玩家畫面。'))
    print('COMSPR full matches',len(matches))
if __name__=='__main__':main()
