#!/usr/bin/env python3
"""Issue22：完整格的獨立 Pillow 合成、整幀及抽樣統計。Docker 限定。"""
import argparse
from collections import Counter
import json
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw
from dax_pics import EGA
from hd_issue22 import ROOT, HD, INV, OUT, sha, write, scale3x_reference
from hd_issue21 import exact_positions
from hd_battle_receipts import Oracle
from hd_wall_receipts import hashes

class CellOracle:
    def __init__(self, theme):
        self.global_cells={};self.cells=[{} for _ in range(121)];self.assets={}
        self.manifest=json.loads((theme/'walls.json').read_text())
        for a in self.manifest['assets']:
            w,h=a['size'];raw=(theme/a['indexed']).read_bytes()
            assert sha(raw)==a['indexed_sha256']
            assert sha((theme/a['png']).read_bytes())==a['png_sha256']
            self.assets[a['key']]=(a,Image.fromarray(np.frombuffer(raw,np.uint8).reshape(h,w)),Image.open(theme/a['png']).convert('RGBA'))
        for c in self.manifest['cells']:
            ref=Image.new('L',(8,8),c['base_color']);pix=Image.new('RGBA',(24,24),(*EGA[c['base_color']],255));owner=Image.new('L',(8,8),0)
            for layer in c['layers']:
                a,src,art=self.assets[layer['key']];x,y=layer['at'];mask=src.point(lambda p:0 if p==13 else 255)
                ref.paste(src,(x,y),mask);owner.paste(255 if a['kind']=='wall' else 0,(x,y,x+src.width,y+src.height),mask)
                if a.get('algorithm')=='scale3x':
                    scaled=scale3x_reference(np.array(src));mask=Image.fromarray(np.where(scaled==13,0,255).astype(np.uint8))
                else:mask=mask.resize((src.width*3,src.height*3),Image.Resampling.NEAREST)
                pix.paste(art,(x*3,y*3),mask)
            raw=ref.tobytes();assert sha(raw)==c['indexed_sha256']
            arr=np.array(ref);owned=np.array(owner)>0
            if not c['paint']:pix=native(raw,8,8)
            row=dict(key=c['key'],raw=raw,pix=np.array(pix),paint=c['paint'],layers=c['layers'],base=c['base_color'],guard=bool(owned.sum()>=16 and len(np.unique(arr[owned]))>=2))
            if c['positions']:
                for pos in c['positions']:self.cells[pos][raw]=row
            else:self.global_cells[raw]=row
        if self.manifest.get('matching')=='portable-complete-cells':
            groups={}
            for c in list(self.global_cells.values())+[r for lookup in self.cells for r in lookup.values()]:groups.setdefault(c['raw'],[]).append(c)
            self.global_cells={}
            for raw,rows in groups.items():
                c=dict(rows[0]);c['guard']=any(r['guard'] for r in rows);c['paint']=all(r['paint'] for r in rows)
                if len({r['pix'].tobytes() for r in rows})>1:
                    c.update(key='ambiguous-portable-'+sha(raw)[:16],pix=np.array(native(raw,8,8)),paint=False)
                self.global_cells[raw]=c

    def matches(self,raw):
        p=np.frombuffer(raw,np.uint8).reshape(200,320);plans=[]
        for pos in range(121):
            x,y=24+pos%11*8,24+pos//11*8;k=p[y:y+8,x:x+8].tobytes()
            plans.append(self.global_cells.get(k) or self.cells[pos].get(k))
        return plans

    def apply(self, raw, out):
        plans=self.matches(raw);mask=np.zeros((200,320),bool)
        for pos,c in enumerate(plans):
            if c is None:continue
            x,y=24+pos%11*8,24+pos//11*8
            out[y*3:(y+8)*3,x*3:(x+8)*3]=c['pix'];mask[y:y+8,x:x+8]=True
        return mask,plans

def native(raw,w=320,h=200):
    arr=np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(h,w)]
    return Image.fromarray(np.dstack((np.repeat(np.repeat(arr,3,0),3,1),np.full((h*3,w*3),255,np.uint8))))

def is_3d(raw):
    # 本作既有3D框線完整比對；不以牆樣守衛把戰鬥或物品文字誤列探索畫面。
    source=(OUT/'issue22-runtime/throtl-initial3/final.bin').read_bytes()
    a=np.frombuffer(raw,np.uint8).reshape(200,320);b=np.frombuffer(source,np.uint8).reshape(200,320)
    return np.array_equal(a[24:112,16:24],b[24:112,16:24]) and np.array_equal(a[112:120,24:112],b[112:120,24:112])

