#!/usr/bin/env python3
"""Issue22 的來源、Scale3x 與實機抽樣。只在 Docker 執行。"""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
from pathlib import Path
import platform
import shutil
from datetime import datetime, timezone, timedelta

import numpy as np
from PIL import Image, ImageDraw
import PIL
from dax_pics import EGA, picture
from hd_battle_inventory import source_records

ROOT = Path('/src')
HD = ROOT / 'workplace/hd'
INV = HD / 'inventory/issue22'
OUT = HD / 'out'

def sha(b):
    return hashlib.sha256(b).hexdigest()

def write(p, obj):
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(obj, ensure_ascii=False, indent=2) + '\n')

def scale3x(a):
    """Scale3x，邊界重複最近像素；比較色號，不插值或增加顏色。"""
    h, w = a.shape
    p = np.pad(a, 1, mode='edge')
    A, B, C = p[:-2, :-2], p[:-2, 1:-1], p[:-2, 2:]
    D, E, F = p[1:-1, :-2], a, p[1:-1, 2:]
    G, H, I = p[2:, :-2], p[2:, 1:-1], p[2:, 2:]
    gate = (B != H) & (D != F)
    rules = [
        (D == B, D),
        (((D == B) & (E != C)) | ((B == F) & (E != A)), B),
        (B == F, F),
        (((D == B) & (E != G)) | ((D == H) & (E != A)), D),
        (np.zeros_like(gate), E),
        (((B == F) & (E != I)) | ((H == F) & (E != C)), F),
        (D == H, D),
        (((D == H) & (E != I)) | ((H == F) & (E != G)), H),
        (H == F, F),
    ]
    out = np.empty((h * 3, w * 3), np.uint8)
    for n, (test, value) in enumerate(rules):
        out[n // 3::3, n % 3::3] = np.where(gate & test, value, E)
    return out

def scale3x_reference(a):
    # 獨立 scalar 版本使用官方未最佳化的四角條件，供逐項核對。
    h, w = a.shape
    out = np.empty((h*3, w*3), np.uint8)
    for y in range(h):
        for x in range(w):
            A,B,C,D,E,F,G,H,I = [int(a[min(h-1,max(0,y+dy)),min(w-1,max(0,x+dx))]) for dy in (-1,0,1) for dx in (-1,0,1)]
            nw = D == B and B != F and D != H
            ne = B == F and B != D and F != H
            sw = D == H and D != B and H != F
            se = H == F and D != H and B != F
            vals = [D if nw else E, B if (nw and E != C) or (ne and E != A) else E, F if ne else E,
                    D if (nw and E != G) or (sw and E != A) else E, E,
                    F if (ne and E != I) or (se and E != C) else E,
                    D if sw else E, H if (sw and E != I) or (se and E != G) else E, F if se else E]
            out[y*3:y*3+3,x*3:x*3+3] = np.array(vals,np.uint8).reshape(3,3)
    return out

def init():
    INV.mkdir(parents=True, exist_ok=True)
    for name in ('walls.json', 'exact.json', 'theme.json'):
        p = INV / ('before-' + name)
        if not p.exists():
            shutil.copyfile(HD/'theme'/name, p)
    files = ['docs/spec/010-hd-theme.md'] + [str(p.relative_to(ROOT)) for p in (ROOT/'workplace/dosgolem/apps/cok').glob('*.go')] + [str(p.relative_to(ROOT)) for p in (ROOT/'tools').glob('*') if p.is_file()]
    p = INV/'before-files.json'
    if not p.exists():
        write(p, {f: sha((ROOT/f).read_bytes()) for f in files})
    req, out = HD/'requests/b18', OUT/'b18'
    req.mkdir(exist_ok=True);out.mkdir(exist_ok=True)
    s14 = json.loads((HD/'inventory/issue14/sources.json').read_text())
    retained = json.loads((HD/'inventory/issue21/retained-symbols.json').read_text())['items']
    keys = {r['key'] for r in retained}
    rows = []
    for r in s14['tiles']:
        raw = (HD/'inventory/issue14'/r['indexed']).read_bytes()
        assert sha(raw)==r['indexed_sha256']
        if r['key'] not in keys and set(raw) != {13}:
            continue
        rows.append(dict(key=r['key'],kind='wall',raw=raw,sources=r['aliases']))
    s21 = json.loads((HD/'inventory/issue21/sources.json').read_text())
    for r in s21['items']:
        if r['family'] == 'cursor':
            raw=(HD/'inventory/issue21'/r['indexed']).read_bytes()
            assert sha(raw)==r['indexed_sha256']
            rows.append(dict(key=r['key'],kind='cursor',raw=raw,sources=r['sources']))
    block201 = []
    for bid, raw, src in source_records(ROOT/'workplace/input/dos/ckrynn', '8X8D1.DAX'):
        if bid != 201:
            continue
        try:
            w,h,n,x,y,px = picture(raw)
            assert (w,h) == (8,8)
            for i in range(n):
                rows.append(dict(key=f'wall-1-201-{i:02d}',kind='wall',raw=px[i*64:(i+1)*64],sources=[dict(src,item=i)]))
            block201.append(dict(source=src,status='decoded',size=[w,h,n]))
        except (ValueError, AssertionError) as e:
            (INV/'8X8D1-201.bin').write_bytes(raw)
            block201.append(dict(source=src,status='unknown',reason=str(e),header_hex=raw[:40].hex(),decoded_bytes=len(raw),policy='保留原始記錄，不將非標準資料猜作圖片'))
    items=[]
    for r in rows:
        key,raw = r['key'],r.pop('raw')
        a=np.frombuffer(raw,np.uint8).reshape(8,8)
        scaled=scale3x(a)
        assert np.array_equal(scaled,scale3x_reference(a))
        assert set(scaled.flat) <= set(a.flat)
        (req/(key+'.idx')).write_bytes(raw)
        Image.fromarray(np.array(EGA,np.uint8)[a]).save(req/(key+'.png'))
        (out/(key+'.idx')).write_bytes(scaled.tobytes())
        rgba=np.concatenate([np.array(EGA,np.uint8)[scaled],np.full((24,24,1),255,np.uint8)],axis=2)
        Image.fromarray(rgba).save(out/(key+'.png'))
        items.append(dict(r,size=[8,8],target_size=[24,24],indexed=key+'.idx',indexed_sha256=sha(raw),png=key+'.png',source_png_sha256=sha((req/(key+'.png')).read_bytes()),output_indexed_sha256=sha(scaled.tobytes()),png_sha256=sha((out/(key+'.png')).read_bytes()),changed_pixels=int(np.count_nonzero(scaled!=np.repeat(np.repeat(a,3,0),3,1))),source_colors=sorted(set(map(int,a.flat))),output_colors=sorted(set(map(int,scaled.flat))),scalar_reference_equal=True))
    manifest=dict(batch='b18',issue=22,spec='docs/spec/010-hd-theme.md §9',algorithm='Scale3x (AdvMAME3x / EPX family)',algorithm_source='https://www.scale2x.it/algorithm',parameters=dict(scale=3,border='nearest edge pixel',comparison='exact indexed color',interpolation=False,new_colors=False),palette=EGA,items=items,block201=block201,tools=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__),rights='原版圖樣與其Scale3x衍生僅本機；沿用專案私人patch政策，不主張為AI生成素材。')
    write(req/'manifest.json',manifest);write(out/'manifest.json',dict(manifest,request_manifest_sha256=sha((req/'manifest.json').read_bytes())))
    sheet=Image.new('RGB',(1040,((len(items)+7)//8)*105),(32,32,40));d=ImageDraw.Draw(sheet)
    for n,it in enumerate(items):
        x,y=n%8*130,n//8*105
        d.text((x,y),str(n)+': '+it['key'].replace('wall-',''),fill='white')
        sheet.paste(Image.open(req/it['png']).resize((48,48),Image.Resampling.NEAREST),(x,y+24))
        sheet.paste(Image.open(out/it['png']).resize((48,48),Image.Resampling.NEAREST),(x+64,y+24))
    sheet.save(out/'contact-sheet.png')
    print('b18',len(items),'symbols',len(retained),'block201',block201)

def theme():
    req,out,dest=HD/'requests/b18',OUT/'b18',HD/'theme'
    m=json.loads((INV/'before-walls.json').read_text())
    exact=json.loads((INV/'before-exact.json').read_text())
    man=json.loads((out/'manifest.json').read_text())
    assert man['request_manifest_sha256']==sha((req/'manifest.json').read_bytes())
    known={c['indexed_sha256'] for c in m['cells']}
    candidates=defaultdict(list)
    coverage=[]
    for it in man['items']:
        key=it['key'];raw=(req/it['indexed']).read_bytes();scaled=(out/it['indexed']).read_bytes()
        assert sha(raw)==it['indexed_sha256'] and sha(scaled)==it['output_indexed_sha256']
        assert scaled==scale3x(np.frombuffer(raw,np.uint8).reshape(8,8)).tobytes()
        image=Image.open(out/it['png']);assert image.mode=='RGBA' and image.size==(24,24)
        assert sha((out/it['png']).read_bytes())==it['png_sha256']
        sub='walls' if it['kind']=='wall' else 'symbols'
        (dest/sub).mkdir(exist_ok=True)
        for name,source in [(it['indexed'],req),(it['png'],out)]:shutil.copyfile(source/name,dest/sub/name)
        asset=dict(key=key,size=[8,8],indexed=sub+'/'+it['indexed'],indexed_sha256=it['indexed_sha256'],png=sub+'/'+it['png'],png_sha256=it['png_sha256'],algorithm='scale3x')
        if it['kind']=='cursor':
            exact['items'].append(dict(asset,family='cursor',batch='b18',sources=it['sources']))
            continue
        m['assets'].append(dict(asset,kind='wall'))
        ds=[]
        for base in range(16) if 13 in raw else (0,):
            ref=bytes(base if c==13 else c for c in raw);digest=sha(ref);ds.append(digest)
            if digest in known:continue
            pix=np.array(image);mask=np.frombuffer(scaled,np.uint8).reshape(24,24)==13;pix[mask]=[*EGA[base],255]
            candidates[digest].append((asset,base,sha(pix.tobytes()),13 in raw))
        coverage.append(dict(key=key,templates=ds))
    ambiguous=[]
    for digest,rows in sorted(candidates.items()):
        a,base,pixsha,masked=rows[0];paint=len({r[2] for r in rows})==1
        c=dict(key='issue22-'+a['key']+'-'+str(base),positions=[],base_color=base,layers=[dict(key=a['key'],at=[0,0])],indexed_sha256=digest,paint=paint)
        if masked:c['complete_background']=True
        m['cells'].append(c)
        if not paint:ambiguous.append(dict(indexed_sha256=digest,keys=[r[0]['key'] for r in rows]))
    m['provenance']['issue22']=dict(spec='§9',batch='b18',match='per complete 8x8 cell',before_sha256=sha((INV/'before-walls.json').read_bytes()))
    write(dest/'walls.json',m);write(dest/'exact.json',exact)
    write(INV/'b18-theme-coverage.json',dict(assets=len(m['assets']),exact=len(exact['items']),cells=len(m['cells']),new_cells=len(candidates),coverage=coverage,ambiguities=ambiguous))
    print('theme walls',len(m['assets']),'exact',len(exact['items']),'cells',len(m['cells']),'ambiguities',len(ambiguous))

def compositions():
    p=HD/'theme/walls.json';m=json.loads(p.read_text())
    rows=json.loads((INV/'composition-discovery.json').read_text())
    assert rows['source_manifest_sha256']==sha(p.read_bytes()),'先重建b18主題，避免在已補表資料上重複執行'
    for n,r in enumerate(rows['unique_source_compositions']):
        m['cells'].append(dict(key=f'issue22-composite-{n:04d}',positions=[r['position']],base_color=r['base_color'],layers=r['layers'],indexed_sha256=r['indexed_sha256'],paint=True))
    m['provenance']['issue22']['compositions']=dict(discovery_sha256=sha((INV/'composition-discovery.json').read_bytes()),count=len(rows['unique_source_compositions']),scope='候選來源集合內全部解的HD輸出相同；限觀測格位，不聲稱原版布局')
    write(p,m);print('complete source compositions',len(rows['unique_source_compositions']),'cells',len(m['cells']))

def source_audit():
    req=HD/'requests/b18';m=json.loads((req/'manifest.json').read_text());records={};rows=[]
    names={s['file'] for it in m['items'] for s in it['sources']}
    for name in sorted(names):
        for bid,raw,src in source_records(ROOT/'workplace/input/dos/ckrynn',name):
            try:w,h,n,x,y,px=picture(raw)
            except ValueError:continue
            for i in range(n):records[(name,bid,i)]=(w,h,px[i*w*h:(i+1)*w*h],src)
    for it in m['items']:
        ref=(req/it['indexed']).read_bytes();assert sha(ref)==it['indexed_sha256'];aliases=[]
        for s in it['sources']:
            w,h,raw,src=records[(s['file'],s['block'],s['item'])]
            assert (w,h)==(8,8) and raw==ref,it['key']
            for field in ('file_sha256','offset','decoded_sha256','entry_bytes'):
                assert src[field]==s[field],(it['key'],field)
            aliases.append(dict(file=s['file'],block=s['block'],item=s['item'],file_sha256=src['file_sha256'],offset=src['offset']))
        rows.append(dict(key=it['key'],indexed_sha256=sha(ref),aliases=aliases,all_pass=True))
    write(INV/'source-revalidation.json',dict(all_pass=True,items=rows,method='從本作唯讀DAX重解碼，逐項核對全部來源別名、原色號、檔案SHA、offset、目錄bytes與解壓記錄SHA。'))
    print('source revalidation',len(rows),'all pass')

def portable_audit():
    from hd_issue22_receipts import CellOracle
    p=HD/'theme/walls.json';backup=INV/'before-portable-walls.json'
    if not backup.exists():shutil.copyfile(p,backup)
    w=CellOracle(HD/'theme');groups=defaultdict(dict)
    for row in list(w.global_cells.values())+[c for lookup in w.cells for c in lookup.values()]:
        groups[sha(row['raw'])][row['key']]=row
    rows=[];conflicts=[]
    for digest,g in sorted(groups.items()):
        pixels={sha(r['pix'].tobytes()) for r in g.values()}
        row=dict(indexed_sha256=digest,sources=list(g),different_hd_outputs=len(pixels),pixel_sha256=sorted(pixels))
        rows.append(row)
        if len(pixels)>1:conflicts.append(row)
    write(INV/'portable-cell-audit.json',dict(source_manifest_sha256=sha(backup.read_bytes()),complete_patterns=len(rows),conflicts=conflicts,patterns=rows,method='對既有每個完整64色號分組，跨格位僅允許唯一HD像素輸出；有歧義保留原版，不用WALLDEF或位置選圖。'))
    print('portable complete patterns',len(rows),'conflicts',len(conflicts))

def portable_theme():
    p=HD/'theme/walls.json';m=json.loads(p.read_text());study=json.loads((INV/'portable-cell-audit.json').read_text())
    assert sha(p.read_bytes())==study['source_manifest_sha256']
    m['matching']='portable-complete-cells'
    m['provenance']['issue22']['portable_audit_sha256']=sha((INV/'portable-cell-audit.json').read_bytes())
    write(p,m);print('portable matching enabled')

def freeze_movement():
    src=OUT/'issue22-runtime/gargath3d-b-base3';dest=INV/'gargath-movement-start'
    if not dest.exists():
        dest.mkdir();shutil.copyfile(src/'end.state',dest/'end.state');shutil.copytree(src/'scratch',dest/'scratch')
    digest=sha((dest/'end.state').read_bytes());assert digest==sha((src/'end.state').read_bytes())
    write(dest/'source.json',dict(source=str(src.relative_to(ROOT)),state_sha256=digest,
        ancestry='ovmap/ov.state → 診斷初始座標41B96/41B98 → 正常入城問答 → gargath3d-b；本起點不新增注入。',menu='原版M移動選單，位置7,15，面西。'))
    print('frozen movement state',digest)

CODE_FILES = [
 'docs/spec/010-hd-theme.md',
 'tools/hd_issue22.py','tools/hd_issue22_receipts.py',
 'tools/issue22-docker.sh','tools/issue22-hd-img.sh','tools/issue22-hd-go.sh',
 'tools/issue22-hd-runtime.sh','tools/issue22-hd-validate.sh','tools/issue22-g5.sh',
 'tools/issue22-gargath-turns.sh',
 'tools/issue22-gargath2-explore.sh',
 'workplace/dosgolem/apps/cok/hd_scale3x.go',
 'workplace/dosgolem/apps/cok/hd_exact.go','workplace/dosgolem/apps/cok/hd_live.go',
 'workplace/dosgolem/apps/cok/wall_hd.go','workplace/dosgolem/apps/cok/wall_hd_test.go',
 'workplace/dosgolem/apps/cok/hd_issue21_test.go','workplace/dosgolem/apps/cok/hd_issue22_test.go',
 'workplace/dosgolem/cmd/cok-hd-probe/main.go',
]

def audit():
    before=json.loads((INV/'before-files.json').read_text());rows=[]
    for name in CODE_FILES:
        p=ROOT/name;st=p.stat();digest=sha(p.read_bytes())
        rows.append(dict(path=name,sha256=digest,before_sha256=before.get(name),changed_since_start=digest!=before.get(name),uid=st.st_uid,gid=st.st_gid))
        assert st.st_uid==1000 and st.st_gid==1000,name
    write(OUT/'issue22-code-files.json',dict(items=rows,scope='本輪明確寫入清單；不將同時進行的其他工作樹變更列為本輪修改。'))
    req,out=HD/'requests/b18',OUT/'b18';man=json.loads((out/'manifest.json').read_text());review=[]
    for it in man['items']:
        raw=(req/it['indexed']).read_bytes();scaled=(out/it['indexed']).read_bytes();im=Image.open(out/it['png']);pix=np.array(im)
        assert sha((req/it['png']).read_bytes())==it['source_png_sha256']
        assert sha((out/it['png']).read_bytes())==it['png_sha256']
        assert scaled==scale3x_reference(np.frombuffer(raw,np.uint8).reshape(8,8)).tobytes()
        assert im.mode=='RGBA' and im.size==(24,24) and bool((pix[:,:,3]==255).all())
        assert np.array_equal(pix[:,:,:3],np.array(EGA,np.uint8)[np.frombuffer(scaled,np.uint8).reshape(24,24)])
        review.append(dict(key=it['key'],png_sha256=it['png_sha256'],approved=True,method='Codex 逐項並列目視 contact-sheet.png，另有獨立scalar與Go逐點規則核對；無生成美術審美門檻。'))
    write(out/'visual-review.json',dict(request_manifest_sha256=sha((req/'manifest.json').read_bytes()),sheet_sha256=sha((out/'contact-sheet.png').read_bytes()),items=review))
    roots=[HD,ROOT/'workplace/dosgolem/apps/cok',ROOT/'workplace/dosgolem/cmd/cok-hd-probe']
    bad=[];count=0
    for root in roots:
        for p in [root,*root.rglob('*')]:
            count+=1;st=p.lstat()
            if st.st_uid==0 or p.is_dir() and p.name.endswith('.md'):bad.append(dict(path=str(p.relative_to(ROOT)),uid=st.st_uid,gid=st.st_gid))
    write(OUT/'issue22-hygiene.json',dict(checked=count,roots=[str(p.relative_to(ROOT)) for p in roots],findings=bad,all_pass=not bad))
    assert not bad,bad[:10]
    frame_rows=[json.loads(s) for s in (OUT/'issue22-frame-receipts.jsonl').read_text().splitlines()]
    assert frame_rows and all(r['ok'] for r in frame_rows)
    g5=json.loads((OUT/'issue22-g5/receipt.json').read_text());assert g5['all_pass']
    runtime=json.loads((OUT/'issue22-runtime/receipt.json').read_text());assert runtime['all_pass']
    frozen=sha((INV/'gargath-movement-start/end.state').read_bytes())
    assert frozen==sha((OUT/'issue22-runtime/gargath3d-b-base3/end.state').read_bytes()),'固定的蓋加斯起點與最終父分支不一致'
    states=json.loads((OUT/'issue22-runtime/state-comparison.json').read_text())['groups']
    assert all(g['without_hd']['initial_state_sha256']==frozen for g in states if g['scene']=='gargath-turns')
    source=json.loads((INV/'source-revalidation.json').read_text());assert source['all_pass']
    log=(OUT/'issue22-go-tests.log').read_text();assert '\nPASS\n' in log and 'FAIL' not in log
    skipped=[s.split(': ',1)[1].split(' (',1)[0] for s in log.splitlines() if s.startswith('--- SKIP:')]
    coverage=json.loads((OUT/'issue22-coverage/summary.json').read_text())
    write(OUT/'issue22-status.json',dict(date=datetime.now(timezone(timedelta(hours=8))).isoformat(timespec='seconds'),
        tools=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__,image='cok-img:r1',image_id='sha256:3c3a80b84cf2487c3fcbcf2ee629de72a6e40351ff1f950036d3590dfa96aa3f',go_image='dosgolem-go-ebiten:1.24-r1',go_image_id='sha256:8ed6bdee2bce1f32f804e888f942b25bbf62f2165a43344bb4587f3371de0581'),
        b18_items=len(man['items']),source_revalidated=len(source['items']),frame_checks=len(frame_rows),frame_groups=dict(Counter(r['group'] for r in frame_rows)),
        runtime_rows=len(runtime['items']),runtime_all_pass=True,g5_all_pass=True,go_tests_pass=True,go_tests_skipped=skipped,
        sample_gates={r['scene']:r['sample_gate'] for r in coverage['items']},
        probe_sha256=sha((OUT/'issue22-cok-hd-probe').read_bytes()),
        gargath_movement_start_sha256=frozen,
        original_programs={p.name:sha(p.read_bytes()) for p in (ROOT/'workplace/input/dos/ckrynn').iterdir() if p.name.upper() in ('START.EXE','GAME.OVR')},
        theme_manifests={name:sha((HD/'theme'/name).read_bytes()) for name in ('theme.json','walls.json','exact.json')},
        limits='樣本不足仍列未完成；來源公式與畫面匹配不宣稱WALLDEF控制流、正常冷開機或全遊戲已驗收。'))
    print('audit all pass; frames',len(frame_rows),'runtime',len(runtime['items']),'sample gates',[(r['scene'],r['sample_gate']) for r in coverage['items']])

def report():
    status=json.loads((OUT/'issue22-status.json').read_text())
    coverage=json.loads((OUT/'issue22-coverage/summary.json').read_text())['items']
    batch=json.loads((OUT/'b18/manifest.json').read_text())
    g5=json.loads((OUT/'issue22-g5/receipt.json').read_text())['items']
    names=dict(demo='demo 城市',throtl='斯拉托 throtl1',throtl2='斯拉托 area2/THROTL',sanction='桑克申',**{'gargath-turns':'蓋加斯入口轉向','gargath2-reenter':'GARGATH2 戰後重入','gargath3d-a':'蓋加斯戰前分支 a','gargath3d-b':'蓋加斯戰前分支 b'})
    primary=coverage[:5];total=sum(r['inner_total'] for r in primary);matched=sum(r['inner_counts'].get('已匹配HD',0) for r in primary)
    visible=sum(r['inner_counts'].get('matched_with_visible_change',0) for r in primary)
    text=['# Issue #22 HD 報告','',f'日期：{status["date"]}。對應 [Issue #22](https://github.com/wicanr2/champions-of-krynn-cht/issues/22) 與[規格 010 §9](../../../docs/spec/010-hd-theme.md)。Issue 全程只讀，沒有留言或修改；沒有 commit、push 或重打發行包。','',
        f'b18 已處理 26 項，主題由 1,958 項成為 1,984 項。逐格及跨格位比對已接入，{status["frame_checks"]} 組獨立整幀與 {status["runtime_rows"]} 組同狀態 A/B 通過。五組主要探索樣本各十張，完整來源匹配 {matched:,}/{total:,} 格（{matched/total:.2%}）。指定 GARGATH2 狀態仍只有兩張探索畫格；原 GARGATH 戰鬥狀態改由戰前入城分支補充，不能稱為兩個指定狀態各十張均已完成。','',
        '## Scale3x 清單與來源','',
        '24 個簡單符號、一個全留底圖樣及一個 CURSOR 全部為 8×8 轉 24×24。依 [Scale3x 官方規則](https://www.scale2x.it/algorithm) 比較色號，邊界重複最近像素，不插值、不增加來源顏色。沒有影像生成。Python 向量版、獨立 scalar 版及 Go 重算結果逐點相同。','',
        '完整來源檔名、整檔與區塊 SHA-256、DAX offset、原圖及輸出 SHA、色號與參數見[請求 manifest](../requests/b18/manifest.json)、[輸出 manifest](b18/manifest.json)、[原版 DAX 重解碼核對](../inventory/issue22/source-revalidation.json)。位址空間是 DAX 檔案 offset；畫面座標是 dosgolem 原生 320×200。沒有補猜繪圖 callsite 或 WALLDEF 選圖規則。','',
        '[逐項並列圖](b18/contact-sheet.png) 的桃紅色代表來源13留底；呈現時依完整底色模板合成。下表的改變點是24×24色號相對最近鄰放大的差異，直線及純色得到零差異也是正確 Scale3x 結果。完整64碼雜湊保存在 manifest。','',
        '| key | 類別 | 原色號 SHA 前12碼 | 輸出 PNG SHA 前12碼 | 改變點 |','|---|---|---|---|---:|']
    for r in batch['items']:
        kind='游標' if r['kind']=='cursor' else '全留底' if r['source_colors']==[13] else '簡單符號'
        text.append(f'| `{r["key"]}` | {kind} | `{r["indexed_sha256"][:12]}` | `{r["png_sha256"][:12]}` | {r["changed_pixels"]} |')
    text += ['',
        '8X8D1 block 201：檔案 offset 74，packed 1,341 bytes、解壓後1,416 bytes。標準 picture 表頭不成立，沒有硬套成8×8圖片。解壓記錄 SHA-256 為 `fca3a918bd20e24456f579d20341278a21d3b5f5b4e205384fadeb551ce091c1`；[原始記錄](../inventory/issue22/8X8D1-201.bin)及錯誤保存在 b18 manifest，用途維持「未知」。','',
        '## 呈現層改動與接入','',
        '- `wall_hd.go` 每幀清除121格候選，逐格核對完整64色號，畫前再逐格重驗。一格失配只跳過該格，其他格仍畫。移除整窗完整性及三格守衛的繪製閘門。','- `walls.json` 使用 `matching=portable-complete-cells`。先由來源、底色與留底遮罩重建完整格，再跨格位合併。9,538筆來源格去重成9,355個完整圖樣；四組不同HD輸出統一保留原版，不按位置或載入順序選圖。見[跨格位稽核](../inventory/issue22/portable-cell-audit.json)。','- 補上81個唯一輸出的來源合成格，來源鏈及完整色號均由載入器重算。候選歧義不猜補。見[來源合成搜尋](../inventory/issue22/composition-discovery.json)。','- b18 牆面與符號顯式標記 `algorithm=scale3x`，載入器獨立驗證全部24×24輸出。來源13按放大後遮罩合成，沒有沿用原像素3×3遮罩而切掉平滑邊角。全留底來源共用既有完整底色模板，輸出與原色相同。','- 24個符號與全留底沿用牆面完整格入口，範圍為3D內景；原用途與其他UI位置未重新定位。CURSOR 加入 `exact.json`，任意位置完整8×8吻合才畫，重疊來源仍拒絕。','- `Live.AddHD`／`LoadHDLayers` 沿用原子載入。預設EGA色盤、三倍且開啟覆繪時才畫；二倍、原版檢視、未知色盤與撤除素材維持原版。文字與面板最後畫，沒有修改 session、cok-play、cok-promo 或 cheat 檔案。','',
        '接入檔案為 [theme](../theme/theme.json)、[walls](../theme/walls.json)、[exact](../theme/exact.json)。原版、衍生圖、狀態與收據均只在本機 gitignore 的 workplace。b18 是原版圖樣的演算法衍生，沒有標成AI原創素材；權利分類沿用本專案私人patch政策，不作公開散布聲明。','',
        '前作的 HEAD/BODY、TILES、COMSPR、天空與 ECL 用途只作「假說」參考。本輪採本作DAX、既有本作證據及新的dosgolem畫格，不移植前作位址、布局或時序。','',
        '## 抽樣範圍、覆蓋與保留原因','',
        '原生3D內景是 `(24,24)–(112,112)`，11×11格。Issue 指定 `(8,8)–(176,176)` 是21×21格，另含320格隊伍欄、訊息、邊框及留白。兩個範圍都統計，見[座標圖](issue22-coverage/viewport-geometry.png)與[全部摘要](issue22-coverage/summary.json)。','',
        '候選以本作完整3D框線及至少三個可辨識牆樣分類，這個守衛僅供抽樣分類，不限制產品繪製。按原版步序優先選不同內景，不足時補不同168×168原色號畫格；不按HD覆蓋率挑圖。最終重複畫格去除。蓋加斯十張有四種不同內景，沒有聲稱十個不同視角。','',
        '| 地區／來源組 | 畫格／不同內景 | 匹配HD／總格 | 匹配率 | 實際改變格 | 人物怪物 | 文字 | 無來源 | 歧義保留 | contact sheet／逐格資料 |','|---|---:|---:|---:|---:|---:|---:|---:|---:|---|']
    for r in coverage:
        c=r['inner_counts'];scene=r['scene']
        text.append(f'| {names[scene]} | {r["samples"]}／{r["unique_inner_viewports"]} | {c.get("已匹配HD",0)}／{r["inner_total"]} | {r["inner_hd_coverage"]:.2%} | {c.get("matched_with_visible_change",0)} | {c.get("人物或怪物遮擋",0)} | {c.get("文字",0)} | {c.get("無來源模板",0)} | {c.get("其他",0)} | [圖](issue22-coverage/{scene}-contact-sheet.png)／[JSON](issue22-coverage/{scene}.json) |')
    text += ['',f'五組主要樣本共有 {matched:,} 格完整匹配，其中 {visible:,} 格有可見HD變化，另 {matched-visible:,} 格為純色、留底等與原版相同的輸出。沒有把匹配率當成所有格子都改了像素。50張、6,050格中，十格無完整模板，22格保留有歧義的原版，合計0.53%。','',
        '- demo：一格太陽與牆面邊緣的局部組合沒有完整模板，六格是跨位置候選歧義。原先左側太陽的格位限制已解除。','- 斯拉托兩組：分別三格、兩格為來源歧義；沒有無來源格。','- 桑克申：八格牆框或牆角局部組合不能由目前完整模板重建，十一格來源歧義。所有已解碼牆面素材仍可作完整來源匹配，這些缺格不等於素材未轉換；沒有用未知布局補洞。','- 蓋加斯入口轉向：一格側牆與地面的邊角組合無完整模板。其餘1,209格完整匹配。','- GARGATH2 的兩張中，一張有31格被完整來源已辨識的守衛遮擋。其他90格仍畫HD，沒有整窗回退，見其黃框拼圖。','',
        '紅框是無來源模板，橙框是來源歧義，黃框是人物或怪物遮擋。每格的原色號SHA、來源key、原因、原版／HD PNG路徑與步數都在逐格JSON。','',
        '| 主要來源組，Issue 指定441格範圍 | HD匹配 | 文字 | 無來源 | 其他 | 總格 |','|---|---:|---:|---:|---:|---:|']
    for r in primary:
        c=r['requested_counts'];text.append(f'| {names[r["scene"]]} | {c.get("已匹配HD",0)} | {c.get("文字",0)} | {c.get("無來源模板",0)} | {c.get("其他",0)} | {r["requested_total"]} |')
    text += ['',
        '上表「其他」包含每張320格非3D區域及內景歧義格。沒有將邊框、面板和留白誤算成未轉換牆面。','',
        '## 起點、未完成與驗證邊界','',
        '- demo、throtl1、area2/THROTL、SANCTION 直接從題目指定state正常續跑。兩個GARGATH指定state都在戰鬥；本輪未從GARGATH原戰鬥狀態取得十張自由探索畫格。','- 蓋加斯主要十張來自戰前入城分支。祖先為 ovmap/ov.state，沿用既有診斷座標注入 `41B96=0d00;41B98=0300`，正常通過入城問答後保存，再以M進移動模式、於入口三格轉向。這是診斷起點續跑，不是未注入的冷開機玩家流程。','- 固定[移動起點與來源鏈](../inventory/issue22/gargath-movement-start/source.json)的SHA為 `88ac5bff23faad10f40722a4c44f04978895675fb092252f655fd1a8bb15c890`。它與最終父分支state逐位元組相同。單次probe的 initial_poke 空字串只代表該次沒有新增注入，不能抹去祖先注入。','- GARGATH2 正常戰後退出、重入的兩張3D後很快進入戰鬥。另以Y、Return保存的[入城診斷](issue22-runtime/gargath2-entry-base3/final.png)在送移動鍵前已進戰鬥。未用狀態注入避戰，也未把戰鬥畫格列成探索樣本。其「同一指定來源十張」仍未完成；a、b分支各九張亦原樣列出，不重複補數。','- block201用途與格式仍未知。牆面、符號的未觀測用途、WALLDEF控制流及未覆蓋局部組合仍未知。沒有把完整模板命中宣稱為原版選圖順序或全遊戲已驗收。','',
        '## 收據與重跑','',
        f'最終探針為 Go 1.24.13，VCS `f0448acad67c5f99148896125587c84db74c933b` 加本機改動；[建置資訊](issue22-go-version.txt)。二進位SHA為 `{status["probe_sha256"]}`。映像完整ID、原版START.EXE／GAME.OVR雜湊及主題雜湊見[status](issue22-status.json)。','',
        '| 檢查 | 結果 |','|---|---|',
        f'| apps/cok | 通過；本輪真實素材與COMSPR測試有執行。[日誌](issue22-go-tests.log)。{len(status["go_tests_skipped"])}個舊私有資料測試因未設定其專用參數明確SKIP，名稱在status |',
        f'| 獨立整幀 | {status["frame_checks"]}組通過：115完整底色來源、116單格失配、1游標、263原版保存／續跑畫格。三倍逐點相同，二倍、原版檢視與輸入色號不變。[JSONL](issue22-frame-receipts.jsonl) |',
        f'| 最終程式A/B | {status["runtime_rows"]}組通過。每組相同二進位、state與輸入；三倍與二倍兩側CPU、1MiB RAM、原色號、256色色盤、暫存檔案及完整終點state相同。二倍畫面零差異，三倍與獨立Pillow預期零差異。[結果](issue22-runtime/receipt.json)／[狀態](issue22-runtime/state-comparison.json) |',
        '| G5繁中七場景 | 完整SHA-256全部與#21相同，疊字範圍外差異0，無缺字。[結果](issue22-g5/receipt.json) |','',
        '| G5場景 | 繁中SHA-256前12碼 |','|---|---|']
    for r in g5:text.append(f'| {r["scene"]} | `{r["actual_sha256"][:12]}` |')
    text += ['',
        '兩側在執行前載入同一state，固定原版亂數記憶體及虛擬時間，沒有重擲或修改正式亂數。種子欄位未另解碼；這裡驗證相同初態與輸入下的呈現不干擾，不宣稱跨全流程骰序驗收。','',
        '在專案根目錄執行。所有程式均由Docker入口執行，Go沿用 workplace/dosgolem/tools/go.sh：','',
        '```sh',
        '# b18與主題可從現存來源及研究索引重建，不須影像生成。',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py init',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py theme',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py compositions',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py portable_audit',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py portable_theme',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py source_audit','',
        'DOSGOLEM_EXTRA_MOUNTS="$PWD/workplace/hd:/hd;$PWD/workplace/probe/issue21-battle-source:/probe:ro" \\',
        'DOSGOLEM_COK_ISSUE21_THEME=/hd/theme DOSGOLEM_COK_ISSUE21_PROBE=/probe \\',
        'DOSGOLEM_COK_ISSUE22_THEME=/hd/theme DOSGOLEM_COK_ISSUE22_BATCH=/hd/out/b18 \\',
        'bash tools/issue22-hd-go.sh test ./apps/cok -count=1 -v','',
        'bash tools/issue22-hd-runtime.sh build',
        'bash tools/issue22-hd-validate.sh core-a',
        'bash tools/issue22-hd-validate.sh core-b',
        'bash tools/issue22-hd-validate.sh gargath',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py freeze_movement',
        'bash tools/issue22-gargath-turns.sh',
        'bash tools/issue22-g5.sh',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22_receipts.py runtime',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22_receipts.py coverage','',
        '# 固定495例的原始資料、預期及雜湊均在cases.json；重播時不重取樣。',
        'DOSGOLEM_GO_CMD=/hd/out/issue22-cok-hd-frame-check \\',
        'DOSGOLEM_EXTRA_MOUNTS="$PWD/workplace/hd:/hd" \\',
        'bash tools/issue22-hd-go.sh -hd /hd/theme \\',
        ' -cases /hd/out/issue22-frames/cases.json \\',
        ' -receipt /hd/out/issue22-frame-receipts.jsonl',
        'bash tools/issue22-hd-img.sh python -B tools/hd_issue22.py audit',
        '```','',
        'GARGATH來源若需從前一步重生，先執行：','',
        '```sh',
        'bash tools/issue22-hd-runtime.sh gargath-entry base 3 4620000000 4455000000 \\',
        " 'Return,M,Right,Y,Y,Return,Return,Return,Return,Return,Return' 15000000",
        'bash tools/issue22-hd-runtime.sh gargath2 base 3 9966000000 9466000000 \\',
        " 'Escape,E,Escape,E,Right,Up,Left,Up,Right,Up,Left,Up,Right,Up,Right,Up,Left,Up,Up,Right,Up,Up'",
        'bash tools/issue22-gargath2-explore.sh prepare',
        '```','',
        '## 改動檔案與清理','',
        '完整19個程式／規格檔及SHA見[code-files](issue22-code-files.json)：','',
        '- 規格：docs/spec/010-hd-theme.md。','- Python：tools/hd_issue22.py、tools/hd_issue22_receipts.py。','- Shell：tools/issue22-docker.sh、issue22-hd-img.sh、issue22-hd-go.sh、issue22-hd-runtime.sh、issue22-hd-validate.sh、issue22-g5.sh、issue22-gargath-turns.sh、issue22-gargath2-explore.sh。','- dosgolem：apps/cok/hd_scale3x.go、hd_exact.go、hd_live.go、wall_hd.go、wall_hd_test.go、hd_issue21_test.go、hd_issue22_test.go；cmd/cok-hd-probe/main.go。','- 本機資料：workplace/hd/requests/b18、out/b18、inventory/issue22、theme及out/issue22-*。','',
        '沒有修改其他規格、CONTEXT.md、AGENTS.md、README、text/、session/、cok-play、cok-promo、cheat或其他儲存庫。本輪未執行commit、push、Release或打包。工作樹中其他同時進行的變更沒有納入本輪清單。','',
        '初期Go包裝器的shell函式未穿過外部timeout，改用PATH中的實體Docker入口補標籤。執行中更新外層腳本曾造成unbound variable與EOF，固定腳本後已用同命令乾淨重跑。蓋加斯取樣也修正為先M進入移動模式。這些是驗證腳本與輸入模式問題，沒有當作遊戲缺陷。','',
        '正式容器均有 cok.issue=22、--rm、無網路、UID/GID1000、最多2 CPU、2GiB、PID與日誌限制及外層逾時，最多三個同時執行。[執行設定](issue22-docker-runtime.jsonl)。[擁有權與異常目錄稽核](issue22-hygiene.json)通過；最終Docker清理檢查另存 issue22-containers-final.txt。沒有prune、rmi、刪除他人容器或清快取。','']
    (OUT/'issue22-report.md').write_text('\n'.join(text))
    print('wrote',OUT/'issue22-report.md')

if __name__ == '__main__':
    p=argparse.ArgumentParser();p.add_argument('action',choices=['init','theme','compositions','source_audit','portable_audit','portable_theme','freeze_movement','audit','report']);a=p.parse_args();globals()[a.action]()
