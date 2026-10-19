#!/usr/bin/env python3
"""術語一致性檢查：術語表 ↔ 中文手冊 ↔ SSI 提示書譯文 ↔ 雜誌攻略 ↔ 地圖攻略點。

- 手冊：術語表 source 為 aNN.md／bNN.md 的列，zh 要出現在該頁；使用者定案改譯的列（note 有「手冊作「X」」）改查 X。
- 提示書（hint-*.md）：英文段出現術語表 en（最長優先、整詞），中文段要出現對應 zh。
- 舊譯名：術語表 note 記的異寫（另見、取代異譯、手冊作）不得出現在 --zh-files（產生後的攻略與地圖點）；
  異寫本身是另一個術語表譯名的不算。
- 雜誌對照表（--magazine-terms）：from 不得是術語表 zh，to 要含術語表 zh（from 等於 to 的保護列略過）。
- 地圖攻略點 label ≤ 8 字（規格 009 §2.1）。
有不一致時結束碼 1。
"""
import argparse
import csv
import re
import sys
from pathlib import Path

# 英文段出現這些 en 多半是一般詞或複合詞的一部分（Potion of Invisibility、Red Mage Scroll、Bout Hall），不視為專名
GENERIC = {'Fighter', 'Cleric', 'Thief', 'Ranger', 'Paladin', 'Knight', 'Mage', 'Magic-User', 'Gold', 'Steel', 'Human',
           'Exit', 'Yes', 'No', 'Look', 'Rest', 'Fix', 'Area', 'View', 'Cast', 'Use', 'Trade', 'Drop', 'Train',
           'Temple', 'Bank', 'Inn', 'Shop', 'Tavern', 'Guard', 'Guards', 'Day', 'Night', 'Light', 'Map', 'Wall', 'Door',
           'Good', 'Evil', 'Neutral', 'Lawful Good', 'Chaotic Evil', 'Ring', 'Wand', 'Rod', 'Staff', 'Potion', 'Scroll',
           'Bracers', 'Mail', 'Plate Mail', 'Chain Mail', 'Long Sword', 'Arrow', 'Arrows', 'Dart', 'Bow', 'Shield', 'Sling',
           'Dagger', 'Club', 'Hammer', 'Mace', 'Spear', 'Lance', 'Gem', 'Gems', 'Jewelry', 'Fear', 'Sleep', 'Hold Person',
           'Strength', 'Dexterity', 'Constitution', 'Intelligence', 'Wisdom', 'Charisma', 'Elf', 'Elves', 'Dwarf', 'Dwarves',
           'Ghost', 'Skeleton', 'Zombie', 'Wight', 'Wraith', 'Spectre', 'Shadow', 'Ghoul', 'Ghast', 'Mummy', 'Vampire',
           'Banded Mail', 'Ring Mail', 'Scale Mail', 'Leather Armor', 'Armor', 'Encamp', 'Save', 'Party', 'Combat',
           'Pool', 'Keep', 'Undead', 'Dragon', 'Special', 'Poison', 'Speed', 'Giant', 'Hall', 'Dead', 'Invisibility',
           'Mage Scroll'}


# 舊譯名字串在這些語境是一般詞，不是該術語：值是允許的上下文（正規式，須含該字串）
ALLOW = {
    '鬼怪': r'妖魔鬼怪',
    '黑暗之后塔克西絲': r'黑暗之后塔克西絲',  # 兩個術語表譯名並列（Dark Queen、Takhisis）
    '賣出': r'賣出',  # 介面短字（底列選單字數受限），術語表全名「變賣物品」
}


def rows(path):
    with open(path, encoding='utf-8', newline='') as f:
        return list(csv.DictReader(f, delimiter='\t', quoting=csv.QUOTE_NONE))


def strip_paren(s):
    return re.sub(r'（.*?）|\(.*?\)', '', s).strip()


def manual_forms(note):
    """note 的「手冊作「X」」→ [X]。"""
    return [strip_paren(m) for m in re.findall(r'手冊作「([^」]+)」', note or '')]


def variants(note):
    """note 記的舊譯名：另見、取代異譯、手冊作。"""
    out = manual_forms(note)
    for m in re.finditer(r'(?:另見|取代異譯)：([^；;]+)', note or ''):
        for v in re.split(r'[、，,]', m.group(1)):
            v = strip_paren(v)
            if v and not v.isascii():  # 頁碼註記（例 b17）不是譯名
                out.append(v)
    return out


