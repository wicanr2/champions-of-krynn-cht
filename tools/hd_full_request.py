#!/usr/bin/env python3
"""Issue #12 全量請求；每批 8–12 個來源區塊，動畫狀態不拆批。

入口：docs/spec/010-hd-theme.md §5。索引只按本作 DAX 雜湊去重。
TITLE 組圖中的原始零件保留供盤點，不將零件另當已觀測的整幅畫面。
"""
import argparse
import json
from pathlib import Path
from hd_coverage import inventory, sha
from hd_request import emit, rgb_png
from dax_pics import EGA


def main():
    ap=argparse.ArgumentParser()
    for n in ('orig','coverage','out'):
        ap.add_argument('--'+n,type=Path,required=True)
    a=ap.parse_args()
    cov=json.loads(a.coverage.read_text())
    reg,files=inventory(a.orig)
    bysha={r['indexed_sha256']:r for r in cov['items']}
    groups={}
    for r in reg.values():
        cr=bysha[r['indexed_sha256']]
        if cr['theme_keys']:
            continue
        al=r['aliases'][0]
        groups.setdefault((al['file'],al['block']),[]).append(r)
    groups=sorted(groups.items())
    n=(len(groups)+11)//12
    q,rem=divmod(len(groups),n)
    assert q>=8 or n==1
    plan=[];start=0
    for i in range(n):
        batch=f'b{i+2:02d}';out=a.out/batch
        if out.exists():
            raise FileExistsError(f'{out} 已存在；不覆寫請求')
        out.mkdir(parents=True)
        size=q+(i<rem);selection=groups[start:start+size];start+=size
        items=[]
        for (fname,bid),rows in selection:
            for r in sorted(rows,key=lambda r:r['aliases'][0]['frame']):
                al=r['aliases'][0];w,h=r['width'],r['height'];k=al['frame']
                stem=f'{Path(fname).stem.lower()}-{bid:03d}'
                key=f'{stem}@{k}' if al['animated'] else stem
                cr=bysha[r['indexed_sha256']]
                rect=cr['evidence'][0]['screen_rect'] if cr['evidence'] else (
                    [24,24,w,h] if fname.startswith('PIC') else [8,8,w,h] if fname.startswith('BIGPIC') else [0,0,w,h])
                extra=dict(runtime_evidence=cr['runtime_evidence'],aliases=r['aliases'],
                    evidence_records=cr['evidence'],position_grade='已證實' if cr['evidence'] else
                    '家族契約，未觀測' if not fname.startswith('TITLE') else '未知，禁止接入',
                    integration_allowed=not fname.startswith('TITLE') or bool(cr['evidence']),
                    component_theme_keys=cr['component_theme_keys'])
                if cr['component_theme_keys']:
                    extra['disposition']='already_covered_composition_component'
                    extra['integration_allowed']=False
                if al['animated']:
                    extra['animation']=dict(rule='base XOR raw_frame；未觀測相位仍為假說',
                        frame=k,base=f'{stem}@0',evidence_state=bool(cr['screens']))
                    if k:
                        # 狀態與原始基底 XOR，與解碼的 delta 一致。
                        base=next(rr for rr in reg.values() if any(x['file']==fname and x['block']==bid and x['frame']==0 for x in rr['aliases']))
                        mask=bytes(int(x!=y) for x,y in zip(r['pixels'],base['pixels']))
                        name=f'{key}-mask-x3.png'
                        rgb_png(out/name,mask,w,h,3,[(0,0,0),(255,255,255)])
                        extra['animation']['mask']=name
                if r['constant']:
                    extra['rendering']='native_exact_constant'
                sources=[dict(x,placed_at=rect[:2],block_size=[w,h]) for x in r['aliases']]
                it=emit(out,3,key,f'{fname} 區塊 {bid} 狀態 {k}',rect,None,r['pixels'],sources,**extra)
                if cr['evidence']:
                    it['evidence_screen']=cr['evidence'][0]['file']
                items.append(it)
        man=dict(batch=batch,issue='#12',spec='docs/spec/010-hd-theme.md §5',scale=3,style='B',
            source_blocks=len(selection),palette=dict(name='本作 EGA 16 色',rgb=EGA,
                evidence='既有原版色號與原版側 PNG 逐色核對，coverage.json'),items=items)
        (out/'manifest.json').write_text(json.dumps(man,ensure_ascii=False,indent=2)+'\n')
        plan.append(dict(batch=batch,source_blocks=len(selection),items=len(items),keys=[it['key'] for it in items]))
        print(batch,len(selection),'來源',len(items),'狀態')
    result=dict(issue='#12',coverage_sha256=sha(a.coverage.read_bytes()),source_sha256=files,batches=plan)
    plan_path=a.out.parent/'inventory/request-plan.json'
    plan_path.parent.mkdir(parents=True,exist_ok=True)
    plan_path.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')


if __name__=='__main__':
    main()
