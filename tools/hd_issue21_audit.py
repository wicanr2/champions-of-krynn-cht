#!/usr/bin/env python3
"""Issue21：交付索引、來源用途分類、閘門負對照與擁有權收據。Docker 限定。"""
import argparse
from collections import Counter
import importlib
import json
import os
from pathlib import Path
import platform
import shutil
import PIL
import numpy as np
from hd_battle_inventory import sha,write

ROOT=Path('/src');HD=ROOT/'workplace/hd';OUT=HD/'out';INV=HD/'inventory/issue21'
FILES=['docs/spec/010-hd-theme.md','tools/hd_issue21.py','tools/hd_issue21_art.py','tools/hd_issue21_walls.py',
 'tools/hd_issue21_receipts.py','tools/hd_issue21_audit.py','tools/hd_battle_receipts.py','tools/issue21-g5.sh',
 'tools/issue21-hd-go.sh','tools/issue21-hd-img.sh','tools/issue21-hd-source.sh','tools/issue21-hd-runtime.sh','tools/issue21-hd-receipt.sh',
 'workplace/dosgolem/apps/cok/hd_exact.go','workplace/dosgolem/apps/cok/hd_issue21_test.go',
 'workplace/dosgolem/apps/cok/actor_hd.go','workplace/dosgolem/apps/cok/effect_hd.go','workplace/dosgolem/apps/cok/hd_layers.go',
 'workplace/dosgolem/apps/cok/hd_live.go','workplace/dosgolem/apps/cok/wall_hd.go','workplace/dosgolem/cmd/cok-hd-probe/main.go']

def controls():
    art=importlib.import_module('hd_issue21_art');original=art.HD
    base=OUT/'issue21-controls';base.mkdir(exist_ok=True)
    def hashes(p):return {str(f.relative_to(p)):sha(f.read_bytes()) for f in p.rglob('*') if f.is_file()}
    original_hashes=hashes(HD/'theme');results=[]
    for kind in ('positive','revoked-review','stale-review','stale-request','stale-png'):
        dest=base/kind
        if dest.exists():shutil.rmtree(dest)
        (dest/'theme').mkdir(parents=True)
        for batch in ('b15','b16'):
            req=dest/'requests'/batch;out=dest/'out'/batch;req.mkdir(parents=True);out.mkdir(parents=True)
            man=json.loads((HD/'requests'/batch/'manifest.json').read_text())
            shutil.copyfile(HD/'requests'/batch/'manifest.json',req/'manifest.json')
            for it in man['items']:
                shutil.copyfile(HD/'requests'/batch/it['indexed'],req/it['indexed'])
                shutil.copyfile(HD/'out'/batch/(it['key']+'.png'),out/(it['key']+'.png'))
            for name in ('validation.json','visual-review.json'):shutil.copyfile(HD/'out'/batch/name,out/name)
        art.HD=dest;art.theme();before=hashes(dest/'theme')
        out=dest/'out/b16';req=dest/'requests/b16'
        if kind=='revoked-review':
            p=out/'visual-review.json';j=json.loads(p.read_text());j['items'][-1]['approved']=False;write(p,j)
        elif kind=='stale-review':
            p=out/'visual-review.json';j=json.loads(p.read_text());j['items'][-1]['png_sha256']='0'*64;write(p,j)
        elif kind=='stale-request':
            p=req/'manifest.json';j=json.loads(p.read_text());j['items'][-1]['description']+='x';write(p,j)
        elif kind=='stale-png':
            j=json.loads((req/'manifest.json').read_text());p=out/(j['items'][-1]['key']+'.png');p.write_bytes(p.read_bytes()+b'changed')
        rejected=False
        try:art.theme()
        except (AssertionError,ValueError):rejected=True
        good=(not rejected if kind=='positive' else rejected) and before==hashes(dest/'theme')
        results.append(dict(control=kind,rejected=rejected,theme_unchanged=before==hashes(dest/'theme'),ok=good))
    art.HD=original;assert hashes(HD/'theme')==original_hashes
    write(OUT/'issue21-controls.json',dict(items=results,all_pass=all(r['ok'] for r in results),original_theme_unchanged=True))
    assert all(r['ok'] for r in results);print('art gates',len(results),'all pass')

