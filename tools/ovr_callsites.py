#!/usr/bin/env python3
"""把 dosgolem probe 紀錄裡的執行期呼叫端換算成 GAME.OVR 檔案位置。

輸入是同一次執行、帶 `-reads-of GAME.OVR` 與 `-call-args ... -call-length-string` 的 probe 紀錄。
每次 overlay 載入是兩筆讀取：程式碼（目的段:0000）接著重定位表；重定位表的檔案位置
緊接程式碼結尾，會被讀進另一個暫用段，這裡把它排除。對每個呼叫端
(步數, CS:IP)，找該步數之前最後一次把程式碼讀到 CS 段的那筆，輸出
`ovr@<檔案位置>+<IP>`。CS 不是 overlay 載入段時輸出 `resident`。
"""
import re
import sys
from collections import defaultdict

READ = re.compile(r'#(\d+)\s+GAME\.OVR\s+handle=\w+ → ([0-9A-F]{4}):0000 要 (\d+) 得 (\d+)')
POS = re.compile(r'#(\d+) read GAME\.OVR @(\d+) \(0x[0-9A-F]+\) (\d+) bytes')
HDR = re.compile(r'^([0-9A-F]{4}:[0-9A-F]{4}) 被呼叫')
CALL = re.compile(r'#(\d+)\s+由 ([0-9A-F]{4}):([0-9A-F]{4})\s.*\| string="(.*)"$')


def main(path):
    text = open(path, encoding='utf-8').read().split('\n')
    pos = {}
    for l in text:
        m = POS.search(l)
        if m:
            pos.setdefault(int(m[1]), (int(m[2]), int(m[3])))
    loads = defaultdict(list)  # seg -> [(step, file_off, size)]
    reads = []
    for l in text:
        m = READ.search(l)
        if m:
            step, seg, got = int(m[1]), int(m[2], 16), int(m[4])
            off, n = pos.get(step, (None, None))
            reads.append((step, seg, off, got))
    prev_end = None
    for step, seg, off, got in sorted(reads):
        if off is not None and off == prev_end:
            prev_end = None  # 重定位表
            continue
        loads[seg].append((step, off, got))
        prev_end = off + got if off is not None else None
    callee = None
    rows = defaultdict(set)
    for l in text:
        h = HDR.match(l)
        if h:
            callee = h[1]
            continue
        m = CALL.search(l)
        if not m or callee is None:
            continue
        step, cs, ip, s = int(m[1]), int(m[2], 16), int(m[3], 16), m[4]
        cands = [x for x in loads.get(cs, []) if x[0] < step]
        if cands:
            _, off, size = cands[-1]
            where = f'ovr@0x{off:05X}+0x{ip:04X}' if off is not None else f'ovr@?+0x{ip:04X}'
            if ip >= size:
                where += '(超出載入長度)'
        else:
            where = 'resident'
        rows[(callee, where, f'{cs:04X}:{ip:04X}')].add(s)
    print('callee\tcallsite\truntime\tsample')
    for (callee, where, rt), ss in sorted(rows.items()):
        sample = sorted(ss, key=len, reverse=True)[0]
        print(f'{callee}\t{where}\t{rt}\t{sample[:50]}')


if __name__ == '__main__':
    main(sys.argv[1])
