import sys
import csv
bad=0
for b in sys.argv[1:]:
    rows=[l.rstrip('\n').split('\t') for l in open(f'../../batches/{b}.tsv',encoding='utf-8')][1:]
    out=[l.rstrip('\n').split('\t') for l in open(f'../../done-ko/{b}.tsv',encoding='utf-8')]
    assert out[0]==['key','translation','note'],out[0]
    out=out[1:]
    assert len(rows)==len(out),(b,len(rows),len(out))
    for r,o in zip(rows,out):
        assert len(o)==3,(b,o)
        assert r[0]==o[0]
        t=o[1]
        assert t==t.strip() and t and '\n' not in t and '\r' not in t
        L=len(r[1])
        if len(t)>L:
            print('LONG',b,r[0],repr(r[1]),L,t,len(t)); bad+=1
        elif len(t)>len(r[1].strip()):
            print('long-vs-stripped',b,repr(r[1]),t)
print('bad',bad)
