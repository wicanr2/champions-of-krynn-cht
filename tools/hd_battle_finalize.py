#!/usr/bin/env python3
"""規格010 §6：同狀態 A/B 與逐項交付索引，全部產物只寫 workplace/hd。"""
import argparse
from collections import Counter
import json
from pathlib import Path
import platform
import shutil
import tempfile
import numpy as np
from PIL import Image
import PIL
from hd_battle_inventory import sha,write

def hashes(path):return {str(p.relative_to(path)):sha(p.read_bytes()) for p in sorted(path.rglob('*')) if p.is_file()}

def runtime(root,prefix='',filename='receipt.json'):
    from hd_battle_receipts import Oracle
    hd=root/'workplace/hd';dest=hd/'out/issue13-runtime';oracle=Oracle(hd/'theme');rows=[]
    for s in (3,2):
        left=dest/f'{prefix}base{s}';right=dest/f'{prefix}hd{s}';a=json.loads((left/'receipt.json').read_text());b=json.loads((right/'receipt.json').read_text())
        fields=('initial_state_sha256','initial_steps','steps','memory_sha256','registers','keys_sent')
        state_equal=all(a[k]==b[k] for k in fields);assert state_equal
        scratch_equal=hashes(left/'scratch')==hashes(right/'scratch');assert scratch_equal
        aa={r['name']:r for r in a['shots']};bb={r['name']:r for r in b['shots']};assert set(aa)==set(bb)
        for name,shot in aa.items():
            raw=(left/(name+'.bin')).read_bytes();same_original=raw==(right/(name+'.bin')).read_bytes();assert same_original and shot['steps']==bb[name]['steps']
            pa=np.array(Image.open(left/(name+'.png')).convert('RGBA'));pb=np.array(Image.open(right/(name+'.png')).convert('RGBA'));changed=np.any(pa!=pb,axis=2)
            predicted,allowed,hits=oracle.expected(raw);mask=np.repeat(np.repeat(allowed,s,0),s,1)
            outside=int(changed[~mask].sum());expected_difference=0
            if s==3:
                want=pa.copy();want[mask]=predicted[mask];expected_difference=int(np.any(want!=pb,axis=2).sum())
            else:expected_difference=int(changed.sum())
            ok=state_equal and scratch_equal and same_original and outside==0 and expected_difference==0
            rows.append(dict(scene=name,scale=s,steps=shot['steps'],display_difference=int(changed.sum()),outside_difference=outside,
                independent_expected_difference=expected_difference,source_matches=hits,state_equal=state_equal,scratch_equal=scratch_equal,same_original=same_original,ok=ok))
        write(dest/f'{prefix}state-comparison-{s}.json',dict(without_hd=a,with_hd=b,scratch_without_hd=hashes(left/'scratch'),scratch_with_hd=hashes(right/'scratch')))
    result=dict(issue=13,scope='dosgolem demo 保存狀態續跑；原版RAM/CPU/色號/暫存存檔與獨立整幀預期',
        seed=('執行前固定同一battle.state及輸入；battle.state由d450.state無注入續跑至1450M生成。' if prefix else '執行前固定同一d450.state及輸入；')+'RNG記憶體與虛擬時鐘從相同狀態還原，沒有另設seed數字或重擲。',
        binary_sha256=sha((hd/'out/cok-hd-probe').read_bytes()),items=rows,all_pass=all(r['ok'] for r in rows))
    assert any(r['scale']==3 and r['display_difference']>0 for r in rows)
    if prefix:assert any(any(k.startswith(('dungcom','wildcom')) for k in r['source_matches']) and r['display_difference']>0 for r in rows if r['scale']==3)
    write(dest/filename,result);print(prefix+'runtime',len(rows),'all_pass',result['all_pass'],'changed3',sum(r['display_difference']>0 for r in rows if r['scale']==3))
    assert result['all_pass']

def battle_runtime(root):runtime(root,'battle-','battle-receipt.json')

