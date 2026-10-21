#!/usr/bin/env python3
"""圓體點陣字型：源泉圓體（GenSenRounded2 TW，SIL OFL 1.1）轉 GOLEMFNT（主專案使用者 2026-10-21 指示）。

--cell 16 給 2 倍畫面（16×16），--cell 24 給 3 倍（24×24）。字集同 eten_font.py（譯文表要顯示的欄加可列印 ASCII）。
- 全形字：以字面框（ascender 880、descender 120 單位）對齊，字級等於格高，水平置中，50% 門檻轉 1 bit。
- ASCII：畫進半格（16 格墨跡在 x=4..11，24 格在 x=6..17），同倚天的半形位置，讓組合字（規格 005 §2）成立；
  太寬的字母水平壓縮到半格。
- 字型沒有的字以 --fallback（Unifont hex，經 eten_font.load_unifont）補，24 格時最近鄰放大；--sample 輸出樣張 PNG。
GOLEMFNT：magic、LE W、H、字數，每字 u32 碼點＋u8 來源標記（7＝本工具、8＝Unifont 補字）＋字模，碼點遞增。
"""
import argparse
import hashlib
import struct
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, str(Path(__file__).resolve().parent))
from eten_font import charset, load_unifont  # noqa: E402

GENSEN = {  # 檔名：SHA-256 前 16 碼（github.com/ButTaiwan/gensen-font v2.100 GenSenRounded2TW-otf.zip）
    'GenSenRounded2TW-M.otf': 'b8286a42c534d31c', 'GenSenRounded2TW-B.otf': '050f9ebf394faab3',
}


def to_bits(img, cell, thr):
    rb = (cell + 7) // 8
    out = bytearray(rb * cell)
    px = img.load()
    for y in range(cell):
        for x in range(cell):
            if px[x, y] >= thr:
                out[y * rb + x // 8] |= 0x80 >> (x % 8)
    return bytes(out)


def full(font, ch, cell, thr):
    g = Image.new('L', (cell, cell), 0)
    d = ImageDraw.Draw(g)
    w = d.textlength(ch, font=font)
    d.text(((cell - w) / 2, cell * 0.88), ch, fill=255, font=font, anchor='ls')
    return to_bits(g, cell, thr)


def half(font, ch, cell, thr):
    h, q = cell // 2, cell // 4
    big = Image.new('L', (cell * 2, cell), 0)
    d = ImageDraw.Draw(big)
    d.text((cell // 2, cell * 0.88), ch, fill=255, font=font, anchor='ls')
    box = big.getbbox()
    g = Image.new('L', (cell, cell), 0)
    if box:
        x0, x1 = box[0], box[2]
        piece = big.crop((x0, 0, x1, cell))
        if piece.width > h:
            piece = piece.resize((h, cell), Image.LANCZOS)
        g.paste(piece, (q + (h - piece.width) // 2, 0))
    return to_bits(g, cell, thr)


def from_unifont(bm16, cell):
    if cell == 16:
        return bm16
    rb = (cell + 7) // 8
    out = bytearray(rb * cell)
    for y in range(cell):
        sy = y * 16 // cell
        row = bm16[sy * 2] << 8 | bm16[sy * 2 + 1]
        for x in range(cell):
            if row & (0x8000 >> (x * 16 // cell)):
                out[y * rb + x // 8] |= 0x80 >> (x % 8)
    return bytes(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('catalogs', nargs='*')
    ap.add_argument('--otf', required=True)
    ap.add_argument('--cell', type=int, choices=(16, 24), required=True)
    ap.add_argument('--threshold', type=int, default=128)
    ap.add_argument('--fallback', help='Unifont hex（字型沒有的字）')
    ap.add_argument('--out')
    ap.add_argument('--sample', help='只畫這段文字的樣張 PNG 到 --out')
    a = ap.parse_args()
    if GENSEN.get(Path(a.otf).name) != hashlib.sha256(Path(a.otf).read_bytes()).hexdigest()[:16]:
        sys.exit(f'{a.otf}：來源身份不符（需 {"、".join(GENSEN)} 之一，雜湊相符）')
    font = ImageFont.truetype(a.otf, a.cell)
    notdef = bytes(font.getmask('\ue000'))  # 私用區字一定不在字型裡，畫出來就是 .notdef 方框
    cmap_ok = lambda ch: bytes(font.getmask(ch)) != notdef  # noqa: E731
    if a.sample:
        cell = a.cell
        img = Image.new('L', (cell * len(a.sample), cell), 0)
        for i, ch in enumerate(a.sample):
            bm = half(font, ch, cell, a.threshold) if ord(ch) < 0x80 else full(font, ch, cell, a.threshold)
            rb = (cell + 7) // 8
            for y in range(cell):
                for x in range(cell):
                    if bm[y * rb + x // 8] & (0x80 >> (x % 8)):
                        img.putpixel((i * cell + x, y), 255)
        img.save(a.out)
        return
    uni = load_unifont(a.fallback) if a.fallback else {}
    chars = charset(a.catalogs, ''.join(chr(c) for c in range(0x20, 0x7F)))
    body, missing, fallback = [], [], 0
    for ch in chars:
        if ord(ch) < 0x80:
            tag, bm = 7, half(font, ch, a.cell, a.threshold)
        elif cmap_ok(ch):
            tag, bm = 7, full(font, ch, a.cell, a.threshold)
        elif ord(ch) in uni:
            tag, bm = 8, from_unifont(uni[ord(ch)], a.cell)
            fallback += 1
        else:
            missing.append(ch)
            continue
        body.append(struct.pack('<IB', ord(ch), tag) + bm)
    if missing:
        sys.exit(f'缺字 {len(missing)}：{"".join(missing[:40])}')
    data = b'GOLEMFNT' + struct.pack('<HHI', a.cell, a.cell, len(body)) + b''.join(body)
    Path(a.out).write_bytes(data)
    print(f'{len(body)} 字（Unifont 補 {fallback}），{a.cell}×{a.cell}，SHA-256 {hashlib.sha256(data).hexdigest()}')


if __name__ == '__main__':
    main()
