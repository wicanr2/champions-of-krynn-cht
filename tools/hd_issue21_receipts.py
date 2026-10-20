#!/usr/bin/env python3
"""Issue21：獨立 Pillow 預期、完整來源正負例與同狀態收據。Docker 限定。"""
import argparse
from collections import Counter
import json
from pathlib import Path
import numpy as np
from PIL import Image
from dax_pics import EGA
from hd_battle_inventory import sha,write
from hd_wall_receipts import CombinedOracle, hashes
from hd_issue21 import exact_positions

ROOT=Path('/src');HD=ROOT/'workplace/hd'
class Issue21Oracle:
    def __init__(self):
        self.base=CombinedOracle(HD/'theme');self.exact=[]
        for it in json.loads((HD/'theme/exact.json').read_text())['items']:
            w,h=it['size'];p=(HD/'theme'/it['indexed']).read_bytes();im=HD/'theme'/it['png'];assert sha(p)==it['indexed_sha256'] and sha(im.read_bytes())==it['png_sha256']
            self.exact.append((it,np.frombuffer(p,np.uint8).reshape(h,w),np.array(Image.open(im).convert('RGBA'))))
    def expected(self,raw):
        out,allowed,hits=self.base.expected(raw);frame=np.frombuffer(raw,np.uint8).reshape(200,320);candidates=[]
        for it,p,im in self.exact:
            for x,y in exact_positions(frame,p):candidates.append((x,y,it,p,im))
        own=np.zeros((200,320),np.uint16)
        for x,y,it,p,im in candidates:h,w=p.shape;own[y:y+h,x:x+w]+=1
        for x,y,it,p,im in candidates:
            h,w=p.shape
            if np.any(own[y:y+h,x:x+w]>1):continue
            out[y*3:(y+h)*3,x*3:(x+w)*3]=im;allowed[y:y+h,x:x+w]=True;hits.append('exact:'+it['key'])
        return out,allowed,hits

