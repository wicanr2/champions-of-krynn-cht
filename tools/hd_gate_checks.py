#!/usr/bin/env python3
"""Issue #12：以實際候選測目視撤銷及驗收雜湊失效的負對照。

入口：規格 010 §5。只在指定 HD 目錄建立短期副本，不修改正式候選。
"""
import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path


def main():
    ap=argparse.ArgumentParser();ap.add_argument('--hd',type=Path,required=True);a=ap.parse_args();root=a.hd.resolve()
    rows=[]
    for case in ('review-revoked','review-digest-stale','request-digest-stale'):
        with tempfile.TemporaryDirectory(dir=root/'inventory') as temp:
            temp=Path(temp);hd=temp/'hd';hd.mkdir();(hd/'b01').symlink_to(root/'out/b01',target_is_directory=True)
            batch=hd/'b02';batch.mkdir()
            for p in (root/'out/b02').iterdir():
                if p.is_file() and p.suffix=='.png':(batch/p.name).symlink_to(p)
            review=json.loads((root/'out/b02/visual-review.json').read_text())
            validation=json.loads((root/'out/b02/validation.json').read_text())
            if case=='review-revoked':review['items'][0]['approved']=False
            elif case=='review-digest-stale':review['items'][0]['png_sha256']='0'*64
            else:validation['request_manifest_sha256']='0'*64
            (batch/'visual-review.json').write_text(json.dumps(review))
            (batch/'validation.json').write_text(json.dumps(validation))
            dest=temp/'theme'
            r=subprocess.run([sys.executable,str(Path(__file__).with_name('hd_theme.py')),
                '--req',str(root/'requests'),'--hd',str(hd),'--out',str(dest),'--batches','b02'],text=True,capture_output=True)
            good=r.returncode!=0 and not (dest/'theme.json').exists()
            rows.append(dict(case=case,ok=good,exit_code=r.returncode,message=r.stderr.strip()))
            assert good,(case,r.stdout,r.stderr)
    (root/'out/issue12-gate-controls.json').write_text(json.dumps(dict(all_pass=True,items=rows),ensure_ascii=False,indent=2)+'\n')
    print('3 個主題驗收失效負對照全部通過')


if __name__=='__main__':main()