def metadata(root):
    hd=root/'workplace/hd';total=[];call_count=0;all_jobs=[]
    runtime_seen=set()
    for name in ('receipt.json','battle-receipt.json'):
        p=hd/'out/issue13-runtime'/name
        if p.exists():
            for row in json.loads(p.read_text())['items']:
                for key in row['source_matches']:runtime_seen.update(key.split('/'))
    for batch in ('b07','b08','b09','b10','b11'):
        req=hd/'requests'/batch/'manifest.json';request=json.loads(req.read_text());out=hd/'out'/batch
        val=json.loads((out/'validation.json').read_text());validation={r['key']:r for r in val['items']};review={r['key']:r for r in json.loads((out/'visual-review.json').read_text())['items']}
        jobs=[]
        for p in sorted((out/'jobs').glob('*.json')):
            j=json.loads(p.read_text());assert sha((out/j['raw']).read_bytes())==j['raw_sha256'];jobs.append(dict(file=str(p.relative_to(out)),sha256=sha(p.read_bytes()),**j))
        call_count+=len(jobs);rows=[]
        all_jobs.extend(dict(batch=batch,**j) for j in jobs)
        for it in request['items']:
            key=it['key'];ph=sha((out/(key+'.png')).read_bytes());assert validation[key]['target_met'] and review[key]['approved'] and ph==validation[key]['png_sha256']==review[key]['png_sha256']
            related=[]
            for j in jobs:
                members=[j['key']] if j.get('key') else [v['key'] for v in json.loads((req.parent/(j['atlas']+'.json')).read_text())['items']]
                if key in members and (not j.get('only') or key in j['only']):related.append(j['file'])
            row=dict(key=key,sources=it['sources'],size=it['target_size'],request_runtime_evidence=it['runtime_evidence'],
                runtime_evidence='observed' if it['runtime_evidence']=='observed' or key in runtime_seen else 'unobserved',observed_in_issue13_runtime=key in runtime_seen,
                evidence=it.get('evidence',[]),png=key+'.png',png_sha256=ph,validation=validation[key],visual_review=review[key],
                source_indexed=it['indexed'],source_indexed_sha256=it['indexed_sha256'],generation_jobs=related,
                selected_job=related[-1] if related else None,reuse=it.get('reuse_candidate'),reuse_archive=it.get('reuse_archive'),
                rendering=it.get('rendering'),status='theme-ready; receipt tools use HDLive; standard frontend awaits integration patch')
            rows.append(row);total.append(dict(batch=batch,**row))
        write(out/'manifest.json',dict(batch=batch,issue=13,spec=request['spec'],scale=3,style='B',request_manifest_sha256=sha(req.read_bytes()),
            generation_calls=len(jobs),jobs=jobs,items=rows,artifacts={str(p.relative_to(out)):sha(p.read_bytes()) for p in sorted(out.glob('*.json')) if p.name!='manifest.json'},
            rights='AI重繪依專案定案可放入原版＋HD patch；原始DAX、畫面、state與原版文字均限本機。',
            limits='來源未觀測者保留unobserved；美術通過不代表取得其正常玩家觸發或原版driver語意。'))
    write(hd/'out/issue13-status.json',dict(issue=13,base_theme_items=97,new_theme_items=len(total),total_theme_items=97+len(total),
        standard_frontend_current_items=97,standard_frontend_integration='out/frontend-integration.patch，尚未套用',
        generated_derivatives=sum(bool(r['generation_jobs']) for r in total),reused=sum(bool(r['reuse']) for r in total),generation_calls=call_count,items=total,
        environment=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__)))
    write(hd/'out/issue13-generation-jobs.json',dict(issue=13,calls=call_count,items=all_jobs,
        note='本輪內建image_gen呼叫；原圖已複製各批out，前作沿用的原圖與提示詞另見各項reuse_archive。'))
    print('items',len(total),'calls',call_count,'reused',sum(bool(r['reuse']) for r in total))

