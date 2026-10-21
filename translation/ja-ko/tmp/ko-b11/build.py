import csv, sys, os
b = sys.argv[1]
base = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako'
tmp = base + '/tmp/ko-b11'
rows = list(csv.reader(open(f'{base}/batches/{b}.tsv', encoding='utf-8'), delimiter='\t'))
hdr, data = rows[0], rows[1:]
tr = [l.rstrip('\n') for l in open(f'{tmp}/{b}.txt', encoding='utf-8')]
notes = {}
p = f'{tmp}/{b}.notes'
if os.path.exists(p):
    for l in open(p, encoding='utf-8'):
        l = l.rstrip('\n')
        if l:
            n, t = l.split('|', 1)
            notes[int(n)] = t
assert len(tr) == len(data), (len(tr), len(data))
out = [['key', 'translation', 'note']]
for i, (r, t) in enumerate(zip(data, tr), 1):
    out.append([r[0], t.strip(), notes.get(i, '')])
os.makedirs(f'{base}/done-ko', exist_ok=True)
with open(f'{base}/done-ko/{b}.tsv', 'w', encoding='utf-8', newline='') as f:
    w = csv.writer(f, delimiter='\t', lineterminator='\n', quoting=csv.QUOTE_NONE, escapechar=None, quotechar=None) if False else None
    for r in out:
        f.write('\t'.join(r) + '\n')