def frames():
    oracle=Issue21Oracle();dest=HD/'out/issue21-frames';dest.mkdir(exist_ok=True);tasks=[]
    for it,p,im in oracle.exact:
        frame=np.full((200,320),8,np.uint8);h,w=p.shape;frame[59:59+h,37:37+w]=p
        tasks.append((it['key'],'source-positive',frame.tobytes()));frame[59,37]^=1
        tasks.append((it['key']+'-one-point-mismatch','source-negative',frame.tobytes()))
    search=json.loads((HD/'inventory/issue21/search.json').read_text())
    # 索引已去重的原版保存畫面，不重命名為本輪玩家流程。
    for screen in search['screens']:
        path=screen['paths'][0];p=ROOT/path
        if p.suffix!='.bin':
            m=json.loads((HD/'screens/normal/manifest.json').read_text());e=next(r for r in m['items'] if r['source']==path);p=HD/'screens/normal'/e['file']
        raw=p.read_bytes();assert sha(raw)==screen['sha256'];tasks.append((path,'saved-original',raw))
    hits=json.loads((HD/'inventory/issue21/comspr-search.json').read_text())['matches']
    for row in hits:tasks.append((row['screen'],'dosgolem-comspr-transient',(ROOT/row['screen']).read_bytes()))
    guard=next(r for r in oracle.base.walls.global_cells.values() if r['guard'])
    # b17每個來源各選一個可達的完整模板；留底顏色只是明列的來源測試條件。
    manifest=oracle.base.walls.manifest;byhash={c['indexed_sha256']:c for c in manifest['cells']};coverage=json.loads((HD/'inventory/issue21/wall-coverage.json').read_text())['coverage']
    sample=[r for r in coverage if any(a['key']==r['key'] for a in json.loads((HD/'requests/b17/manifest.json').read_text())['items'])]
    masked=[r for r in coverage if len(r['templates'])>1]
    sample+=masked[::max(1,len(masked)//24)]
    for r in sample:
        c=byhash[r['templates'][0]];pos=(c['positions'] or [60])[0]
        target=next((v for v in oracle.base.walls.global_cells.values() if sha(v['raw'])==r['templates'][0]),None)
        if target is None:target=next(v for v in oracle.base.walls.cells[pos].values() if sha(v['raw'])==r['templates'][0])
        frame=np.full((200,320),8,np.uint8)
        for n in range(121):
            x,y=24+n%11*8,24+n//11*8;frame[y:y+8,x:x+8]=np.frombuffer(guard['raw'],np.uint8).reshape(8,8)
        x,y=24+pos%11*8,24+pos//11*8;frame[y:y+8,x:x+8]=np.frombuffer(target['raw'],np.uint8).reshape(8,8)
        tasks.append((r['key']+'-complete-background','wall-source-positive',frame.tobytes()))
    cases=[]
    for n,(name,group,raw) in enumerate(tasks):
        expected,allowed,matches=oracle.expected(raw);native=np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(200,320)];original=np.dstack((np.repeat(np.repeat(native,3,0),3,1),np.full((600,960),255,np.uint8)))
        stem=f'{n:04d}';idx=dest/(stem+'.bin');ep=dest/(stem+'-expected.png');op=dest/(stem+'-original.png');idx.write_bytes(raw);Image.fromarray(expected).save(ep);Image.fromarray(original).save(op)
        pp=lambda p:'/hd/'+str(p.relative_to(HD))
        cases.append(dict(name=name,group=group,indexed=pp(idx),indexed_sha256=sha(raw),expected=pp(ep),expected_sha256=sha(ep.read_bytes()),original=pp(op),original_sha256=sha(op.read_bytes()),matches=matches))
        if n%80==0:print('independent frames',n,'/',len(tasks),flush=True)
    write(dest/'cases.json',dict(cases=cases,scope='來源衍生案例與原版執行畫格分組；Pillow獨立預期'))
    print('cases',len(cases),Counter(c['group'] for c in cases))

def runtime():
    oracle=Issue21Oracle();dest=HD/'out/issue21-runtime';rows=[];groups=[]
    for scene in ('portrait','battle'):
        for scale in (3,2):
            a=dest/f'{scene}-base{scale}';b=dest/f'{scene}-hd{scale}';ra=json.loads((a/'receipt.json').read_text());rb=json.loads((b/'receipt.json').read_text())
            keys=('initial_state_sha256','initial_steps','initial_poke','steps','memory_sha256','registers','keys_sent','keys','key_names','key_from','key_every','shots')
            assert all(ra[k]==rb[k] for k in keys),(scene,scale,'state');assert hashes(a/'scratch')==hashes(b/'scratch')
            groups.append(dict(scene=scene,scale=scale,without_hd=ra,with_hd=rb,scratch=hashes(a/'scratch'),same_final_state=True))
            changed=0
            for shot in ra['shots']:
                name=shot['name'];raw=(a/(name+'.bin')).read_bytes();assert raw==(b/(name+'.bin')).read_bytes()
                pa=np.array(Image.open(a/(name+'.png')).convert('RGBA'));pb=np.array(Image.open(b/(name+'.png')).convert('RGBA'));delta=np.any(pa!=pb,axis=2)
                if scale==2:assert not delta.any();expected_diff=0;hits=[]
                else:
                    predicted,mask,hits=oracle.expected(raw);mask=np.repeat(np.repeat(mask,3,0),3,1)
                    want=pa.copy();want[mask]=predicted[mask];expected_diff=int(np.any(want!=pb,axis=2).sum());assert expected_diff==0,(scene,name,expected_diff)
                    changed+=bool(delta.any())
                rows.append(dict(scene=scene,scale=scale,name=name,indexed_sha256=sha(raw),changed_pixels=int(delta.sum()),expected_diff=expected_diff,matches=hits,ok=True))
            if scale==3:assert changed>0,(scene,'no HD change')
    write(dest/'state-comparison.json',dict(groups=groups))
    write(dest/'receipt.json',dict(all_pass=True,items=rows,scope='相同state於執行前固定初始亂數記憶體與虛擬時鐘，無注入無輸入無重擲；終點CPU、1MiB RAM、暫存檔案及每個原版色號畫格相同。不是新冷啟動玩家流程。'))
    print('runtime',len(rows),'all pass')

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['frames','runtime']);a=ap.parse_args();globals()[a.action]()
