#!/usr/bin/env python3
"""規格010 §7：本作牆面來源與完整3D視窗證據。只在Docker執行。"""
import argparse
from collections import Counter, defaultdict
import json
from pathlib import Path
import platform
import numpy as np
from PIL import Image, ImageDraw
import PIL
from dax_pics import EGA, picture
from hd_battle_inventory import source_records, sha, write

LAYOUTS = [(0,1,2),(2,1,4),(6,1,4),(10,3,4),(22,2,8),
           (38,2,8),(54,7,8),(110,2,11),(132,2,11),(154,1,2)]

def rgb(px, size):
    return Image.fromarray(np.array(EGA,np.uint8)[np.frombuffer(px,np.uint8).reshape(size[1],size[0])])

def load_sources(root):
    orig=root/'workplace/input/dos/ckrynn';tiles={};defs=[];pictures=[];files={}
    for name in ('WALLDEF1','WALLDEF2','WALLDEF3','8X8D1','8X8D2','8X8D3','SKY','HEAD2','HEAD3','BODY2','BODY3','TILES'):
        files[name+'.DAX']=sha((orig/(name+'.DAX')).read_bytes())
        for bid,raw,src in source_records(orig,name+'.DAX'):
            if name.startswith('WALLDEF'):
                assert len(raw) in (780,1560)
                defs.append(dict(source=src,raw=raw))
                continue
            try:w,h,n,x,y,px=picture(raw)
            except ValueError:
                assert name=='8X8D1' and bid==201
                continue
            if name.startswith('8X8D'):
                for i in range(n):
                    p=px[i*64:(i+1)*64]
                    r=tiles.setdefault(sha(p),dict(key=f'wall-{name[-1]}-{bid:03d}-{i:02d}',size=[8,8],raw=p,aliases=[]))
                    r['aliases'].append(dict(src,item=i,picture_size=[w,h,n]))
            else:
                for i in range(n):pictures.append(dict(key=f'{name.lower()}-{bid:03d}-{i:02d}',size=[w,h],raw=px[i*w*h:(i+1)*w*h],source=dict(src,item=i)))
    return list(tiles.values()),defs,pictures,files

