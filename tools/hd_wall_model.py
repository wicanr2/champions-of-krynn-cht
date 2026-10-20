#!/usr/bin/env python3
"""規格010 §7：WALLDEF來源與SKY合成的獨立逐格模型。"""
from collections import defaultdict
import json
from pathlib import Path
import numpy as np
from hd_wall_inventory import load_sources, saved_screens, LAYOUTS, rgb
from hd_battle_inventory import sha,write

def backgrounds(pictures):
    sky={p['source']['block']:np.frombuffer(p['raw'],np.uint8).reshape(p['size'][1],p['size'][0]) for p in pictures if p['key'].startswith('sky-')}
    results=[];seen=set()
    for color in (0,11,15):
        for mountain in (False,True):
            for sun in [None]+[(x,y) for y in range(-8,41,8) for x in range(0,81,8)]:
                px=np.full((88,88),color,np.uint8);owners=np.full((88,88),-1,np.int32);coords=np.zeros((88,88,2),np.int32)
                layers=[]
                if mountain:layers.append((250,0,0))
                if sun:layers.append((251,*sun))
                layers.append((252,0,40))
                for bid,x,y in layers:
                    q=sky[bid];h,w=q.shape
                    for yy in range(max(0,y),min(88,y+h)):
                        for xx in range(max(0,x),min(88,x+w)):
                            v=q[yy-y,xx-x]
                            if v!=13:px[yy,xx]=v;owners[yy,xx]=bid;coords[yy,xx]=[xx-x,yy-y]
                digest=sha(px.tobytes())
                if digest in seen:continue
                seen.add(digest);results.append(dict(color=color,mountain=mountain,sun=sun,px=px,owners=owners,coords=coords,layers=layers))
    return results

def tilearray(panel):
    return panel.reshape(11,8,11,8).transpose(0,2,1,3).reshape(121,8,8)

class Model:
    def __init__(self,root):
        self.tiles,self.defs,self.pictures,self.files=load_sources(root)
        j=json.loads((root/'workplace/hd/inventory/issue14/sources.json').read_text())
        used={t['key'] for t in j['tiles'] if t['used_by_walldef']}
        self.tiles=[t for t in self.tiles if t['key'] in used and any(c!=13 for c in t['raw'])]
        self.tiles.sort(key=lambda t:t['key'])
        self.raw=np.array([np.frombuffer(t['raw'],np.uint8).reshape(8,8) for t in self.tiles]);self.mask=self.raw!=13
        self.bgs=backgrounds(self.pictures)
        self.cache={}
        for bg in self.bgs:
            bg['cells']=tilearray(bg['px'])
        self.guards=np.array([len(set(t['raw'])-{13})>=2 and sum(c!=13 for c in t['raw'])>=16 for t in self.tiles])

    def cell(self,px,bg):
        key=(px.tobytes(),bg.tobytes())
        if key not in self.cache:
            choices=np.flatnonzero(np.where(self.mask,self.raw,bg).reshape(len(self.raw),64).__eq__(px.reshape(1,64)).all(axis=1))
            self.cache[key]=(list(map(int,choices)),bool(np.array_equal(px,bg)))
        return self.cache[key]

    def match(self,panel):
        cells=tilearray(panel);found=[]
        # 預先依非13前景篩選，背景候選只檢查留底點。
        possible=[np.flatnonzero(((self.raw==px)|~self.mask).all(axis=(1,2))) for px in cells]
        for bi,bg in enumerate(self.bgs):
            plan=[];guards=0
            for ci,(px,back,choices) in enumerate(zip(cells,bg['cells'],possible)):
                qs=self.raw[choices];mask=self.mask[choices]
                valid=choices[np.all((qs==px)|(~mask&(back==px)),axis=(1,2))]
                background=bool(np.array_equal(px,back))
                composed=None
                if not len(valid) and not background:
                    signature=(px.tobytes(),back.tobytes())
                    if signature not in self.cache:
                        chain=None
                        for top in sorted(choices,key=lambda n:-int(self.mask[n].sum())):
                            visible=~self.mask[top]
                            fits=np.all(~visible|((self.raw==px)|(~self.mask&(back==px))),axis=(1,2))
                            bottoms=np.flatnonzero(fits)
                            if len(bottoms):
                                bottom=max(bottoms,key=lambda n:int((self.mask[n]&visible).sum()))
                                chain=[int(top),int(bottom)];break
                        self.cache[signature]=chain
                    composed=self.cache[signature]
                    if composed is None:break
                # 留底圖塊會與背景形成相同最終基準。多種來源可解釋時不畫該格。
                if background:plan.append(-1)
                elif len(valid):
                    n=max(valid,key=lambda n:int(self.mask[n].sum()));n=int(n);plan.append(n);guards+=int(self.guards[n])
                else:
                    plan.append(composed);guards+=int(any(self.guards[n] for n in composed))
            else:
                if guards>=3:found.append(dict(background=bi,plan=plan,guards=guards))
        return found

