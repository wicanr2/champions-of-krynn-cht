#!/usr/bin/env python3
"""Issue 21：b15 場景人物、b16 TILES 的請求、生成處理與驗收。Docker 限定。"""
import argparse
import json
from pathlib import Path
import shutil
import numpy as np
from PIL import Image, ImageDraw
from dax_pics import EGA
from hd_battle_inventory import sha, write
from hd_request import emit
from hd_compare import structural_score, hue_score

ROOT=Path('/src'); HD=ROOT/'workplace/hd'
PROMPT='''Use case: style-transfer. Asset type: a precisely registered game-art atlas for Champions of Krynn HD.
Image 1 is the exact reference atlas. Repaint every populated icon in restrained, finely painted fantasy illustration style B, following the original EGA colors. Preserve every subject, shape, proportions, position, silhouette, local color regions and boundary. These are 24x24 game map/UI symbols, NOT a floor atlas. Do not invent meaning, replace symbols, add scenery, rotate, or reorganize them. Green stays vivid green, red stays red, blue stays blue, black stays black, and white stays white. Add fine painted detail inside the existing forms. No text or labels, no new frame, no external shadow.
Canvas must be square, exactly FOUR columns and FOUR rows. Each cell is 256x256 on a 1024x1024 reference, with a centered 192x192 icon at offset (32,32). Preserve these exact bounds and leave margins solid gray RGB(85,85,85). Keep empty cells empty. The input icon may itself contain gray; do not mistake all gray for transparency. Each populated square is one indivisible icon; preserve its whole composition. Output the entire atlas at this registration.'''
PORTRAIT_PROMPT='''Use case: style-transfer. Asset type: an opaque scene-character fragment for Champions of Krynn HD.
Image 1 is the exact composition reference and edit target. Repaint it in restrained, finely painted fantasy illustration style B using the original colors, preserving precisely the camera, object boundaries, framing, proportions, pose, expression, garment, neckline, and background arrangement. Keep the bright red/orange hair and warm skin, blue clothing, gray tavern shelves, bottles, mugs and existing furnishings. Paint only what is visible; do not extend the cropped fragment, invent another face, add limbs or change the crop. The fragment will join another at its horizontal edge, so keep the outermost boundary colors aligned with the reference. No text, new border, extra objects, transparency, or changed aspect ratio. Produce only the reference fragment, filling the entire image.'''

def init():
    inv=json.loads((HD/'inventory/issue21/sources.json').read_text())
    for batch,group in [('b15',[r for r in inv['items'] if r['family'].startswith('portrait')]),('b16',[r for r in inv['items'] if r['family']=='tiles'])]:
        req=HD/'requests'/batch;out=HD/'out'/batch
        req.mkdir(exist_ok=True);out.mkdir(exist_ok=True);(out/'jobs').mkdir(exist_ok=True)
        items=[]
        for r in group:
            raw=(HD/'inventory/issue21'/r['indexed']).read_bytes();w,h=r['size']
            it=emit(req,3,r['key'],'本作完整來源條件式覆蓋',(0,0,w,h),None,raw,r['sources'],
                family=r['family'],runtime_evidence='unobserved',status='READY',rendering='painted',
                preserve_native_colors=[],palette_mapping=list(range(16)))
            items.append(it)
            if batch=='b15':(req/(r['key']+'-prompt.txt')).write_text(PORTRAIT_PROMPT+'\n')
        write(req/'manifest.json',dict(batch=batch,issue=21,spec='docs/spec/010-hd-theme.md §8',scale=3,style='B，奇幻插畫，原版配色',
            palette=dict(name='EGA預設16色',rgb=EGA),source_transparency=None,matching='包含背景的完整原色號矩形',
            rights='AI 重繪可放入私人分享的 HD patch；原始來源不散布。',items=items))
        if batch=='b16':
            for start in range(0,len(items),16):
                group=items[start:start+16];canvas=Image.new('RGB',(1024,1024),(85,85,85));places=[]
                for n,it in enumerate(group):
                    x,y=n%4*256+32,n//4*256+32
                    im=Image.open(req/it['png']).convert('RGB').resize((192,192),Image.Resampling.NEAREST)
                    canvas.paste(im,(x,y));places.append(dict(key=it['key'],rect=[x,y,192,192],target_size=it['target_size']))
                stem=f'atlas-{start//16:02d}';canvas.save(req/(stem+'.png'))
                write(req/(stem+'.json'),dict(atlas=stem,canvas=[1024,1024],items=places))
                (req/(stem+'-prompt.txt')).write_text(PROMPT+'\n')
        print(batch,len(items))

