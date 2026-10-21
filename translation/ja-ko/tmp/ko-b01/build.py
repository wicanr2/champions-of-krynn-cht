import csv, sys, unicodedata, re
base='/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/'
bad=0
for b in sys.argv[1:]:
    rows=list(csv.reader(open(base+f'batches/{b}.tsv',encoding='utf-8'),delimiter='\t',quoting=csv.QUOTE_NONE))[1:]
    src=[l.rstrip('\n') for l in open(base+f'tmp/ko-b01/{b}.src',encoding='utf-8') if l.strip()]
    if len(src)!=len(rows):
        print(b,'count mismatch',len(src),len(rows)); bad+=1; continue
    out=['key\ttranslation\tnote']
    for i,(r,l) in enumerate(zip(rows,src)):
        p=l.split('\t')
        if len(p)<2 or len(p)>3: print(b,i,'cols',repr(l)); bad+=1; continue
        k,t=p[0],p[1]; n=p[2] if len(p)==3 else ''
        if 'ecl.'+k!=r[0]: print(b,i,'key mismatch',k,r[0]); bad+=1; continue
        t=unicodedata.normalize('NFC',t)
        if t!=t.strip() or not t: print(b,r[0],'space/empty',repr(t)); bad+=1
        if len(t)>len(r[1]): print(b,r[0],'TOO LONG',len(t),len(r[1]),t,'|',r[1]); bad+=1
        for ch in t:
            if not (ord(ch)<128 or 0xAC00<=ord(ch)<=0xD7A3): print(b,r[0],'badchar',repr(ch),t); bad+=1; break
        # placeholders / digits preserved
        if re.findall(r'\d+',r[1]) and re.findall(r'\d+',r[1])!=re.findall(r'\d+',t): print(b,r[0],'digits',r[1],t); bad+=1
        if re.search(r'[(]',t): print(b,r[0],'PAREN',t)
        out.append(f'{r[0]}\t{t}\t{n}')
    if bad==0 or True:
        open(base+f'done-ko/{b}.tsv','w',encoding='utf-8').write('\n'.join(out)+'\n')
print('bad',bad)
