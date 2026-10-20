#!/usr/bin/env python3
"""Issue21：完整來源背景模板、孤立圖源盤點與可達性稽核。Docker 限定。"""
import argparse
from collections import Counter, defaultdict
import json
from pathlib import Path
import shutil
import numpy as np
from PIL import Image, ImageDraw
from dax_pics import EGA
from hd_battle_inventory import sha,write
from hd_request import emit
from hd_wall_art import PROMPT, wall_structure

ROOT=Path('/src');HD=ROOT/'workplace/hd';OUT=HD/'out';INV=HD/'inventory/issue21'

def init():
    rows=[r for r in json.loads((HD/'inventory/issue14/sources.json').read_text())['tiles'] if not r['used_by_walldef']]
    write(INV/'unselected-8x8.json',rows);req=HD/'requests/b17';out=OUT/'b17'
    req.mkdir(exist_ok=True);out.mkdir(exist_ok=True);(out/'jobs').mkdir(exist_ok=True)
    items=[];retained=[]
    for n,r in enumerate(rows):
        raw=(HD/'inventory/issue14'/r['indexed']).read_bytes()
        # 前19個來源為箭頭與單像素輪廓。維持原始符號清晰度，不做材質重畫。
        if n<19 or len(set(raw)-{13})<=1:
            retained.append(dict(key=r['key'],sources=r['aliases'],indexed_sha256=sha(raw),reason='方向箭頭與單像素輪廓保留原貌' if n<19 else '單一實色加留底，維持原色與輪廓'))
            continue
        mapped=bytes(8 if c==13 else c for c in raw)
        it=emit(req,3,r['key'],'未由WALLDEF定位的8×8來源，僅完整色號條件式呈現',(0,0,8,8),None,mapped,r['aliases'],
            raw_indexed=r['key']+'.raw.idx',raw_indexed_sha256=sha(raw),runtime_evidence='unobserved',status='READY',
            transparent_source=13,rendering='painted',preserve_native_colors=[])
        (req/it['raw_indexed']).write_bytes(raw);items.append(it)
    write(req/'manifest.json',dict(batch='b17',issue=21,spec='docs/spec/010-hd-theme.md §8.3',scale=3,kind='unassigned-wall-symbols',items=items,
        palette=dict(name='EGA預設16色；來源13留底，美術參考以8表示',rgb=EGA),source_transparency=13,
        rights='AI重繪沿用本專案HD patch定案，原始資料僅本機。'))
    write(INV/'retained-symbols.json',dict(items=retained,reason='圖形用途未定位，不把箭頭、單像素線及單色標記換成插畫材質。字模及CURSOR另保留原版。'))
    for start in range(0,len(items),64):
        canvas=Image.new('RGB',(1024,1024),(85,85,85));places=[]
        for n,it in enumerate(items[start:start+64]):
            x,y=n%8*128+16,n//8*128+16
            im=Image.open(req/it['png']).convert('RGB').resize((96,96),Image.Resampling.NEAREST);canvas.paste(im,(x,y))
            places.append(dict(key=it['key'],rect=[x,y,96,96],target_size=it['target_size']))
        stem=f'atlas-{start//64:02d}';canvas.save(req/(stem+'.png'));write(req/(stem+'.json'),dict(atlas=stem,canvas=[1024,1024],items=places))
        (req/(stem+'-prompt.txt')).write_text(PROMPT.replace('fantasy dungeon wall texture fragments','fantasy architectural texture fragments and geometric symbols')+'\n')
    print('b17',len(items),'retained',len(retained))

def prepare():
    req=HD/'requests/b17';out=OUT/'b17';man=json.loads((req/'manifest.json').read_text());bykey={r['key']:r for r in man['items']}
    for path in sorted((out/'jobs').glob('*.json')):
        job=json.loads(path.read_text());p=out/job['raw'];assert sha(p.read_bytes())==job['raw_sha256'];im=Image.open(p).convert('RGBA');W,H=im.size
        if job.get('key'):parts=[dict(key=job['key'],rect=[0,0,W,H],target_size=bykey[job['key']]['target_size'])];cw,ch=W,H
        else:
            atlas=json.loads((req/(job['atlas']+'.json')).read_text());parts=atlas['items'];cw,ch=atlas['canvas'];assert W==H
        for part in parts:
            if job.get('only') and part['key'] not in job['only']:continue
            it=bykey[part['key']];x,y,w,h=part['rect'];tile=im.crop((round(x*W/cw),round(y*H/ch),round((x+w)*W/cw),round((y+h)*H/ch))).resize((24,24),Image.Resampling.LANCZOS)
            arr=np.array(tile);arr[:,:,3]=255;raw=np.frombuffer((req/it['raw_indexed']).read_bytes(),np.uint8).reshape(8,8);base=np.array(EGA,np.uint8)[np.where(raw==13,8,raw)];big=np.repeat(np.repeat(raw==13,3,0),3,1);arr[big,:3]=EGA[8]
            nearest=np.repeat(np.repeat(base,3,0),3,1);arr[0,:,:3]=nearest[0];arr[-1,:,:3]=nearest[-1];arr[:,0,:3]=nearest[:,0];arr[:,-1,:3]=nearest[:,-1]
            Image.fromarray(arr).save(out/(it['key']+'.png'))

