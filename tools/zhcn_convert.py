#!/usr/bin/env python3
"""由繁中正式譯文產生簡中表（規格 015 §4）。在 cok-opencc:r1 內執行（tools/docker/opencc.Dockerfile）。

- 一般文字用 OpenCC tw2sp（台灣正體到大陸簡體，含詞彙轉換）。
- 專有名詞（glossary.tsv 的 person、place、race、monster、class，及 translit-names.tsv 的中文名）先換成佔位符，
  單獨用 t2s 逐字轉簡體，句子轉完再放回，詞彙轉換不改到譯名。
- 轉完套 zhcn-fixes.tsv（from、to、note）逐條修正兩岸用語。
- 檢查：每列 {…} 佔位符、\\n 數量不變；轉換後比繁中長的列列出（固定格數的介面字串可能放不下）。

輸出（寫回 --text）：<表>.zh-CN.tsv、composed-templates.zh-CN.tsv、menu-masters.zh-CN.tsv、map-titles.zh-CN.tsv、
translit-zh-CN/（音譯表，逐字轉簡體）。--check 只比對既有輸出是否最新，不寫檔。
"""
import argparse
import csv
import io
import re
import sys
from pathlib import Path

import opencc

LANG = 'zh-CN'
TABLES = ['ecl-text', 'ui-text', 'name', 'journal', 'walkthrough', 'map-guide', 'host-text']
TEXT_COLS = {'translation', 'title', 'body', 'label', 'summary'}
PROPER = {'person', 'place', 'race', 'monster', 'class'}
HOST_EXTRA = {'lang.name': '简中', 'journal.ref': ' %d 则%s', 'journal.refdot': '。'}
TOKEN = re.compile(r'\{[^}]*\}')

prose = opencc.OpenCC('tw2sp')
chars = opencc.OpenCC('t2s')


def read(path):
    with open(path, encoding='utf-8', newline='') as f:
        rows = list(csv.reader(f, delimiter='\t', quoting=csv.QUOTE_NONE))
    return rows[0], rows[1:]


def dump(header, rows):
    out = io.StringIO()
    w = csv.writer(out, delimiter='\t', quoting=csv.QUOTE_NONE, lineterminator='\n', escapechar=None)
    w.writerow(header)
    w.writerows(rows)
    return out.getvalue()


class Converter:
    def __init__(self, text):
        h, rows = read(text / 'glossary.tsv')
        terms = {r[h.index('zh')] for r in rows if r[h.index('category')] in PROPER and r[h.index('zh')]}
        h, rows = read(text / 'translit-names.tsv')
        terms |= {r[h.index('chinese')] for r in rows if r[h.index('chinese')]}
        self.terms = sorted(terms, key=len, reverse=True)  # 長的先換，避免短名吃掉長名的一部分
        self.fix = []
        fix = text / 'zhcn-fixes.tsv'
        if fix.exists():
            h, rows = read(fix)
            self.fix = [(r[h.index('from')], r[h.index('to')]) for r in rows]
        self.longer = []

    def prose(self, s):
        if not s:
            return s
        slots = []

        def hold(m):
            slots.append(chars.convert(m.group(0)))
            return chr(0xE000 + len(slots) - 1)
        pat = re.compile('|'.join(re.escape(t) for t in self.terms)) if self.terms else None
        held = pat.sub(hold, s) if pat else s
        out = prose.convert(held)
        out = re.sub('[-]', lambda m: slots[ord(m.group(0)) - 0xE000], out)
        for a, b in self.fix:
            out = out.replace(a, b)
        if TOKEN.findall(out) != TOKEN.findall(s) or out.count('\\n') != s.count('\\n'):
            sys.exit(f'佔位符或換行改變：{s!r} → {out!r}')
        if len(out) > len(s):
            self.longer.append((s, out))
        return out


def convert_table(conv, text, name, cols=TEXT_COLS, charwise=False):
    h, rows = read(text / name)
    idx = [i for i, c in enumerate(h) if c in cols]
    out = []
    for r in rows:
        r = list(r)
        for i in idx:
            if i < len(r):
                r[i] = chars.convert(r[i]) if charwise else conv.prose(r[i])
        out.append(r)
    return h, out


def build(text):
    conv = Converter(text)
    files = {}
    for t in TABLES:
        h, rows = convert_table(conv, text, f'{t}.zh-TW.tsv')
        if t == 'host-text':  # 語言名稱與手札編號格式由這裡給，不取繁中轉換的結果
            rows = [r for r in rows if r[0] not in HOST_EXTRA]
            rows += [[k, v] + [''] * (len(h) - 2) for k, v in HOST_EXTRA.items()]
        files[f'{t}.{LANG}.tsv'] = dump(h, rows)
    for stem in ('composed-templates', 'menu-masters'):
        files[f'{stem}.{LANG}.tsv'] = dump(*convert_table(conv, text, f'{stem}.tsv'))
    h, rows = read(text / 'map-geo.tsv')
    files[f'map-titles.{LANG}.tsv'] = dump(['map', 'title'], [[r[h.index('map')], conv.prose(r[h.index('title')])] for r in rows])
    # 音譯資料：譯音字逐字轉簡體；允許字集同樣轉換（檔名照 translit.Load 的固定名稱）
    sub = f'translit-{LANG}/'
    files[sub + 'translit-table.tsv'] = dump(*convert_table(conv, text, 'translit-table.tsv', {'zh', 'zh_female', 'zh_initial'}, True))
    files[sub + 'translit-names.tsv'] = dump(*convert_table(conv, text, 'translit-names.tsv', {'chinese'}, True))
    files[sub + 'translit-chars.zh-TW.tsv'] = dump(*convert_table(conv, text, 'translit-chars.zh-TW.tsv', {'translation'}, True))
    files[sub + 'translit-arpabet.tsv'] = (text / 'translit-arpabet.tsv').read_text(encoding='utf-8')
    return files, conv


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--text', type=Path, required=True)
    ap.add_argument('--check', action='store_true')
    ap.add_argument('--longer', type=Path, help='把轉換後變長的列寫到這個 TSV（審查用）')
    a = ap.parse_args()
    files, conv = build(a.text)
    stale = []
    for name, body in files.items():
        p = a.text / name
        if a.check:
            if not p.exists() or p.read_text(encoding='utf-8') != body:
                stale.append(name)
            continue
        p.parent.mkdir(exist_ok=True)
        p.write_text(body, encoding='utf-8')
    if a.longer:
        a.longer.write_text(dump(['zh-TW', 'zh-CN'], conv.longer), encoding='utf-8')
    print(f'簡中表 {len(files)} 份；轉換後變長 {len(conv.longer)} 列；保護專有名詞 {len(conv.terms)} 個')
    if stale:
        sys.exit('簡中表不是最新，重跑 tools/zhcn-convert.sh：' + '、'.join(stale))


if __name__ == '__main__':
    main()
