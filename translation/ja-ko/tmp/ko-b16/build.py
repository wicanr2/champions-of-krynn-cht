import sys,unicodedata
b=sys.argv[1]
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
rows=[l.rstrip('\n').split('\t') for l in open(base+'batches/%s.tsv'%b,encoding='utf-8')]
hdr,rows=rows[0],rows[1:]
tr={}
for l in open(base+'tmp/ko-b16/%s.txt'%b,encoding='utf-8'):
    l=l.rstrip('\n')
    if not l: continue
    n,t,*note=l.split('|')
    tr[int(n)]=(t,'|'.join(note))
errs=[]
out=['key\ttranslation\tnote']
for i,r in enumerate(rows,1):
    if i not in tr: errs.append('missing %d'%i); continue
    t,note=tr[i]
    if t!=t.strip() or '\t' in t or '\n' in t: errs.append('ws %d'%i)
    if len(t)>len(r[1]): errs.append('len %d: %d>%d %s'%(i,len(t),len(r[1]),t))
    out.append('%s\t%s\t%s'%(r[0],t,note))
if len(tr)!=len(rows): errs.append('count %d vs %d'%(len(tr),len(rows)))
print('\n'.join(errs) or 'OK', len(rows))
if not errs or '--force' in sys.argv:
    open(base+'done-ko/%s.tsv'%b,'w',encoding='utf-8').write('\n'.join(out)+'\n')