class Issue22Oracle:
    def __init__(self):
        self.walls=CellOracle(HD/'theme');self.other=Oracle(HD/'theme');self.exact=[]
        for it in json.loads((HD/'theme/exact.json').read_text())['items']:
            w,h=it['size'];p=(HD/'theme'/it['indexed']).read_bytes();im=HD/'theme'/it['png']
            assert sha(p)==it['indexed_sha256'] and sha(im.read_bytes())==it['png_sha256']
            self.exact.append((it,np.frombuffer(p,np.uint8).reshape(h,w),np.array(Image.open(im).convert('RGBA'))))

    def expected(self, raw):
        out=np.array(native(raw));allowed,plans=self.walls.apply(raw,out)
        # 遊戲順序為 walls、combat、actors、encounters、exact；獨立舊oracle涵蓋中間三項。
        other,mask,hits=self.other.expected(raw);out[np.repeat(np.repeat(mask,3,0),3,1)]=other[np.repeat(np.repeat(mask,3,0),3,1)];allowed|=mask
        frame=np.frombuffer(raw,np.uint8).reshape(200,320);candidates=[];own=np.zeros((200,320),np.uint16)
        for it,p,im in self.exact:
            for x,y in exact_positions(frame,p):
                candidates.append((x,y,it,p,im));h,w=p.shape;own[y:y+h,x:x+w]+=1
        for x,y,it,p,im in candidates:
            h,w=p.shape
            if np.any(own[y:y+h,x:x+w]>1):continue
            out[y*3:(y+h)*3,x*3:(x+w)*3]=im;allowed[y:y+h,x:x+w]=True;hits.append('exact:'+it['key'])
        return out,allowed,hits,plans

def overview():
    o=Issue22Oracle();rows=[]
    for p in sorted((OUT/'issue22-runtime').glob('*-base3/receipt.json')):
        r=json.loads(p.read_text());seen=set();complete=[];guard_counts=[]
        for s in r['shots']:
            raw=(p.parent/(s['name']+'.bin')).read_bytes();a=np.frombuffer(raw,np.uint8).reshape(200,320)
            plans=o.walls.matches(raw);guards=sum(c is not None and c['guard'] for c in plans)
            guard_counts.append(guards)
            if guards>=3 and is_3d(raw):
                sig=sha(a[24:112,24:112].tobytes());seen.add(sig);complete.append((s['name'],sum(c is not None for c in plans),guards))
        rows.append(dict(scene=p.parent.name,shots=len(r['shots']),three_d_candidate_frames=len(complete),unique_inner_viewports=len(seen),frames=complete,guard_counts=guard_counts))
    write(INV/'sampling-overview.json',rows)
    print([{k:v for k,v in r.items() if k not in ('frames','guard_counts')} for r in rows])

