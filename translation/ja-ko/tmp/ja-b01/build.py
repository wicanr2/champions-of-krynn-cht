import sys,csv
bn=sys.argv[1]
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
inp=[l.rstrip('\n').split('\t') for l in open(base+f'batches/{bn}.tsv',encoding='utf-8')][1:]
work=[l.rstrip('\n') for l in open(base+f'tmp/ja-b01/{bn}.work',encoding='utf-8') if l.strip()]
out=['key\ttranslation\tnote']
err=0
if len(work)!=len(inp): print('COUNT',len(work),len(inp)); err=1
for i,(r,w) in enumerate(zip(inp,work)):
    p=w.split('||')
    if len(p)==2: p.append('')
    if len(p)!=3: print('FIELDS',i,w); err=1; continue
    k,t,n=p
    if k!=r[0]: print('KEY',i,k,r[0]); err=1
    if t!=t.strip() or '\t' in t or not t: print('WS',i,k); err=1
    if len(t)>len(r[1].strip()): print('LEN',k,len(t),len(r[1].strip()),t); err=1
    out.append(f'{k}\t{t}\t{n}')
if not err:
    open(base+f'done-ja/{bn}.tsv','w',encoding='utf-8').write('\n'.join(out)+'\n'); print('OK',len(out)-1)
