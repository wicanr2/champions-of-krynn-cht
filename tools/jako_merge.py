#!/usr/bin/env python3
"""合併日文、韓文翻譯批次成正式譯文表（規格 015 §5）。

輸入（workplace/tr-jako）：batches/bNN.tsv（key、original…）、done-<語言>/bNN.tsv（key、translation、note）、
glossary-<語言>.tsv（en、target、basis、note）、aux-<語言>/ 的 composed-templates、menu-masters、host-text、
map-titles、journal、walkthrough、map-guide。
輸出（--text）：ecl-text、ui-text、name、composed-templates、menu-masters、host-text、map-titles、journal、
walkthrough、map-guide 各 <表>.<語言>.tsv。

檢查（任一不過就失敗，不寫檔）：每批 key 與順序同輸入；譯文無 tab、頭尾無空白；佔位符 {…} 與繁中一致；
ECL／介面譯文格數不超過原文字元數；選單母串每詞不超過原詞格數；map.dirs 四個字。
名稱表依 name_catalog.py 的規則（monster、person、place、item、spell，原形與全大寫都收雜湊），
同一雜湊有兩種譯名時取術語表中先出現的一筆，並列在輸出。
"""
import argparse
import csv
import hashlib
import re
import sys
from pathlib import Path

TOKEN = re.compile(r'\{[^}]*\}')
NAME_CATS = {'monster', 'person', 'place', 'item', 'spell'}
JOURNAL_REF = {'ja': ('%d項%s', '。'), 'ko': ('%d항%s', '.')}  # 日文譯文收在「手記第」、韓文收在「…제」


def read(p):
    with open(p, encoding='utf-8', newline='') as f:
        rows = list(csv.reader(f, delimiter='\t', quoting=csv.QUOTE_NONE))
    return rows[0], rows[1:]


def dicts(p):
    h, rows = read(p)
    return [dict(zip(h, r + [''] * (len(h) - len(r)))) for r in rows]


def dump(header, rows):
    return '\n'.join('\t'.join(r) for r in [header] + rows) + '\n'


def fail(msg):
    sys.exit(f'jako_merge：{msg}')


FOREIGN = {'ko': re.compile(r'[\u3040-\u30ff\u3400-\u9fff]'),  # 韓文譯文不該有假名或漢字
           'ja': re.compile(r'[\u1100-\u11ff\uac00-\ud7af]')}  # 日文譯文不該有韓文
LANG = None
BAD = []


