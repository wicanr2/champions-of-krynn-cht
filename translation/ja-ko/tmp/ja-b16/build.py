import sys, os
BASE = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako'
TMP = BASE + '/tmp/ja-b16'

def rd(p):
    with open(p, encoding='utf-8', newline='') as f:
        t = f.read()
    return [l.split('\t') for l in t.split('\n') if l != '']

trial = {x[0]: x[2] for x in rd(BASE + '/trial-ja.tsv')[1:]}
gloss = {}
for x in rd(BASE + '/glossary-ja.tsv')[1:]:
    gloss.setdefault(x[0], x[1])

ok = True
for b in sys.argv[1:]:
    inp = rd(f'{BASE}/batches/{b}.tsv')[1:]
    with open(f'{TMP}/{b}.txt', encoding='utf-8', newline='') as f:
        lines = [l for l in f.read().split('\n') if l != '']
    if len(lines) != len(inp):
        print(b, 'LINE COUNT MISMATCH', len(lines), len(inp)); ok = False; continue
    out = ['key\ttranslation\tnote']
    for i, (row, ln) in enumerate(zip(inp, lines), 1):
        tr, _, note = ln.partition('@@')
        out.append('\t'.join([row[0], tr, note]))
        orig = row[1]
        if '\t' in tr or '\n' in tr or '\r' in tr or '\t' in note:
            print(b, i, 'CTRL'); ok = False
        if tr != tr.strip():
            print(b, i, 'WS'); ok = False
        if not tr:
            print(b, i, 'EMPTY'); ok = False
        if len(tr) > len(orig):
            print(b, i, 'LONG', len(tr), len(orig), tr); ok = False
        if row[0] in trial and trial[row[0]] != tr:
            print(b, i, 'TRIAL DIFF', tr, trial[row[0]]); ok = False
        if row[0].startswith('ui.') and orig.strip() in gloss and gloss[orig.strip()] != tr:
            print(b, i, 'GLOSS DIFF', repr(orig), tr, gloss[orig.strip()])
        # fullwidth ASCII check
        for ch in tr:
            if 0xFF01 <= ord(ch) <= 0xFF5E and ch not in '！？：（）':
                print(b, i, 'FULLWIDTH', ch, tr); ok = False
    with open(f'{BASE}/done-ja/{b}.tsv', 'w', encoding='utf-8', newline='') as f:
        f.write('\n'.join(out) + '\n')
    # verify
    chk = rd(f'{BASE}/done-ja/{b}.tsv')
    assert chk[0] == ['key', 'translation', 'note']
    assert len(chk) == len(inp) + 1, (len(chk), len(inp))
    for r, o in zip(chk[1:], inp):
        assert len(r) == 3, r
        assert r[0] == o[0]
    print(b, 'rows', len(chk) - 1)
print('ALL OK' if ok else 'PROBLEMS')
