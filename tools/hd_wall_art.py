#!/usr/bin/env python3
"""規格010 §7：牆面與SKY請求、影像圖集處理及驗收。Docker限定。"""
import argparse
import json
from pathlib import Path
import numpy as np
from PIL import Image,ImageDraw
from dax_pics import EGA
from hd_request import emit,rgb_png
from hd_wall_inventory import load_sources,rgb
from hd_battle_inventory import sha,write
from hd_compare import structural_score,hue_score,edges,lowpass

_SOURCE_CALIBRATION={}
def source_calibration(orig):
    key=(orig.shape,orig.tobytes())
    if key not in _SOURCE_CALIBRATION:
        scores=[]
        for delta in (0,-1,1):
            p=np.clip(orig.astype(np.int16)+delta,0,255).astype(np.uint8)
            scores.append(structural_score(orig,np.repeat(np.repeat(p,3,0),3,1)))
        _SOURCE_CALIBRATION[key]=scores
    return _SOURCE_CALIBRATION[key]

def wall_structure(orig,hd):
    """§7.3–7.4：來源正對照不穩定才改用連續梯度，判定不依候選分數。"""
    legacy=structural_score(orig,hd)
    lum=lambda a:a[...,0]*.299+a[...,1]*.587+a[...,2]*.114
    filtered=lowpass(orig)
    if min(source_calibration(orig))>=.4:return legacy,legacy,'legacy-percentile'
    h,w=orig.shape[:2];sy,sx=hd.shape[0]//h,hd.shape[1]//w
    small=hd[...,:3].reshape(h,sy,w,sx,3).mean(axis=(1,3))
    def gradient(a):
        p=lum(a)
        return np.abs(np.diff(p,axis=1))[:-1,:]+np.abs(np.diff(p,axis=0))[:,:-1]
    a,b=gradient(filtered),gradient(lowpass(small));denom=np.maximum(a,b).sum()
    score=float(np.minimum(a,b).sum()/denom) if denom>0 else 0.
    return score,legacy,'continuous-gradient-jaccard-calibrated-source'

PROMPT='''Use case: style-transfer.
Asset type: a precisely registered 8 by 8 atlas of fantasy dungeon wall texture fragments for Champions of Krynn HD. Image 1 is the exact reference atlas, not an example layout.
Repaint every populated square in a restrained, finely painted fantasy illustration style B, using the original local colors. These are tiny stone, brick, wood, metal, roof, doorway and diagonal perspective wall fragments, not characters or icons. Preserve the precise positions, proportions, perspective edges, mortar lines, stripes, highlights and dark regions of EVERY fragment. Keep black black, cyan cyan, green green, gray gray, ochre ochre, red red, blue blue. Do not turn the colorful walls brown or desaturate them.
Keep the canvas exactly square, with exactly eight columns and eight rows. Each cell is 128 by 128 on the 1024 reference, with a centered 96 by 96 fragment at offset 16,16. Keep those exact square bounds. The flat gray margins and gray masked-out regions must stay flat gray RGB 85,85,85. Leave empty cells empty. No numbers, labels, text, bevel frames or extra scenery.
Tile continuity is critical: do not move the edge pixels, silhouette, corners or line endpoints. The eight-pixel original pattern defines exact neighboring joins. Paint subtle surface detail inside the original shapes. No dramatic lighting or cast shadows outside the original objects. Adjacent pieces and left/right perspective pairs must join without a visible new seam. Do not add objects or decorations. Keep each fragment pinned to its reference position.'''

