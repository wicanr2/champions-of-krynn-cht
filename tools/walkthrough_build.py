#!/usr/bin/env python3
"""由雜誌攻略轉錄（reference/walkthrough-zh/softworld-*.md）產生 text/walkthrough.zh-TW.tsv（規格 008）。

- `## 第 N 章 標題` 開新章；正文段落（空行分隔）照錄，跨頁接續。
- 地圖圖例另成一段「地圖：名稱」：`> 地圖：名稱。…`、短的 `> 圖說：名稱` 開新圖，其後的引用列是圖例條目；
  `--map-pages` 指定的雜誌頁整頁都是地圖（第 30 期轉錄成一般列），頁首第一列是圖名、`N.` 列是條目。
- 刪去頁首重複的雜誌標題、作者署名、待續、HTML 註解。
輸出欄位 chapter（1 起連號）、title、body（段落以字面 \\n 分隔）、source（期數與雜誌頁碼）。
"""
import argparse
import re
from pathlib import Path

PAGE = re.compile(r'<!-- page: \S+（雜誌頁碼 (\d+)） -->')
CHAPTER = re.compile(r'^## (第 ?\d+ ?章)\s*(.+)$')
SKIP = re.compile(r'^(克萊恩英豪.*|當正義為光明劃下休止符|完全攻略.*|攻略完結篇|完結篇|／.*|待續.*)$')
ITEM = re.compile(r'^\d+[.．]')
# 遊戲字型是 Big5：雜誌的異體字換成標準字，全形空白換半形（轉錄檔保留原貌）
BIG5 = str.maketrans({'爲': '為', '强': '強', '眞': '真', '裏': '裡', '啓': '啟', '‧': '·', '\u3000': ' ', '—': '－', '–': '-'})


def load_terms(path):
    """雜誌譯名 → 術語表譯名（text/walkthrough-terms.tsv：from、to、en、reason）。轉錄檔保留原貌，只在產生面板文字時換。

    from 等於 to 的列用來保護較長的字串（例「幽靈騎士大陸全圖」是雜誌圖名，不換成死亡騎士）。
    """
    if not path or not path.exists():
        return {}
    out = {}
    for line in path.read_text(encoding='utf-8').splitlines()[1:]:
        c = line.split('\t')
        if len(c) >= 2 and c[0]:
            out[c[0].translate(BIG5)] = c[1]
    return out


def apply_terms(text, terms):
    """一次掃過、最長者優先，換過的字不再被換。單一註記的括號（例「卡拉瑪（Kalaman）」）保留原文；含逗號的括號是清單，照換。"""
    if not terms:
        return text
    pat = re.compile('( ?)(' + '|'.join(re.escape(k) for k in sorted(terms, key=len, reverse=True)) + ')( ?)')

    def sub(m):
        # 雜誌在英文名前後留的空白：換成中文後，旁邊不是英數字就去掉
        s = m.string
        lead = m.group(1) if m.start() > 0 and s[m.start() - 1].isascii() and s[m.start() - 1].isalnum() else ''
        tail = m.group(3) if m.end() < len(s) and s[m.end()].isascii() and s[m.end()].isalnum() else ''
        return lead + terms[m.group(2)] + tail
    parts = re.split(r'(（[^（），]*）)', text)
    return ''.join(p if i % 2 else pat.sub(sub, p) for i, p in enumerate(parts))


class Section:
    def __init__(self, title, issue, page):
        self.title, self.issue, self.pages, self.paras, self.cur = title, issue, [page], [], []

    def add_page(self, page):
        if page not in self.pages:
            self.pages.append(page)

    def line(self, text):
        self.cur.append(text)

    def end_para(self, force=False):
        # 跨欄、跨頁的斷句：段落沒有以句尾標點結束就接著下一列（轉錄時頁標記前後有空行）
        if self.cur and (force or ''.join(self.cur)[-1] in '。！？」』…）)：'):
            self.paras.append(''.join(self.cur))
            self.cur = []

    def source(self):
        p = self.pages
        span = f'p.{p[0]}' if len(p) == 1 else f'p.{p[0]}-{p[-1]}'
        return f'軟體世界 {self.issue} 期 {span}'