def validate():
    from hd_compare import hue_score
    req=HD/'requests/b17';out=OUT/'b17';man=json.loads((req/'manifest.json').read_text());rows=[]
    for it in man['items']:
        p=out/(it['key']+'.png');im=Image.open(p);arr=np.array(im);idx=np.frombuffer((req/it['indexed']).read_bytes(),np.uint8).reshape(8,8);orig=np.array(EGA,np.uint8)[idx]
        score,legacy,method=wall_structure(orig,arr);hm,hf=hue_score(orig,arr);ok=score>=.4 and hm<30 and im.mode=='RGBA' and im.size==(24,24) and bool((arr[:,:,3]==255).all())
        rows.append(dict(key=it['key'],png_sha256=sha(p.read_bytes()),structural_score=score,legacy_score=legacy,method=method,hue_median_degrees=hm,target_met=ok))
    write(out/'validation.json',dict(request_manifest_sha256=sha((req/'manifest.json').read_bytes()),items=rows,all_pass=all(r['target_met'] for r in rows)))
    print('b17 pass',sum(r['target_met'] for r in rows),'/',len(rows));print([(r['key'],round(r['structural_score'],3),round(r['hue_median_degrees'],1)) for r in rows if not r['target_met']])
    for start in range(0,len(man['items']),32):
        sheet=Image.new('RGB',(1280,((min(32,len(man['items'])-start)+7)//8)*120),(32,32,40));d=ImageDraw.Draw(sheet)
        for n,it in enumerate(man['items'][start:start+32]):
            x,y=n%8*160,n//8*120;d.text((x,y),it['key'],fill='white')
            for k,p in enumerate((req/it['reference_png'],out/(it['key']+'.png'))):sheet.paste(Image.open(p).convert('RGB').resize((64,64),Image.Resampling.NEAREST),(x+k*80,y+30))
        sheet.save(out/f'review-{start//32:02d}.png')

def supplement():
    theme=HD/'theme';backup=OUT/'issue21-walls-before.json';p=theme/'walls.json'
    if not backup.exists():shutil.copyfile(p,backup)
    m=json.loads(backup.read_text());existing={c['indexed_sha256'] for c in m['cells']};assets={a['key']:a for a in m['assets']}
    if (OUT/'b17/visual-review.json').exists():
        req=HD/'requests/b17';out=OUT/'b17';man=json.loads((req/'manifest.json').read_text());val=json.loads((out/'validation.json').read_text());rev=json.loads((out/'visual-review.json').read_text());assert val['all_pass'] and val['request_manifest_sha256']==sha((req/'manifest.json').read_bytes()) and rev['request_sha256']==sha((req/'manifest.json').read_bytes())
        vr={r['key']:r for r in rev['items']};nr={r['key']:r for r in val['items']}
        for it in man['items']:
            raw=(req/it['raw_indexed']).read_bytes();png=out/(it['key']+'.png');ph=sha(png.read_bytes());assert vr[it['key']]['approved'] and vr[it['key']]['png_sha256']==ph==nr[it['key']]['png_sha256']
            shutil.copyfile(req/it['raw_indexed'],theme/'walls'/(it['key']+'.idx'));shutil.copyfile(png,theme/'walls'/png.name)
            a=dict(key=it['key'],kind='wall',size=[8,8],indexed='walls/'+it['key']+'.idx',indexed_sha256=sha(raw),png='walls/'+png.name,png_sha256=ph);assets[a['key']]=a;m['assets'].append(a)
    groups=defaultdict(list);coverage=[]
    for a in m['assets']:
        if a['kind']!='wall':continue
        raw=(theme/a['indexed']).read_bytes();bases=range(16) if 13 in raw else (0,);refs=[]
        for base in bases:
            ref=bytes(base if c==13 else c for c in raw);digest=sha(ref);refs.append(digest)
            if digest in existing:continue
            png=np.array(Image.open(theme/a['png']).convert('RGBA'));mask=np.repeat(np.repeat(np.frombuffer(raw,np.uint8).reshape(8,8)==13,3,0),3,1);png[mask]=[*EGA[base],255]
            groups[digest].append((a,base,ref,sha(png.tobytes())))
        coverage.append(dict(key=a['key'],templates=refs,raw_sha256=sha(raw)))
    ambiguity=[]
    for n,(digest,rows) in enumerate(sorted(groups.items())):
        a,base,ref,pixsha=rows[0];paint=len({r[3] for r in rows})==1
        spec=dict(key=f'issue21-complete-{n:05d}',positions=[],base_color=base,layers=[dict(key=a['key'],at=[0,0])],indexed_sha256=digest,paint=paint)
        if 13 in (theme/a['indexed']).read_bytes():spec['complete_background']=True
        m['cells'].append(spec)
        if not paint:ambiguity.append(dict(indexed_sha256=digest,sources=[r[0]['key'] for r in rows],policy='原版完整模板，不選美術'))
    m['provenance']['issue21']=dict(original_manifest_sha256=sha(backup.read_bytes()),method='source13 留底列舉16個完整底色；既有模板優先，新增歧義畫原版',coverage='inventory/issue21/wall-coverage.json')
    write(p,m)
    known={c['indexed_sha256'] for c in m['cells']}
    assert all(all(d in known for d in r['templates']) for r in coverage)
    write(INV/'wall-coverage.json',dict(old_assets=len(json.loads(backup.read_text())['assets']),assets=len(m['assets']),
        old_cells=len(json.loads(backup.read_text())['cells']),new_cells=len(groups),total_cells=len(m['cells']),
        coverage=coverage,ambiguities=ambiguity,all_source_templates_reachable=True,
        limits='完整模板可達性，不是原版實機觀測；97個WALLDEF布局仍未知，未知多層背景仍回退。'))
    print('walls',len(m['assets']),'cells',len(m['cells']),'ambiguous',len(ambiguity))

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['init','prepare','validate','supplement']);a=ap.parse_args();globals()[a.action]()