def init(root):
    hd=root/'workplace/hd';inv=json.loads((hd/'inventory/issue14/sources.json').read_text());tiles,defs,pics,files=load_sources(root)
    used={t['key'] for t in inv['tiles'] if t['used_by_walldef']}
    groups={'b13':[t for t in tiles if t['key'] in used and any(v!=13 for v in t['raw'])],
            'b14':[p for p in pics if p['key'].startswith('sky-')]}
    for batch,rows in groups.items():
        req=hd/'requests'/batch;out=hd/'out'/batch;req.mkdir(exist_ok=True);out.mkdir(exist_ok=True);(out/'jobs').mkdir(exist_ok=True)
        items=[]
        for r in rows:
            key=r['key'];w,h=r['size'];raw=r['raw'];mapped=bytes(8 if c==13 else c for c in raw)
            it=emit(req,3,key,'WALLDEF／8X8D牆面來源' if batch=='b13' else 'SKY來源',
                (24,24,w,h),None,mapped,r.get('aliases',[r.get('source')]),
                raw_indexed=key+'.raw.idx',raw_indexed_sha256=sha(raw),transparent_source=13,
                preserve_native_colors=[],rendering='native_exact_constant' if len(set(mapped))==1 else 'painted',
                status='DRAFT',runtime_evidence='見inventory/issue14/model.json',
                mask=key+'.mask',mask_sha256=sha(bytes(int(c!=13) for c in raw)))
            (req/it['raw_indexed']).write_bytes(raw);(req/it['mask']).write_bytes(bytes(int(c!=13) for c in raw))
            if it['rendering']=='native_exact_constant':
                Image.open(req/it['reference_png']).convert('RGBA').save(out/(key+'.png'))
            items.append(it)
        write(req/'manifest.json',dict(batch=batch,issue=14,spec='docs/spec/010-hd-theme.md §7',scale=3,
            kind='wall-tiles' if batch=='b13' else 'sky',palette=dict(name='EGA 預設16色；13留底以灰色表示',rgb=EGA),items=items))
        pending=[it for it in items if it['rendering']!='native_exact_constant']
        if batch=='b13':
            for start in range(0,len(pending),64):
                group=pending[start:start+64];canvas=Image.new('RGB',(1024,1024),(85,85,85));places=[]
                for n,it in enumerate(group):
                    x,y=n%8*128+16,n//8*128+16
                    im=Image.open(req/it['png']).convert('RGB').resize((96,96),Image.Resampling.NEAREST);canvas.paste(im,(x,y))
                    places.append(dict(key=it['key'],rect=[x,y,96,96],target_size=it['target_size']))
                stem=f'atlas-{start//64:02d}';canvas.save(req/(stem+'.png'))
                write(req/(stem+'.json'),dict(atlas=stem,canvas=[1024,1024],items=places))
                (req/(stem+'-prompt.txt')).write_text(PROMPT+'\n')
        print(batch,len(items),'AI',len(pending),'atlases',(len(pending)+63)//64)

def prepare(root,only_job=None):
    hd=root/'workplace/hd'
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;out=hd/'out'/batch;j=json.loads((req/'manifest.json').read_text());bykey={it['key']:it for it in j['items']}
        for job in sorted((out/'jobs').glob('*.json')):
            if only_job and job.name!=only_job:continue
            desc=json.loads(job.read_text());raw=out/desc['raw']
            assert sha(raw.read_bytes())==desc['raw_sha256']
            image=Image.open(raw).convert('RGBA');W,H=image.size
            if desc.get('atlas'):
                atlas=json.loads((req/(desc['atlas']+'.json')).read_text());places=atlas['items'];cw,ch=atlas['canvas']
            else:
                it=bykey[desc['key']];places=[dict(key=it['key'],rect=[0,0,W,H],target_size=it['target_size'])];cw,ch=W,H
            for p in places:
                if desc.get('only') and p['key'] not in desc['only']:continue
                it=bykey[p['key']];x,y,w,h=p['rect'];crop=(round(x*W/cw),round(y*H/ch),round((x+w)*W/cw),round((y+h)*H/ch))
                im=image.crop(crop).resize(tuple(p['target_size']),Image.Resampling.LANCZOS);im.putalpha(255)
                arr=np.array(im);sw,sh=it['size'];src=np.frombuffer((req/it['raw_indexed']).read_bytes(),np.uint8).reshape(sh,sw)
                mask=np.repeat(np.repeat(src==13,3,0),3,1);arr[mask,:3]=EGA[8]
                # 一個輸出像素寬的來源邊界固定接縫，不改內部線條。
                if batch=='b13':
                    native=np.repeat(np.repeat(np.array(EGA,np.uint8)[np.where(src==13,8,src)],3,0),3,1)
                    border=np.zeros(arr.shape[:2],bool);border[0,:]=border[-1,:]=True;border[:,0]=border[:,-1]=True
                    arr[border,:3]=native[border]
                Image.fromarray(arr).save(out/(it['key']+'.png'))
        print(batch,'PNG',sum((out/(it['key']+'.png')).is_file() for it in j['items']))

def review(root):
    hd=root/'workplace/hd';overall=True
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;out=hd/'out'/batch;j=json.loads((req/'manifest.json').read_text());pending=[];rows=[]
        for it in j['items']:
            p=out/(it['key']+'.png')
            if not p.exists():pending.append(it['key']);continue
            arr=np.array(Image.open(p).convert('RGBA'));w,h=it['size'];src=np.frombuffer((req/it['indexed']).read_bytes(),np.uint8).reshape(h,w)
            orig=np.array(EGA,np.uint8)[src];constant=it['rendering']=='native_exact_constant'
            structure,legacy,method=wall_structure(orig,arr);hue=hue_score(orig,arr)[0]
            raw=np.frombuffer((req/it['raw_indexed']).read_bytes(),np.uint8).reshape(h,w)
            mask=np.repeat(np.repeat(raw==13,3,0),3,1);off=int(np.any(arr[...,:3]!=EGA[8],axis=2)[mask].sum())
            good=arr.shape==(h*3,w*3,4) and (arr[...,3]==255).all() and off==0 and hue<30 and (structure>=.4 or constant and np.all(arr[...,:3]==orig[0,0]))
            rows.append(dict(key=it['key'],png_sha256=sha(p.read_bytes()),structural_score=structure,legacy_structural_score=legacy,structure_method=method,hue=hue,mask_difference=off,target_met=bool(good)))
        passed=not pending and all(r['target_met'] for r in rows);overall &= passed
        write(out/'validation.json',dict(request_sha256=sha((req/'manifest.json').read_bytes()),all_pass=passed,pending=pending,items=rows))
        print(batch,'pending',len(pending),'failed',[r['key'] for r in rows if not r['target_met']])
        # 每頁64項原版／HD並排，逐項目視後另寫綁定SHA的審查紀錄。
        complete=[it for it in j['items'] if (out/(it['key']+'.png')).is_file()]
        if batch=='b13':
            for start in range(0,len(complete),64):
                page=complete[start:start+64];sheet=Image.new('RGB',(1024,1024),'#303030');draw=ImageDraw.Draw(sheet)
                for n,it in enumerate(page):
                    x,y=n%8*128,n//8*128;draw.text((x+2,y+2),str(start+n),fill='white')
                    for delta,path in ((0,req/it['reference_png']),(64,out/(it['key']+'.png'))):
                        im=Image.open(path).convert('RGB').resize((60,60),Image.Resampling.NEAREST);sheet.paste(im,(x+delta,y+28))
                sheet.save(out/f'review-{start//64:02d}.png')
    if not overall:raise SystemExit(1)

def retry(root):
    hd=root/'workplace/hd';req=hd/'requests/b13';out=hd/'out/b13'
    man=json.loads((req/'manifest.json').read_text());table={it['key']:it for it in man['items']}
    rows=[r for r in json.loads((out/'validation.json').read_text())['items'] if not r['target_met']]
    used=list(req.glob('redo-*.json'));start=len(used)
    for offset in range(0,len(rows),16):
        group=rows[offset:offset+16];canvas=Image.new('RGB',(1024,1024),(85,85,85));places=[]
        for n,row in enumerate(group):
            it=table[row['key']];x,y=n%4*256+32,n//4*256+32
            canvas.paste(Image.open(req/it['png']).convert('RGB').resize((192,192),Image.Resampling.NEAREST),(x,y))
            places.append(dict(key=it['key'],rect=[x,y,192,192],target_size=it['target_size']))
            old=out/(it['key']+'.png');archive=out/(it['key']+f'-failed-{start+offset//16:02d}.png')
            if not archive.exists():archive.write_bytes(old.read_bytes())
        stem=f'redo-{start+offset//16:02d}';canvas.save(req/(stem+'.png'));write(req/(stem+'.json'),dict(atlas=stem,canvas=[1024,1024],items=places))
    write(out/'retry-plan.json',dict(failed=rows,atlases=[f'redo-{start+n:02d}' for n in range((len(rows)+15)//16)]))
    print('retry',len(rows),'atlases',(len(rows)+15)//16)

def calibrate(root):
    hd=root/'workplace/hd';req=hd/'requests/b13';man=json.loads((req/'manifest.json').read_text());rows=[];null_source=None
    for it in man['items']:
        w,h=it['size'];raw=(req/it['indexed']).read_bytes();orig=np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(h,w)]
        if len(set(raw))==1:continue
        nn=np.repeat(np.repeat(orig,3,0),3,1);score,legacy,method=wall_structure(orig,nn)
        assert score==1,(it['key'],score,legacy,method)
        scores=source_calibration(orig);probes=[]
        for delta in (-1,1):
            p=np.clip(orig.astype(np.int16)+delta,0,255).astype(np.uint8);effective=wall_structure(orig,np.repeat(np.repeat(p,3,0),3,1))[0];assert effective>=.4,(it['key'],delta,effective)
            probes.append(dict(delta=delta,effective_score=effective))
        rows.append(dict(key=it['key'],source_sha256=sha(raw),self_score=score,legacy_self_score=legacy,legacy_minus_one=scores[1],legacy_plus_one=scores[2],perturbations=probes,method=method))
        if scores[0]==0 and null_source is None:null_source=(it['key'],orig)
    assert null_source is not None
    control_key,original=null_source;same=np.repeat(np.repeat(original,3,0),3,1)
    controls=[]
    for name,p in (('same',same),('shift-nine-output-pixels',np.roll(same,9,axis=1)),('erased-flat',np.full_like(same,85))):
        score,legacy,method=wall_structure(original,p);controls.append(dict(kind=name,source_key=control_key,input_sha256=sha(p.tobytes()),score=score,legacy=legacy,method=method))
        assert score==1 if name=='same' else score<.4
    write(hd/'out/issue14-structure-calibration.json',dict(spec='010 §7.3',tool_sha256=sha(Path(__file__).read_bytes()),
        request_sha256=sha((req/'manifest.json').read_bytes()),nonconstant_sources=len(rows),all_self_scores_one=True,
        source_null_count=sum(r['legacy_self_score']==0 for r in rows),unstable_source_count=sum(r['method']=='continuous-gradient-jaccard-calibrated-source' for r in rows),items=rows,controls=controls))
    print('calibration',len(rows),'source-null',sum(r['legacy_self_score']==0 for r in rows),'unstable',sum(r['method']=='continuous-gradient-jaccard-calibrated-source' for r in rows),controls)

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['init','prepare','review','retry','calibrate']);ap.add_argument('--root',type=Path,required=True);ap.add_argument('--job');a=ap.parse_args()
    if a.action=='prepare':prepare(a.root,a.job)
    else:globals()[a.action](a.root)
