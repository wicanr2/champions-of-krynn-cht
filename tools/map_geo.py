#!/usr/bin/env python3
"""由 GEO1..3.DAX 產生 text/map-geo.tsv（規格 009 §2.1）：每塊地圖的鍵、0x400 本體雜湊、名稱。

只存雜湊，不存原版位元組。名稱取自 --titles（`map<TAB>title`，人工對應；沒對到的留空）。
"""
import argparse
import hashlib
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from ecl_text_catalog import dax_blocks  # noqa: E402


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--titles', type=Path)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    titles = {}
    if a.titles and a.titles.exists():
        for line in a.titles.read_text(encoding='utf-8').splitlines()[1:]:
            k, t = line.split('\t')[:2]
            titles[k] = t
    rows = ['map\tpayload_sha256\ttitle']
    for n in (1, 2, 3):
        for bid, blk in dax_blocks((a.orig / f'GEO{n}.DAX').read_bytes()):
            if len(blk) != 0x402:
                sys.exit(f'GEO{n} 區塊 {bid:#x} 長度 {len(blk)}')
            key = f'{n}/{bid:02x}'
            rows.append(f'{key}\t{hashlib.sha256(blk[2:]).hexdigest()}\t{titles.get(key, "")}')
            # 同一塊 GEO 給兩個區用時，名稱表以 `鍵@ECL 區塊` 另列（例 3/50@51 達西默），沿用同一雜湊
            for q in sorted(k for k in titles if k.startswith(key + '@')):
                rows.append(f'{q}\t{hashlib.sha256(blk[2:]).hexdigest()}\t{titles[q]}')
    a.out.write_text('\n'.join(rows) + '\n', encoding='utf-8')
    print(f'{len(rows) - 1} 塊寫到 {a.out}')


if __name__ == '__main__':
    main()