def audit():
    inputs=[]
    for p in sorted((ROOT/'workplace/input/dos/ckrynn').iterdir()):
        if not p.is_file():continue
        name=p.stem
        if p.suffix.upper()!='.DAX':kind='原版程式或設定，不改動'
        elif name.startswith(('ECL','GEO','WALLDEF','ITEM','MON')):kind='腳本、幾何、布局或資料記錄，非獨立圖片'
        elif name in ('MIDI','SOUND'):kind='音訊資料，不屬HD圖片'
        elif name=='CURSOR':kind='8×8指標保留原貌，完整命中見search.json'
        else:kind='已盤點圖源，按家族manifest核對'
        inputs.append(dict(file=p.name,bytes=p.stat().st_size,sha256=sha(p.read_bytes()),classification=kind))
    write(INV/'all-input-files.json',dict(items=inputs,
        nonpicture_record=dict(file='8X8D1.DAX',block=201,decoded_bytes=1416,status='未知；標準picture解碼拒絕，不猜作場景圖片',policy='維持原版；未接HD，未改字型或UI'),
        other_sources=dict(original_8x8_unique=1439,previous_wall_art=1348,new_wall_art=66,retained_symbols=24,fully_transparent=1),
        references=['curse_of_the_azure_bonds/docs/knowledge/gold-box-graphics.md','docs/spec/324-combat-terrain-atlases.md',
          'docs/spec/round-57-indexed-picture-and-walldef.md','docs/spec/round-64-dungeon-floor.md',
          'docs/spec/round-72-combat-sprite-extraction.md','docs/spec/round-82-bigpic-event.md','docs/knowledge/gold-box-state.md'],
        reference_grade='前作語意對本作皆為假說；本作像素來源、位置與呈現條件另由dosgolem核對。'))
    sources=json.loads((INV/'sources.json').read_text())
    current={r['file']:r['sha256'] for r in inputs};assert all(current[k]==v for k,v in sources['files'].items())
    old=json.loads((HD/'inventory/issue14/sources.json').read_text());assert all(current[k]==v for k,v in old['files'].items())
    coverage=json.loads((INV/'wall-coverage.json').read_text());used={r['key'] for r in coverage['coverage']}
    status=json.loads((OUT/'issue14-status.json').read_text());prior=[r['key'] for r in status['items'] if r.get('runtime_evidence')=='unobserved-source']
    assert len(prior)==1225 and all(k in used for k in prior)
    raw_by_key={r['key']:r['raw_sha256'] for r in coverage['coverage']}
    assert all(raw_by_key[r['key']]==r['source_indexed_sha256'] for r in status['items'] if r['key'] in prior)
    rows=[]
    for name in FILES:
        p=ROOT/name;rows.append(dict(file=name,sha256=sha(p.read_bytes()),uid=p.stat().st_uid,gid=p.stat().st_gid))
    write(OUT/'issue21-code-files.json',dict(items=rows))
    arts=[]
    for batch in ('b15','b16','b17'):
        j=json.loads((OUT/batch/'manifest.json').read_text());v=json.loads((OUT/batch/'validation.json').read_text());assert v['all_pass']
        arts.append(dict(batch=batch,count=len(j['items']),generation_calls=len(j['jobs']),structural_min=min(r['structural_score'] for r in v['items']),hue_median_max=max(r['hue_median_degrees'] for r in v['items'])))
    frames=[json.loads(s) for s in (OUT/'issue21-frame-receipts.jsonl').read_text().splitlines()];assert len(frames)==389 and all(r['ok'] for r in frames)
    runtime=json.loads((OUT/'issue21-runtime/receipt.json').read_text());g5=json.loads((OUT/'issue21-g5.json').read_text());assert runtime['all_pass'] and g5['all_pass']
    tests=(OUT/'issue21-go-tests.log').read_text();assert '\nPASS\n' in tests and '\nFAIL' not in tests
    write(OUT/'issue21-status.json',dict(issue=21,date='2026-10-20',art=arts,new_art_total=sum(a['count'] for a in arts),total_theme_items=1958,
        frames=dict(count=len(frames),all_pass=True),runtime=dict(count=len(runtime['items']),all_pass=True),g5=g5,
        wall_templates_checked=9079,new_wall_cells=coverage['new_cells'],ambiguous_new_cells=len(coverage['ambiguities']),
        old_unobserved_sources_covered=len(prior),
        original_input_hashes_unchanged=True,unit_tests='PASS',legacy_private_tests_skipped=[l for l in tests.splitlines() if l.startswith('--- SKIP:')],
        binary_sha256={p.name:sha(p.read_bytes()) for p in (OUT/'issue21-cok-hd-probe',OUT/'issue21-cok-hd-frame-check')},
        tools=dict(go=(OUT/'issue21-go-version.txt').read_text().strip(),python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__),
        image_ids=dict(img='sha256:3c3a80b84cf2487c3fcbcf2ee629de72a6e40351ff1f950036d3590dfa96aa3f',go='sha256:8ed6bdee2bce1f32f804e888f942b25bbf62f2165a43344bb4587f3371de0581'),
        git_heads=dict(main='6ac6e1689d1d6d9578cf84061ecb49511699d8ad',dosgolem='3a757f193de5869ad32f14d5af24fb171b1dd917'),
        limits='未提交差異由code-files逐項SHA固定；狀態續跑不是新冷開機玩家流程。'))
    print('audit: art',sum(a['count'] for a in arts),'frames',len(frames),'runtime',len(runtime['items']),'G5 all pass')

def wall_regression():
    from hd_wall_receipts import WallOracle
    from hd_wall_inventory import saved_screens
    old=OUT/'issue21-old-theme';old.mkdir(exist_ok=True)
    shutil.copyfile(OUT/'issue21-walls-before.json',old/'walls.json')
    if not (old/'walls').exists():(old/'walls').symlink_to('../../theme/walls',target_is_directory=True)
    before=WallOracle(old);after=WallOracle(HD/'theme');rows=[]
    for name,raw in saved_screens(ROOT):
        a,ah=before.apply(raw);b,bh=after.apply(raw)
        if a is not None:assert b is not None and np.array_equal(a,b),name
        rows.append(dict(screen=name,before_active=a is not None,after_active=b is not None,old_known_pixels_unchanged=True))
    write(OUT/'issue21-old-wall-regression.json',dict(items=rows,screens=len(rows),old_known=sum(r['before_active'] for r in rows),all_pass=True))
    print('old wall regression',len(rows),'known',sum(r['before_active'] for r in rows),'all pass')

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['controls','audit','wall_regression']);a=ap.parse_args();globals()[a.action]()
