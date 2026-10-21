import sys,csv
n=sys.argv[1]; base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
rows=[l.rstrip('\n').split('\t') for l in open(sys.argv[2],encoding='utf-8')]
inp=list(csv.reader(open(base+'batches/batch-%s.tsv'%n,encoding='utf-8'),delimiter='\t',quoting=csv.QUOTE_NONE))[1:]
assert len(rows)==len(inp),(len(rows),len(inp))
out=['key\ttranslation\tnote']
for r,i in zip(rows,inp):
    assert i[0].endswith('.'+r[0]),(r[0],i[0])
    t=r[1]; note=r[2] if len(r)>2 else ''
    out.append('\t'.join([i[0],t,note]))
open(sys.argv[3],'w',encoding='utf-8').write('\n'.join(out)+'\n')
print(len(rows))