def discover():
    w=CellOracle(HD/'theme');unknown={};occurrences=Counter()
    for p in sorted((OUT/'issue22-runtime').glob('*-base3/receipt.json')):
        for s in json.loads(p.read_text())['shots']:
            raw=(p.parent/(s['name']+'.bin')).read_bytes()
            if not is_3d(raw):continue
            a=np.frombuffer(raw,np.uint8).reshape(200,320)
            plans=w.matches(raw)
            if sum(c is not None and c['guard'] for c in plans)<3:continue
            for pos,c in enumerate(plans):
                if c is not None:continue
                x,y=24+pos%11*8,24+pos//11*8;t=a[y:y+8,x:x+8].tobytes()
                unknown.setdefault((pos,t),str((p.parent/(s['name']+'.bin')).relative_to(ROOT)));occurrences[(pos,t)]+=1
    walls=[(a,np.array(ref).reshape(64),np.array(pix)) for a,ref,pix in w.assets.values() if a['kind']=='wall']
    refs=np.array([r[1] for r in walls]);masks=refs!=13;found=[];ambiguous=[];missing=[]
    for ni,((pos,raw),screen) in enumerate(unknown.items()):
        t=np.frombuffer(raw,np.uint8);upper=np.where(np.all((~masks)|(refs==t),axis=1)&masks.any(axis=1)&(~masks).any(axis=1))[0]
        solutions={}
        def offer(base,layers,pix):
            solutions.setdefault(sha(pix.tobytes()),dict(base_color=int(base),layers=layers))
        for u in upper:
            ua,ur,up=walls[u];um=masks[u]
            # 已由來源模型重建的天空或較下層完整格，包含全部背景點。
            for lower in list(w.cells[pos].values()):
                if not lower['paint'] or len(lower['layers'])>=6:continue
                lr=np.frombuffer(lower['raw'],np.uint8)
                if not np.array_equal(lr[~um],t[~um]):continue
                pix=lower['pix'].copy();bm=np.repeat(np.repeat(um.reshape(8,8),3,0),3,1)
                if ua.get('algorithm')=='scale3x':bm=scale3x_reference(ur.reshape(8,8))!=13
                pix[bm]=up[bm];offer(lower['base'],lower['layers']+[dict(key=ua['key'],at=[0,0])],pix)
            # 兩個牆面來源及一個完整單色底；全64點均須由來源公式重建。
            compatible=np.all((~masks)|um|(refs==t),axis=1)&np.any(masks&(~um),axis=1)
            for l in np.where(compatible)[0]:
                la,lr,lp=walls[l];rest=(~um)&(~masks[l]);colors=np.unique(t[rest])
                if len(colors)>1:continue
                base=int(colors[0]) if len(colors) else 0
                pix=np.empty((24,24,4),np.uint8);pix[:]=[*EGA[base],255]
                for a,r,art in (walls[l],walls[u]):
                    bm=scale3x_reference(r.reshape(8,8))!=13 if a.get('algorithm')=='scale3x' else np.repeat(np.repeat((r!=13).reshape(8,8),3,0),3,1)
                    pix[bm]=art[bm]
                offer(base,[dict(key=la['key'],at=[0,0]),dict(key=ua['key'],at=[0,0])],pix)
        row=dict(position=pos,indexed_sha256=sha(raw),screen=screen,occurrences=occurrences[(pos,raw)],different_outputs=len(solutions))
        if len(solutions)==1:found.append(dict(row,**next(iter(solutions.values()))))
        elif solutions:ambiguous.append(row)
        else:missing.append(row)
        if ni%100==0:print('discover',ni,'/',len(unknown),flush=True)
    write(INV/'composition-discovery.json',dict(source_manifest_sha256=sha((HD/'theme/walls.json').read_bytes()),unique_source_compositions=found,ambiguous=ambiguous,no_source_composition=missing,scope='來源13留底、最多兩個8×8來源或已知天空格加牆面；只保留所有解產出同一HD像素者。未直接將畫面色號當模板。'))
    print('unique',len(found),'ambiguous',len(ambiguous),'missing',len(missing))

def frames():
    o=Issue22Oracle();tasks=[];dest=OUT/'issue22-frames';dest.mkdir(exist_ok=True)
    # 每個Scale3x完整來源與留底背景；其餘格刻意未知，證明不需整窗守衛。
    coverage=json.loads((INV/'b18-theme-coverage.json').read_text())['coverage']
    for r in coverage:
        for digest in r['templates']:
            candidates=[c for c in o.walls.global_cells.values() if sha(c['raw'])==digest]
            if not candidates:candidates=[c for lookup in o.walls.cells for c in lookup.values() if sha(c['raw'])==digest]
            c=candidates[0];pos=next((n for n,lookup in enumerate(o.walls.cells) if lookup.get(c['raw']) is c),60)
            a=np.full((200,320),5,np.uint8);x,y=24+pos%11*8,24+pos//11*8;a[y:y+8,x:x+8]=np.frombuffer(c['raw'],np.uint8).reshape(8,8)
            tasks.append((r['key']+'-'+digest[:8],'scale3x-source-complete',a.tobytes()))
            a[y,x]^=1;tasks.append((r['key']+'-'+digest[:8]+'-mismatch','source-single-cell-mismatch',a.tobytes()))
    for it,p,im in o.exact:
        if it.get('algorithm')!='scale3x':continue
        a=np.full((200,320),5,np.uint8);h,w=p.shape;a[59:59+h,37:37+w]=p
        tasks.append((it['key'],'scale3x-cursor',a.tobytes()));a[59,37]^=1
        tasks.append((it['key']+'-mismatch','source-single-cell-mismatch',a.tobytes()))
    # 所有本輪已保存原版畫格，而非只選HD正例。
    for p in sorted((OUT/'issue22-runtime').glob('*-base3/receipt.json')):
        for s in json.loads(p.read_text())['shots']:
            f=p.parent/(s['name']+'.bin');tasks.append((str(f.relative_to(HD)),'dosgolem-state-continuation',f.read_bytes()))
    cases=[]
    for n,(name,group,raw) in enumerate(tasks):
        expected,allowed,hits,plans=o.expected(raw)
        idx=dest/f'{n:04d}.bin';ep=dest/f'{n:04d}-expected.png';op=dest/f'{n:04d}-original.png'
        idx.write_bytes(raw);Image.fromarray(expected).save(ep);native(raw).save(op)
        pp=lambda p:'/hd/'+str(p.relative_to(HD))
        cases.append(dict(name=name,group=group,indexed=pp(idx),indexed_sha256=sha(raw),expected=pp(ep),expected_sha256=sha(ep.read_bytes()),original=pp(op),original_sha256=sha(op.read_bytes()),wall_cells=sum(c is not None for c in plans),matches=hits))
        if n%100==0:print('frames',n,'/',len(tasks),flush=True)
    write(dest/'cases.json',dict(cases=cases,scope='獨立Pillow預期；來源案例與原版state續跑分開列記'))
    print('cases',len(cases),dict(Counter(c['group'] for c in cases)))

