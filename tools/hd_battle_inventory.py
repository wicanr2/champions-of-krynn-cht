#!/usr/bin/env python3
"""規格 010 §6，Issue #13。來源與保存畫面盤點，不啟動遊戲。"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import sys
import struct

import numpy as np
from PIL import Image
from dax_pics import EGA, dax, picture, frames
from hd_request import source

FILES = ('DUNGCOM', 'WILDCOM', 'RANDCOM', 'CPIC1', 'CPIC2', 'CPIC3',
         'CHEAD', 'CBODY', 'COMSPR', 'SPRIT1', 'SPRIT2', 'SPRIT3',
         'HEAD2', 'HEAD3', 'BODY2', 'BODY3', 'SKY', 'TILES')
def sha(b): return hashlib.sha256(b).hexdigest()
def write(p, obj): p.write_text(json.dumps(obj, ensure_ascii=False, indent=2)+'\n')
def source_records(orig,name):
    """DAX 目錄可重複 block ID，必須保留序號與檔案offset。"""
    data=(orig/name).read_bytes();header=struct.unpack_from('<H',data)[0]+2;blocks=dax(data)
    assert (header-2)%9==0 and len(blocks)==(header-2)//9
    for n,(bid,raw) in enumerate(blocks):
        entry=2+n*9;check,offset,decoded,packed=struct.unpack_from('<BIHH',data,entry);assert bid==check and len(raw)==decoded
        yield bid,raw,dict(file=name,file_sha256=sha(data),block=bid,record_index=n,entry_offset=entry,
            entry_bytes=data[entry:entry+9].hex(),offset=header+offset,packed_size=packed,decoded_size=decoded,decoded_sha256=sha(raw))

def repair_provenance(root):
    hd=root/'workplace/hd';dest=hd/'inventory/issue13';p=dest/'inventory.json';before=p.read_bytes();j=json.loads(before)
    backup=dest/'provenance-before.json'
    if not backup.exists():backup.write_bytes(before)
    cached={};changes=[]
    for row in j['items']:
        aliases=[];seen=set()
        for old in row['aliases']:
            name=old['file']
            if name not in cached:cached[name]=list(source_records(root/'workplace/input/dos/ckrynn',name))
            found=[]
            for bid,raw,src in cached[name]:
                if bid!=old['block']:continue
                try:pics=[picture(raw)]
                except ValueError:pics=[f[1:] for f in frames(raw)]
                w,h,count,x,y,px=pics[old['frame']];item=old['item'];native=px[item*w*h:(item+1)*w*h]
                if [w,h]==row['size'] and sha(native)==row['raw_sha256']:
                    found.append(dict(src,frame=old['frame'],item=item,header_xy=[x,y]))
            assert found,(row['key'],old,len(found))
            for new in found:
                location=(new['file'],new['record_index'],new['frame'],new['item'])
                if location not in seen:aliases.append(new);seen.add(location)
            if len(found)>1:
                changes.append(dict(key=row['key'],before=old,after_candidates=found,
                    note='同ID目錄項的本幀像素相同，保留全部來源定位，不猜測原版選擇。'))
            elif old.get('offset')!=found[0]['offset'] or old.get('decoded_sha256')!=found[0]['decoded_sha256']:
                changes.append(dict(key=row['key'],before=old,after=found[0]))
        row['aliases']=aliases
    write(p,j)
    request_changes=[]
    for batch in ('b07','b08','b09','b10','b11','b12'):
        req=hd/'requests'/batch/'manifest.json';old=req.read_bytes();man=json.loads(old)
        table={r['key']:r for r in j['items']}
        for it in man['items']:
            it['sources']=table[it['key']]['aliases']
            if it.get('rendering')=='native_exact_constant':it['generation']='原版純色保真，非AI生成'
        archived=dest/f'{batch}-request-before-provenance.json'
        if not archived.exists():archived.write_bytes(old)
        write(req,man);request_changes.append(dict(batch=batch,before_sha256=sha(old),after_sha256=sha(req.read_bytes())))
    correction=dest/'provenance-correction.json'
    if not correction.exists():write(correction,dict(reason='SPRIT2 block14、SPRIT3 block35 各有重複ID。舊dict按ID覆寫，改以record_index與offset定位。',
        inventory_before_sha256=sha(before),inventory_after_sha256=sha(p.read_bytes()),changes=changes,requests=request_changes,
        pixels_and_art_unchanged=True,original_driver_selection='unknown；呈現器僅按當幀完整前景匹配，不按block ID選圖。'))
    print('provenance corrected',len(changes),'aliases; pixels unchanged')
def kind(n):
    if n in ('DUNGCOM','WILDCOM','RANDCOM'): return 'terrain'
    if n.startswith('CPIC'): return 'actor'
    if n=='CHEAD': return 'head'
    if n=='CBODY': return 'body'
    if n=='COMSPR': return 'effect'
    if n.startswith('SPRIT'): return 'encounter'
    return 'other'

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--root',type=Path,required=True)
    ap.add_argument('--previous',type=Path,required=True)
    ap.add_argument('--repair-provenance',action='store_true')
    a=ap.parse_args(); root=a.root; hd=root/'workplace/hd'; orig=root/'workplace/input/dos/ckrynn'
    if a.repair_provenance:repair_provenance(root);return
    dest=hd/'inventory/issue13';dest.mkdir(exist_ok=True)
    registry={}; files={}
    for name in FILES:
        path=orig/(name+'.DAX');files[path.name]=sha(path.read_bytes())
        for block,raw,src in source_records(orig,path.name):
            try: pics=[picture(raw)]
            except ValueError: pics=[f[1:] for f in frames(raw)]
            for frame,(w,h,count,x,y,px) in enumerate(pics):
                for item in range(count):
                    rawpx=px[item*w*h:(item+1)*w*h]; k=kind(name)
                    signature=(k,w,h,sha(rawpx))
                    row=registry.setdefault(signature,dict(key=f'{name.lower()}-{block:03d}-{frame:02d}-{item:02d}',
                        kind=k,size=[w,h],raw_sha256=sha(rawpx),aliases=[],colors=dict(Counter(rawpx)),
                        evidence=[],full_matches=0,foreground_matches=0))
                    row['aliases'].append(dict(src,frame=frame,item=item,header_xy=[x,y]))
                    row['raw']=rawpx
    rows=list(registry.values())
    # 原版色號來源；絕不從畫面採樣建立基準。
    for row in rows:
        px=row['raw'];mapped=bytes(8 if c==0 else 0 if c==8 else c for c in px) if row['kind'] in ('actor','head','body','effect') else px
        row['indexed']=row['key']+'.idx';row['indexed_sha256']=sha(mapped)
        (dest/row['indexed']).write_bytes(mapped)
        row['array']=np.frombuffer(mapped,np.uint8).reshape(row['size'][1],row['size'][0])
    screens=[]
    man=json.loads((hd/'screens/normal/manifest.json').read_text())
    for it in man['items']:
        raw=(hd/'screens/normal'/it['file']).read_bytes();assert sha(raw)==it['indexed_sha256']
        screens.append((it['source'],raw,np.frombuffer(raw,np.uint8).reshape(200,320)))
    terrain={r['indexed_sha256']:r for r in rows if r['kind']=='terrain'}
    chars={};heads=[r for r in rows if r['kind']=='head'];bodies=[r for r in rows if r['kind']=='body']
    for body in bodies:
        for head in heads:
            p=body['array'].copy();q=head['array'];p[:q.shape[0]][q!=8]=q[q!=8]
            for flip in (False,True):
                v=p[:,::-1] if flip else p
                chars.setdefault(sha(v.tobytes()),[]).append(dict(head=head['key'],body=body['key'],flip=flip))
    battle=[]; character_matches=[]; foreground=[]
    for row in rows:
        if row['kind'] not in ('actor','effect'):continue
        p=row['array'];mask=p!=8;yy,xx=np.nonzero(mask)
        if len(xx)<8:continue
        for flip in (False,True):
            q=p[:,::-1] if flip else p;m=q!=8;ys,xs=np.nonzero(m)
            ids=np.linspace(0,len(xs)-1,min(8,len(xs)),dtype=int)
            foreground.append((row,q,m,ys[ids],xs[ids],flip))
    for name,raw,screen in screens:
        hits=[]
        for y in range(8,153,24):
            for x in range(8,153,24):
                tile=screen[y:y+24,x:x+24];r=terrain.get(sha(tile.tobytes()))
                if r:
                    hits.append(dict(key=r['key'],x=x,y=y));r['full_matches']+=1
                    if len(r['evidence'])<8:r['evidence'].append(dict(screen=name,x=x,y=y,method='完整矩形'))
        guards=sum(len(terrain[sha(screen[h['y']:h['y']+24,h['x']:h['x']+24].tobytes())]['colors'])>1 for h in hits)
        if guards>=3:
            battle.append(dict(screen=name,sha256=sha(raw),terrain=hits))
            for y in range(8,153,24):
                for x in range(8,153,24):
                    sig=sha(screen[y:y+24,x:x+24].tobytes())
                    if sig in chars:
                        match=dict(screen=name,x=x,y=y,sha256=sig,pairs=chars[sig]);character_matches.append(match)
                        for pair in chars[sig]:
                            for key in (pair['head'],pair['body']):
                                r=next(r for r in rows if r['key']==key);r['full_matches']+=1
                                if len(r['evidence'])<8:r['evidence'].append(match)
            for r,p,mask,yy,xx,flip in foreground:
                h,w=p.shape
                for y in range(8,177-h,24):
                    for x in range(8,177-w,24):
                        q=screen[y:y+h,x:x+w]
                        if np.array_equal(q[yy,xx],p[yy,xx]) and np.array_equal(q[mask],p[mask]):
                            full=bool(np.array_equal(q,p));r['foreground_matches']+=1;r['full_matches']+=full
                            if len(r['evidence'])<8:r['evidence'].append(dict(screen=name,x=x,y=y,flip=flip,full=full,
                                background_colors=list(map(int,np.unique(q[~mask]))),method='全部非背景前景'))
        # 3D 與其他家族只量測，不預設透明色或換色。
        for r in rows:
            if r['kind'] not in ('encounter','other'):continue
            p=r['array'];h,w=p.shape
            if w>112 or h>120:continue
            for y in range(8,129-h,8):
                for x in range(8,177-w,8):
                    q=screen[y:y+h,x:x+w]
                    if np.array_equal(q,p):
                        r['full_matches']+=1
                        if len(r['evidence'])<8:r['evidence'].append(dict(screen=name,x=x,y=y,method='原始完整矩形'))
    # 與前作同色號才列為重繪沿用候選，正式接入仍經本作數值與目視。
    old={}
    for path in sorted((a.previous/'workplace/hd/requests').glob('b*/manifest.json')):
        j=json.loads(path.read_text())
        for it in j['items']:
            idx=path.parent/it['indexed'];png=a.previous/'workplace/hd/out'/path.parent.name/(it['key']+'.png')
            if idx.exists() and png.exists():old.setdefault((tuple(it['size']),sha(idx.read_bytes())),dict(batch=path.parent.name,key=it['key'],png=str(png.relative_to(a.previous)),png_sha256=sha(png.read_bytes())))
    for r in rows:
        r['reuse_candidate']=old.get((tuple(r['size']),r['indexed_sha256']))
        r['foreground']=None if r['kind']=='other' else int(np.count_nonzero(r['array']!=(0 if r['kind']=='encounter' else 8)))
        r['runtime_evidence']='observed' if r['evidence'] else 'unobserved'
        del r['raw'],r['array']
    summary={n:dict(raw_items=sum(sum(s['file']==n+'.DAX' for s in r['aliases']) for r in rows),
        unique=sum(any(s['file']==n+'.DAX' for s in r['aliases']) for r in rows),
        observed=sum(bool(r['evidence']) and any(s['file']==n+'.DAX' for s in r['aliases']) for r in rows)) for n in FILES}
    write(dest/'inventory.json',dict(issue=13,spec='docs/spec/010-hd-theme.md §6',tool_sha256=sha(Path(__file__).read_bytes()),
        sources=files,address_space='DAX 檔案 offset；畫面為 dosgolem 320×200 色號',summary=summary,items=rows,
        battle_screens=battle,character_matches=character_matches,screen_count=len(screens),
        limits='保存畫面分析；不是新執行收據。未觀測語意保持未知。'))
    print(json.dumps(summary,ensure_ascii=False));print('battle',len(battle),'character',len(character_matches),'unique',len(rows),'reuse',sum(bool(r['reuse_candidate']) for r in rows),flush=True)

if __name__=='__main__':main()
