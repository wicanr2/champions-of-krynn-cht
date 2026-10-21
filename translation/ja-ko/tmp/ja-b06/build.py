import csv, sys, unicodedata
base = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
b = sys.argv[1]
src = list(csv.reader(open(base + 'batches/%s.tsv' % b, encoding='utf-8'), delimiter='\t', quoting=csv.QUOTE_NONE))[1:]
lines = [l.rstrip('\n') for l in open(base + 'tmp/ja-b06/%s.txt' % b, encoding='utf-8') if l.strip()]
out = []
for l in lines:
    parts = l.split(' ## ')
    k = 'ecl.' + parts[0].strip()
    tr = parts[1].strip()
    note = parts[2].strip() if len(parts) > 2 else ''
    out.append((k, tr, note))
ok = True
if len(out) != len(src):
    print('COUNT MISMATCH', len(out), len(src)); ok = False
sk = [r[0] for r in src]
ok_k = [o[0] for o in out]
for i, (a, c) in enumerate(zip(sk, ok_k)):
    if a != c:
        print('KEY MISMATCH at', i, a, c); ok = False; break
sd = {r[0]: r for r in src}
missing = [k for k in sk if k not in ok_k]
print('missing', missing[:5])
for k, tr, note in out:
    if '\t' in tr or '\n' in tr or tr != tr.strip() or not tr:
        print('FORMAT', k, repr(tr)); ok = False
    o = sd.get(k)
    if o and len(tr) > len(o[1]):
        print('LEN', k, len(tr), len(o[1]), repr(o[1]), tr); ok = False
if ok:
    with open(base + 'done-ja/%s.tsv' % b, 'w', encoding='utf-8', newline='') as f:
        f.write('key\ttranslation\tnote\n')
        for k, tr, note in out:
            f.write('%s\t%s\t%s\n' % (k, tr, note))
    print('written', b, len(out))
else:
    print('NOT written')
