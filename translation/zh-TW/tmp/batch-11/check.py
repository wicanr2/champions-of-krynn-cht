import sys
for n in sys.argv[1:]:
    B='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
    inp=open(B+f'batches/batch-{n}.tsv',encoding='utf-8').read().rstrip('\n').split('\n')[1:]
    out=open(B+f'done/batch-{n}.tsv',encoding='utf-8').read()
    assert out.endswith('\n') and not out.endswith('\n\n')
    ol=out.rstrip('\n').split('\n')
    assert ol[0]=='key\ttranslation\tnote'
    ol=ol[1:]
    assert len(ol)==len(inp),(n,len(ol),len(inp))
    bad=[]
    for i,(a,b) in enumerate(zip(inp,ol)):
        c=b.split('\t')
        if len(c)!=3 or c[0]!=a.split('\t')[0]: bad.append(('cols/key',b));continue
        t=c[1]
        if t!=t.strip() or not t or '\r' in b: bad.append(('ws',b))
        try: (t+c[2]).encode('big5')
        except UnicodeEncodeError as e: bad.append(('big5',t,e.reason, t[e.start:e.end]))
    print(n,len(ol),'noise',sum(1 for l in ol if l.split('\t')[1]=='-'),'bad',bad)
