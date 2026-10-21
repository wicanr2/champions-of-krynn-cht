import sys
base = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
n, src = sys.argv[1], sys.argv[2]
rows = [l.rstrip('\n').split('\t') for l in open(base+f'batches/batch-{n}.tsv', encoding='utf-8')][1:]
tr = [l.rstrip('\n') for l in open(src, encoding='utf-8') if l.strip()]
assert len(rows) == len(tr), (len(rows), len(tr))
with open(base+f'done/batch-{n}.tsv', 'w', encoding='utf-8', newline='') as f:
    f.write('key\ttranslation\tnote\n')
    for r, t in zip(rows, tr):
        p = t.split('|')
        assert len(p) == 2, t
        f.write(f'{r[0]}\t{p[0].strip()}\t{p[1].strip()}\n')
print(n, len(rows), sum(1 for t in tr if t.startswith('-|')))
