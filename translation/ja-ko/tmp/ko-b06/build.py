import csv,sys
b=sys.argv[1]
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
rows=list(csv.reader(open(base+f'batches/{b}.tsv',encoding='utf-8',newline=''),delimiter='\t',quoting=csv.QUOTE_NONE))
hdr,rows=rows[0],rows[1:]
src=open(f'{b}.txt',encoding='utf-8').read().split('\n')
if src and src[-1]=='': src.pop()
if len(src)!=len(rows): print('LEN MISMATCH',len(src),len(rows)); sys.exit(1)
out=['key\ttranslation\tnote']; bad=0
for i,(r,l) in enumerate(zip(rows,src),1):
    t,n=l.split('|||',1)
    t=t.strip(); n=n.strip()
    assert '\t' not in t+n
    if len(t)>len(r[1]) and '길이 제한' not in n: print('LONG',i,r[0],len(t),len(r[1]),r[1],'=>',t); bad+=1
    if not t: print('EMPTY',i); bad+=1
    out.append(f'{r[0]}\t{t}\t{n}')
if '--write' in sys.argv and not bad:
    open(base+f'done-ko/{b}.tsv','w',encoding='utf-8').write('\n'.join(out)+'\n'); print('written')
print('rows',len(rows),'bad',bad)
