import csv,re
B='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
src={}
for n in (104,105,106,107):
    src[n]=[l.split('\t')[0] for l in open(B+'batches/batch-%d.tsv'%n,encoding='utf-8').read().split('\n')[1:] if l]
exec(open(B+'tmp/batch-104/gen.py').read().split('for n in')[0])
allk={k.split('.',1)[1] for v in src.values() for k in v}
print('unknown keys',[k for k in M if k not in allk])
for n in (104,105,106,107):
    t=open(B+'done/batch-%d.tsv'%n,encoding='utf-8').read()
    L=t.split('\n'); assert L[-1]==''; L=L[:-1]
    assert L[0]=='key\ttranslation\tnote'
    assert [l.split('\t')[0] for l in L[1:]]==src[n],n
    for l in L[1:]:
        c=l.split('\t'); assert len(c)==3,l
        assert c[1]==c[1].strip() and c[1]!='' ,l
        c[1].encode('big5'); c[2].encode('big5')
    print(n,'ok',len(L)-1)
