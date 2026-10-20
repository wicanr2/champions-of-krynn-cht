#!/usr/bin/env python3
"""規格 010 §6，獨立 Pillow 整幀預期。來源案例與執行畫面分組。"""
import argparse
import json
import multiprocessing
from pathlib import Path
import numpy as np
from PIL import Image
from dax_pics import EGA
from hd_battle_inventory import sha,write
from hd_issue21 import exact_positions

class Oracle:
    def __init__(self,theme):
        self.theme=theme;self.terrain=[];self.actors=[];self.characters={};self.effects=[];self.encounters=[];self.fixed=[]
        def load(it,raw=False):
            w,h=it.get('size',it.get('screen_rect',[0,0,0,0])[2:]);idx=(theme/it['indexed']).read_bytes();assert sha(idx)==it['indexed_sha256'];assert len(idx)==w*h
            image=theme/it['png'];assert sha(image.read_bytes())==it['png_sha256'];return np.frombuffer(idx,np.uint8).reshape(h,w),np.array(Image.open(image).convert('RGBA'))
        t=json.loads((theme/'theme.json').read_text())
        for it in t['items']:
            p,im=load(it);self.fixed.append((it,p,im))
        for filename,attr in (('combat.json','terrain'),('actors.json','actors'),('effects.json','effects'),('encounters.json','encounters')):
            for it in json.loads((theme/filename).read_text())['items']:
                p,im=load(it)
                if attr=='actors':
                    for flip in (False,True):getattr(self,attr).append((it,p[:,::-1] if flip else p,im[:,::-1] if flip else im))
                else:getattr(self,attr).append((it,p,im))
        originals=self.effects;self.effects=[];seen=set()
        for flip in (False,True):
            for it,p,im in originals:
                q=p[:,::-1] if flip else p;r=im[:,::-1] if flip else im
                if q.tobytes() in seen:continue
                seen.add(q.tobytes());self.effects.append((it,q,r))
        self.effect_lookup={p.tobytes():(it,p,im) for it,p,im in self.effects}
        t=json.loads((theme/'characters.json').read_text());heads=[(it,*load(it)) for it in t['heads']];bodies=[(it,*load(it)) for it in t['bodies']]
        self.heads=heads;self.bodies=bodies
        for bi,bp,bim in bodies:
            for hi,hp,him in heads:
                p=bp.copy();h,w=hp.shape;p[:h][hp!=8]=hp[hp!=8]
                im=bim.copy();mask=np.repeat(np.repeat(hp!=8,3,0),3,1);im[:h*3][mask]=him[mask]
                for flip in (False,True):
                    q=p[:,::-1] if flip else p;r=im[:,::-1] if flip else im
                    assert q.tobytes() not in self.characters
                    self.characters[q.tobytes()]=(q,r,bi['key']+'/'+hi['key'])
        self.terrain_lookup={p.tobytes():(it,p,im) for it,p,im in self.terrain}

    def character_matrix(self):
        rows=[]
        for raw,(p,im,key) in self.characters.items():
            pixels=im.copy();mask=np.repeat(np.repeat(p==8,3,0),3,1);pixels[mask]=0
            rows.append(dict(key=key,indexed_sha256=sha(raw),rgba_sha256=sha(pixels.tobytes())))
        return dict(count=len(rows),scope='來源組合與鏡像，不是原版執行畫面',items=rows)

    def expected(self,raw):
        frame=np.frombuffer(raw,np.uint8).reshape(200,320);native=np.array(EGA,np.uint8)[frame]
        out=np.empty((600,960,4),np.uint8);out[...,:3]=np.repeat(np.repeat(native,3,0),3,1);out[...,3]=255
        allowed=np.zeros((200,320),bool);hits=[];terrain=[];candidates=[];enc=[]
        def paste(x,y,p,im,mask):
            h,w=p.shape;m=np.repeat(np.repeat(mask,3,0),3,1);out[y*3:(y+h)*3,x*3:(x+w)*3][m]=im[m];allowed[y:y+h,x:x+w]|=mask
        for y in range(8,153,24):
            for x in range(8,153,24):
                t=self.terrain_lookup.get(frame[y:y+24,x:x+24].tobytes())
                if t:terrain.append((x,y,*t))
        guards=sum(len(np.unique(p))>1 and np.count_nonzero(p!=8)>=16 for x,y,it,p,im in terrain)
        if guards>=3:
            for x,y,it,p,im in terrain:paste(x,y,p,im,np.ones(p.shape,bool));hits.append(it['key'])
            for it,p,im in self.actors:
                h,w=p.shape;mask=p!=8;yy,xx=np.nonzero(mask);n=len(xx)//2
                for y in range(8,177-h,24):
                    for x in range(8,177-w,24):
                        q=frame[y:y+h,x:x+w]
                        if q[yy[0],xx[0]]!=p[yy[0],xx[0]] or q[yy[n],xx[n]]!=p[yy[n],xx[n]]:continue
                        if np.array_equal(q[mask],p[mask]):candidates.append((x,y,it['key'],p,im,mask))
            for y in range(8,153,24):
                for x in range(8,153,24):
                    tile=frame[y:y+24,x:x+24].tobytes();row=self.characters.get(tile)
                    if row:
                        p,im,key=row;candidates.append((x,y,key,p,im,p!=8))
            # 規格010 §8.4：COMSPR 的完整模板可位於移動格之間。
            for it,p,im in self.effects:
                for x,y in exact_positions(frame[8:176,8:176],p):
                    candidates.append((x+8,y+8,it['key'],p,im,p!=8))
            ownership=np.zeros((200,320),np.uint16)
            for x,y,key,p,im,m in candidates:h,w=p.shape;ownership[y:y+h,x:x+w]+=m
            for x,y,key,p,im,m in candidates:
                h,w=p.shape
                if np.any(ownership[y:y+h,x:x+w][m]>1):continue
                paste(x,y,p,im,m);hits.append(key)
        for it,p,im in self.encounters:
            x,y=it['position'];h,w=p.shape;m=p!=0;q=frame[y:y+h,x:x+w];mapped=np.where(p==13,0,p)
            if np.array_equal(q[m],mapped[m]):enc.append((x,y,it['key'],p,im,m))
        own=np.zeros((200,320),np.uint16)
        for x,y,key,p,im,m in enc:h,w=p.shape;own[y:y+h,x:x+w]+=m
        for x,y,key,p,im,m in enc:
            h,w=p.shape
            if not np.any(own[y:y+h,x:x+w][m]>1):paste(x,y,p,im,m);hits.append(key)
        for it,p,im in self.fixed:
            x,y,w,h=it['screen_rect']
            if np.array_equal(frame[y:y+h,x:x+w],p):paste(x,y,p,im,np.ones(p.shape,bool));hits.append(it['key'])
        return out,allowed,hits

