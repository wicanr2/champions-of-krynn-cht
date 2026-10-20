#!/usr/bin/env python3
"""規格010 §7：獨立Pillow整幀預期、來源負例與dosgolem A/B。"""
import argparse
from collections import Counter
import json
import shutil
from pathlib import Path
import numpy as np
from PIL import Image
from hd_battle_inventory import sha,write
from hd_battle_receipts import Oracle
from hd_wall_inventory import saved_screens
from dax_pics import EGA

class WallOracle:
    def __init__(self,theme):
        self.global_cells={};self.cells=[{} for _ in range(121)];self.assets={}
        m=json.loads((theme/'walls.json').read_text());self.manifest=m
        for it in m['assets']:
            w,h=it['size'];raw=(theme/it['indexed']).read_bytes();assert sha(raw)==it['indexed_sha256']
            assert sha((theme/it['png']).read_bytes())==it['png_sha256']
            self.assets[it['key']]=(it,Image.fromarray(np.frombuffer(raw,np.uint8).reshape(h,w)),Image.open(theme/it['png']).convert('RGBA'))
        for it in m['cells']:
            # Pillow有界paste獨立處理負偏移及來源13遮罩，與Go逐點迴圈分開。
            ref=Image.new('L',(8,8),it['base_color']);pix=Image.new('RGBA',(24,24),(*EGA[it['base_color']],255));owner=Image.new('L',(8,8),0)
            for layer in it['layers']:
                spec,src,art=self.assets[layer['key']];x,y=layer['at'];mask=src.point(lambda p:0 if p==13 else 255)
                ref.paste(src,(x,y),mask);owner.paste(255 if spec['kind']=='wall' else 0,(x,y,x+src.width,y+src.height),mask)
                pix.paste(art,(x*3,y*3),mask.resize((src.width*3,src.height*3),Image.Resampling.NEAREST))
            raw=ref.tobytes();assert sha(raw)==it['indexed_sha256'];arr=np.array(ref);owned=np.array(owner)>0
            guard=owned.sum()>=16 and len(np.unique(arr[owned]))>=2
            if not it['paint']:pix=Image.fromarray(np.dstack((np.repeat(np.repeat(np.array(EGA,np.uint8)[arr],3,0),3,1),np.full((24,24),255,np.uint8))))
            row=dict(key=it['key'],raw=raw,pix=np.array(pix),guard=guard,paint=it['paint'])
            if it['positions']:
                for pos in it['positions']:
                    assert raw not in self.cells[pos];self.cells[pos][raw]=row
            else:
                assert raw not in self.global_cells;self.global_cells[raw]=row

    def apply(self,raw):
        p=np.frombuffer(raw,np.uint8).reshape(200,320);plans=[]
        for pos in range(121):
            x,y=24+pos%11*8,24+pos//11*8;key=p[y:y+8,x:x+8].tobytes();it=self.global_cells.get(key) or self.cells[pos].get(key)
            if it is None:return None,[]
            plans.append(it)
        if sum(it['guard'] for it in plans)<3:return None,[]
        panel=np.empty((264,264,4),np.uint8)
        for pos,it in enumerate(plans):x,y=pos%11*24,pos//11*24;panel[y:y+24,x:x+24]=it['pix']
        return panel,[it['key'] for it in plans if it['guard']]

class CombinedOracle:
    def __init__(self,theme):self.other=Oracle(theme);self.walls=WallOracle(theme)
    def expected(self,raw):
        out,allowed,hits=self.other.expected(raw);panel,wallhits=self.walls.apply(raw)
        if panel is not None:
            mask=np.repeat(np.repeat(allowed[24:112,24:112],3,0),3,1);region=out[72:336,72:336];region[~mask]=panel[~mask]
            allowed[24:112,24:112]=True;hits+=['walls:'+k for k in wallhits]
        return out,allowed,hits