def build(path, issue, map_pages):
    chapters, maps = [], []
    chap = mp = None
    page = 0
    map_page_first = False
    for raw in path.read_text(encoding='utf-8').splitlines():
        line = re.sub(r'<!--(?! page).*?-->', '', raw).strip()
        if m := PAGE.match(line):
            page = int(m.group(1))
            for s in (chap, mp):
                if s:
                    s.end_para()
            map_page_first = page in map_pages
            if chap and not map_page_first:
                chap.add_page(page)
            continue
        if not line:
            if chap:
                chap.end_para()
            if mp:
                mp.end_para()
            continue
        if SKIP.match(line):
            continue
        if m := CHAPTER.match(line):
            if chap:
                chap.end_para()
            chap = Section(f'{m.group(1).replace(" ", "")}　{m.group(2).strip()}', issue, page)
            chapters.append(chap)
            continue
        if page in map_pages:  # 整頁地圖（一般列）
            if map_page_first and not line.startswith('>'):
                mp = Section(f'地圖：{line}', issue, page)
                maps.append(mp)
                map_page_first = False
                continue
            text = line.lstrip('> ').removeprefix('圖說：').strip()
            if mp and text and not text.startswith('KEY'):
                mp.line(text)
                mp.end_para(force=True)
            continue
        if line.startswith('>'):
            text = line.lstrip('> ').strip()
            if m := re.match(r'^地圖：([^。]+)', text):
                mp = Section(f'地圖：{m.group(1)}', issue, page)
                maps.append(mp)
                continue
            body = text.removeprefix('圖說：').strip()
            if text.startswith('圖說：') and not ITEM.match(body) and len(body) <= 12:
                mp = Section(f'地圖：{body}', issue, page)
                maps.append(mp)
                continue
            if mp and body and not body.startswith('KEY'):
                mp.line(body)
                mp.end_para(force=True)
            continue
        if chap:
            chap.line(line)
    for s in chapters + maps:
        s.end_para(force=True)
    return chapters, maps


HINT_ORDER = ['intro', 'overland', 'gargath', 'kalaman', 'vingaard', 'throtl', 'cekos', 'cekos-cont', 'turef', 'quazle',
              'slaveship', 'hct', 'tunnels', 'dragonpit', 'cerberus', 'cerberus-cont', 'graveyard', 'dulcimer', 'voicewood',
              'dargaard1', 'dargaard2', 'dargaard3', 'shipwreck', 'cursedvillage', 'funhouse', 'fathertrees', 'challenge',
              'monsters', 'journal', 'magicitems']


def hint_sections(hint_dir):
    """SSI 提示書翻譯（hint-<代號>.md 的「## 中文」段）；續篇（-cont）併入前一段。"""
    out = []
    for code in HINT_ORDER:
        f = hint_dir / f'hint-{code}.md'
        if not f.exists():
            continue
        lines = f.read_text(encoding='utf-8').splitlines()
        title = lines[0].lstrip('# ').split('（')[0].strip() if lines else code
        body, on, cur = [], False, []
        for line in lines[1:]:
            if line.startswith('## '):
                on = line.strip() == '## 中文'
                continue
            if not on:
                continue
            t = line.strip()
            if not t:
                if cur:
                    body.append(''.join(cur)); cur = []
                continue
            if t.startswith('#'):
                if cur:
                    body.append(''.join(cur)); cur = []
                body.append(f'【{t.lstrip("# ").strip()}】')
                continue
            if re.match(r'^(\d+|[A-Z])[.．]', t) or t.startswith(('- ', '* ')):
                if cur:
                    body.append(''.join(cur)); cur = []
                body.append(t.lstrip('-* ').strip())
                continue
            cur.append(t.replace('**', ''))
        if cur:
            body.append(''.join(cur))
        body = [b.replace('**', '') for b in body if b]
        if code.endswith('-cont') and out:
            out[-1][1].extend(body)
        else:
            out.append([f'提示書：{title}', body])
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--src', required=True, type=Path, help='轉錄目錄')
    ap.add_argument('--out', required=True, type=Path)
    ap.add_argument('--hint-dir', type=Path, help='SSI 提示書翻譯（reference/hintbook-zh/）')
    ap.add_argument('--terms', type=Path, help='雜誌譯名對照（text/walkthrough-terms.tsv）')
    a = ap.parse_args()
    terms = load_terms(a.terms)
    plan = [('softworld-16.md', 16, set()), ('softworld-17.md', 17, set()), ('softworld-18.md', 18, set())]
    rows = ['chapter\ttitle\tbody\tsource']
    n = 0
    for name, issue, map_pages in plan:
        chapters, maps = build(a.src / name, issue, map_pages)
        for s in chapters + maps:
            body = apply_terms('\\n'.join(p.replace('\t', ' ') for p in s.paras if p).translate(BIG5), terms)
            if not body:
                continue
            n += 1
            rows.append(f'{n}\t{apply_terms(s.title.translate(BIG5), terms)}\t{body}\t{s.source()}')
    if a.hint_dir:
        for title, body in hint_sections(a.hint_dir):
            text = '\\n'.join(p.replace('\t', ' ') for p in body).translate(BIG5)
            if text:
                n += 1
                rows.append(f'{n}\t{title.translate(BIG5)}\t{text}\tSSI 官方提示書')
    a.out.write_text('\n'.join(rows) + '\n', encoding='utf-8')
    print(f'{n} 段寫到 {a.out}')


if __name__ == '__main__':
    main()
