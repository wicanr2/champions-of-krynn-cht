import sys
b=sys.argv[1]
rows=[l.rstrip('\n').split('\t') for l in open(f'../../batches/{b}.tsv',encoding='utf-8')][1:]
tr=[l.rstrip('\n').split('\t') for l in open(f'{b}.txt',encoding='utf-8')]
print(len(rows),len(tr))
for i in range(max(len(rows),len(tr))):
    o=rows[i][1] if i<len(rows) else '<none>'
    t=tr[i][0] if i<len(tr) else '<none>'
    print(i+1,repr(o),'=>',t)