def split_hint(text):
    en, zh, cur = [], [], None
    for line in text.splitlines():
        if line.startswith('## '):
            cur = {'## 英文原文': en, '## 中文': zh}.get(line.strip())
            continue
        if cur is not None:
            cur.append(line)
    return '\n'.join(en), '\n'.join(zh)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--glossary', required=True, type=Path)
    ap.add_argument('--hint-dir', type=Path)
    ap.add_argument('--manual-dir', type=Path)
    ap.add_argument('--zh-files', nargs='*', default=[], type=Path, help='產生後的中文檔，檢查舊譯名')
    ap.add_argument('--map-guide', type=Path)
    ap.add_argument('--magazine-terms', type=Path)
    a = ap.parse_args()
    gl = [r for r in rows(a.glossary) if r['en'] and r['zh']]
    zhs = {r['zh'] for r in gl}
    bad = 0

    if a.manual_dir:
        n = miss = decided = 0
        for r in gl:
            src = (r['source'] or '').strip()
            if not re.fullmatch(r'[ab]\d\d\.md', src):
                continue
            n += 1
            page = (a.manual_dir / src).read_text(encoding='utf-8')
            if r['zh'] in page:
                continue
            old = manual_forms(r['note'])
            if old and any(o in page for o in old):
                decided += 1
                continue
            miss += 1
            print(f'手冊\t{src}\t{r["en"]}\t{r["zh"]}\t該頁未見')
        print(f'# 手冊：{n} 列；{decided} 列為定案改譯（手冊舊譯在該頁）；{miss} 列對不上')
        bad += miss

    if a.hint_dir:
        terms = sorted(((r['en'], r) for r in gl if r['en'] not in GENERIC and re.search(r'[A-Za-z]', r['en'])),
                       key=lambda t: -len(t[0]))
        n = miss = 0
        for f in sorted(a.hint_dir.glob('hint-*.md')):
            en, zh = split_hint(f.read_text(encoding='utf-8'))
            rest = en
            for t, r in terms:
                pat = re.compile(r'(?<![A-Za-z])' + re.escape(t) + r"(?:'s|s)?(?![A-Za-z])")
                if not pat.search(rest):
                    continue
                rest = pat.sub(' ', rest)
                n += 1
                if r['zh'] not in zh:
                    used = [v for v in variants(r['note']) if v in zh]
                    miss += 1
                    print(f'提示書\t{f.name}\t{t}\t{r["zh"]}\t中文段未見；異寫：{"、".join(used) or "無"}')
        print(f'# 提示書：{n} 處術語，{miss} 處中文段未用術語表譯名')
        bad += miss

    if a.zh_files:
        old = {}
        for r in gl:
            for v in variants(r['note']):
                # 異寫本身是別的譯名、或是現行譯名的一部分（例 索斯 ⊂ 索斯王）不算舊譯名
                if len(v) >= 2 and v not in zhs and not any(v in z for z in zhs):
                    old.setdefault(v, r)
        miss = 0
        for f in a.zh_files:
            text = f.read_text(encoding='utf-8')
            for v, r in sorted(old.items(), key=lambda t: -len(t[0])):
                c = text.count(v) - (len(re.findall(ALLOW[v], text)) if v in ALLOW else 0)
                if c:
                    miss += 1
                    print(f'舊譯名\t{f.name}\t{v}\t→ {r["zh"]}（{r["en"]}）\t{c} 處')
        print(f'# 舊譯名：{miss} 處')
        bad += miss

    if a.magazine_terms and a.magazine_terms.exists():
        miss = 0
        zh_en = {}
        for r in gl:
            zh_en.setdefault(r["zh"], set()).add(r["en"].lower())
        for r in rows(a.magazine_terms):
            if r['from'] == r['to']:  # 保護列
                continue
            # from 是別的術語的譯名可以（例 幽靈騎士 是遊戲中文名，敵方單位 Death Knight 才換）
            if r['en'].lower() in zh_en.get(r['from'], ()):
                miss += 1
                print(f'雜誌對照\t{r["from"]}\t→ {r["to"]}\tfrom 已是術語表譯名')
            if r['to'] not in zhs and not any(len(z) >= 2 and z in r['to'] for z in zhs):
                miss += 1
                print(f'雜誌對照\t{r["from"]}\t→ {r["to"]}\tto 不在術語表')
        print(f'# 雜誌對照：{miss} 列有問題')
        bad += miss

    if a.map_guide:
        long_ = [r for r in rows(a.map_guide) if len(r['label']) > 8]
        for r in long_:
            print(f'地圖\t{r["map"]} {r["n"]}\t{r["label"]}\t超過 8 字')
        print(f'# 地圖攻略點：{len(long_)} 個 label 超過 8 字')
        bad += len(long_)
    sys.exit(1 if bad else 0)


if __name__ == '__main__':
    main()