def frames(root):
    hd=root/'workplace/hd';dest=hd/'out/issue14-frames'
    if dest.exists():shutil.rmtree(dest)
    dest.mkdir();oracle=CombinedOracle(hd/'theme');tasks=[]
    for name,raw in saved_screens(root):tasks.append((name,'saved-original' if 'issue14-runtime' not in name else 'dosgolem-state-continuation',raw))
    # 每個觀測視窗的一點失配、偏移、清除及恢復，與原版遊戲擷取分組。
    models=json.loads((hd/'inventory/issue14/model.json').read_text())
    for row in models['results']:
        p=root/row['screen']
        if p.suffix=='.png':
            old=json.loads((hd/'screens/normal/manifest.json').read_text());entry=next(it for it in old['items'] if it['source']==row['screen']);raw=(hd/'screens/normal'/entry['file']).read_bytes()
        else:raw=p.read_bytes()
        changed=bytearray(raw);changed[24*320+24]=16
        tasks.append((row['screen']+'-unknown-pixel','derived-negative',bytes(changed)))
        arr=np.frombuffer(raw,np.uint8).reshape(200,320).copy();arr[24:112,25:113]=arr[24:112,24:112].copy();arr[24:112,24]=0
        tasks.append((row['screen']+'-offset','derived-negative',arr.tobytes()))
    # 通用不留底牆面各來源組抽樣，驗證格位、至少三個守衛與新素材撤除邊界。
    guards=[r for r in oracle.walls.global_cells.values() if r['guard']][:3]
    refs=list(oracle.walls.global_cells.values());sample=refs[::max(1,len(refs)//30)]
    for it in sample:
        arr=np.zeros((200,320),np.uint8)
        for pos,g in zip((12,13,14),guards):x,y=24+pos%11*8,24+pos//11*8;arr[y:y+8,x:x+8]=np.frombuffer(g['raw'],np.uint8).reshape(8,8)
        arr[64:72,64:72]=np.frombuffer(it['raw'],np.uint8).reshape(8,8)
        tasks.append((it['key'],'source-derived-positive',arr.tobytes()))
    cases=[]
    for n,(name,group,raw) in enumerate(tasks):
        # 非EGA負例只驗證牆面回退，Go的完整色盤負例另由單元測試覆蓋。
        if max(raw)>15:
            raw=bytes(15 if c>15 else c for c in raw)
            if oracle.walls.apply(raw)[0] is not None:continue
        expected,allowed,hits=oracle.expected(raw);src=np.array(EGA,np.uint8)[np.frombuffer(raw,np.uint8).reshape(200,320)]
        original=np.dstack((np.repeat(np.repeat(src,3,0),3,1),np.full((600,960),255,np.uint8)))
        stem=f'{n:04d}';idx=dest/(stem+'.bin');ep=dest/(stem+'-expected.png');op=dest/(stem+'-original.png');idx.write_bytes(raw)
        Image.fromarray(expected).save(ep,compress_level=1);Image.fromarray(original).save(op,compress_level=1)
        def path(p):return '/hd/'+str(p.relative_to(hd))
        cases.append(dict(name=name,group=group,indexed=path(idx),indexed_sha256=sha(raw),expected=path(ep),expected_sha256=sha(ep.read_bytes()),original=path(op),original_sha256=sha(op.read_bytes()),matches=hits))
    write(dest/'cases.json',dict(cases=cases,scope='來源案例與保存原版畫面分組；Pillow獨立合成，不代替原版執行。'))
    print('frames',len(cases),dict(Counter(c['group'] for c in cases)))

def hashes(path):return {str(p.relative_to(path)):sha(p.read_bytes()) for p in sorted(path.rglob('*')) if p.is_file()}

def preview(root):
    hd=root/'workplace/hd';dest=hd/'out/issue14-preview';dest.mkdir(exist_ok=True);oracle=WallOracle(hd/'out/issue14-theme-preview')
    model=json.loads((hd/'inventory/issue14/model.json').read_text());cases=[]
    for area,substring in (('throtl','throtl-source/final'),('gargath','gargath-entry/'),('sanction','sanction-source/')):
        entry=next(r for r in model['results'] if substring in r['screen']);raw=(root/entry['screen']).read_bytes();panel,hits=oracle.apply(raw);assert panel is not None
        p=np.frombuffer(raw,np.uint8).reshape(200,320);native=np.dstack((np.repeat(np.repeat(np.array(EGA,np.uint8)[p],3,0),3,1),np.full((600,960),255,np.uint8)));out=native.copy();out[72:336,72:336]=panel
        idx=dest/(area+'.bin');ep=dest/(area+'-hd.png');op=dest/(area+'-original.png');idx.write_bytes(raw);Image.fromarray(out).save(ep);Image.fromarray(native).save(op)
        sheet=Image.new('RGB',(528,264));sheet.paste(Image.fromarray(native[72:336,72:336]).convert('RGB'),(0,0));sheet.paste(Image.fromarray(panel).convert('RGB'),(264,0));sheet.save(dest/(area+'-compare.png'))
        def pp(p):return '/hd/'+str(p.relative_to(hd))
        cases.append(dict(name=area,group='source-verified-runtime-preview',indexed=pp(idx),indexed_sha256=sha(raw),expected=pp(ep),expected_sha256=sha(ep.read_bytes()),original=pp(op),original_sha256=sha(op.read_bytes())))
    write(dest/'cases.json',dict(cases=cases))
    print('three source-verified viewport previews')

def runtime(root):
    hd=root/'workplace/hd';dest=hd/'out/issue14-runtime';oracle=CombinedOracle(hd/'theme');rows=[];groups=[]
    for area in ('throtl','gargath','sanction'):
        changed3=0
        for scale in (3,2):
            left=dest/f'{area}-base{scale}';right=dest/f'{area}-hd{scale}';a=json.loads((left/'receipt.json').read_text());b=json.loads((right/'receipt.json').read_text())
            fields=('initial_state_sha256','initial_steps','initial_poke','steps','memory_sha256','registers','keys_sent','keys','key_names','key_from','key_every')
            assert all(a[k]==b[k] for k in fields),area
            assert hashes(left/'scratch')==hashes(right/'scratch'),area
            assert a['shots']==b['shots'],area
            group=dict(area=area,scale=scale,without_hd=a,with_hd=b,scratch_without_hd=hashes(left/'scratch'),scratch_with_hd=hashes(right/'scratch'))
            groups.append(group)
            for shot in a['shots']:
                name=shot['name'];raw=(left/(name+'.bin')).read_bytes();assert raw==(right/(name+'.bin')).read_bytes()
                pa=np.array(Image.open(left/(name+'.png')).convert('RGBA'));pb=np.array(Image.open(right/(name+'.png')).convert('RGBA'));difference=np.any(pa!=pb,axis=2)
                predicted,allowed,hits=oracle.expected(raw);mask=np.repeat(np.repeat(allowed,scale,0),scale,1)
                outside=int(difference[~mask].sum());expected_diff=int(difference.sum())
                if scale==3:
                    want=pa.copy();want[mask]=predicted[mask];expected_diff=int(np.any(want!=pb,axis=2).sum())
                assert outside==0 and expected_diff==0,(area,name,scale,outside,expected_diff)
                wall=any(k.startswith('walls:') for k in hits)
                if scale==3 and wall and difference.any():changed3+=1
                rows.append(dict(area=area,name=name,steps=shot['steps'],scale=scale,display_difference=int(difference.sum()),outside_difference=outside,independent_expected_difference=expected_diff,wall_match=wall,source_matches=hits,ok=True))
        assert changed3>0,(area,'沒有三倍牆面HD正例')
    write(dest/'state-comparison.json',dict(groups=groups))
    write(dest/'receipt.json',dict(issue=14,binary_sha256=sha((hd/'out/issue14-cok-hd-probe').read_bytes()),all_pass=True,
        seed='兩側在執行前由相同state固定原版RNG記憶體與虛擬時鐘，無重擲；Gargath座標注入明列initial_poke。',
        scope='既有狀態續跑，含診斷歷史；非新冷開機。終點RAM、CPU、原版色號與暫存層比較，非逐畫格RAM。',items=rows))
    print('runtime',len(rows),'all pass','walls3',sum(r['scale']==3 and r['wall_match'] and r['display_difference']>0 for r in rows))

if __name__=='__main__':
    ap=argparse.ArgumentParser();ap.add_argument('action',choices=['frames','runtime','preview']);ap.add_argument('--root',required=True,type=Path);a=ap.parse_args();globals()[a.action](a.root)
