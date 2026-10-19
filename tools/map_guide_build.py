#!/usr/bin/env python3
"""由攻略面板資料（text/walkthrough.zh-TW.tsv）的座標產生地圖攻略點 text/map-guide.zh-TW.tsv（幽靈騎士規格 009 §2.1）。

《軟體世界》攻略以「（x，y）」標出事件位置（畫面座標列的 x,y）。每章依章名對到 GEO 地圖（CHAPTER_MAP，
地圖身分見 text/map-titles.tsv 與 docs/re/006），每個座標取所在句子當說明、句首詞組當標籤。
同一地圖同一座標只留第一筆。第 1 章第一屯墾區的座標實為斯拉托（攻略把斯拉托的事件接在第 1 章後半），以句中出現的位置歸屬。
輸出欄位：map、n、x、y、label、summary、source。
"""
import argparse
import re
from pathlib import Path

CHAPTER_MAP = [  # (章名關鍵字, GEO 鍵)
    ('斯拉托墓穴', '1/22'), ('斯拉托', '1/20'), ('第一屯墾區', '1/20'),
    ('蓋加斯城堡', '2/31'), ('蓋加斯', '2/30'), ('達加德爵士之墓', '2/43'),
    ('第三屯墾區', '2/44'), ('傑勒克', '2/32'), ('奈拉卡監獄', '2/42'), ('奈拉卡', '2/40'),
    ('桑克申', '3/50'), ('胡爾齊德', '3/51'), ('杜爾加斯特', '3/52'), ('飛行堡壘', '3/60'), ('科南', '3/61'),
]
COORD = re.compile(r'（\s*(\d{1,2})\s*[，,]\s*(\d{1,2})\s*）')
SENT = re.compile(r'[^。！？\\]*?（\s*\d{1,2}\s*[，,]\s*\d{1,2}\s*）[^。！？]*[。！？」]?')


def chapter_map(title):
    for k, m in CHAPTER_MAP:
        if k in title:
            return m
    return None


def label_of(sent):
    s = COORD.sub('', sent)
    s = re.sub(r'^(走到|來到|到了|到|在|於|進入|一進到|一進入|前往|再到|再往|往)', '', s.strip('，、 '))
    s = re.split(r'[，。！？；：「]', s)[0].strip()
    return s[:10] if s else '事件'


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--walkthrough', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    rows, seen, counts = [], set(), {}
    for line in a.walkthrough.read_text(encoding='utf-8').splitlines()[1:]:
        ch, title, body, src = line.split('\t')[:4]
        m = chapter_map(title)
        if not m or title.startswith('地圖'):
            continue
        for para in body.split('\\n'):
            for sent in SENT.findall(para):
                for x, y in COORD.findall(sent):
                    x, y = int(x), int(y)
                    if x > 15 or y > 15 or (m, x, y) in seen:
                        continue
                    seen.add((m, x, y))
                    counts[m] = counts.get(m, 0) + 1
                    summary = sent.strip('，、 ').replace('\t', ' ')
                    rows.append((m, counts[m], x, y, label_of(sent), summary, f'{src}（{title}）'))
    rows.sort(key=lambda r: (r[0], r[1]))
    a.out.write_text('map\tn\tx\ty\tlabel\tsummary\tsource\n' + ''.join('\t'.join(map(str, r)) + '\n' for r in rows), encoding='utf-8')
    print(f'{len(rows)} 點：' + '、'.join(f'{k} {v}' for k, v in sorted(counts.items())))


if __name__ == '__main__':
    main()
