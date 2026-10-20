#!/usr/bin/env python3
"""規格010 §7：來源重播、美術閘門負例與交付索引。"""
import argparse
import json
from pathlib import Path
import platform
import shutil
import tempfile
from collections import Counter
import numpy as np
import PIL
from PIL import Image
from hd_battle_inventory import sha,write,source_records
from dax_pics import picture
from hd_wall_art import prepare
from hd_wall_theme import compile_theme

def hashes(path):return {str(p.relative_to(path)):sha(p.read_bytes()) for p in sorted(path.rglob('*')) if p.is_file()}

def controls(root):
    hd=root/'workplace/hd';orig=root/'workplace/input/dos/ckrynn';cache={};aliases=0;selected={}
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;m=json.loads((req/'manifest.json').read_text())
        for it in m['items']:
            target=(req/it['raw_indexed']).read_bytes();assert sha(target)==it['raw_indexed_sha256']
            for src in it['sources']:
                if src['file'] not in cache:cache[src['file']]=list(source_records(orig,src['file']))
                bid,raw,position=cache[src['file']][src['record_index']]
                for f in ('file_sha256','entry_offset','entry_bytes','offset','packed_size','decoded_size','decoded_sha256'):assert position[f]==src[f],(it['key'],f)
                assert bid==src['block'];w,h,n,x,y,px=picture(raw);assert [w,h]==it['size'];p=px[src['item']*w*h:(src['item']+1)*w*h];assert p==target
                aliases+=1
            p=hd/'out'/batch/(it['key']+'.png');selected[str(p.relative_to(hd))]=sha(p.read_bytes())
    prepare(root)
    assert {p:sha((hd/p).read_bytes()) for p in selected}==selected
    negatives=[]
    with tempfile.TemporaryDirectory(dir=hd/'out',prefix='issue14-gates-') as td:
        rr=Path(td);hh=rr/'workplace/hd';(hh/'inventory/issue14').mkdir(parents=True)
        shutil.copyfile(hd/'inventory/issue14/model.json',hh/'inventory/issue14/model.json')
        for batch in ('b13','b14'):
            req=hh/'requests'/batch;out=hh/'out'/batch;req.mkdir(parents=True);out.mkdir(parents=True)
            for p in (hd/'requests'/batch).glob('*.idx'):shutil.copyfile(p,req/p.name)
            shutil.copyfile(hd/'requests'/batch/'manifest.json',req/'manifest.json')
            m=json.loads((req/'manifest.json').read_text())
            for it in m['items']:shutil.copyfile(hd/'out'/batch/(it['key']+'.png'),out/(it['key']+'.png'))
            for name in ('validation.json','visual-review.json'):shutil.copyfile(hd/'out'/batch/name,out/name)
        compile_theme(rr);before=hashes(hh/'theme')
        for kind in ('revoked-review','expired-review-sha','stale-request','expired-png'):
            p=hh/('requests/b13/manifest.json' if kind=='stale-request' else 'out/b13/visual-review.json');saved=p.read_bytes();j=json.loads(saved);png=None;pngbefore=None
            if kind=='revoked-review':j['items'][0]['approved']=False
            elif kind=='expired-review-sha':j['items'][0]['png_sha256']='expired'
            elif kind=='stale-request':j['items'][0]['sources'][0]['offset']+=1
            else:
                key=j['items'][0]['key'];png=hh/'out/b13'/(key+'.png');pngbefore=png.read_bytes();png.write_bytes(pngbefore+b'expired')
            write(p,j);refused=False
            try:compile_theme(rr)
            except AssertionError:refused=True
            assert refused and before==hashes(hh/'theme'),kind
            negatives.append(dict(kind=kind,rejected=True,theme_unchanged=True));p.write_bytes(saved)
            if png is not None:png.write_bytes(pngbefore)
    write(hd/'out/issue14-controls.json',dict(source_items=len(selected),source_aliases=aliases,source_replay_identical=True,processed_replay_identical=True,negative_controls=negatives))
    print('source items',len(selected),'aliases',aliases,'replay identical; four gates rejected')

