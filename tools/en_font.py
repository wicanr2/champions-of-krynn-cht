#!/usr/bin/env python3
"""英文版字型（規格 015 §3.1）：Courier Prime Bold 水平拉寬填滿原版方形字格，輸出 GOLEMFNT。

每個可列印 ASCII 字先以 (cell×2/3)×cell 的窄格畫出（字高含上伸與下伸置中），再水平拉成 cell×cell，
以 50% 門檻轉成 1 bit。--cell 16 給 2 倍畫面（cok-en.golemfnt），--cell 24 給 3 倍（cok-en-3x.golemfnt）。
GOLEMFNT：magic、LE W、H、字數，每字 u32 碼點＋u8 來源標記（6＝本工具）＋字模，碼點遞增。
"""
import argparse
import hashlib
import struct
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

COURIER = ('CourierPrime-Bold.ttf', 'ff1f38786c849d1c')  # google/fonts ofl/courierprime，SHA-256 前 16 碼


def glyph(font, ch, cell):
    narrow = cell * 2 // 3
    top, bottom = font.getbbox('Ay|')[1], font.getbbox('Ay|')[3]
    dy = (cell - (bottom - top)) / 2 - top
    g = Image.new('L', (narrow, cell), 0)
    d = ImageDraw.Draw(g)
    d.text(((narrow - d.textlength(ch, font=font)) / 2, dy), ch, fill=255, font=font)
    g = g.resize((cell, cell), Image.LANCZOS)
    rb = (cell + 7) // 8
    out = bytearray(rb * cell)
    px = g.load()
    for y in range(cell):
        for x in range(cell):
            if px[x, y] >= 128:
                out[y * rb + x // 8] |= 0x80 >> (x % 8)
    return bytes(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--ttf', required=True)
    ap.add_argument('--cell', type=int, choices=(16, 24), required=True)
    ap.add_argument('--out', required=True)
    a = ap.parse_args()
    b = Path(a.ttf).read_bytes()
    if hashlib.sha256(b).hexdigest()[:16] != COURIER[1]:
        raise SystemExit(f'{a.ttf}：來源身份不符（需 {COURIER[0]}，SHA-256 {COURIER[1]}…）')
    font = ImageFont.truetype(a.ttf, round(a.cell * 26 / 24))
    chars = [chr(c) for c in range(0x20, 0x7F)]
    body = b''.join(struct.pack('<IB', ord(ch), 6) + glyph(font, ch, a.cell) for ch in chars)
    data = b'GOLEMFNT' + struct.pack('<HHI', a.cell, a.cell, len(chars)) + body
    Path(a.out).write_bytes(data)
    print(f'{len(chars)} 字，{a.cell}×{a.cell}，{len(data)} bytes，SHA-256 {hashlib.sha256(data).hexdigest()}')


if __name__ == '__main__':
    main()
