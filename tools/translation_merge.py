#!/usr/bin/env python3
"""驗證批次譯文並合併為正式 TSV（移植自拯救地球 tools/ecl_translation_merge.py，只支援 zh-TW）。

每個 done/batch-NN.tsv 的 key 集合與順序必須與 batches/batch-NN.tsv 完全相同；
譯文須單行、NFC、非空、無頭尾空白、無全形空白、Big5 可編碼（開發字型為倚天）。
`-` 表示該列是雜訊，不進正式檔。正式檔只收 catalog 內的 key；既有譯文除非批次有新值，否則保留。
note 欄非空的列另列出，供人工確認。
"""
import argparse
import csv
import sys
import unicodedata
from pathlib import Path

SOURCE = 'batch-editorial'


def read_tsv(path, header):
    rows = list(csv.reader(path.open(encoding='utf-8'), delimiter='\t', quoting=csv.QUOTE_NONE))
    if not rows or rows[0][:len(header)] != header:
        raise SystemExit(f'{path}: 表頭應為 {header}')
    return rows[1:]


def check(text, where):
    errs = []
    if not text:
        errs.append('空白')
    if any(c in text for c in '\t\r\n'):
        errs.append('含 tab／換行')
    if unicodedata.normalize('NFC', text) != text:
        errs.append('非 NFC')
    if text != text.strip(' '):
        errs.append('含頭尾空白')
    if '　' in text:
        errs.append('含全形空白')
    if any(unicodedata.category(c) == 'Cc' for c in text):
        errs.append('控制字元')
    try:
        text.encode('big5')
    except UnicodeEncodeError as e:
        errs.append(f'非 Big5：{text[e.start:e.end]!r}')
    return [f'{where}：{e}' for e in errs]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--events', required=True, type=Path)
    ap.add_argument('--batches', required=True, type=Path)
    ap.add_argument('--done', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    ap.add_argument('--check-only', action='store_true')
    a = ap.parse_args()
    catalog = {r[0] for r in read_tsv(a.events, ['event_key', 'original_length', 'original_sha256', 'sources'])}
    merged = {}
    if a.out.exists():
        for k, t, s in read_tsv(a.out, ['key', 'translation', 'source']):
            merged[k] = (t, s)
    errors, notes, garbage, done_batches = [], [], 0, 0
    for done in sorted(a.done.glob('batch-[0-9]*.tsv')):
        want = [r[0] for r in read_tsv(a.batches / done.name, ['key', 'original'])]
        got = read_tsv(done, ['key', 'translation'])
        if [r[0] for r in got] != want:
            errors.append(f'{done.name}：key 集合或順序與輸入不同（{len(got)}／{len(want)}）')
            continue
        done_batches += 1
        for r in got:
            k, t = r[0], (r[1] if len(r) > 1 else '')
            if len(r) > 2 and r[2]:
                notes.append(f'{done.name}:{k}：{r[2]}')
            if t == '-':
                garbage += 1
                merged.pop(k, None)
                continue
            if k not in catalog:
                errors.append(f'{done.name}：{k} 不在 catalog')
                continue
            e = check(t, f'{done.name}:{k}')
            if e:
                errors.extend(e)
                continue
            merged[k] = (t, SOURCE)
    for e in errors:
        print(e, file=sys.stderr)
    for n in notes:
        print('註記', n, file=sys.stderr)
    print(f'批次 {done_batches}，譯文 {len(merged)}，雜訊 {garbage}，註記 {len(notes)}，錯誤 {len(errors)}', file=sys.stderr)
    if errors:
        sys.exit(1)
    if not a.check_only:
        lines = ['key\ttranslation\tsource'] + [f'{k}\t{t}\t{s}' for k, (t, s) in sorted(merged.items())]
        a.out.write_text('\n'.join(lines) + '\n', encoding='utf-8')


if __name__ == '__main__':
    main()
