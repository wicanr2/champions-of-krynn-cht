import sys
batch, mfile, nfile, inp, out = sys.argv[1:6]
def load(p):
    d = {}
    try:
        for l in open(p, encoding='utf-8').read().split('\n'):
            if not l: continue
            parts = l.split('|')
            d[str(int(parts[0]))] = parts[1:]
    except FileNotFoundError:
        pass
    return d
m = load(mfile); n = load(nfile)
rows = [l.split('\t') for l in open(inp, encoding='utf-8').read().rstrip('\n').split('\n')][1:]
used = set(); res = ['key\ttranslation\tnote']
for r in rows:
    suf = str(int(r[0].split('.')[-1]))
    assert suf in m, ('missing', r[0])
    used.add(suf)
    note = n.get(suf, [''])[0]
    tr = m[suf][0]
    if len(m[suf]) > 1 and m[suf][1]: note = m[suf][1]
    res.append(f'{r[0]}\t{tr}\t{note}')
assert used == set(m), ('extra', set(m) - used)
open(out, 'w', encoding='utf-8').write('\n'.join(res) + '\n')
print(len(rows), 'rows;', sum(1 for r in res[1:] if r.split('\t')[1] == '-'), 'noise')
