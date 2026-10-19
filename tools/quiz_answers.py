#!/usr/bin/env python3
"""由原版題目表與英文手札 OCR 的逐字座標算出手冊查驗題答案（規格 006 本作版）。

題目表在 START.EXE（docs/re/005）：每筆 5 bytes `[位移, 第幾字, 第幾則, 頁, 答案長度]`，以「第 30 則第 6 字第 32 頁」
（實機已答對 TO）定位整段連續紀錄。答案字取自 `pdftotext -bbox` 的逐字座標：每頁依欄位（四欄）由上而下排序，
找到 `Entry N` 標題後從下一個字起算第 W 個字；行尾連字號斷字併回一字。每題以表中的答案長度核對，不符時報錯。
輸出 text/quiz-answers.tsv（page、entry、word、answer），只含答案字。
"""
import argparse
import html
import re
import sys
from pathlib import Path

# OCR 排序錯誤的題目，以頁面圖人工核對（docs/re/005）：第 67 則旁的地圖圖例「KEY. A rough map…」被排進正文。
OVERRIDES = {(38, 67, 8): 'CARAMON'}

WORD = re.compile(r'<word xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)" yMax="([\d.]+)">(.*?)</word>')


def quiz_table(exe):
    d = exe.read_bytes()
    anchor = [i for i in range(len(d) - 5) if d[i + 1:i + 5] == bytes([6, 30, 32, 2])]
    if len(anchor) != 1:
        sys.exit(f'題目表錨點命中 {len(anchor)} 處')
    ok = lambda o: 1 <= d[o + 1] <= 20 and 1 <= d[o + 2] <= 88 and 20 <= d[o + 3] <= 60 and 1 <= d[o + 4] <= 20
    s = anchor[0]
    while ok(s - 5) and d[s - 5] < d[s]:
        s -= 5
    recs, o = [], s
    while ok(o):
        recs.append(tuple(d[o + 1:o + 5]))
        o += 5
    return recs


def entries(bbox):
    out = {}
    for page in bbox.read_text(encoding='utf-8').split('<page ')[1:]:
        words = [(float(a), float(b), float(c), html.unescape(t)) for a, b, c, _, t in WORD.findall(page)]
        cols = [[] for _ in range(4)]
        for x, y, x2, t in words:
            cols[min(3, int(x // 198))].append((y, x, t))
        seq = []
        for c in cols:
            # 同一行的字 yMin 會差 1–2 點（標題編號字級較大），以 3 點容差分行，行內依 x 排序
            lines = []
            for y, x, t in sorted(c):
                if lines and y - lines[-1][0] < 3:
                    lines[-1][1].append((x, t))
                else:
                    lines.append([y, [(x, t)]])
            for _, ws in lines:
                seq += [t for _, t in sorted(ws)]
        cur = None
        for i, t in enumerate(seq):
            kind = 'journal' if t.lower().endswith('entry') else 'tavern' if t.lower().endswith('tale') else None
            if kind and i + 1 < len(seq) and seq[i + 1].isdigit():
                cur = (kind, int(seq[i + 1])); out[cur] = []; skip = True
                continue
            if t.lower() in ('journal', 'tavern') and i + 1 < len(seq) and seq[i + 1].lower() in ('entry', 'tale'):
                continue
            if cur is None:
                continue
            if skip:
                skip = False
                continue
            out[cur].append(t)
    return out


def joined(tokens):
    res = []
    for t in tokens:
        if res and res[-1].endswith('-') and len(res[-1]) > 1:
            res[-1] = res[-1][:-1] + t
        else:
            res.append(t)
    return [w for w in res if re.search(r'[A-Za-z0-9]', w)]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--exe', required=True, type=Path)
    ap.add_argument('--bbox', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    ents = entries(a.bbox)
    rows, bad = [], 0
    for w, e, p, n in quiz_table(a.exe):
        # 第 44 頁起是酒館傳聞（Tavern Tale），遊戲仍以 journal entry 稱呼
        ws = joined(ents.get(('tavern' if p >= 44 else 'journal', e), []))
        ans = re.sub(r'[^A-Za-z0-9]', '', ws[w - 1]).upper() if len(ws) >= w else ''
        ans = OVERRIDES.get((p, e, w), ans)
        flag = '' if len(ans) == n else f'  ← 長度不符（表 {n}）'
        bad += bool(flag)
        print(f'p{p} e{e} w{w}: {ans}{flag}')
        rows.append(f'{p}\t{e}\t{w}\t{ans}')
    a.out.write_text('page\tentry\tword\tanswer\n' + '\n'.join(rows) + '\n', encoding='utf-8')
    print(f'{len(rows)} 題，長度不符 {bad}')
    sys.exit(1 if bad else 0)


if __name__ == '__main__':
    main()
