#!/usr/bin/env python3
"""Issue #12：保存所有生成版本，整幅縮放，依來源動畫遮罩接合。

入口：docs/spec/010-hd-theme.md §5。raw 不覆寫；無裁切與自動對位。
"""
import argparse
import json
import shutil
import tempfile
from pathlib import Path
from PIL import Image, ImageFilter
from hd_coverage import sha


def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--root',type=Path,required=True)
    ap.add_argument('--generated',type=Path,default=Path('/kb/generated_images'))
    a=ap.parse_args();root=a.root
    # Issue #12 重播只處理本輪批次，避免改寫戰鬥／遭遇圖的獨立交付索引。
    for req in sorted((root/'requests').glob('b0[2-6]/manifest.json')):
        man=json.loads(req.read_text());batch=man['batch']
        if batch=='b01':continue
        out=root/'out'/batch;out.mkdir(exist_ok=True)
        reuse=out/'reuse.json'
        for previous in json.loads(reuse.read_text()) if reuse.exists() else []:
            archived=out/previous['archived_png']
            assert sha(archived.read_bytes())==previous['previous_png_sha256']
            shutil.copyfile(archived,out/f'{previous["key"]}.png')
        jobs=sorted((out/'jobs').glob('*.json'))
        selected={}
        for job in jobs:
            j=json.loads(job.read_text());key=j['key']
            if j.get('status')!='generated':continue
            src=a.generated/Path(j['generated_path']).parent.name/Path(j['generated_path']).name
            raw=out/f'{key}-{job.stem}-raw.png'
            if not raw.exists():shutil.copyfile(src,raw)
            assert sha(raw.read_bytes())==sha(src.read_bytes())
            selected[key]=raw
        for it in sorted(man['items'],key=lambda it:bool(it.get('animation',{}).get('mask'))):
            key=it['key'];raw=selected.get(key)
            if not raw:continue
            im=Image.open(raw).convert('RGBA').resize(tuple(it['target_size']),Image.Resampling.LANCZOS)
            im.putalpha(255)
            anim=it.get('animation',{})
            if anim.get('mask'):
                base=out/f'{anim["base"]}.png'
                if not base.exists():continue
                mask=Image.open(req.parent/anim['mask']).convert('L').filter(ImageFilter.MaxFilter(13))
                im=Image.composite(im,Image.open(base).convert('RGBA'),mask)
            # 驗收可與其他批次生成並行；只讓讀者看到完整 PNG。
            with tempfile.NamedTemporaryFile(dir=out,suffix='.png',delete=False) as tmp:
                name=Path(tmp.name)
            try:
                im.save(name)
                name.replace(out/f'{key}.png')
            finally:
                name.unlink(missing_ok=True)
        rows=[]
        for it in man['items']:
            p=out/f'{it["key"]}.png'
            rows.append(dict(key=it['key'],status='candidate' if p.exists() else 'missing',
                png_sha256=sha(p.read_bytes()) if p.exists() else None,
                runtime_evidence=it['runtime_evidence']))
        output=dict(issue='#12',batch=batch,spec=man['spec'],scale=3,style='B',
            request_sha256=sha(req.read_bytes()),items=rows,processing='tools/hd_prepare.py',
            rights_note='AI 生成，依專案定案可放入原版＋HD patch。前作引用另記來源。')
        (out/'manifest.json').write_text(json.dumps(output,ensure_ascii=False,indent=2)+'\n')


if __name__=='__main__':main()
