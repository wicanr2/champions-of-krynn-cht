#!/usr/bin/env python3
"""由 START.EXE、GAME.OVR 靜態抽取介面字串（Pascal 長度前綴），產生不含原文的 catalog。

判準沿拯救地球 tools/engine_fragments.py：長度 1–80、全部可列印 ASCII、含兩個以上連續英文字母。
這是啟發式，會收到程式碼中的偶然位元組；執行期以整串雜湊比對，雜訊永遠不會命中。

輸出：
  --events  text/ui-text-events.tsv（key、長度、SHA-256、來源；不含原文）
  --source  workplace 內的 key→原文對照（只供翻譯，不入版控）
  --anchors 選用：執行期抓到的字串（每行一則），檢查哪些不在 catalog
"""
import argparse
import hashlib
import re
from pathlib import Path


def extract(orig):
    found = {}
    for name in ('START.EXE', 'GAME.OVR'):
        d = (orig / name).read_bytes()
        for i in range(len(d) - 1):
            n = d[i]
            if 1 <= n <= 80 and i + 1 + n <= len(d):
                s = d[i + 1:i + 1 + n]
                if all(32 <= c < 127 for c in s) and re.search(rb'[A-Za-z]{2}', s):
                    found.setdefault(s.decode('ascii'), []).append(f'{name}:{i}')
    return found


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--events', required=True, type=Path)
    ap.add_argument('--source', required=True, type=Path)
    ap.add_argument('--anchors', type=Path)
    a = ap.parse_args()
    found = extract(a.orig)
    ev = ['event_key\toriginal_length\toriginal_sha256\tsources']
    src = ['key\toriginal\tsources']
    for t, locs in found.items():
        raw = t.encode('ascii')
        key = f'ui.{hashlib.sha256(raw).hexdigest()[:12]}'
        ev.append(f'{key}\t{len(raw)}\t{hashlib.sha256(raw).hexdigest()}\t{";".join(locs[:4])}')
        src.append(f'{key}\t{t}\t{";".join(locs[:4])}')
    keys = [l.split('\t')[0] for l in ev[1:]]
    if len(keys) != len(set(keys)):
        raise SystemExit('duplicate key')
    a.events.write_text('\n'.join(ev) + '\n', encoding='utf-8')
    a.source.write_text('\n'.join(src) + '\n', encoding='utf-8')
    print(f'{len(ev) - 1} strings')
    if a.anchors:
        miss = [l for l in a.anchors.read_text(encoding='utf-8').split('\n') if l and l not in found]
        hits = sum(1 for l in a.anchors.read_text(encoding='utf-8').split('\n') if l and l in found)
        print(f'anchors: {hits} hit, {len(miss)} miss')
        for m in miss:
            print('  miss', repr(m))


if __name__ == '__main__':
    main()
