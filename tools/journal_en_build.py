#!/usr/bin/env python3
"""組英文手札表（規格 015 §3.3）：合併 workplace/journal/en/*.tsv 的逐字轉錄成 journal.en.tsv。

輸入每列 kind（journal、tavern）、n、text、note；輸出 entry（手札 N、酒店傳聞 1000+N）、title、body、source，
與 journal.zh-TW.tsv 同欄。編號要齊（手札 1–88、酒店傳聞 1–57），重複或缺漏就失敗。
輸出是手冊原文，只放 workplace 與本機 text/（.gitignore），打包時才放進發行包。
"""
import argparse
import csv
import sys
from pathlib import Path

WANT = {'journal': range(1, 89), 'tavern': range(1, 58)}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--src', type=Path, required=True)
    ap.add_argument('--out', type=Path, required=True)
    a = ap.parse_args()
    got = {}
    for p in sorted(a.src.glob('*.tsv')):
        with open(p, encoding='utf-8', newline='') as f:
            for r in csv.DictReader(f, delimiter='\t', quoting=csv.QUOTE_NONE):
                k = (r['kind'], int(r['n']))
                if k in got:
                    sys.exit(f'{p.name}：{k} 重複（另見 {got[k][1]}）')
                got[k] = (r['text'].strip(), p.name)
    missing = [f'{k}{n}' for k, ns in WANT.items() for n in ns if (k, n) not in got]
    if missing:
        sys.exit('缺：' + '、'.join(missing))
    rows = []
    for kind, ns in WANT.items():
        for n in ns:
            text = got[(kind, n)][0]
            if text in ('', '[map]'):  # 只有地圖的條目（同繁中「本則為地圖，見原版手札」）
                text = '(This entry is a map. See the printed journal.)'
            text = text.replace('\u2014', '--')  # 英文字型只有可列印 ASCII：長破折號照打字機慣例寫兩個連字號
            if any(not (' ' <= c <= '~') for c in text.replace('\\n', '')):
                sys.exit(f'{kind}{n} 含可列印 ASCII 以外的字元')
            if '\t' in text:
                sys.exit(f'{kind}{n} 含 tab')
            entry, title = (n, f'Journal Entry {n}') if kind == 'journal' else (1000 + n, f'Tavern Tale {n}')
            rows.append(f'{entry}\t{title}\t{text}\tjournal-pdf')
    a.out.write_text('entry\ttitle\tbody\tsource\n' + '\n'.join(rows) + '\n', encoding='utf-8')
    print(f'{len(rows)} 則 → {a.out}')


if __name__ == '__main__':
    main()
