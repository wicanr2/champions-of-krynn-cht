import sys
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/'
tr={}
for f in ['part1.txt','part51.txt','part2.txt','part3.txt']:
    for line in open('/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/tmp/journal-ko/'+f,encoding='utf-8'):
        line=line.rstrip('\n')
        if not line: continue
        k,v=line.split(' ::: ',1)
        assert '\t' not in v
        if k in tr and f!='part2.txt': raise SystemExit('dup '+k)
        if k in tr and f=='part2.txt' and k!='51': raise SystemExit('dup '+k)
        tr[int(k)]=v
rows=[l.rstrip('\n').split('\t') for l in open(base+'journal/journal.en.tsv',encoding='utf-8')]
out=['\t'.join(rows[0])]
for r in rows[1:]:
    n=int(r[0])
    title=('여행 수기 제%d항' if n<1000 else '주점 소문 제%d항')%(n if n<1000 else n-1000)
    out.append('\t'.join([r[0],title,tr[n].strip(),r[3]]))
open(base+'tr-jako/aux-ko/journal.tsv','w',encoding='utf-8').write('\n'.join(out)+'\n')