def clean(s, where, pad=False):
    if '\t' in s or '\n' in s or (not pad and s != s.strip()):
        fail(f'{where}：譯文有 tab、換行或頭尾空白：{s!r}')
    if LANG and FOREIGN[LANG].search(s):
        BAD.append(f'{where}：混入其他文字：{s}')
    return s


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--lang', choices=('ja', 'ko'), required=True)
    ap.add_argument('--work', type=Path, required=True)
    ap.add_argument('--text', type=Path, required=True)
    a = ap.parse_args()
    global LANG
    L, W, T = a.lang, a.work, a.text
    LANG = L
    files, report = {}, []

    zh = {}
    for t in ('ecl-text', 'ui-text'):
        for r in dicts(T / f'{t}.zh-TW.tsv'):
            zh[r['key']] = r['translation']
    out = {'ecl': [], 'ui': []}
    seen = set()
    for b in sorted((W / 'batches').glob('b*.tsv')):
        src = dicts(b)
        done = W / f'done-{L}' / b.name
        if not done.exists():
            fail(f'缺 {done}')
        got = dicts(done)
        if [r['key'] for r in got] != [r['key'] for r in src]:
            fail(f'{done.name}：key 或順序與輸入不同')
        for s, g in zip(src, got):
            tr = clean(g['translation'], f'{done.name} {s["key"]}')
            if tr in ('', '-'):
                fail(f'{done.name} {s["key"]}：沒有譯文')
            if TOKEN.findall(tr) != TOKEN.findall(zh[s['key']]):
                fail(f'{done.name} {s["key"]}：佔位符與繁中不同')
            if len(tr) > len(s['original']):
                # 介面字串（路徑 A、C 固定格數）是硬限制；ECL 劇情由路徑 B 在視窗內換行，只列警告
                msg = f'{done.name} {s["key"]}：譯文 {len(tr)} 格超過原文 {len(s["original"])} 字元'
                if s['key'].startswith('ui.'):
                    fail(msg)
                report.append('警告 ' + msg)
            out['ecl' if s['key'].startswith('ecl.') else 'ui'].append([s['key'], tr, 'machine'])
            seen.add(s['key'])
    if seen != set(zh):
        fail(f'譯文 key 與繁中不一致：少 {len(set(zh) - seen)}、多 {len(seen - set(zh))}')
    files[f'ecl-text.{L}.tsv'] = dump(['key', 'translation', 'source'], out['ecl'])
    files[f'ui-text.{L}.tsv'] = dump(['key', 'translation', 'source'], out['ui'])

    # 名稱表：術語表 en → 目標語
    gl = dicts(T / 'glossary.tsv')
    gt = dicts(W / f'glossary-{L}.tsv')
    if [r['en'] for r in gl] != [r['en'] for r in gt]:
        fail(f'glossary-{L}.tsv 的 en 欄與 glossary.tsv 不同')
    names = {}
    for g, t in zip(gl, gt):
        en, tgt = g['en'].strip(), clean(t['target'], f'glossary-{L} {g["en"]}')
        if g['category'] not in NAME_CATS or not en or not tgt:
            continue
        for form in {en, en.upper()}:
            raw = form.encode('ascii')
            h = hashlib.sha256(raw).hexdigest()
            if h in names and names[h] != tgt:
                report.append(f'名稱衝突 {form}：取 {names[h]}，捨 {tgt}')
                continue
            names[h] = tgt
    ev = {r['event_key'] for r in dicts(T / 'name-events.tsv')}
    rows = [[f'name.{h[:12]}', v, 'glossary'] for h, v in sorted(names.items())]
    if {r[0] for r in rows} != ev:
        fail('名稱表的 key 與 name-events.tsv 不同（術語表的 en 或分類變了？）')
    files[f'name.{L}.tsv'] = dump(['key', 'translation', 'source'], rows)

    aux = W / f'aux-{L}'
    # 組句模板
    zt, zrows = read(T / 'composed-templates.tsv')
    at, arows = read(aux / 'composed-templates.tsv')
    if [r[0] for r in arows] != [r[0] for r in zrows]:
        fail('composed-templates：template 欄與繁中不同')
    for z, x in zip(zrows, arows):
        clean(x[1], f'composed-templates {x[0]}', pad=True)  # 模板可帶對齊用的頭尾空白
        if sorted(TOKEN.findall(x[1])) != sorted(TOKEN.findall(z[1])):
            fail(f'composed-templates {x[0]!r}：佔位符與繁中不同')
    files[f'composed-templates.{L}.tsv'] = dump(zt, [[x[0], x[1]] + z[2:] for z, x in zip(zrows, arows)])
    # 選單母串：沿用繁中表的 master、index、長度、雜湊，換 translation
    zm = dicts(T / 'menu-masters.tsv')
    am = dicts(aux / 'menu-masters.tsv')
    if [(r['master'], r['index']) for r in am] != [(r['master'], r['index']) for r in zm]:
        fail('menu-masters：master／index 與繁中不同')
    mrows = []
    for z, x in zip(zm, am):
        tr = clean(x['translation'], f'menu-masters {z["master"]}', pad=True)
        if len(tr) > int(z['original_length']):
            fail(f'menu-masters {z["master"]}#{z["index"]}：{tr!r} 超過 {z["original_length"]} 格')
        mrows.append([z['master'], z['index'], z['original_length'], z['original_sha256'], tr])
    files[f'menu-masters.{L}.tsv'] = dump(['master', 'index', 'original_length', 'original_sha256', 'translation'], mrows)
    # 前端文字
    hz = dicts(T / 'host-text.zh-TW.tsv')
    hx = {r['key']: r for r in dicts(aux / 'host-text.tsv')}
    hrows = []
    for z in hz:
        if z['key'] in ('journal.ref', 'journal.refdot'):
            continue
        x = hx.get(z['key'])
        if x is None:
            fail(f'host-text 缺 {z["key"]}')
        tr = clean(x['translation'], f'host-text {z["key"]}')
        if sorted(TOKEN.findall(tr)) != sorted(TOKEN.findall(z['translation'])):
            fail(f'host-text {z["key"]}：佔位符與繁中不同')
        hrows.append([z['key'], tr, z.get('note', '')])
    if len([c for c in hx['map.dirs']['translation']]) != 4:
        fail('host-text map.dirs 要四個字')
    ref, dot = JOURNAL_REF[L]
    hrows += [['journal.ref', ref, '規格 015：手札編號補字'], ['journal.refdot', dot, '規格 015']]
    files[f'host-text.{L}.tsv'] = dump(['key', 'translation', 'note'], hrows)
    # 地圖名、手札、攻略、地圖攻略點
    mt = dicts(aux / 'map-titles.tsv')
    if [r['map'] for r in mt] != [r['map'] for r in dicts(T / 'map-titles.en.tsv')]:
        fail('map-titles：map 欄與英文表不同')
    files[f'map-titles.{L}.tsv'] = dump(['map', 'title'], [[r['map'], clean(r['title'], 'map-titles')] for r in mt])
    for name, keycol, cols in (('journal', 'entry', ('entry', 'title', 'body', 'source')),
                               ('walkthrough', 'chapter', ('chapter', 'title', 'body', 'source')),
                               ('map-guide', 'n', ('map', 'n', 'x', 'y', 'label', 'summary', 'source'))):
        ref_rows = dicts(T / f'{name}.zh-TW.tsv')
        got = dicts(aux / f'{name}.tsv')
        if [(r.get('map', ''), r[keycol]) for r in got] != [(r.get('map', ''), r[keycol]) for r in ref_rows]:
            fail(f'{name}：編號或順序與繁中不同')
        for r in got:
            for c in cols:
                if '\t' in r[c]:
                    fail(f'{name} {r[keycol]}：{c} 含 tab')
        if name == 'map-guide':
            for r in got:
                if len(r['label']) > 8:
                    fail(f'map-guide {r["map"]}#{r["n"]}：label 超過 8 格')
        files[f'{name}.{L}.tsv'] = dump(list(cols), [[r[c] for c in cols] for r in got])

    for name in ('journal', 'walkthrough', 'map-guide'):
        for line in files[f'{name}.{L}.tsv'].split('\n')[1:]:
            f = line.split('\t')
            text = ' '.join(f[1:3]) if name != 'map-guide' else ' '.join(f[4:6])  # 標題與內文；地圖點為 label、summary
            if FOREIGN[L].search(text):
                BAD.append(f'{name}：混入其他文字：{line[:60]}')
    if BAD:
        fail('\n'.join(BAD[:40]) + f'\n共 {len(BAD)} 處')
    for n, body in files.items():
        (T / n).write_text(body, encoding='utf-8')
    for line in report:
        print(line)
    print(f'{L}：{len(files)} 份表；ECL {len(out["ecl"])}、介面 {len(out["ui"])}、名稱 {len(rows)}')


if __name__ == '__main__':
    main()
