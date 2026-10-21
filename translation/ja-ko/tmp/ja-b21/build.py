import csv,sys
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
trial={}
for r in list(csv.reader(open(base+'trial-ja.tsv',encoding='utf-8'),delimiter='\t',quoting=csv.QUOTE_NONE))[1:]:
    trial[r[0]]=r[2]
bad=0
for b in ['b21','b22','b23','b24','b25']:
    rows=[l.rstrip('\n').split('\t') for l in open(base+'batches/%s.tsv'%b,encoding='utf-8')][1:]
    tr=[l.rstrip('\n').split('|',2) for l in open(base+'tmp/ja-b21/%s.txt'%b,encoding='utf-8')]
    assert len(rows)==len(tr),(b,len(rows),len(tr))
    out=['key\ttranslation\tnote']
    for i,(r,t) in enumerate(zip(rows,tr)):
        assert int(t[0])==i+1,(b,i)
        k,orig=r[0],r[1]
        s=t[1]; note=t[2] if len(t)>2 else ''
        if k in trial and trial[k]!=s: print('TRIAL DIFF',b,i+1,k,trial[k],s); bad+=1
        if s!=s.strip() or '\t' in s or '\n' in s or not s: print('WS',b,i+1,repr(s)); bad+=1
        if len(s)>len(orig): print('LEN raw',b,i+1,len(s),len(orig),s); bad+=1
        elif len(s)>len(orig.strip()): print('len>stripped',b,i+1,len(s),len(orig.strip()),s)
        out.append('\t'.join([k,s,note]))
    open(base+'done-ja/%s.tsv'%b,'w',encoding='utf-8').write('\n'.join(out)+'\n')
print('bad',bad)