def render_case(job):
    n,name,group,raw=job;dest=_DEST;oracle=_ORACLE
    stem=f'{n:04d}';idx=dest/(stem+'.bin');idx.write_bytes(raw);expected,allowed,hits=oracle.expected(raw)
    native=np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(200,320)];native=np.repeat(np.repeat(native,3,0),3,1);native=np.dstack((native,np.full((600,960),255,np.uint8)))
    ep=dest/(stem+'-expected.png');op=dest/(stem+'-original.png');Image.fromarray(expected).save(ep,compress_level=1);Image.fromarray(native).save(op,compress_level=1)
    if n%100==0:print('預期整幀',n,flush=True)
    return dict(name=name,group=group,indexed=str(idx),indexed_sha256=sha(raw),expected=str(ep),expected_sha256=sha(ep.read_bytes()),original=str(op),original_sha256=sha(op.read_bytes()),matches=hits,allowed_pixels=int(allowed.sum()))

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',required=True,type=Path);a=ap.parse_args();hd=a.root/'workplace/hd';dest=hd/'out/issue13-frames';dest.mkdir(exist_ok=True);oracle=Oracle(hd/'theme');cases=[]
    tasks=[]
    def save(name,group,raw):tasks.append((len(tasks),name,group,raw))
    base=np.full((200,320),8,np.uint8)
    guards=[row for row in oracle.terrain if len(np.unique(row[1]))>1][:3]
    for n,(_,p,_) in enumerate(guards):base[8:32,8+n*24:32+n*24]=p
    groups=[('terrain',oracle.terrain),('actor',oracle.actors),('effect',oracle.effects),('encounter',oracle.encounters)]
    for kind,items in groups:
        for n,(it,p,im) in enumerate(items):
            frame=base.copy() if kind!='encounter' else np.full((200,320),3,np.uint8)
            x,y=(it['position'] if kind=='encounter' else (80,80));h,w=p.shape
            if kind=='encounter':m=p!=0;view=frame[y:y+h,x:x+w];view[m]=np.where(p==13,0,p)[m];anchor=tuple(map(int,np.argwhere(m)[0]))
            else:frame[y:y+h,x:x+w]=p;anchor=tuple(map(int,np.argwhere(p!=8)[0])) if np.any(p!=8) else (0,0)
            save(f'{kind}-{it["key"]}-{n}-positive','source-derived',frame.tobytes())
            ay,ax=anchor;frame[y+ay,x+ax]^=1;save(f'{kind}-{it["key"]}-{n}-mismatch','source-derived',frame.tobytes())
    # 每個頭部與身體各抽一個完整來源組合，不宣稱全部組合都在遊戲使用。
    pairs=[(oracle.bodies[0],h) for h in oracle.heads]+[(b,oracle.heads[0]) for b in oracle.bodies]
    for (bi,bp,bim),(hi,hp,him) in pairs:
        p=bp.copy();p[:hp.shape[0]][hp!=8]=hp[hp!=8];frame=base.copy();frame[80:104,80:104]=p;save(bi['key']+'/'+hi['key'],'source-derived-character',frame.tobytes())
    man=json.loads((hd/'screens/normal/manifest.json').read_text())
    for it in man['items']:save(it['source'],'saved-original', (hd/'screens/normal'/it['file']).read_bytes())
    for p in sorted((hd/'out/issue13-runtime/demo-source').glob('*.bin')):save(str(p.relative_to(hd)),'dosgolem-demo-continuation',p.read_bytes())
    global _ORACLE,_DEST
    _ORACLE,_DEST=oracle,dest
    # Docker 的兩個 CPU 各一個工作程序；圖像內容與案例順序不變。
    with multiprocessing.get_context('fork').Pool(2) as pool:cases=list(pool.imap(render_case,tasks,chunksize=4))
    # 收據在 Go 容器掛載 /hd 時直接使用。
    for c in cases:
        for field in ('indexed','expected','original'):c[field]='/hd/'+str(Path(c[field]).relative_to(hd))
    write(dest/'cases.json',dict(cases=cases,source_matrices=len(oracle.characters),scope='Pillow獨立bytes比對；來源案例非原版執行畫面。'))
    write(dest/'character-matrix.json',oracle.character_matrix())
    print('cases',len(cases),'character matrices',len(oracle.characters))

if __name__=='__main__':main()
