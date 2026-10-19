#!/usr/bin/env python3
"""HD 批次驗收與並列原型（G7，#8；規格 010 §3）。

讀請求 manifest（target_size、原版參考圖）與 Codex 輸出目錄，逐項逐風格檢查：
- 檔案存在、PNG 為 RGBA、尺寸等於 target_size、alpha 全為 255（首批皆不透明）
- 對位：HD 縮回 1 倍後的亮度邊緣與原版邊緣重合率（相對分數，供人判讀）
- 色相：HD 縮回 1 倍後與原版同像素的色相差（規格 010 §2.4 配色跟原版一致）
輸出 `<out>/compare-<key>.png`：原版最近鄰放大｜風格 A｜風格 B，並印出檢查表。
"""
import argparse
import json
import struct
import sys
import zlib
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parent))
from dax_pics import EGA  # noqa: E402


def read_png(p):
    """以 PNG IHDR 保留原色彩型態，Pillow 解碼成 RGBA；避免逐點 Python 解濾波。"""
    from PIL import Image
    d = Path(p).read_bytes()
    if d[:8] != b'\x89PNG\r\n\x1a\n' or d[12:16] != b'IHDR':
        raise ValueError(f'{p}：不是 PNG')
    w, h, bd, ct, _, _, il = struct.unpack('>IIBBBBB', d[16:29])
    if bd != 8 or il or ct not in (0, 2, 3, 4, 6):
        raise ValueError(f'{p}：不支援 bit depth {bd}、color type {ct} 或交錯 {il}')
    with Image.open(p) as im:
        px = np.array(im.convert('RGBA'), dtype=np.uint8)
    if px.shape != (h, w, 4):
        raise ValueError(f'{p}：PNG 尺寸不合')
    return ct, px


def write_png(p, rgb):
    h, w, _ = rgb.shape
    raw = b''.join(b'\x00' + rgb[y].tobytes() for y in range(h))

    def chunk(t, d):
        return struct.pack('>I', len(d)) + t + d + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
    Path(p).write_bytes(b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
                        + chunk(b'IDAT', zlib.compress(raw, 6)) + chunk(b'IEND', b''))


def edges(lum):
    gx = np.abs(np.diff(lum, axis=1))[:-1, :]
    gy = np.abs(np.diff(lum, axis=0))[:, :-1]
    g = gx + gy
    return g > np.percentile(g, 85)


def align_score(orig_rgb, hd_rgb):
    """HD 以區塊平均縮回原版尺寸，比較強邊緣重合率；另試 ±2 像素平移，回 (原位分數, 最佳分數, 最佳平移)。"""
    h, w, _ = orig_rgb.shape
    H, W, _ = hd_rgb.shape
    s = H // h
    small = hd_rgb[:h * s, :w * s, :3].reshape(h, s, w, s, 3).mean(axis=(1, 3))
    lum = lambda a: a[..., 0] * .299 + a[..., 1] * .587 + a[..., 2] * .114  # noqa: E731
    eo, eh = edges(lum(orig_rgb.astype(float))), edges(lum(small))
    best = (0, (0, 0))
    base = 0
    for dy in range(-2, 3):
        for dx in range(-2, 3):
            sh = np.roll(np.roll(eh, dy, 0), dx, 1)
            sc = (sh & eo).sum() / max(1, eo.sum())
            if (dy, dx) == (0, 0):
                base = sc
            if sc > best[0]:
                best = (sc, (dx, dy))
    return base, best[0], best[1]


def hue_score(orig_rgb, hd_rgb):
    """色相對照：HD 以區塊平均縮回 1 倍，在原版與 HD 都有彩度的像素比色相差（度）。回 (中位數, 超過 30 度的比例)。"""
    h, w, _ = orig_rgb.shape
    s = hd_rgb.shape[0] // h
    small = hd_rgb[:h * s, :w * s, :3].reshape(h, s, w, s, 3).mean(axis=(1, 3))

    def hs(a):
        a = a.astype(float) / 255
        mx, mn = a.max(axis=2), a.min(axis=2)
        d = mx - mn
        sat = np.where(mx > 0, d / np.maximum(mx, 1e-9), 0)
        r, g, b = a[..., 0], a[..., 1], a[..., 2]
        dd = np.maximum(d, 1e-9)
        hue = np.where(mx == r, (g - b) / dd % 6, np.where(mx == g, (b - r) / dd + 2, (r - g) / dd + 4)) * 60
        return hue, sat
    ho, so = hs(orig_rgb)
    hh, sh = hs(small)
    m = (so > 0.3) & (sh > 0.15)
    if not m.any():
        return 0.0, 0.0
    dh = np.abs(ho - hh)[m]
    dh = np.minimum(dh, 360 - dh)
    return float(np.median(dh)), float((dh > 30).mean())


