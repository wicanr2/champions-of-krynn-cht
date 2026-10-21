import sys, csv
base = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
n = sys.argv[1]
mp = {}
for line in open(base + 'tmp/batch-101/map-%s.txt' % n, encoding='utf-8'):
    line = line.rstrip('\n')
    if not line.strip() or line.startswith('#'):
        continue
    parts = line.split(' ||| ')
    o = parts[0]
    t = parts[1]
    note = parts[2] if len(parts) > 2 else ''
    assert o not in mp, o
    mp[o] = (t, note)
rows = list(csv.reader(open(base + 'batches/batch-%s.tsv' % n, encoding='utf-8', newline=''), delimiter='\t', quoting=csv.QUOTE_NONE))
hdr = rows[0]
used = set()
out = ['key\ttranslation\tnote']
unm = []
for r in rows[1:]:
    k, o = r[0], r[1]
    s = o.strip()
    if s in mp:
        used.add(s)
        t, note = mp[s]
        out.append('%s\t%s\t%s' % (k, t, note))
    else:
        out.append('%s\t-\t' % k)
        unm.append(s)
missing = set(mp) - used
print('mapped unused:', missing)
open(base + 'done/batch-%s.tsv' % n, 'w', encoding='utf-8', newline='').write('\n'.join(out) + '\n')
print(n, 'rows', len(rows) - 1, 'noise', len(unm))
if '-v' in sys.argv:
    for u in unm: print('  -', repr(u))
