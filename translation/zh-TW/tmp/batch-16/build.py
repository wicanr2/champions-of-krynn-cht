import sys
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/'
jobs=[('16','b16.txt','ecl.2.67.'),('17','b17.txt','ecl.2.67.'),('18','b18.txt','ecl.2.68.'),('19','b19.txt','ecl.3.80.'),('20','b20.txt','ecl.3.80.')]
for n,f,p in jobs:
    ins=[l.rstrip('\n').split('\t')[0] for l in open(base+'batches/batch-%s.tsv'%n,encoding='utf-8')][1:]
    rows=[l.rstrip('\n') for l in open(base+'tmp/batch-16/'+f,encoding='utf-8') if l.strip()]
    assert len(ins)==len(rows),(n,len(ins),len(rows))
    out=['key\ttranslation\tnote']
    for k,r in zip(ins,rows):
        parts=r.split('|')
        assert len(parts)>=2,r
        kk,tr=parts[0],parts[1]; note='|'.join(parts[2:])
        assert k==p+kk,(n,k,kk)
        out.append('\t'.join([k,tr.strip(),note.strip()]))
    open(base+'done/batch-%s.tsv'%n,'w',encoding='utf-8').write('\n'.join(out)+'\n')
    print(n,len(out)-1)
