import csv,sys
b='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
for n in sys.argv[1:]:
    i=[l.split('\t')[0] for l in open(b+'batches/batch-%s.tsv'%n,encoding='utf-8').read().split('\n') if l][1:]
    L=open(b+'done/batch-%s.tsv'%n,encoding='utf-8').read().split('\n')
    assert L[-1]==''; L=L[:-1]
    assert L[0]=='key\ttranslation\tnote'
    L=L[1:]; assert len(L)==len(i),(len(L),len(i)); noise=0
    for k,l in zip(i,L):
        p=l.split('\t'); assert len(p)==3 and p[0]==k,l
        t=p[1]; assert t and t==t.strip() and '\r' not in l
        if t=='-': noise+=1
        for ch in t:
            try: ch.encode('big5')
            except Exception: print('BIG5 FAIL',n,k,repr(ch))
    print(n,len(L),'noise',noise)