SCENES=('demo','throtl','throtl2','sanction','gargath-turns','gargath2-reenter','gargath3d-a','gargath3d-b')

def runtime():
    o=Issue22Oracle();dest=OUT/'issue22-runtime';rows=[];groups=[]
    for scene in SCENES:
        if not (dest/f'{scene}-hd2/receipt.json').exists():
            raise RuntimeError('A/B尚未完成：'+scene)
        reference=None
        for scale in (3,2):
            a,b=dest/f'{scene}-base{scale}',dest/f'{scene}-hd{scale}'
            ra,rb=json.loads((a/'receipt.json').read_text()),json.loads((b/'receipt.json').read_text())
            fields=('binary_sha256','initial_state_sha256','initial_steps','initial_poke','steps','memory_sha256','palette_sha256','registers','keys_sent','keys','key_names','key_from','key_every','shots')
            assert all(ra[k]==rb[k] for k in fields),(scene,scale,'原版狀態不同')
            if reference is not None:
                assert all(ra[k]==reference[k] for k in fields),(scene,'二倍與三倍的原版狀態不同')
            else:reference=ra
            assert ra['binary_sha256']==sha((OUT/'issue22-cok-hd-probe').read_bytes())
            ha,hb=hashes(a/'scratch'),hashes(b/'scratch');assert ha==hb,(scene,'暫存檔案不同')
            end_a,end_b=sha((a/'end.state').read_bytes()),sha((b/'end.state').read_bytes());assert end_a==end_b,(scene,'完整state不同')
            groups.append(dict(scene=scene,scale=scale,without_hd=ra,with_hd=rb,scratch=ha,end_state_sha256=end_a,same_final_state=True))
            changed=0
            for s in ra['shots']:
                name=s['name'];raw=(a/(name+'.bin')).read_bytes();assert raw==(b/(name+'.bin')).read_bytes()
                pa,pb=np.array(Image.open(a/(name+'.png')).convert('RGBA')),np.array(Image.open(b/(name+'.png')).convert('RGBA'))
                delta=np.any(pa!=pb,axis=2);expected_diff=0;outside=0
                if scale==2:
                    assert not delta.any(),(scene,name,'二倍變動')
                else:
                    predicted,mask,hits,plans=o.expected(raw);mask=np.repeat(np.repeat(mask,3,0),3,1)
                    want=pa.copy();want[mask]=predicted[mask]
                    expected_diff=int(np.any(want!=pb,axis=2).sum());outside=int(delta[~mask].sum())
                    assert expected_diff==0 and outside==0,(scene,name,expected_diff,outside)
                    changed+=bool(delta.any())
                rows.append(dict(scene=scene,scale=scale,name=name,indexed_sha256=s['indexed_sha256'],palette_sha256=s['palette_sha256'],changed_pixels=int(delta.sum()),expected_diff=expected_diff,outside_difference=outside,ok=True))
            assert scale==2 or changed>0,(scene,'缺少三倍HD正例')
    write(dest/'state-comparison.json',dict(groups=groups))
    write(dest/'receipt.json',dict(all_pass=True,binary_sha256=sha((OUT/'issue22-cok-hd-probe').read_bytes()),items=rows,scope='同一最終二進位、起始state、輸入、虛擬時間與亂數記憶體；CPU、1MiB RAM、色盤、原色號、暫存檔案及完整終點state相同。既有state續跑，不是冷開機驗收。'))
    print('A/B',len(rows),'all pass')

