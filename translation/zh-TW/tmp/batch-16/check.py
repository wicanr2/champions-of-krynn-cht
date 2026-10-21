import csv
for n in ['16','17','18','19','20']:
    a=[l.rstrip('\n').split('\t') for l in open('batches/batch-%s.tsv'%n,encoding='utf-8')][1:]
    b=[l.rstrip('\n').split('\t') for l in open('done/batch-%s.tsv'%n,encoding='utf-8')]
    assert b[0]==['key','translation','note']
    b=b[1:]
    assert len(a)==len(b)
    noise=0
    for x,y in zip(a,b):
        assert len(y)==3,y
        assert x[0]==y[0]
        t=y[1]
        assert t and t==t.strip() and '\r' not in t
        t.encode('big5')
        y[2].encode('big5')
        if t=='-': noise+=1
    print(n,len(b),'noise',noise,'ok')