def metadata(root):
    hd=root/'workplace/hd';allrows=[];alljobs=[];model=json.loads((hd/'inventory/issue14/model.json').read_text());seen=set(model['used_tiles'])
    wallman=json.loads((hd/'requests/b13/manifest.json').read_text());wallrefs={}
    for it in wallman['items']:wallrefs[it['key']]=np.frombuffer((hd/'requests/b13'/it['raw_indexed']).read_bytes(),np.uint8).reshape(8,8)
    skyman=json.loads((hd/'requests/b14/manifest.json').read_text());skyrefs={}
    for it in skyman['items']:skyrefs[it['sources'][0]['block']]=np.frombuffer((hd/'requests/b14'/it['raw_indexed']).read_bytes(),np.uint8).reshape(it['size'][1],it['size'][0])
    for scene in model['results']:
        match=scene['matches'][0];owners=np.zeros((88,88),np.uint16);covered=np.zeros((88,88),bool)
        for bid,x,y in match['bg']['layers']:
            src=skyrefs[bid];h,w=src.shape;L,R=max(0,x),min(88,x+w);T,B=max(0,y),min(88,y+h)
            owners[T:B,L:R][src[T-y:B-y,L-x:R-x]!=13]=bid
        for pos,part in enumerate(match['plan']):
            keys=[part] if isinstance(part,str) else part if isinstance(part,list) else []
            x,y=pos%11*8,pos//11*8
            for key in keys:covered[y:y+8,x:x+8]|=wallrefs[key]!=13
        for bid in np.unique(owners[~covered]):
            if bid:seen.add(f'sky-{bid:03d}-00')
    for batch in ('b13','b14'):
        req=hd/'requests'/batch;out=hd/'out'/batch;m=json.loads((req/'manifest.json').read_text());v=json.loads((out/'validation.json').read_text());assert v['all_pass'];num={r['key']:r for r in v['items']};vis={r['key']:r for r in json.loads((out/'visual-review.json').read_text())['items']}
        jobs=[]
        for p in sorted((out/'jobs').glob('*.json')):
            j=json.loads(p.read_text());assert sha((out/j['raw']).read_bytes())==j['raw_sha256'];jobs.append(dict(file=str(p.relative_to(out)),sha256=sha(p.read_bytes()),**j))
        rows=[]
        for it in m['items']:
            key=it['key'];png=out/(key+'.png');digest=sha(png.read_bytes());assert digest==num[key]['png_sha256']==vis[key]['png_sha256'] and vis[key]['approved']
            with Image.open(png) as im:
                assert im.format=='PNG' and im.mode=='RGBA' and list(im.size)==it['target_size']
                alpha=im.getchannel('A').getextrema();assert alpha==(255,255)
            related=[]
            for job in jobs:
                keys=[job['key']] if job.get('key') else [p['key'] for p in json.loads((req/(job['atlas']+'.json')).read_text())['items']]
                if key in keys and (not job.get('only') or key in job['only']):related.append(job['file'])
            row=dict(key=key,size=it['target_size'],png=key+'.png',png_sha256=digest,png_mode='RGBA',alpha_range=[255,255],sources=it['sources'],source_indexed=it['raw_indexed'],source_indexed_sha256=it['raw_indexed_sha256'],
                rendering=it['rendering'],generation_jobs=related,selected_job=related[-1] if related else None,validation=num[key],visual_review=vis[key],
                runtime_evidence='observed-pattern' if key in seen else 'unobserved-source',
                evidence_limit='只證實完整色號重建，原版driver選圖與時序未知。',status='theme-ready')
            rows.append(row);allrows.append(dict(batch=batch,**row))
        write(out/'manifest.json',dict(batch=batch,issue=14,scale=3,style='B',spec='docs/spec/010-hd-theme.md §7',
            request_manifest_sha256=sha((req/'manifest.json').read_bytes()),generation_calls=len(jobs),jobs=jobs,items=rows,
            rights='依專案定案，AI重繪可放入原版＋HD patch。原始DAX、state與原版擷取僅留本機。'))
        alljobs.extend(dict(j,batch=batch) for j in jobs)
    art=Counter(r['runtime_evidence'] for r in allrows);wall=json.loads((hd/'theme/walls.json').read_text())
    write(hd/'out/issue14-status.json',dict(issue=14,base_theme_items=490,new_theme_items=len(allrows),total_theme_items=490+len(allrows),
        generated_items=sum(bool(r['generation_jobs']) for r in allrows),constant_exact_items=sum(r['rendering']=='native_exact_constant' for r in allrows),
        generation_calls=len(alljobs),source_patterns=dict(art),global_cells=sum(not c['positions'] for c in wall['cells']),position_cells=sum(bool(c['positions']) for c in wall['cells']),
        items=allrows,limits='主題來源覆蓋不代表全遊戲或所有牆面組合已驗收；未知格回退原版。'))
    write(hd/'out/issue14-generation-jobs.json',dict(issue=14,calls=len(alljobs),jobs=alljobs))
    files=[root/'docs/spec/010-hd-theme.md']+sorted((root/'tools').glob('hd_wall_*.py'))+sorted((root/'tools').glob('issue14-hd-*.sh'))
    dg=root/'workplace/dosgolem';files += [dg/p for p in ('apps/cok/wall_hd.go','apps/cok/wall_hd_test.go','apps/cok/hd_layers.go','apps/cok/hd_live.go','cmd/cok-hd-probe/main.go')]
    write(hd/'out/issue14-code-files.json',dict(files=[dict(file=str(p.relative_to(root)),sha256=sha(p.read_bytes())) for p in files]))
    write(hd/'out/issue14-inputs.json',dict(sources=json.loads((hd/'inventory/issue14/sources.json').read_text())['files'],
        source_inventory_sha256=sha((hd/'inventory/issue14/sources.json').read_bytes()),model_sha256=sha((hd/'inventory/issue14/model.json').read_bytes()),
        input_scope='DAX檔案offset；畫面為dosgolem320×200色號。狀態與亂數條件見out/issue14-runtime/state-comparison.json。',
        tools=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__)))
    print('metadata',len(allrows),'items',len(alljobs),'calls total',490+len(allrows))

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['controls','metadata']);ap.add_argument('--root',required=True,type=Path);a=ap.parse_args();globals()[a.action](a.root)
