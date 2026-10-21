import csv
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
for n in ('101','102','103'):
    inp=[l.rstrip('\n').split('\t') for l in open(base+'batches/batch-%s.tsv'%n,encoding='utf-8')]
    raw=open(base+'done/batch-%s.tsv'%n,encoding='utf-8').read()
    assert raw.endswith('\n') and not raw.endswith('\n\n')
    out=[l.split('\t') for l in raw.split('\n')[:-1]]
    assert out[0]==['key','translation','note']
    assert len(out)==len(inp),(len(out),len(inp))
    noise=0
    for i,(a,b) in enumerate(zip(inp[1:],out[1:])):
        assert a[0]==b[0],(i,a[0],b[0])
        assert len(b)==3,b
        t=b[1]
        assert t==t.strip() and t and '\r' not in t
        t.encode('big5')
        b[2].encode('big5') if False else None
        if t=='-': noise+=1
    print(n,'rows',len(out)-1,'noise',noise,'ok')