def face_probe(root,model,panel):
    j=json.loads((root/'workplace/hd/inventory/issue14/sources.json').read_text());tiles={t['key']:t for t in model.tiles}
    placements={0:(4,range(-1,12,2)),1:(3,(-1,1,4)),2:(3,(6,9,11)),3:(3,(-2,1,4,7,10)),4:(1,(-4,-1,2)),5:(1,(7,10,13)),6:(1,(-5,2,9)),7:(0,(-7,0)),8:(0,(9,16)),9:(4,range(0,11,2))}
    matches=[]
    for f in j['faces']:
        if not f['resolved']:continue
        w,h=f['size'];source=np.full((h,w),13,np.uint8)
        for n,key in enumerate(f['tiles']):
            if key not in tiles:continue
            source[n//(w//8)*8:n//(w//8)*8+8,n%(w//8)*8:n%(w//8)*8+8]=np.frombuffer(tiles[key]['raw'],np.uint8).reshape(8,8)
        row,cols=placements[f['layout']]
        for col in cols:
            x,y=col*8,row*8;L,R=max(0,x),min(88,x+w);T,B=max(0,y),min(88,y+h)
            if L>=R or T>=B:continue
            p=source[T-y:B-y,L-x:R-x];q=panel[T:B,L:R];mask=p!=13
            if mask.sum()<16 or len(set(map(int,p[mask])))<2:continue
            if np.array_equal(p[mask],q[mask]):matches.append(dict(file=f['source']['file'],block=f['source']['block'],wall=f['wall'],layout=f['layout'],position=[x+24,y+24],foreground=int(mask.sum()),span=f['span']))
    return matches

def main():
    import argparse
    ap=argparse.ArgumentParser();ap.add_argument('--root',required=True,type=Path);ap.add_argument('--faces',action='store_true');a=ap.parse_args()
    m=Model(a.root);dest=a.root/'workplace/hd/inventory/issue14';rows=[];seen=set();used=set()
    for path,raw in saved_screens(a.root):
        p=np.frombuffer(raw,np.uint8).reshape(200,320)[24:112,24:112];digest=sha(p.tobytes())
        if digest in seen:continue
        seen.add(digest);matches=m.match(p)
        if matches:
            match=matches[0]
            for n in match['plan']:
                if isinstance(n,list):used.update(m.tiles[k]['key'] for k in n)
                elif n>=0:used.add(m.tiles[n]['key'])
            row=dict(screen=path,screen_sha256=sha(raw),panel_sha256=digest,matches=[dict(bg={k:v for k,v in m.bgs[x['background']].items() if k in ('color','mountain','sun','layers')},plan=[[m.tiles[k]['key'] for k in n] if isinstance(n,list) else m.tiles[n]['key'] if n>=0 else n for n in x['plan']],guards=x['guards']) for x in matches])
            if a.faces:row['faces']=face_probe(a.root,m,p)
            rows.append(row);print(path,'bg',len(matches),'guards',match['guards'],'ambig',match['plan'].count(-2),flush=True)
    write(dest/'model.json',dict(issue=14,screens_tested=len(seen),matched=len(rows),profiles=len(m.bgs),source_tiles=len(m.tiles),used_tiles=sorted(used),results=rows))
    print('matched',len(rows),'unique used',len(used),'profiles',len(m.bgs),flush=True)

if __name__=='__main__':main()
