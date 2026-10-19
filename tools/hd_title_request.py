#!/usr/bin/env python3
"""Issue #12：以真實交替標題畫面核對 TITLE 2+4 完整組圖。

入口：規格 010 §5；不從未觀測零件猜測組圖。
"""
import argparse
import json
from pathlib import Path
from hd_request import source,block_pixels,emit
from hd_coverage import sha


def main():
    ap=argparse.ArgumentParser()
    for n in ('orig','screen','batch'):ap.add_argument('--'+n,type=Path,required=True)
    a=ap.parse_args()
    raw,base=source(a.orig,'TITLE.DAX',2,placed_at=[0,0],block_size=[320,200])
    w,h,px=block_pixels(raw);assert (w,h)==(320,200)
    canvas=bytearray(px)
    raw,logo=source(a.orig,'TITLE.DAX',4,placed_at=[0,8],block_size=[320,88])
    w,h,px=block_pixels(raw);assert (w,h)==(320,88)
    for y in range(h):canvas[(8+y)*320:(9+y)*320]=px[y*320:(y+1)*320]
    screen=a.screen.read_bytes();assert len(screen)==64000 and bytes(canvas)==screen,'組圖不等於原版畫面'
    man=json.loads((a.batch/'manifest.json').read_text())
    for it in man['items']:
        if it['key']=='title-004':
            it.update(disposition='composition_component',integration_allowed=False,
                runtime_evidence='observed',evidence_screen=str(a.screen),screen_rect=[0,8,320,88],position_grade='已證實',
                evidence_records=[dict(file=str(a.screen),screen_sha256=sha(screen),screen_rect=[0,8,320,88])])
            for s in it['sources']:s['placed_at']=[0,8]
    it=emit(a.batch,3,'title-dragonlance','冷啟動 DragonLance 交替標題，TITLE 2+4',
        (0,0,320,200),None,canvas,[base,logo],runtime_evidence='observed',integration_allowed=True,
        position_grade='已證實',evidence_records=[dict(file=str(a.screen),screen_sha256=sha(screen),screen_rect=[0,0,320,200])])
    it['evidence_screen']=str(a.screen)
    man['items']=[x for x in man['items'] if x['key']!=it['key']]+[it]
    (a.batch/'manifest.json').write_text(json.dumps(man,ensure_ascii=False,indent=2)+'\n')
    print(it['key'],it['indexed_sha256'],'與真實畫面逐色號相同')


if __name__=='__main__':main()
