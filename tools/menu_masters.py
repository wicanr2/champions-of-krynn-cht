#!/usr/bin/env python3
"""由介面字串產生底列選單母串表（規格 003 §2.2.1）。

選單模組收的選項字串是一整串母串（例 `Ready Use Trade Deposit Drop Halve Join Sell Id Exit`），
引擎依情況只畫其中幾個詞，所以畫面上的詞列常是母串的子序列，整串雜湊查不到。
本工具挑出「以單一空白或 @ 分隔的多詞字串、不是其他字串的片段、譯文詞數相同」者，
逐詞輸出 text/menu-masters.tsv（master、index、original_length、original_sha256、translation），不含原文。
"""
import argparse
import csv
import hashlib
import re
from pathlib import Path

WORDS = re.compile(r"[A-Za-z][A-Za-z0-9'\-]*(?:[ @][A-Za-z0-9][A-Za-z0-9'\-]*)+")


def read(path, enc):
    with open(path, encoding=enc, newline='') as f:
        return list(csv.DictReader(f, delimiter='\t', quoting=csv.QUOTE_NONE))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--source', required=True, type=Path, help='workplace/text/ui-text-source.tsv（含原文，不進版控）')
    ap.add_argument('--translations', required=True, type=Path, help='text/ui-text.zh-TW.tsv')
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    csv.field_size_limit(1 << 20)
    src = {r['key']: r['original'] for r in read(a.source, 'latin-1')}
    tr = {r['key']: r['translation'] for r in read(a.translations, 'utf-8')}
    origs = set(src.values())
    out = ['master\tindex\toriginal_length\toriginal_sha256\ttranslation']
    for key in sorted(src):
        o = src[key]
        if key not in tr or not WORDS.fullmatch(o):
            continue
        if any(o != p and o in p for p in origs):
            continue  # 抽取時的滑動片段
        en, zh = re.split('[ @]', o), re.split('[ @]', tr[key].strip())
        if len(en) != len(zh) or any(w.isdigit() for w in en):
            continue
        for i, (w, z) in enumerate(zip(en, zh)):
            raw = w.encode('ascii')
            out.append(f'{key}\t{i}\t{len(raw)}\t{hashlib.sha256(raw).hexdigest()}\t{z}')
    a.out.write_text('\n'.join(out) + '\n', encoding='utf-8')


if __name__ == '__main__':
    main()
