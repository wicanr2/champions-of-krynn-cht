"""以發行字型的字模畫應用程式圖示：「幽靈騎士」2×2 排列（規格 012；取自拯救地球 tools/appicon.py）。

    python3 tools/appicon.py <字型.golemfnt> <輸出.png> <邊長>

只用標準庫；圖示不含任何原版素材。
"""
import struct
import sys
import zlib

TEXT = "幽靈騎士"
BG = (0x10, 0x14, 0x30)
FG = (0xF0, 0xC8, 0x50)


def load_glyphs(path, wanted):
    data = open(path, "rb").read()
    if data[:8] != b"GOLEMFNT":
        raise SystemExit("不是 GOLEMFNT")
    w, h, n = struct.unpack_from("<HHI", data, 8)
    rb, off, out = (w + 7) // 8, 16, {}
    for _ in range(n):
        cp, _src = struct.unpack_from("<IB", data, off)
        off += 5
        if chr(cp) in wanted:
            out[chr(cp)] = data[off:off + rb * h]
        off += rb * h
    missing = [c for c in wanted if c not in out]
    if missing:
        raise SystemExit(f"字型缺字：{missing}")
    return w, h, out


def png(path, size, pix):
    raw = b"".join(b"\0" + bytes(pix[y * size * 3:(y + 1) * size * 3]) for y in range(size))
    def chunk(tag, body):
        return struct.pack(">I", len(body)) + tag + body + struct.pack(">I", zlib.crc32(tag + body))
    out = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
    out += chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b"")
    open(path, "wb").write(out)


def main():
    font, out, size = sys.argv[1], sys.argv[2], int(sys.argv[3])
    w, h, glyphs = load_glyphs(font, TEXT)
    rb = (w + 7) // 8
    scale = max(1, size // (2 * w + 4))
    margin = (size - 2 * w * scale) // 2
    pix = bytearray(BG * (size * size))
    for i, ch in enumerate(TEXT):
        ox, oy = margin + (i % 2) * w * scale, margin + (i // 2) * h * scale
        g = glyphs[ch]
        for gy in range(h):
            for gx in range(w):
                if g[gy * rb + gx // 8] & (0x80 >> (gx % 8)):
                    for dy in range(scale):
                        for dx in range(scale):
                            x, y = ox + gx * scale + dx, oy + gy * scale + dy
                            if 0 <= x < size and 0 <= y < size:
                                pix[(y * size + x) * 3:(y * size + x) * 3 + 3] = bytes(FG)
    png(out, size, pix)


if __name__ == "__main__":
    main()