def lowpass(rgb):
    """規格 010 §20.2：對稱 [1,2,1]/4 分離卷積，抑制一像素網點。"""
    rgb = rgb.astype(float)
    for axis in (0, 1):
        padded = np.pad(rgb, [(1, 1) if k == axis else (0, 0) for k in range(3)], mode='edge')
        if axis == 0:
            rgb = (padded[:-2] + 2 * padded[1:-1] + padded[2:]) / 4
        else:
            rgb = (padded[:, :-2] + 2 * padded[:, 1:-1] + padded[:, 2:]) / 4
    return rgb


def structural_score(orig_rgb, hd_rgb):
    """原始邊緣指標另存；雙側去網點後比較同位置輪廓，不搜尋最佳平移。"""
    h, w = orig_rgb.shape[:2]
    sy, sx = hd_rgb.shape[0] // h, hd_rgb.shape[1] // w
    small = hd_rgb[..., :3].reshape(h, sy, w, sx, 3).mean(axis=(1, 3))
    filtered = np.repeat(np.repeat(lowpass(small), sy, 0), sx, 1)
    return float(align_score(lowpass(orig_rgb), filtered)[0])


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--req', required=True, type=Path)
    ap.add_argument('--hd', required=True, type=Path)
    ap.add_argument('--out', required=True, type=Path)
    ap.add_argument('--styles', default='A,B', help='逗號分隔；- 表示 <key>.png')
    a = ap.parse_args()
    a.out.mkdir(parents=True, exist_ok=True)
    man = json.loads((a.req / 'manifest.json').read_text(encoding='utf-8'))
    ok = True
    for it in man['items']:
        key, (W, H) = it['key'], it['target_size']
        w, h = it['size']
        idx = np.frombuffer((a.req / it['indexed']).read_bytes(), np.uint8).reshape(h, w)
        orig = np.array(EGA, np.uint8)[idx]
        s = W // w
        panels = [np.repeat(np.repeat(orig, s, 0), s, 1)]
        for st in a.styles.split(','):
            f = a.hd / (f'{key}.png' if st == '-' else f'{key}-{st}.png')
            if not f.exists():
                print(f'{key}-{st}：缺檔 失敗')
                ok = False
                panels.append(np.zeros((H, W, 3), np.uint8))
                continue
            ct, px = read_png(f)
            good = ct == 6 and px.shape[:2] == (H, W) and (px[..., 3] == 255).all()
            msg = f'{key}-{st}：{px.shape[1]}×{px.shape[0]} 色彩型態 {ct} alpha {"全不透明" if (px[..., 3] == 255).all() else "含透明"}'
            if px.shape[:2] == (H, W):
                base, best, sh = align_score(orig, px)
                msg += f' 邊緣重合 {base:.2f}（±2 最佳 {best:.2f} 於 {sh}）'
                hm, hf = hue_score(orig, px)
                msg += f' 色相差中位 {hm:.0f} 度、超過 30 度 {hf:.0%}'
                panels.append(px[..., :3])
                anim = it.get('composition') or it.get('animation') or {}
                if anim.get('mask'):
                    # 動畫狀態（規格 010 §12）：XOR 遮罩向外擴 6 像素以外，必須與 @0 逐像素相同
                    from PIL import Image, ImageFilter
                    m = np.asarray(Image.open(a.req / anim['mask']).convert('L').filter(ImageFilter.MaxFilter(13))) > 0
                    bf = a.hd / (f'{anim["base"]}.png' if st == '-' else f'{anim["base"]}-{st}.png')
                    off = int((read_png(bf)[1][..., :3] != px[..., :3]).any(axis=2)[~m].sum()) if bf.exists() else -1
                    msg += f' 遮罩外與 {anim["base"]} 不同 {off} 像素'
                    good &= off == 0
            else:
                panels.append(np.zeros((H, W, 3), np.uint8))
            print(msg + (' 通過' if good else ' 失敗'))
            ok &= good
        gap = np.full((H, 8, 3), 40, np.uint8)
        sheet = panels[0]
        for p in panels[1:]:
            sheet = np.concatenate([sheet, gap, p], axis=1)
        write_png(a.out / f'compare-{key}.png', np.ascontiguousarray(sheet))
    sys.exit(0 if ok else 1)


if __name__ == '__main__':
    main()