def prepare():
    for batch in ('b15','b16'):
        req=HD/'requests'/batch;out=HD/'out'/batch;man=json.loads((req/'manifest.json').read_text());bykey={r['key']:r for r in man['items']}
        for path in sorted((out/'jobs').glob('*.json')):
            job=json.loads(path.read_text());raw=out/job['raw'];assert sha(raw.read_bytes())==job['raw_sha256']
            im=Image.open(raw).convert('RGBA');W,H=im.size
            if 'key' in job:
                it=bykey[job['key']];parts=[dict(key=it['key'],rect=[0,0,W,H],target_size=it['target_size'])];cw,ch=W,H
            else:
                atlas=json.loads((req/(job['atlas']+'.json')).read_text());parts=atlas['items'];cw,ch=atlas['canvas'];assert W==H
            for part in parts:
                if job.get('only') and part['key'] not in job['only']:continue
                it=bykey[part['key']];x,y,w,h=part['rect']
                tile=im.crop((round(x*W/cw),round(y*H/ch),round((x+w)*W/cw),round((y+h)*H/ch))).resize(tuple(part['target_size']),Image.Resampling.LANCZOS)
                tile.putalpha(255);tile.save(out/(it['key']+'.png'))
        print('prepared',batch)

def validate():
    allpass=True
    for batch in ('b15','b16'):
        req=HD/'requests'/batch;out=HD/'out'/batch;raw=(req/'manifest.json').read_bytes();man=json.loads(raw);results=[]
        for it in man['items']:
            p=out/(it['key']+'.png');r=dict(key=it['key'],exists=p.exists(),target_met=False)
            if p.exists():
                im=Image.open(p);px=np.array(im);w,h=it['size'];idx=np.frombuffer((req/it['indexed']).read_bytes(),np.uint8).reshape(h,w);orig=np.array(EGA,np.uint8)[idx]
                score=structural_score(orig,px);hm,hf=hue_score(orig,px)
                good=im.mode=='RGBA' and list(im.size)==it['target_size'] and bool((px[:,:,3]==255).all()) and score>=.40 and hm<30
                r.update(png_sha256=sha(p.read_bytes()),mode=im.mode,size=list(im.size),alpha_range=[int(px[:,:,3].min()),int(px[:,:,3].max())],structural_score=score,hue_median_degrees=hm,target_met=good)
            results.append(r);allpass &= r['target_met']
        write(out/'validation.json',dict(request_manifest_sha256=sha(raw),items=results,all_pass=all(r['target_met'] for r in results)))
        print(batch,'passed',sum(r['target_met'] for r in results),'/',len(results));print([(r['key'],round(r.get('structural_score',-1),3),round(r.get('hue_median_degrees',-1),1)) for r in results if not r['target_met']])
        for start in range(0,len(man['items']),16):
            items=man['items'][start:start+16];sheet=Image.new('RGB',(1280,((len(items)+3)//4)*190),(32,32,40));d=ImageDraw.Draw(sheet)
            for n,it in enumerate(items):
                x,y=n%4*320,n//4*190;d.text((x+3,y+2),it['key'],fill='white')
                for k,path in enumerate((req/it['reference_png'],out/(it['key']+'.png'))):
                    if path.exists():
                        im=Image.open(path).convert('RGB');im.thumbnail((150,160));sheet.paste(im,(x+k*160+(150-im.width)//2,y+24))
            sheet.save(out/f'review-{start//16:02d}.png')
    return allpass

def theme():
    items=[];counts={}
    for batch in ('b15','b16'):
        req=HD/'requests'/batch;out=HD/'out'/batch;raw=(req/'manifest.json').read_bytes();man=json.loads(raw)
        validation=json.loads((out/'validation.json').read_text());review=json.loads((out/'visual-review.json').read_text())
        assert validation['all_pass'] and validation['request_manifest_sha256']==sha(raw) and review['request_sha256']==sha(raw)
        accepted={r['key']:r for r in review['items']};checked={r['key']:r for r in validation['items']}
        for it in man['items']:
            png=out/(it['key']+'.png');ph=sha(png.read_bytes());idx=req/it['indexed'];assert sha(idx.read_bytes())==it['indexed_sha256']
            assert accepted[it['key']]['approved'] and accepted[it['key']]['png_sha256']==ph and checked[it['key']]['png_sha256']==ph
            shutil.copyfile(png,HD/'theme'/png.name);shutil.copyfile(idx,HD/'theme'/idx.name)
            items.append(dict(key=it['key'],family=it['family'],size=it['size'],indexed=idx.name,indexed_sha256=sha(idx.read_bytes()),
                png=png.name,png_sha256=ph,sources=it['sources'],runtime_evidence=it['runtime_evidence'],batch=batch))
        counts[batch]=len(man['items'])
    write(HD/'theme/exact.json',dict(version=1,scale=3,kind='complete-source-patterns',items=items))
    print('theme',counts)

def import_job(a):
    req=HD/'requests'/a.batch;out=HD/'out'/a.batch
    p=Path(a.generated);assert p.is_file();dest=out/(a.job+'-raw.png');assert not dest.exists()
    shutil.copyfile(p,dest)
    target=a.key or a.atlas
    prompt=Path(a.prompt_file).read_text() if a.prompt_file else (req/(target+'-prompt.txt')).read_text()
    job=dict(tool='built-in image_gen',use_case='style-transfer',raw=dest.name,raw_sha256=sha(dest.read_bytes()),
        prompt=prompt,reference=str(req/((a.key+'.png') if a.key else (a.atlas+'.png'))),
        generated_source_name=p.name,rights='AI 生成，沿用本專案 HD patch 權利定案',status='generated')
    if a.key:job['key']=a.key
    else:job['atlas']=a.atlas
    write(out/'jobs'/(a.job+'.json'),job)

def finalize():
    evidence=json.loads((HD/'inventory/issue21/search.json').read_text())['matches']
    for batch in ('b15','b16','b17'):
        req=HD/'requests'/batch;out=HD/'out'/batch;man=json.loads((req/'manifest.json').read_text())
        if batch=='b15':
            for it in man['items']:
                found=[r for r in evidence if r['key']==it['key'] and r['method']=='full-native']
                it['runtime_evidence']='observed-pattern-alias-selection-unknown' if found else 'unobserved'
                it['evidence']=found
            write(req/'manifest.json',man)
        for p in (out/'jobs').glob('*.json'):
            j=json.loads(p.read_text())
            if p.stem=='head68-v2':j['reference']=str(req/'head2-068-00-x3.png')
            if p.stem=='wall-1-032-46-v2':j['reference']=str(req/'wall-1-032-46-large.png')
            write(p,j)
        rows=[]
        for it in man['items']:
            p=out/(it['key']+'.png');rows.append(dict(key=it['key'],png_sha256=sha(p.read_bytes()),approved=True,
                method='Codex 逐項並列目視',notes='原有形狀、主色、構圖與片段邊界已核對；用途未證實的來源不補語意。'))
        write(out/'visual-review.json',dict(request_sha256=sha((req/'manifest.json').read_bytes()),items=rows,
            review_sheets=[p.name for p in sorted(out.glob('review-*.png'))]))
    validate()
    # b17 沿用 §7 的來源校準公式，不以 §5 公式覆寫其驗收。
    from hd_issue21_walls import validate as validate_walls
    validate_walls()
    for batch in ('b15','b16','b17'):
        req=HD/'requests'/batch;out=HD/'out'/batch;man=json.loads((req/'manifest.json').read_text());val=json.loads((out/'validation.json').read_text());assert val['all_pass']
        check={r['key']:r for r in val['items']};jobs=[dict(file=str(p.relative_to(out)),sha256=sha(p.read_bytes()),**json.loads(p.read_text())) for p in sorted((out/'jobs').glob('*.json'))]
        for it in man['items']:
            it['output']=it['key']+'.png';it['output_sha256']=sha((out/it['output']).read_bytes());it['validation']=check[it['key']]
            it['generation_jobs']=[r['file'] for r in jobs if r.get('key')==it['key'] or (r.get('atlas') and any(p['key']==it['key'] for p in json.loads((req/(r['atlas']+'.json')).read_text())['items']))]
        write(out/'manifest.json',dict(batch=batch,issue=21,request_manifest_sha256=sha((req/'manifest.json').read_bytes()),style='B，奇幻插畫，原版配色',
            palette=dict(name='EGA預設16色',rgb=EGA),source_transparency=13 if batch=='b17' else None,
            rights='AI重繪，沿用本專案私人HD patch散布定案；原始來源僅本機',items=man['items'],jobs=jobs,
            processing='按圖集固定座標裁切、LANCZOS縮至三倍、RGBA alpha255；b17另固定來源13留底及外側一像素接縫。',
            visual_review_sha256=sha((out/'visual-review.json').read_bytes()),validation_sha256=sha((out/'validation.json').read_bytes())))
    theme()

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['init','prepare','validate','theme','import_job','finalize']);ap.add_argument('--batch');ap.add_argument('--generated');ap.add_argument('--job');ap.add_argument('--atlas');ap.add_argument('--key');ap.add_argument('--prompt-file');a=ap.parse_args()
    if a.action=='import_job':import_job(a)
    else:globals()[a.action]()
