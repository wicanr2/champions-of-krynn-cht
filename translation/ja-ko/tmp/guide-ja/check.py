import csv, re

BASE = '/home/anr2/cht/golden_box/champions_of_krynn'
OUT = BASE + '/workplace/tr-jako/aux-ja/'

def rd(p):
    with open(p, encoding='utf-8', newline='') as f:
        return [l.rstrip('\n') for l in f]

bad = re.compile('[０-９Ａ-Ｚａ-ｚ\U0001f000-\U0001faff☀-➿—―]')
hw_kana = re.compile('[｡-ﾟ]')
ok = True
for name, ncol, orig in (('walkthrough.tsv', 4, 'walkthrough.zh-TW.tsv'), ('map-guide.tsv', 7, 'map-guide.zh-TW.tsv')):
    ja = rd(OUT + name)
    zh = rd(BASE + '/text/' + orig)
    print(name, 'lines', len(ja), 'orig', len(zh))
    if len(ja) != len(zh):
        ok = False
    for i, (a, b) in enumerate(zip(ja, zh), 1):
        fa = a.split('\t'); fb = b.split('\t')
        if len(fa) != ncol or len(fb) != ncol:
            print('cols', name, i, len(fa), len(fb)); ok = False; continue
        for j, f in enumerate(fa):
            if f != f.strip():
                print('ws', name, i, j); ok = False
            if bad.search(f) or hw_kana.search(f):
                print('badchar', name, i, j, bad.findall(f)); ok = False
            if '\r' in f or '\x0b' in f:
                print('ctl', name, i, j); ok = False
        if i == 1:
            if fa != fb:
                print('header differs'); ok = False
            continue
        if name == 'walkthrough.tsv':
            if fa[0] != fb[0]:
                print('chapter', i); ok = False
            if not re.fullmatch(r'ソフトワールド誌 \d+号 p\.[\d-]+', fa[3]):
                print('source fmt', i, fa[3]); ok = False
            if re.sub(r'.* (\d+) 期 p\.(.*)', r'\1 \2', fb[3]) != re.sub(r'.* (\d+)号 p\.(.*)', r'\1 \2', fa[3]):
                print('source page mismatch', i); ok = False
        else:
            for k in (0, 1, 2, 3, 6):
                if fa[k] != fb[k]:
                    print('untouched col changed', i, k); ok = False
            if len(fa[4]) > 8:
                print('label too long', i, fa[4], len(fa[4])); ok = False
            if not fa[4] or not fa[5]:
                print('empty', i); ok = False
            # same coordinates in summary as source
            if re.findall(r'[（(](\d+)[,，](\d+)[）)]', fb[5]) != re.findall(r'[（(](\d+)[,，](\d+)[）)]', fa[5]):
                print('coord mismatch', i, re.findall(r'\((\d+)', fb[5])[:0]); ok = False
print('ALL OK' if ok else 'PROBLEMS')
