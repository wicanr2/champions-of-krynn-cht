import sys,csv
b=sys.argv[1]
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
inp=list(csv.reader(open(base+'batches/%s.tsv'%b,encoding='utf-8'),delimiter='\t',quotechar=None))[1:]
dr=[l.rstrip('\n') for l in open(base+'tmp/ja-b%s/%s.draft'%(sys.argv[2],b),encoding='utf-8') if l.strip()]
bad=0
if len(dr)!=len(inp): print('COUNT',len(dr),len(inp)); bad=1
out=['key\ttranslation\tnote']
for i,(r,l) in enumerate(zip(inp,dr)):
    p=l.split('##')
    if len(p)!=3: print('FIELDS',i,l); bad=1; continue
    if not r[0].endswith('.'+p[0]) : print('KEY',i,r[0],p[0]); bad=1
    t=p[1]
    if t!=t.strip() or '\t' in t or not t: print('WS',i,r[0]); bad=1
    n=len(r[1].rstrip(' ')) if False else len(r[1])
    if len(t)>n: print('LEN',r[0],len(t),n,t); bad=1
    out.append('%s\t%s\t%s'%(r[0],t,p[2]))
if not bad and len(sys.argv)>3:
    open(base+'done-ja/%s.tsv'%b,'w',encoding='utf-8').write('\n'.join(out)+'\n'); print('written',len(out)-1)
print('bad',bad)
