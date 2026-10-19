#!/usr/bin/env python3
"""用程式碼位元組把執行期 overlay 段對回 GAME.OVR 的檔案位置。

輸入：GAME.OVR 與 probe `-dump-seg` 傾印的段（`seg-<段>_0000.bin`）。
重定位會改寫段值 word，所以不要求整段相同：在傾印中取多個 8 bytes 錨點，
於 GAME.OVR 找候選起點，再以前 N bytes 的相同比例評分。
"""
import sys
from pathlib import Path


def best(ovr, seg, n=512):
    n = min(n, len(seg))
    cands = set()
    for a in range(0, min(256, n - 8), 8):
        anchor = seg[a:a + 8]
        p = ovr.find(anchor)
        while p >= 0:
            cands.add(p - a)
            p = ovr.find(anchor, p + 1)
    scored = []
    for c in cands:
        if c < 0 or c + n > len(ovr):
            continue
        same = sum(1 for i in range(n) if ovr[c + i] == seg[i])
        scored.append((same / n, c))
    return sorted(scored, reverse=True)[:3]


def main(ovr_path, *dumps):
    ovr = Path(ovr_path).read_bytes()
    for d in dumps:
        seg = Path(d).read_bytes()
        res = best(ovr, seg)
        print(Path(d).name, ' '.join(f'ovr@0x{c:05X}:{r:.3f}' for r, c in res) or '無候選')


if __name__ == '__main__':
    main(*sys.argv[1:])
