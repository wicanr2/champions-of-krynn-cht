#!/usr/bin/env python3
"""在原版畫面（320×200 色號 .bin，probe -dump-at 產出）裡找 DAX 圖片的出現位置（G7 來源證據，#8）。

每張圖片（含動畫每幀、每個 item）在每個畫面做完全比對：圖片的 4 bit 色號與畫面色號逐像素相同；
遮罩色（16，未使用）不比。8X8D 圖塊太小、太常見，不比。
輸出 TSV：file、block、frame、item、width、height、screen、x、y。
"""
import argparse
import sys
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parent))
from dax_pics import dax, frames, picture  # noqa: E402

SKIP = ('8X8D', 'ECL', 'GEO', 'MON1', 'MUSIC', 'SOUND', 'ITEM0', 'WALLDEF')


def pictures(orig, only=()):
    for f in sorted(orig.glob('*.DAX')):
        if f.name.startswith(SKIP) or (only and not f.name.startswith(only)):
            continue
        for bid, d in dax(f.read_bytes()):
            try:
                w, h, items, _, _, px = picture(d)
                pics = [(0, w, h, items, px)]
            except ValueError:
                try:
                    pics = [(k, w, h, items, px) for k, (_, w, h, items, x, y, px) in enumerate(frames(d))]
                except ValueError:
                    continue
            for k, w, h, items, px in pics:
                for it in range(items):
                    a = np.frombuffer(px[it * w * h:(it + 1) * w * h], np.uint8).reshape(h, w)
                    yield f.name, bid, k, it, a


def find(screen, pic):
    H, W = screen.shape
    h, w = pic.shape
    if h > H or w > W or (pic == pic.flat[0]).all():
        return []
    # 先用左上一列縮小候選，再逐一全比
    row = pic[0]
    hits = []
    for y in range(H - h + 1):
        line = screen[y]
        for x in np.flatnonzero(line[:W - w + 1] == row[0]):
            if (line[x:x + w] == row).all() and (screen[y:y + h, x:x + w] == pic).all():
                hits.append((int(x), y))
    return hits


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--screens', nargs='+', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    ap.add_argument('--only', default='', help='只搜這些檔名開頭，逗號分隔（例：PIC,BIGPIC,CPIC）')
    a = ap.parse_args()
    screens = [(p, np.frombuffer(p.read_bytes(), np.uint8).reshape(200, 320) & 15) for p in a.screens if p.stat().st_size == 64000]
    pics = list(pictures(a.orig, tuple(p for p in a.only.split(',') if p)))
    rows = ['file\tblock\tframe\titem\twidth\theight\tscreen\tx\ty']
    for name, bid, k, it, pic in pics:
        for p, s in screens:
            for x, y in find(s, pic):
                rows.append(f'{name}\t{bid}\t{k}\t{it}\t{pic.shape[1]}\t{pic.shape[0]}\t{p}\t{x}\t{y}')
    a.out.write_text('\n'.join(rows) + '\n', encoding='utf-8')
    print(f'{len(pics)} 張圖 × {len(screens)} 畫面，命中 {len(rows) - 1}')


if __name__ == '__main__':
    main()
