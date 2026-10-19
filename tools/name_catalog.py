#!/usr/bin/env python3
"""由 text/glossary.tsv 的 monster、person、place 類別產生名稱 catalog（規格 004 §2.3）。

遊戲以全大寫印怪物名（例 DREAD WOLF），術語表用一般大小寫；兩種寫法都收雜湊。
輸出 text/name-events.tsv（key、長度、SHA-256、來源）與 text/name.zh-TW.tsv（key、譯文、來源），不含原文。
同一雜湊對到不同譯文時報錯。
"""
import argparse
import hashlib
import sys
from pathlib import Path

# item、spell 也收：物品名由詞組合（例 Ring Mail），詞組要先於單詞命中（apps/cok compose.go item）
CATS = {'monster', 'person', 'place', 'item', 'spell'}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--glossary', required=True, type=Path)
    ap.add_argument('--events', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    rows = [l.split('\t') for l in a.glossary.read_text(encoding='utf-8').split('\n')[1:] if l]
    seen = {}
    for r in rows:
        en, zh, cat = r[0].strip(), r[1].strip(), r[2].strip()
        if cat not in CATS or not en or not zh:
            continue
        for form in {en, en.upper()}:
            raw = form.encode('ascii', 'strict')
            h = hashlib.sha256(raw).hexdigest()
            if h in seen and seen[h][1] != zh:
                sys.exit(f'雜湊衝突：{form!r} 對到 {seen[h][1]} 與 {zh}')
            seen[h] = (len(raw), zh)
    ev = ['event_key\toriginal_length\toriginal_sha256\tsources']
    tr = ['key\ttranslation\tsource']
    for h, (n, zh) in sorted(seen.items()):
        key = f'name.{h[:12]}'
        ev.append(f'{key}\t{n}\t{h}\tglossary')
        tr.append(f'{key}\t{zh}\tglossary')
    a.events.write_text('\n'.join(ev) + '\n', encoding='utf-8')
    a.out.write_text('\n'.join(tr) + '\n', encoding='utf-8')
    print(f'{len(seen)} 筆')


if __name__ == '__main__':
    main()
