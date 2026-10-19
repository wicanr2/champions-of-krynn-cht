#!/usr/bin/env python3
"""把原文對照切成翻譯批次（規格 002；工作法見 batch-subagent-localization）。

ECL 來源（`ECLn:block:offset`）同一 block 放同一批；介面來源（`START.EXE:offset`、`GAME.OVR:offset`）
同一檔依位置排序。依檔內位置排序，超過上限就切開。來源檔若沒有 tier 欄，tier 寫 `-`。
每列附前後各一則原文，方便接句。輸出在 workplace，含原文，不入版控。
欄位：key、original、prev、next、tier、source
"""
import argparse
import csv
from pathlib import Path


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--source', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    ap.add_argument('--max', type=int, default=130)
    ap.add_argument('--start', type=int, default=0, help='批次編號起點')
    a = ap.parse_args()
    rows = list(csv.reader(a.source.open(encoding='utf-8'), delimiter='\t', quoting=csv.QUOTE_NONE))[1:]
    blocks = {}
    for r in rows:
        key, text, sources = r[0], r[1], r[2]
        tier = r[3] if len(r) > 3 else '-'
        parts = sources.split(';')[0].split(':')
        if len(parts) == 3:
            group, o = (f'ECL{int(parts[0][3:])}', int(parts[1])), int(parts[2])
        else:
            group, o = (parts[0], 0), int(parts[1])
        blocks.setdefault(group, []).append((o, key, text, tier, sources))
    a.out.mkdir(parents=True, exist_ok=True)
    idx = a.start
    for bk in sorted(blocks):
        items = sorted(blocks[bk])
        for start in range(0, len(items), a.max):
            chunk = items[start:start + a.max]
            idx += 1
            lines = ['key\toriginal\tprev\tnext\ttier\tsource']
            for i, (o, key, text, tier, src) in enumerate(items):
                if not (start <= i < start + a.max):
                    continue
                prev = items[i - 1][2] if i > 0 else ''
                nxt = items[i + 1][2] if i + 1 < len(items) else ''
                lines.append('\t'.join([key, text, prev, nxt, tier, src.split(';')[0]]))
            (a.out / f'batch-{idx:02d}.tsv').write_text('\n'.join(lines) + '\n', encoding='utf-8')
            print(f'batch-{idx:02d} {bk[0]}:{bk[1]} {len(chunk)}')


if __name__ == '__main__':
    main()