def controls(root):
    from hd_battle_art import prepare,theme
    from hd_battle_inventory import source_records
    from dax_pics import picture,frames,EGA
    hd=root/'workplace/hd';orig=root/'workplace/input/dos/ckrynn';sources=0;items=0;cache={}
    selected={}
    for batch in ('b07','b08','b09','b10','b11'):
        req=hd/'requests'/batch;man=json.loads((req/'manifest.json').read_text())
        for it in man['items']:
            target=(req/it['indexed']).read_bytes()
            for src in it['sources']:
                name=src['file']
                if name not in cache:cache[name]=list(source_records(orig,name))
                bid,raw,origin=cache[name][src['record_index']];assert bid==src['block']
                for field in ('file_sha256','offset','packed_size','decoded_size','decoded_sha256'):assert origin[field]==src[field]
                try:pics=[picture(raw)]
                except ValueError:pics=[f[1:] for f in frames(raw)]
                w,h,count,x,y,px=pics[src['frame']];p=px[src['item']*w*h:(src['item']+1)*w*h]
                assert [w,h]==it['size'],(it['key'],src,[w,h],it['size'])
                mapped=bytes(it['palette_mapping'][c] for c in p);assert mapped==target
                if batch=='b11':assert p==(req/it['raw_indexed']).read_bytes()
                sources+=1
            png=np.array(Image.open(req/it['png']).convert('RGB'));assert np.array_equal(png,np.array(EGA,np.uint8)[np.frombuffer(target,np.uint8).reshape(h,w)])
            p=hd/'out'/batch/(it['key']+'.png');selected[str(p.relative_to(hd))]=sha(p.read_bytes());items+=1
    prepare(root)
    replay={name:sha((hd/name).read_bytes()) for name in selected};assert replay==selected
    negative=[]
    # 複製最小驗收輸入至自有暫存目錄；不修改正式核准紀錄。
    with tempfile.TemporaryDirectory(dir=hd/'out',prefix='issue13-gate-') as td:
        rr=Path(td);hh=rr/'workplace/hd';(hh/'theme').mkdir(parents=True)
        for batch in ('b07','b08','b09','b10','b11'):
            rq=hh/'requests'/batch;oo=hh/'out'/batch;rq.mkdir(parents=True);oo.mkdir(parents=True)
            for p in (hd/'requests'/batch).glob('*.idx'):shutil.copyfile(p,rq/p.name)
            shutil.copyfile(hd/'requests'/batch/'manifest.json',rq/'manifest.json')
            for p in (hd/'out'/batch).glob('*.png'):
                if not p.name.startswith(('atlas-','review-')) and not p.name.endswith('-raw.png'):shutil.copyfile(p,oo/p.name)
            for name in ('validation.json','visual-review.json'):shutil.copyfile(hd/'out'/batch/name,oo/name)
        theme(rr);baseline=hashes(hh/'theme')
        for kind in ('revoked-review','expired-review-sha','stale-request','expired-png'):
            p=hh/('requests/b07/manifest.json' if kind=='stale-request' else 'out/b07/visual-review.json');before=p.read_bytes();obj=json.loads(before);png=None;original_png=None
            if kind=='revoked-review':obj['items'][0]['approved']=False
            elif kind=='expired-review-sha':obj['items'][0]['png_sha256']='expired'
            elif kind=='stale-request':obj['items'][0]['sources'][0]['offset']+=1
            else:
                key=json.loads((hh/'requests/b07/manifest.json').read_text())['items'][0]['key'];png=hh/'out/b07'/(key+'.png');original_png=png.read_bytes();png.write_bytes(original_png+b'changed')
            write(p,obj)
            refused=False
            try:theme(rr)
            except AssertionError:refused=True
            assert refused and hashes(hh/'theme')==baseline,kind
            p.write_bytes(before)
            if png:png.write_bytes(original_png)
            negative.append(dict(case=kind,refused=True,theme_unchanged=True))
    write(hd/'out/issue13-controls.json',dict(source_items=items,source_aliases=sources,source_replay_exact=True,
        processed_png_replay_exact=True,processed_png_count=len(selected),negative_controls=negative,all_pass=True))
    print('source items',items,'aliases',sources,'PNG replay',len(selected),'negative',len(negative))

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('action',choices=['runtime','battle_runtime','metadata','controls']);p.add_argument('--root',required=True,type=Path);a=p.parse_args();globals()[a.action](a.root)
