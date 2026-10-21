import csv, re, sys, os

BASE = '/home/anr2/cht/golden_box/champions_of_krynn'
TMP = BASE + '/workplace/tr-jako/tmp/guide-ja/'
OUT = BASE + '/workplace/tr-jako/aux-ja/'
os.makedirs(OUT, exist_ok=True)

def read_tsv(p):
    with open(p, encoding='utf-8', newline='') as f:
        return list(csv.reader(f, delimiter='\t', quoting=csv.QUOTE_NONE))

def write_tsv(p, rows):
    with open(p, 'w', encoding='utf-8', newline='') as f:
        for r in rows:
            f.write('\t'.join(r) + '\n')

errs = []

# walkthrough
src = read_tsv(BASE + '/text/walkthrough.zh-TW.tsv')
recs = {}
cur = None
for fn in ('walkthrough-1.txt', 'walkthrough-2.txt'):
    for line in open(TMP + fn, encoding='utf-8').read().split('\n'):
        if not line.strip():
            continue
        if line.startswith('@@ '):
            cur = int(line[3:])
            recs[cur] = {'T': '', 'S': '', 'P': []}
        elif line.startswith('T: '):
            recs[cur]['T'] = line[3:]
        elif line.startswith('S: '):
            recs[cur]['S'] = line[3:]
        elif line.startswith('P: '):
            recs[cur]['P'].append(line[3:])
        else:
            errs.append('bad line: ' + line[:40])
out = [src[0]]
for r in src[1:]:
    n = int(r[0])
    if n not in recs:
        errs.append('missing walkthrough %d' % n)
        continue
    d = recs[n]
    body = '\\n'.join(d['P'])
    zp = r[2].count('\\n') + 1
    if zp != len(d['P']):
        errs.append('walkthrough %d paragraph count zh=%d ja=%d' % (n, zp, len(d['P'])))
    out.append([r[0], d['T'], body, d['S']])
write_tsv(OUT + 'walkthrough.tsv', out)

# map-guide
src = read_tsv(BASE + '/text/map-guide.zh-TW.tsv')
lines = [l for l in open(TMP + 'map-guide.txt', encoding='utf-8').read().split('\n') if l.strip()]
if len(lines) != len(src) - 1:
    errs.append('map-guide lines %d vs %d' % (len(lines), len(src) - 1))
out = [src[0]]
for r, l in zip(src[1:], lines):
    lab, summ = l.split(' ||| ')
    out.append([r[0], r[1], r[2], r[3], lab, summ, r[6]])
write_tsv(OUT + 'map-guide.tsv', out)

print('\n'.join(errs) if errs else 'build ok')
