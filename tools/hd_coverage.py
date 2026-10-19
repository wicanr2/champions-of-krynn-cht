#!/usr/bin/env python3
"""Issue #12：DAX 色號狀態、既有執行畫面與主題的可重播覆蓋盤點。

入口：docs/spec/010-hd-theme.md §5。原始動畫幀與 XOR 候選分開記錄；
未見畫面的來源不升級成執行證據。相同色號以尺寸及 SHA-256 去重。
"""
import argparse
import hashlib
import json
from pathlib import Path
import numpy as np
from PIL import Image
from dax_pics import dax, picture, frames, EGA
from hd_request import offsets


def sha(data):
    return hashlib.sha256(data).hexdigest()


def inventory(orig):
    registry, files = {}, {}
    for path in sorted(orig.glob('*.DAX')):
        if not path.name.startswith(('PIC', 'BIGPIC', 'TITLE')):
            continue
        data = path.read_bytes()
        files[path.name] = sha(data)
        for bid, raw in dax(data):
            try:
                w, h, count, x, y, px = picture(raw)
                assert count == 1, (path, bid, count)
                states = [(0, px, None)]
                animated = False
            except ValueError:
                fr = frames(raw)
                w, h = fr[0][1:3]
                assert all(f[1:4] == (w, h, 1) for f in fr)
                base = fr[0][-1]
                states = [(k, base if k == 0 else bytes(a ^ b for a, b in zip(base, f[-1])),
                           None if k == 0 else f[-1]) for k, f in enumerate(fr)]
                animated = True
            for k, px, mask in states:
                assert len(px) == w*h and max(px) < 16
                key = (w, h, sha(px))
                row = registry.setdefault(key, dict(width=w, height=h, indexed_sha256=key[2],
                    aliases=[], evidence=[], screens=0, theme_keys=[], component_theme_keys=[],
                    constant=len(set(px)) == 1, pixels=px))
                row['aliases'].append(dict(file=path.name, file_sha256=files[path.name], block=bid,
                    frame=k, frames=len(states), animated=animated, **offsets(data)[bid],
                    decoded_sha256=sha(raw), raw_frame_sha256=sha(fr[k][-1]) if animated else sha(px),
                    state_rule='base XOR raw_frame' if animated and k else 'base',
                    mask_sha256=sha(mask) if mask else None))
    return registry, files


def main():
    ap = argparse.ArgumentParser()
    for name in ('orig', 'probe', 'theme', 'out'):
        ap.add_argument('--'+name, required=True, type=Path)
    ap.add_argument('--screens', type=Path)
    a = ap.parse_args()
    registry, files = inventory(a.orig)
    theme = json.loads((a.theme/'theme.json').read_text())
    for it in theme['items']:
        x, y, w, h = it['screen_rect']
        raw = (a.theme/it['indexed']).read_bytes()
        assert sha(raw) == it['indexed_sha256']
        row = registry.get((w,h,sha(raw)))
        if row:
            row['theme_keys'].append(it['key'])
        for r in registry.values():
            if any(s['file'] == al['file'] and s['block'] == al['block']
                   for s in it.get('sources', []) for al in r['aliases']):
                r['component_theme_keys'].append(it['key'])
    paths = set(a.probe.rglob('*.bin'))
    if a.screens:
        # normal/ 是本工具鏈另存的正規化副本，不重複計為新原版畫面。
        paths.update(p for p in a.screens.rglob('*.bin') if 'normal' not in p.relative_to(a.screens).parts)
    paths = {p for p in paths if p.stat().st_size == 64000}
    paths.update(a.probe.rglob('*-base.png'))
    paths = sorted(paths)
    title_rows = [r for r in registry.values() if any(al['file'].startswith('TITLE') for al in r['aliases'])]
    for n, p in enumerate(paths):
        if p.suffix == '.png':
            rgb=np.array(Image.open(p).convert('RGB'))
            sy,sx=rgb.shape[0]//200,rgb.shape[1]//320
            if sy != sx or sy < 1 or rgb.shape[:2] != (200*sy,320*sx):
                raise ValueError(f'{p} 原版畫面尺寸不合')
            small=rgb[::sy,::sx]
            if not np.array_equal(rgb,np.repeat(np.repeat(small,sy,0),sx,1)):
                raise ValueError(f'{p} 並非原版整數倍畫面')
            screen=np.full((200,320),255,np.uint8)
            for k,color in enumerate(EGA):
                screen[np.all(small==color,axis=2)]=k
            if (screen==255).any():
                raise ValueError(f'{p} 有非 EGA 色，不能猜測原版色號')
        else:
            screen = np.frombuffer(p.read_bytes(), np.uint8).reshape(200,320) & 15
        for w,h,x,y in ((88,88,24,24),(304,120,8,8),(320,200,0,0),(240,88,40,8)):
            r = registry.get((w,h,sha(screen[y:y+h,x:x+w].tobytes())))
            if r:
                record(r,p,x,y)
        # TITLE 的非全幅區塊沒有先假定座標；以逐列第一列比對縮小搜尋。
        for r in title_rows:
            w,h=r['width'],r['height']
            if (w,h) in ((320,200),(240,88)):
                continue
            px=np.frombuffer(r['pixels'],np.uint8).reshape(h,w)
            for y in range(201-h):
                for x in range(0,321-w,8):
                    if np.array_equal(screen[y,x:x+w],px[0]) and np.array_equal(screen[y:y+h,x:x+w],px):
                        record(r,p,x,y)
        if (n+1)%2000 == 0:
            print(f'已掃描 {n+1}/{len(paths)}',flush=True)
    rows=[]
    for r in registry.values():
        r.pop('pixels')
        r['runtime_evidence']='observed' if r['screens'] else 'unobserved'
        r['evidence_grade']='已證實完整畫面色號' if r['screens'] else '來源已解碼；XOR 相位與執行位置未觀測'
        rows.append(r)
    summary={}
    for family in ('PIC','BIGPIC','TITLE'):
        group=[r for r in rows if any(al['file'].startswith(family) for al in r['aliases'])]
        summary[family]=dict(unique_states=len(group),aliases=sum(len(r['aliases']) for r in group),
            integrated=sum(bool(r['theme_keys']) for r in group),
            component_covered=sum(bool(r['component_theme_keys']) for r in group),
            observed=sum(bool(r['screens']) for r in group),unobserved=sum(not r['screens'] for r in group))
    result=dict(issue='#12',spec='docs/spec/010-hd-theme.md §5',source_sha256=files,
        screen_files=len(paths),theme_items=len(theme['items']),summary=summary,items=rows)
    a.out.parent.mkdir(parents=True,exist_ok=True)
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(summary,ensure_ascii=False))


def record(row,path,x,y):
    evidence=dict(file=str(path),screen_sha256=sha(path.read_bytes()),screen_rect=[x,y,row['width'],row['height']])
    if any(e['file']==str(path) and e['screen_rect']==evidence['screen_rect'] for e in row['evidence']):
        return
    row['screens']+=1
    if len(row['evidence'])<5:
        row['evidence'].append(evidence)


if __name__=='__main__':
    main()
