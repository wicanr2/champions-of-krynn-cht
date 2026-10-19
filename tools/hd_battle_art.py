#!/usr/bin/env python3
"""規格 010 §6：Issue #13 請求、圖集拆圖、沿用證據與主題閘門。"""
import argparse
import json
import shutil
from pathlib import Path
import numpy as np
from PIL import Image,ImageDraw
from dax_pics import EGA
from hd_request import emit,rgb_png
from hd_battle_inventory import sha,write

BATCH={'terrain':'b07','actor':'b08','head':'b09','body':'b09','effect':'b10','encounter':'b11','other':'b12'}
KINDS={'b07':'combat-terrain','b08':'combat-actors','b09':'combat-characters','b10':'combat-effects','b11':'encounter-sprites','b12':'other-draft'}
def init(root,previous):
    hd=root/'workplace/hd';inv=hd/'inventory/issue13';j=json.loads((inv/'inventory.json').read_text());groups={b:[] for b in KINDS}
    for r in j['items']:
        batch=BATCH[r['kind']];req=hd/'requests'/batch;out=hd/'out'/batch
        req.mkdir(exist_ok=True);out.mkdir(exist_ok=True)
        key=r['key'];w,h=r['size'];idx=(inv/r['indexed']).read_bytes()
        x,y=(r['evidence'][0]['x'],r['evidence'][0]['y']) if r['evidence'] else (8,8)
        it=emit(req,3,key,'、'.join(f'{s["file"]}:{s["block"]}@{s["frame"]}/{s["item"]}' for s in r['aliases']),
            (x,y,w,h),None,idx,r['aliases'],kind=r['kind'],runtime_evidence=r['runtime_evidence'],
            evidence=r['evidence'],preserve_native_colors=[8] if batch in ('b07','b08','b09','b10') else [],
            rendering='native_exact_constant' if len(set(idx))==1 else 'painted',
            reuse_candidate=r['reuse_candidate'],palette_mapping=[8,1,2,3,4,5,6,7,0,9,10,11,12,13,14,15] if batch in ('b08','b09','b10') else list(range(16)),
            background_index=8 if batch in ('b07','b08','b09','b10') else None,
            status='READY' if batch in ('b07','b08','b09','b10') else 'DRAFT')
        rgb_png(req/f'{key}-x12.png',idx,w,h,12)
        if r['reuse_candidate']:
            old=r['reuse_candidate'];arc=out/'reused'/key;arc.mkdir(parents=True,exist_ok=True)
            oldout=previous/'workplace/hd/out'/old['batch'];oldreq=previous/'workplace/hd/requests'/old['batch']
            src=previous/old['png'];assert sha(src.read_bytes())==old['png_sha256']
            shutil.copyfile(src,out/f'{key}.png');shutil.copyfile(src,arc/'selected.png')
            for name in ('manifest.json','generation-jobs.json','visual-review.json','validation.json','prepare.py','registration.json'):
                if (oldout/name).is_file():shutil.copyfile(oldout/name,arc/name)
            # 所有歷史候選與提示詞留在本輪；不只留下最終選圖。
            for p in oldout.glob(old['key']+'-*-raw.png'):shutil.copyfile(p,arc/p.name)
            shutil.copyfile(oldreq/'manifest.json',arc/'request-manifest.json')
            it['reuse_archive']=str(arc.relative_to(out));it['generation']='原版純色保真，非AI生成' if it['rendering']=='native_exact_constant' else '前作內建 image_gen，完整色號相同後沿用'
        elif it['rendering']=='native_exact_constant' and it['status']=='READY':
            Image.open(req/it['reference_png']).convert('RGBA').save(out/f'{key}.png')
        groups[batch].append(it)
    for b,items in groups.items():
        req=hd/'requests'/b;out=hd/'out'/b
        write(req/'manifest.json',dict(batch=b,issue=13,spec='docs/spec/010-hd-theme.md §6',kind=KINDS[b],scale=3,
            palette=dict(name='EGA 預設16色',rgb=EGA),items=items))
        if b not in ('b07','b08','b10'):continue
        pending=[it for it in items if not (out/f'{it["key"]}.png').exists()]
        # 等尺寸一組，維持來源比例；每張圖集至多十六件。
        bysize={}
        for it in pending:bysize.setdefault(tuple(it['size']),[]).append(it)
        sheetnum=0
        for size,group in sorted(bysize.items()):
            for start in range(0,len(group),16):
                part=group[start:start+16];cols=4;rows=4;cell=256
                canvas=Image.new('RGB',(cols*cell,rows*cell),(85,85,85));draw=ImageDraw.Draw(canvas);placements=[]
                w,h=size;s=min(192//w,192//h);W,H=w*s,h*s;ox,oy=(cell-W)//2,(cell-H)//2
                for n,it in enumerate(part):
                    x,y=(n%cols)*cell+ox,(n//cols)*cell+oy
                    p=Image.open(req/it['png']).convert('RGB').resize((W,H),Image.Resampling.NEAREST);canvas.paste(p,(x,y))
                    # 標號在裁切區之外，生成時明確要求保留格線座標。
                    draw.text(((n%cols)*cell+6,(n//cols)*cell+6),str(n+1),fill=(255,255,255))
                    placements.append(dict(key=it['key'],rect=[x,y,W,H],target_size=it['target_size']))
                stem=f'atlas-{sheetnum:02d}';sheetnum+=1;canvas.save(req/(stem+'.png'))
                write(req/(stem+'.json'),dict(batch=b,atlas=stem,canvas=[1024,1024],cols=cols,rows=rows,
                    items=placements,reference=stem+'.png',source_shape=size))
        print(b,'items',len(items),'new',len(pending),'atlases',sheetnum)

def encounter_requests(root):
    hd=root/'workplace/hd';inv=hd/'inventory/issue13';j=json.loads((inv/'inventory.json').read_text())
    observed=json.loads((inv/'encounter-probe.json').read_text())['matches'];req=hd/'requests/b11';out=hd/'out/b11';items=[]
    for r in j['items']:
        if r['kind']!='encounter':continue
        key=r['key'];w,h=r['size'];raw=(inv/r['indexed']).read_bytes()
        positions={tuple(s['header_xy']) for s in r['aliases']};assert len(positions)==1
        hx,hy=positions.pop();x,y=24+hx*8,24+hy*8;assert 24<=x and x+w<=112 and 24<=y and y+h<=112
        idx=bytes(8 if c==0 else 0 if c==13 else c for c in raw)
        mask=bytes(int(c!=0) for c in raw);rawname=key+'.raw.idx';maskname=key+'.mask'
        (req/rawname).write_bytes(raw);(req/maskname).write_bytes(mask)
        evidence=[e for e in observed if e['key']==key]
        it=emit(req,3,key,'SPRIT 遭遇圖，來源0不畫、13→0',(x,y,w,h),None,idx,r['aliases'],
            kind='encounter',runtime_evidence='observed' if evidence else 'unobserved',evidence=evidence,
            raw_indexed=rawname,raw_indexed_sha256=sha(raw),foreground_mask=maskname,foreground_mask_sha256=sha(mask),
            palette_mapping=[8,1,2,3,4,5,6,7,8,9,10,11,12,0,14,15],background_index=8,
            preserve_native_colors=[8],status='READY',rendering='painted',position=[x,y])
        items.append(it)
    write(req/'manifest.json',dict(batch='b11',issue=13,spec='docs/spec/010-hd-theme.md §6.3',kind='encounter-sprites',scale=3,
        palette=dict(name='EGA 預設16色；灰底僅供美術參考',rgb=EGA),items=items))
    for start in range(0,len(items),16):
        group=items[start:start+16];canvas=Image.new('RGB',(1024,1024),(85,85,85));draw=ImageDraw.Draw(canvas);placements=[]
        for n,it in enumerate(group):
            w,h=it['size'];s=min(192//w,192//h);W,H=w*s,h*s;x=n%4*256+(256-W)//2;y=n//4*256+(256-H)//2
            im=Image.open(req/it['png']).convert('RGB').resize((W,H),Image.Resampling.NEAREST);canvas.paste(im,(x,y))
            draw.text((n%4*256+6,n//4*256+6),str(n+1),fill='white');placements.append(dict(key=it['key'],rect=[x,y,W,H],target_size=it['target_size']))
        stem=f'atlas-{start//16:02d}';canvas.save(req/(stem+'.png'));write(req/(stem+'.json'),dict(batch='b11',atlas=stem,canvas=[1024,1024],cols=4,rows=4,items=placements,reference=stem+'.png'))
    print('b11',len(items),'items')

def import_generated(root,generated):
    hd=root/'workplace/hd'
    for p in sorted(p for b in ('b07','b08','b09','b10','b11') for p in (hd/'out'/b/'jobs').glob('*.json')):
        j=json.loads(p.read_text());src=generated/Path(j['generated_path']).parent.name/Path(j['generated_path']).name;dest=p.parent.parent/j['raw']
        if not dest.exists():shutil.copyfile(src,dest)
        assert sha(src.read_bytes())==sha(dest.read_bytes())
        j['raw_sha256']=sha(dest.read_bytes());write(p,j)

def prepare(root):
    hd=root/'workplace/hd'
    for batch in ('b07','b08','b09','b10','b11'):
        req=hd/'requests'/batch;out=hd/'out'/batch;man=json.loads((req/'manifest.json').read_text())
        bykey={it['key']:it for it in man['items']}
        for job in sorted((out/'jobs').glob('*.json')):
            j=json.loads(job.read_text())
            if j.get('status')!='generated':continue
            raw=out/j['raw'];assert sha(raw.read_bytes())==j['raw_sha256']
            im=Image.open(raw).convert('RGBA');W,H=im.size
            if j.get('key'):
                it=bykey[j['key']];parts=[dict(key=it['key'],rect=[0,0,W,H],target_size=it['target_size'])];baseW,baseH=W,H
            else:
                atlas=json.loads((req/(j['atlas']+'.json')).read_text());assert W==H and W>=512,(raw,im.size)
                parts=atlas['items'];baseW,baseH=1024,1024
            for p in parts:
                if j.get('only') and p['key'] not in j['only']:continue
                it=bykey[p['key']];x,y,w,h=p['rect'];rect=(round(x*W/baseW),round(y*H/baseH),round((x+w)*W/baseW),round((y+h)*H/baseH))
                tile=im.crop(rect).resize(tuple(p['target_size']),Image.Resampling.LANCZOS);tile.putalpha(255)
                arr=np.array(tile);sw,sh=it['size'];idx=np.frombuffer((req/it['indexed']).read_bytes(),np.uint8).reshape(sh,sw)
                mask=np.repeat(np.repeat(idx==8,3,0),3,1);arr[mask,:3]=EGA[8]
                Image.fromarray(arr).save(out/(it['key']+'.png'))
        print(batch,'outputs',sum((out/(it['key']+'.png')).exists() for it in man['items']))

def sheets(root):
    hd=root/'workplace/hd'
    for batch in ('b07','b08','b09','b10','b11'):
        req=hd/'requests'/batch;out=hd/'out'/batch;j=json.loads((req/'manifest.json').read_text())
        for start in range(0,len(j['items']),16):
            group=j['items'][start:start+16];canvas=Image.new('RGB',(1280,4*200),(30,30,35));d=ImageDraw.Draw(canvas)
            for n,it in enumerate(group):
                ox,oy=n%4*320,n//4*200;d.text((ox+4,oy+3),it['key'],fill='white')
                for k,p in enumerate((req/it['reference_png'],out/(it['key']+'.png'))):
                    if not p.exists():continue
                    im=Image.open(p).convert('RGB');im.thumbnail((150,170),Image.Resampling.NEAREST)
                    canvas.paste(im,(ox+k*160+(150-im.width)//2,oy+23))
            canvas.save(out/f'review-{start//16:02d}.png')

def theme(root):
    hd=root/'workplace/hd';theme=hd/'theme';counts={}
    for batch,filename in (('b07','combat.json'),('b08','actors.json'),('b09','characters.json'),('b10','effects.json'),('b11','encounters.json')):
        req=hd/'requests'/batch;out=hd/'out'/batch;raw=(req/'manifest.json').read_bytes();man=json.loads(raw)
        validation=json.loads((out/'validation.json').read_text());review=json.loads((out/'visual-review.json').read_text())
        assert validation['all_pass'] and validation['request_manifest_sha256']==sha(raw)
        accepted={r['key']:r for r in review['items']};results={r['key']:r for r in validation['items']};items=[]
        assert review['request_sha256']==sha(raw)
        for it in man['items']:
            key=it['key'];png=out/(key+'.png');idx=req/it['indexed'];ph=sha(png.read_bytes())
            assert accepted[key]['approved'] and accepted[key]['png_sha256']==ph
            assert results[key]['target_met'] and results[key]['png_sha256']==ph
            assert sha(idx.read_bytes())==it['indexed_sha256']
            if batch=='b11':
                idx=req/it['raw_indexed'];assert sha(idx.read_bytes())==it['raw_indexed_sha256']
            shutil.copyfile(png,theme/(key+'.png'));shutil.copyfile(idx,theme/idx.name)
            items.append(dict(key=key,size=it['size'],png=key+'.png',png_sha256=ph,indexed=it['indexed'],
                indexed_sha256=sha(idx.read_bytes()),sources=it['sources'],batch=batch,runtime_evidence=it['runtime_evidence']))
            items[-1]['indexed']=idx.name
            if batch=='b11':items[-1]['position']=it['position']
        t=dict(version=1,scale=3,kind=KINDS[batch],grid=[8,8,7,7,24],background_index=8,
            request_manifest_sha256=sha(raw),validation_sha256=sha((out/'validation.json').read_bytes()),visual_review_sha256=sha((out/'visual-review.json').read_bytes()))
        if batch=='b09':
            t['heads']=[r for r in items if r['key'].startswith('chead')];t['bodies']=[r for r in items if r['key'].startswith('cbody')]
        else:t['items']=items
        write(theme/filename,t);counts[batch]=len(items)
    print(counts)

def main():
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['init','encounter_requests','import_generated','prepare','sheets','theme']);ap.add_argument('--root',required=True,type=Path);ap.add_argument('--previous',type=Path);ap.add_argument('--generated',type=Path,default=Path('/generated'))
    a=ap.parse_args()
    if a.action=='init':init(a.root,a.previous)
    elif a.action=='import_generated':import_generated(a.root,a.generated)
    else:globals()[a.action](a.root)

if __name__=='__main__':main()
