#!/usr/bin/env python3
"""Issue #12：保存逐項驗收、原圖、提示詞與主題的交付索引。

入口：docs/spec/010-hd-theme.md §5。只寫 workplace/hd。
"""
import argparse
import json
import platform
from pathlib import Path
import PIL
import numpy as np
from hd_coverage import sha


def read(path,default):
    return json.loads(path.read_text()) if path.exists() else default


def main():
    ap=argparse.ArgumentParser();ap.add_argument('--hd',type=Path,required=True);a=ap.parse_args();root=a.hd
    theme=read(root/'theme/theme.json',{'items':[]})
    selected={it['key'] for it in theme['items']}
    total=[]
    for req in sorted((root/'requests').glob('b0[2-6]/manifest.json')):
        request=read(req,{})
        out=root/'out'/request['batch']
        validation=read(out/'validation.json',{'items':[]});val={r['key']:r for r in validation['items']}
        review=read(out/'visual-review.json',{'items':[]});vis={r['key']:r for r in review['items']}
        reuse={r['key']:r for r in read(out/'reuse.json',[])}
        jobs=[dict(file=str(p.relative_to(out)),sha256=sha(p.read_bytes()),**read(p,{})) for p in sorted((out/'jobs').glob('*.json'))]
        raws=[dict(file=p.name,sha256=sha(p.read_bytes())) for p in sorted(out.glob('*-raw.png'))]
        rows=[]
        for it in request['items']:
            k=it['key'];p=out/(k+'.png');vr=val.get(k,{});manual=vis.get(k,{})
            integrated=k in selected
            if integrated:
                assert p.exists() and vr['target_met'] and manual['approved']
                assert sha(p.read_bytes())==vr['png_sha256']==manual['png_sha256']
                assert validation['request_manifest_sha256']==sha(req.read_bytes())
            status='integrated' if integrated else it.get('disposition') or 'pending'
            reasons=[]
            if not integrated and not it.get('disposition'):
                if not p.exists():reasons.append('尚無輸出')
                elif not vr.get('target_met'):reasons.append('數值閘門未通過')
                if not manual.get('approved'):reasons.append('目視尚未核准')
                if not it.get('integration_allowed'):reasons.append('來源位置未知')
            row=dict(source_key=k,sources=it['sources'],aliases=it.get('aliases',[]),
                target_size=it['target_size'],screen_rect=it['screen_rect'],runtime_evidence=it['runtime_evidence'],
                status=status,reasons=reasons,output_file=p.name if p.exists() else None,
                output_sha256=sha(p.read_bytes()) if p.exists() else None,validation=vr,visual_review=manual,
                generation_jobs=[j['file'] for j in jobs if j['key']==k],reuse=reuse.get(k))
            rows.append(row);total.append(dict(batch=request['batch'],**row))
        artifacts={str(p.relative_to(out)):sha(p.read_bytes()) for p in sorted(out.rglob('*'))
            if p.is_file() and (p.suffix=='.json' or p.name.endswith('-raw.png')) and p.name!='manifest.json'}
        result=dict(batch=request['batch'],issue='#12',spec=request['spec'],scale=3,style='B',
            request_manifest_sha256=sha(req.read_bytes()),items=rows,jobs=jobs,raws=raws,artifacts=artifacts,
            integrated_count=sum(r['status']=='integrated' for r in rows),
            environment=dict(image='cok-img:r1',python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__),
            rights_note='AI 重繪素材依專案定案可放入原版＋HD patch；原版來源與執行畫面仍限本機。')
        (out/'manifest.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (root/'out/issue12-status.json').write_text(json.dumps(dict(issue='#12',theme_items=len(theme['items']),items=total),ensure_ascii=False,indent=2)+'\n')
    print('主題',len(theme['items']),'項；新請求',len(total),'項；已接入',sum(r['status']=='integrated' for r in total))


if __name__=='__main__':main()
