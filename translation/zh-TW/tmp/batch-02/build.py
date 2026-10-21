import csv,sys
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
for n in ['02','03','04','05','06']:
    inp=list(csv.reader(open(base+f'batches/batch-{n}.tsv',encoding='utf-8',newline=''),delimiter='\t',quoting=csv.QUOTE_NONE))[1:]
    m={}
    for ln in open(base+f'tmp/batch-02/b{n}.txt',encoding='utf-8').read().split('\n'):
        if not ln: continue
        k,t,*note=ln.split('|')
        assert k not in m,k
        m[k]=(t,'|'.join(note))
    keys=[r[0] for r in inp]
    assert set(m)==set(keys),(n,set(m)^set(keys))
    out=['key\ttranslation\tnote']
    for k in keys:
        t,note=m[k]; out.append(f'{k}\t{t}\t{note}')
    open(base+f'done/batch-{n}.tsv','w',encoding='utf-8',newline='').write('\n'.join(out)+'\n')
    print(n,len(keys))
