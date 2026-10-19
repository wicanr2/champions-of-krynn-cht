#!/usr/bin/env python3
"""規格010 §6：產生 live.go 最小接線供審查；不修改原工作樹。"""
import argparse
import difflib
import json
from pathlib import Path
from hd_battle_inventory import sha,write

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',required=True,type=Path);a=ap.parse_args();root=a.root
    app=root/'workplace/dosgolem/apps/cok';out=root/'workplace/hd/out';patch=[];mapping={};records=[]
    for name in ('live.go','hd.go'):
        before=(app/name).read_text();after=before
        if name=='live.go':
            pairs=[('type Live struct {','type Live struct {\n\t*HDLayers // 規格010 §6，僅讀原版畫面。'),
                ('return &Live{Layer:', 'return &Live{HDLayers: &HDLayers{}, Layer:'),
                ('func (l *Live) Frame(indexed []byte, palette [256][3]uint8) {','func (l *Live) Frame(indexed []byte, palette [256][3]uint8) {\n\tl.HDLayers.Frame(indexed, palette)'),
                ('if overlay {\n\t\tl.Layer.Draw(out, scale, missing)','if overlay {\n\t\tl.HDLayers.Draw(out, l.indexed, scale)\n\t\tl.Layer.Draw(out, scale, missing)')]
        else:
            pairs=[('for _, it := range items {\n\t\tl.Layer.Watch(it.Watcher())','layers, count, err := LoadHDLayers(dir)\n\tif err != nil { return err }\n\tfor _, it := range items {\n\t\tl.Layer.Watch(it.Watcher())'),
                ('l.HDItems += len(items)','l.HDLayers = layers\n\tl.HDItems += len(items) + count')]
        for old,new in pairs:
            assert after.count(old)==1,(name,old);after=after.replace(old,new)
        path=out/(name.removesuffix('.go')+'-with-hd.go');path.write_text(after)
        mapping['/src/apps/cok/'+name]='/hd/out/'+path.name
        patch.extend(difflib.unified_diff(before.splitlines(True),after.splitlines(True),fromfile='a/apps/cok/'+name,tofile='b/apps/cok/'+name))
        records.append(dict(file='apps/cok/'+name,before_sha256=sha(before.encode()),after_sha256=sha(after.encode())))
    (out/'frontend-integration.patch').write_text(''.join(patch));write(out/'frontend-overlay.json',dict(Replace=mapping))
    write(out/'frontend-integration.json',dict(status='awaiting-write-scope-confirmation',files=records,
        reason='使用者禁止修改live.go；此處只產生可審查差異，沒有套用。',music_hooks='所有配樂程式與掛點原樣保留'))
    print(''.join(patch))

if __name__=='__main__':main()
