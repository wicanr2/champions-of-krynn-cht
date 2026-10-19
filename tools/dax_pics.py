#!/usr/bin/env python3
"""盤點原版 DAX 圖檔（G7 HD 來源辨識，#8）。

格式依 golden-box-remake-engine `dax/dax.go`（DOS RLE）與 `graphics/picture.go`（17 bytes 表頭＋4 bit 色號，
高半位元組在前；寬單位 8 像素）。每個區塊試解為圖片，輸出：
- `<out>/inventory.tsv`：file、block、width、height、items、x、y、decoded_sha256、status
- `<out>/<FILE>/<block>-<item>.png`：EGA 預設 16 色、1 倍原尺寸（只在 workplace，不進 Git）
- `<out>/<FILE>.png`：該檔全部區塊的縮圖總表（2 倍）
"""
import argparse
import hashlib
import struct
import zlib
from pathlib import Path

EGA = [(0, 0, 0), (0, 0, 170), (0, 170, 0), (0, 170, 170), (170, 0, 0), (170, 0, 170), (170, 85, 0), (170, 170, 170),
       (85, 85, 85), (85, 85, 255), (85, 255, 85), (85, 255, 255), (255, 85, 85), (255, 85, 255), (255, 255, 85),
       (255, 255, 255)]


def png(path, w, h, rgb):
    raw = b''.join(b'\x00' + bytes(rgb[y * w * 3:(y + 1) * w * 3]) for y in range(h))

    def chunk(t, d):
        return struct.pack('>I', len(d)) + t + d + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
    Path(path).write_bytes(b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
                           + chunk(b'IDAT', zlib.compress(raw, 6)) + chunk(b'IEND', b''))


def rle(packed, expected):
    out = bytearray()
    pos = 0
    while pos < len(packed) and len(out) < expected:
        c = packed[pos] - 256 if packed[pos] > 127 else packed[pos]
        pos += 1
        if c >= 0:
            out += packed[pos:pos + c + 1]
            pos += c + 1
        else:
            out += bytes([packed[pos]]) * (-c)
            pos += 1
    if len(out) != expected:
        raise ValueError(f'RLE {len(out)} != {expected}')
    return bytes(out)


def dax(data):
    hdr = struct.unpack_from('<H', data)[0] + 2
    blocks = []
    for pos in range(2, hdr, 9):
        bid, off, dsize, psize = struct.unpack_from('<BIHH', data, pos)
        start = hdr + off
        blocks.append((bid, rle(data[start:start + psize], dsize)))
    return blocks


def picture(d):
    """回 (w, h, items, x, y, pixels)；不像圖片就丟 ValueError。

    8X8D 圖塊集表頭的 items 欄存 255，實際是 256 個（長度 17＋256×32）：以長度推得的件數為準。
    """
    if len(d) < 17:
        raise ValueError('短於表頭')
    h, wu, x, y, items = struct.unpack_from('<HHhhB', d)
    w = wu * 8
    if w and h and items and (len(d) - 17) % (w * h // 2) == 0 and (len(d) - 17) // (w * h // 2) == items + 1:
        items += 1
    if not w or not h or not items or len(d) != 17 + items * w * h // 2:
        raise ValueError(f'尺寸不合 {w}x{h} items={items} len={len(d)}')
    px = bytearray()
    for b in d[17:]:
        px += bytes((b >> 4, b & 15))
    return w, h, items, x, y, bytes(px)


def frames(d):
    """PIC／SPRIT 動畫：首位元組幀數，每幀 4 bytes（延遲，未證實）＋圖片表頭＋像素。回 [(delay, picture…)]。"""
    n, pos, out = d[0], 1, []
    if not 1 <= n <= 64:
        raise ValueError(f'幀數 {n}')
    for _ in range(n):
        if pos + 21 > len(d):
            raise ValueError('幀超出區塊')
        delay = struct.unpack_from('<I', d, pos)[0]
        h, wu = struct.unpack_from('<HH', d, pos + 4)
        items = d[pos + 12]
        size = 17 + items * wu * 8 * h // 2
        out.append((delay,) + picture(d[pos + 4:pos + 4 + size]))
        pos += 4 + size
    if pos != len(d):
        raise ValueError(f'幀長度合計 {pos} != {len(d)}')
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    a.out.mkdir(parents=True, exist_ok=True)
    rows = ['file\tblock\twidth\theight\titems\tx\ty\tdecoded_sha256\tstatus']
    for f in sorted(a.orig.glob('*.DAX')):
        try:
            blocks = dax(f.read_bytes())
        except Exception as e:  # noqa: BLE001
            rows.append(f'{f.name}\t\t\t\t\t\t\t\t容器解不開：{e}')
            continue
        tiles = []
        for bid, d in blocks:
            sha = hashlib.sha256(d).hexdigest()
            try:
                pics = [('ok',) + picture(d)]
            except ValueError as e:
                try:
                    pics = [(f'幀 {k} 延遲 {fr[0]}',) + fr[1:] for k, fr in enumerate(frames(d))]
                except ValueError as e2:
                    rows.append(f'{f.name}\t{bid}\t\t\t\t\t\t{sha}\t非圖片：{e}；非動畫：{e2}')
                    continue
            d_out = a.out / f.stem
            d_out.mkdir(exist_ok=True)
            for k, (status, w, h, items, x, y, px) in enumerate(pics):
                rows.append(f'{f.name}\t{bid}\t{w}\t{h}\t{items}\t{x}\t{y}\t{sha}\t{status}')
                for it in range(items):
                    p = px[it * w * h:(it + 1) * w * h]
                    rgb = bytearray()
                    for c in p:
                        rgb += bytes(EGA[c])
                    png(d_out / f'{bid:03d}-{k}-{it}.png', w, h, rgb)
                    tiles.append((w, h, rgb))
        if not tiles:
            continue
        # 縮圖總表：依序橫排，寬 640，2 倍
        sw = 640
        lines, cur, cw, ch = [], [], 0, 0
        for w, h, rgb in tiles:
            if cw + w * 2 + 4 > sw and cur:
                lines.append((cur, ch)); cur, cw, ch = [], 0, 0
            cur.append((cw, w, h, rgb)); cw += w * 2 + 4; ch = max(ch, h * 2 + 4)
        if cur:
            lines.append((cur, ch))
        H = sum(c for _, c in lines)
        canvas = bytearray([40, 40, 60] * sw * H)
        oy = 0
        for line, ch in lines:
            for ox, w, h, rgb in line:
                for yy in range(h * 2):
                    for xx in range(min(w * 2, sw - ox)):
                        s = ((yy // 2) * w + xx // 2) * 3
                        t = ((oy + yy) * sw + ox + xx) * 3
                        canvas[t:t + 3] = rgb[s:s + 3]
            oy += ch
        png(a.out / f'{f.stem}.png', sw, H, canvas)
    (a.out / 'inventory.tsv').write_text('\n'.join(rows) + '\n', encoding='utf-8')
    ok = sum(1 for r in rows[1:] if '\t非圖片' not in r)
    print(f'{len(rows) - 1} 區塊，圖片 {ok}')


if __name__ == '__main__':
    main()