def saved_screens(root):
    # 既有索引正規化185張原版畫面；保留原始來源路徑，不當作新擷取。
    hd=root/'workplace/hd';man=json.loads((hd/'screens/normal/manifest.json').read_text())
    result=[]
    for it in man['items']:
        px=(hd/'screens/normal'/it['file']).read_bytes();assert sha(px)==it['indexed_sha256']
        result.append((it['source'],px))
    # 只讀已完成的固定來源探針，不把A/B輸出或中斷的嘗試回灌成模型輸入。
    for name in ('gargath-entry','gargath-source','sanction-source','throtl-source'):
        for p in sorted((hd/'out/issue14-runtime'/name).glob('*.bin')):
            if p.stat().st_size==64000:result.append((str(p.relative_to(root)),p.read_bytes()))
    return result

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);a=ap.parse_args()
    root=a.root;dest=root/'workplace/hd/inventory/issue14';dest.mkdir(exist_ok=True)
    tiles,defs,pictures,files=load_sources(root)
    groups={}
    for t in tiles:
        for s in t['aliases']:
            groups[(s['file'],s['block'],s['item'])]=t
    faces=[];used=set()
    # 每780 bytes含五種牆，各156個global symbol。連續記錄的符號跨70格分段。
    for d in defs:
        src=d['source'];bid=src['block'];n=len(d['raw'])//780
        blocks=[bid] if n==1 else [bid*10+i+1 for i in range(n)]
        bankfile='8X8D'+src['file'][7]+'.DAX'
        for wall in range(n*5):
            for layout,(start,cols,rows) in enumerate(LAYOUTS):
                ids=d['raw'][wall*156+start:wall*156+start+cols*rows];parts=[];valid=True
                for symbol in ids:
                    if symbol==0:parts.append(None);continue
                    if symbol<=45:loc=('8X8D1.DAX',203,symbol-1)
                    else:
                        band=(symbol-46)//70;item=(symbol-46)%70
                        loc=(bankfile,blocks[band],item) if band<len(blocks) else None
                    t=groups.get(loc)
                    if t is None:valid=False;parts.append(None)
                    else:parts.append(t['key']);used.add(t['key'])
                faces.append(dict(source=src,wall=wall,layout=layout,span=[wall*156+start,len(ids)],
                    size=[cols*8,rows*8],symbols=list(ids),tiles=parts,resolved=valid))
    bykey={t['key']:t for t in tiles}
    for t in tiles:
        t['used_by_walldef']=t['key'] in used
        (dest/(t['key']+'.idx')).write_bytes(t['raw'])
    # 一份總表保留所有符號；只將WALLDEF可定位的符號列為牆面候選。
    cols=20;cell=48;rows=(len(tiles)+cols-1)//cols
    sheet=Image.new('RGB',(cols*cell,rows*cell),'#303030');draw=ImageDraw.Draw(sheet)
    for i,t in enumerate(tiles):
        x,y=i%cols*cell,i//cols*cell
        sheet.paste(rgb(t['raw'],(8,8)).resize((32,32),Image.Resampling.NEAREST),(x,y+12))
        draw.text((x,y),str(i),fill='white' if t['key'] in used else 'gray')
    sheet.save(dest/'tiles.png')
    # 保存完整原始零件，不套透明或配色假說。
    for t in pictures:rgb(t['raw'],t['size']).resize((t['size'][0]*3,t['size'][1]*3),Image.Resampling.NEAREST).save(dest/(t['key']+'.png'))
    write(dest/'sources.json',dict(issue=14,address_space='DAX檔案offset；dosgolem 320×200色號',files=files,
        tools=dict(python=platform.python_version(),pillow=PIL.__version__,numpy=np.__version__,tool_sha256=sha(Path(__file__).read_bytes())),
        tiles=[dict({k:v for k,v in t.items() if k!='raw'},indexed=t['key']+'.idx',indexed_sha256=sha(t['raw']),colors=dict(Counter(t['raw']))) for t in tiles],
        definitions=[dict(source=d['source'],bytes=len(d['raw']),records=len(d['raw'])//780) for d in defs],faces=faces,
        pictures=[dict({k:v for k,v in p.items() if k!='raw'},indexed_sha256=sha(p['raw'])) for p in pictures]))
    screens=saved_screens(root)
    panels={}
    for path,px in screens:
        arr=np.frombuffer(px,np.uint8).reshape(200,320);panel=arr[24:112,24:112].copy()
        row=panels.setdefault(sha(panel.tobytes()),dict(panel=panel,screens=[]));row['screens'].append(path)
    # 先量測每格與來源前景的精確命中；不作產品契約。
    walltiles=[t for t in tiles if t['key'] in used]
    tarrays=np.array([np.frombuffer(t['raw'],np.uint8).reshape(8,8) for t in walltiles])
    masks=tarrays!=13
    nonempty=masks.any(axis=(1,2))
    results=[]
    for digest,r in panels.items():
        cells=[];unmatched=[];guards=0
        for gy in range(11):
            for gx in range(11):
                p=r['panel'][gy*8:gy*8+8,gx*8:gx*8+8]
                indices=np.flatnonzero(((tarrays==p)|~masks).all(axis=(1,2))&nonempty)
                matches=[walltiles[i]['key'] for i in indices]
                const=len(set(p.flat))==1
                if not matches and not const:unmatched.append(dict(x=gx,y=gy,colors=dict(Counter(map(int,p.flat))),sha256=sha(p.tobytes())))
                if matches and len(set(p.flat))>=2:guards+=1
                cells.append(dict(x=gx,y=gy,keys=matches,constant=int(p[0,0]) if const else None))
        results.append(dict(sha256=digest,screens=r['screens'],guards=guards,unmatched=unmatched,cells=cells,
            bands=[dict(Counter(map(int,r['panel'][y].flat))) for y in (0,8,16,24,32,40,44,46,48,56,64,72,80,87)]))
    write(dest/'probe.json',dict(screens=len(screens),panels=len(panels),wall_tiles=len(walltiles),results=results))
    print('tiles',len(tiles),'used',len(used),'faces',len(faces),'resolved',sum(f['resolved'] for f in faces),'panels',len(panels))
    for r in results:
        if r['guards']>=3:print('panel',r['screens'][0],r['guards'],'unmatched',len(r['unmatched']),'bands',r['bands'][:1])

if __name__=='__main__':main()