def coverage():
    o=Issue22Oracle();dest=OUT/'issue22-coverage';dest.mkdir(exist_ok=True)
    discovery=json.loads((INV/'composition-discovery.json').read_text())
    ambiguous={(r['position'],r['indexed_sha256']) for r in discovery['ambiguous']}
    summaries=[]
    for scene in SCENES:
        a,b=OUT/'issue22-runtime'/f'{scene}-base3',OUT/'issue22-runtime'/f'{scene}-hd3'
        r=json.loads((a/'receipt.json').read_text());candidate=[];seen=set();seen_inner=set();selected=[];extra=[]
        for s in r['shots']:
            raw=(a/(s['name']+'.bin')).read_bytes();p=np.frombuffer(raw,np.uint8).reshape(200,320);plans=o.walls.matches(raw)
            if not is_3d(raw) or sum(c is not None and c['guard'] for c in plans)<3:continue
            sig=sha(p[8:176,8:176].tobytes());inner=sha(p[24:112,24:112].tobytes())
            if sig in seen:continue
            seen.add(sig);item=(s,raw,plans,inner)
            if inner not in seen_inner:seen_inner.add(inner);candidate.append(item)
            else:extra.append(item)
        selected=(candidate+extra)[:10]
        sheet=Image.new('RGB',(1100,((len(selected)+1)//2)*310),(28,28,36));draw=ImageDraw.Draw(sheet)
        totals=Counter();outer_totals=Counter();frame_rows=[]
        for n,(s,raw,plans,inner) in enumerate(selected):
            name=s['name'];p=np.frombuffer(raw,np.uint8).reshape(200,320)
            pa,pb=np.array(Image.open(a/(name+'.png')).convert('RGBA')),np.array(Image.open(b/(name+'.png')).convert('RGBA'))
            other,othermask,otherhits=o.other.expected(raw)
            personmask=np.zeros((200,320),bool)
            for it,src,art in o.other.encounters:
                px,py=it['position'];hh,ww=src.shape;mask=src!=0;q=p[py:py+hh,px:px+ww]
                if np.array_equal(q[mask],np.where(src==13,0,src)[mask]):personmask[py:py+hh,px:px+ww]|=mask
            for it,src,art in o.exact:
                if it.get('family') not in ('portrait-head','portrait-body'):continue
                hh,ww=src.shape
                for px,py in exact_positions(p,src):personmask[py:py+hh,px:px+ww]=True
            counts=Counter();cells=[]
            for pos,c in enumerate(plans):
                x,y=24+pos%11*8,24+pos//11*8
                cell=p[y:y+8,x:x+8].tobytes();key=(pos,sha(cell));subtype=''
                if c is not None and c['paint']:
                    reason='已匹配HD';subtype='來源完整吻合'
                elif np.any(personmask[y:y+8,x:x+8]):
                    reason='人物或怪物遮擋';subtype='其他HD家族的完整來源已辨識，牆面格不完整'
                elif c is not None or key in ambiguous:
                    reason='其他';subtype='來源候選產生不同HD，保留原版'
                else:
                    reason='無來源模板';subtype='完整64色號未被來源模板或已驗合成重建'
                changed=bool(np.any(pa[y*3:(y+8)*3,x*3:(x+8)*3]!=pb[y*3:(y+8)*3,x*3:(x+8)*3]))
                counts[reason]+=1
                if reason=='已匹配HD':counts['matched_with_visible_change' if changed else 'matched_identical_to_native']+=1
                cells.append(dict(position=pos,rect=[x,y,8,8],indexed_sha256=sha(cell),source_key=None if c is None else c['key'],reason=reason,detail=subtype,visible_hd_change=changed))
            outer=[];oc=Counter(counts)
            for gy in range(21):
                for gx in range(21):
                    x,y=8+gx*8,8+gy*8
                    if 24<=x<112 and 24<=y<112:continue
                    text_zone=(136<=x<176 and 16<=y<112) or (136<=x<176 and 120<=y<128) or (8<=x<176 and 136<=y<176)
                    reason='文字' if text_zone and np.any(p[y:y+8,x:x+8]!=0) else '其他'
                    detail='非3D的隊伍、座標或訊息文字格' if reason=='文字' else '非3D的面板、邊框或留白'
                    outer.append(dict(rect=[x,y,8,8],reason=reason,detail=detail));oc[reason]+=1
            assert sum(counts.get(k,0) for k in ('已匹配HD','人物或怪物遮擋','文字','無來源模板','其他'))==121
            assert sum(oc.get(k,0) for k in ('已匹配HD','人物或怪物遮擋','文字','無來源模板','其他'))==441
            totals.update(counts);outer_totals.update(oc)
            frame_rows.append(dict(name=name,steps=s['steps'],indexed_sha256=sha(raw),palette_sha256=s['palette_sha256'],inner_viewport_sha256=inner,requested_viewport_sha256=sha(p[8:176,8:176].tobytes()),base_png=str((a/(name+'.png')).relative_to(ROOT)),hd_png=str((b/(name+'.png')).relative_to(ROOT)),inner_counts=dict(counts),requested_counts=dict(oc),inner_cells=cells,outer_cells=outer,other_source_matches=otherhits))
            x,y=n%2*550,n//2*310;draw.text((x+4,y+2),scene+' '+name,fill='white');draw.text((x+4,y+16),f'HD {counts["已匹配HD"]}/121; unknown {counts["無來源模板"]}; actor {counts["人物或怪物遮擋"]}; other {counts["其他"]}',fill='white')
            sheet.paste(Image.fromarray(pa[72:336,72:336]).convert('RGB'),(x+4,y+38));sheet.paste(Image.fromarray(pb[72:336,72:336]).convert('RGB'),(x+278,y+38))
            for c in cells:
                if c['reason']=='已匹配HD':continue
                cx,cy,_,_=c['rect'];L,T=x+278+(cx-24)*3,y+38+(cy-24)*3
                color={'人物或怪物遮擋':'yellow','其他':'orange','無來源模板':'red','文字':'cyan'}[c['reason']]
                draw.rectangle((L,T,L+23,T+23),outline=color,width=2)
        sheet_path=dest/(scene+'-contact-sheet.png');sheet.save(sheet_path)
        total=len(selected)*121;outer_total=len(selected)*441
        summary=dict(scene=scene,samples=len(selected),sample_gate=len(selected)>=10,unique_inner_viewports=len({r['inner_viewport_sha256'] for r in frame_rows}),eligible_distinct_inner_viewports=len(candidate),eligible_distinct_requested_viewports=len(candidate)+len(extra),inner_total=total,inner_counts=dict(totals),inner_hd_coverage=totals['已匹配HD']/total if total else 0,requested_total=outer_total,requested_counts=dict(outer_totals),requested_hd_coverage=outer_totals['已匹配HD']/outer_total if outer_total else 0,contact_sheet=str(sheet_path.relative_to(ROOT)),initial_state_sha256=r['initial_state_sha256'],initial_steps=r['initial_steps'],initial_poke=r['initial_poke'],key_names=r['key_names'])
        write(dest/(scene+'.json'),dict(summary=summary,frames=frame_rows));summaries.append(summary)
    write(dest/'summary.json',dict(items=summaries,inner_rect=[24,24,88,88],requested_rect=[8,8,168,168],selection='按原版收據順序，優先10個不同內景；不足時補不同168×168畫格，另列實際不同視角數。門檻及候選在比較HD覆蓋率前固定，不按覆蓋率選幀。',legend='已匹配HD含純色或全留底等與原版相同的輸出；實際改變另計。紅框無來源、黃框人物怪物遮擋、橙框來源歧義。外圍320格按非3D文字與其他面板分類。'))
    p=OUT/'issue22-runtime/demo-base3';r=json.loads((p/'receipt.json').read_text());raw=(p/(r['shots'][0]['name']+'.bin')).read_bytes()
    im=native(raw).convert('RGB');d=ImageDraw.Draw(im)
    d.rectangle((24,24,527,527),outline='cyan',width=3);d.rectangle((72,72,335,335),outline='yellow',width=3)
    d.text((24,530),'ROI (8,8)..(176,176): 21 x 21 cells',fill='cyan',stroke_width=1,stroke_fill='black')
    d.text((72,340),'3D (24,24)..(112,112): 11 x 11 cells',fill='yellow',stroke_width=1,stroke_fill='black')
    im.save(dest/'viewport-geometry.png')
    print([{k:s[k] for k in ('scene','samples','unique_inner_viewports','inner_hd_coverage','inner_counts')} for s in summaries])

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('action',choices=['overview','discover','frames','runtime','coverage']);a=p.parse_args();globals()[a.action]()
