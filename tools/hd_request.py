#!/usr/bin/env python3
"""產生 HD 重繪請求批次（G7，#8；規格 010）：原版圖 PNG＋manifest.json（AGENTS.md 第 6 節欄位）。

每個項目由 DAX 區塊組成，依定位證據（tools/hd_locate.py 的 locate.tsv）貼到畫面位置，
再以實際畫面色號核對：組好的圖必須與證據畫面同區域逐像素相同，否則中止。
輸出：
- `<out>/<key>.png`：原版 1 倍（EGA 預設 16 色）
- `<out>/<key>-x<倍率>.png`：最近鄰放大，供構圖參考
- `<out>/<key>.idx`：原版色號（逐列，每像素 1 byte），供驗收比對
- `<out>/manifest.json`
"""
import argparse
import hashlib
import json
import struct
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from dax_pics import EGA, dax, frames, picture, png  # noqa: E402

# key：說明、畫面位置 (x, y, w, h)、證據畫面、組成 [(DAX 檔, 區塊, 貼上 x, 貼上 y, 取用列 y0, y1)]
BATCHES = {
    # 本作首批（G8，#9）：證據畫面在 workplace/probe/ 底下的 hd-screens（docs/re/009）
    'b01': [
        ('title', '標題畫面：TITLE 區塊 2 全幅背景，區塊 3 標題字樣貼在 (40,8)', (0, 0, 320, 200), '../hd/screens/title.bin',
         [('TITLE.DAX', 2, 0, 0, 0, 200), ('TITLE.DAX', 3, 40, 8, 0, 88)]),
        ('camp', '視窗圖：營火（PIC1 區塊 59，PIC2、PIC3 同圖）', (24, 24, 88, 88), '../hd/screens/camp.bin',
         [('PIC1.DAX', 59, 24, 24, 0, 88)]),
        ('outpost', '視窗圖：卡爾爵士肖像（PIC1 區塊 57，屯墾區畫面）', (24, 24, 88, 88), '../hd/screens/outpost.bin',
         [('PIC1.DAX', 57, 24, 24, 0, 88)]),
        ('solace', '大圖：索拉斯最後家園旅店（BIGPIC1 區塊 112）', (8, 8, 304, 120), '../hd/screens/solace.bin',
         [('BIGPIC1.DAX', 112, 8, 8, 0, 120)]),
        ('caravan', '大圖：龍人襲擊商隊（BIGPIC1 區塊 114）', (8, 8, 304, 120), '../hd/screens/caravan.bin',
         [('BIGPIC1.DAX', 114, 8, 8, 0, 120)]),
    ],
}
ANIM = {}  # 動畫圖（PIC 多幀）首批不做
SOURCE_BATCHES = {}  # 只有來源、沒有執行畫面證據的圖（幽靈騎士 b12 起）首批不做

def anim_states(raw, w, h):
    """回 [(幀號, 色號狀態, XOR 遮罩)]，重複的狀態只留第一個。"""
    fr = [px[:w * h] for _, _, _, _, _, _, px in frames(raw)]
    out, seen = [(0, bytes(fr[0]), None)], {bytes(fr[0])}
    for k in range(1, len(fr)):
        st = bytes(a ^ b for a, b in zip(fr[0], fr[k]))
        if st not in seen:
            seen.add(st)
            out.append((k, st, bytes(1 if c else 0 for c in fr[k])))
    return out


def offsets(data):
    hdr = struct.unpack_from('<H', data)[0] + 2
    out = {}
    for pos in range(2, hdr, 9):
        bid, off, dsize, psize = struct.unpack_from('<BIHH', data, pos)
        out[bid] = {'offset': hdr + off, 'packed_size': psize, 'decoded_size': dsize}
    return out


def block_pixels(raw):
    try:
        w, h, items, _, _, px = picture(raw)
    except ValueError:
        _, w, h, items, _, _, px = frames(raw)[0]
    return w, h, px[:w * h]


def source(orig, fname, bid, **extra):
    data = (orig / fname).read_bytes()
    raw = dict(dax(data))[bid]
    return raw, {'file': fname, 'file_sha256': hashlib.sha256(data).hexdigest(), 'block': bid,
                 **offsets(data)[bid], 'decoded_sha256': hashlib.sha256(raw).hexdigest(), **extra}


def rgb_png(path, canvas, W, H, s, pal=None):
    pal = pal or EGA
    big = bytearray()
    for yy in range(H * s):
        for xx in range(W * s):
            big += bytes(pal[canvas[(yy // s) * W + xx // s]])
    png(path, W * s, H * s, big)


def emit(out, s, key, desc, rect, screen, canvas, sources, **extra):
    X, Y, W, H = rect
    rgb_png(out / f'{key}.png', canvas, W, H, 1)
    rgb_png(out / f'{key}-x{s}.png', canvas, W, H, s)
    (out / f'{key}.idx').write_bytes(bytes(canvas))
    return {'key': key, 'description': desc, 'screen_rect': [X, Y, W, H], 'size': [W, H],
            'target_size': [W * s, H * s], 'evidence_screen': f'workplace/probe/{screen}' if screen else None,
            'png': f'{key}.png', 'png_sha256': hashlib.sha256((out / f'{key}.png').read_bytes()).hexdigest(),
            'reference_png': f'{key}-x{s}.png', 'indexed': f'{key}.idx',
            'indexed_sha256': hashlib.sha256(bytes(canvas)).hexdigest(), 'transparent': None, 'sources': sources, **extra}


def region(probe, screen, rect):
    X, Y, W, H = rect
    scr = (probe / screen).read_bytes()
    return bytes(scr[(Y + yy) * 320 + X + xx] & 15 for yy in range(H) for xx in range(W))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--probe', required=True, type=Path)
    ap.add_argument('--batch', required=True)
    ap.add_argument('--scale', type=int, default=4)
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    a.out.mkdir(parents=True, exist_ok=True)
    s = a.scale
    items = []
    for key, desc, (X, Y, W, H), screen, parts in BATCHES[a.batch]:
        canvas = bytearray(W * H)
        sources = []
        for fname, bid, px_, py_, y0, y1 in parts:
            raw, src = source(a.orig, fname, bid, rows_used=[y0, y1], placed_at=[px_, py_])
            w, h, px = block_pixels(raw)
            src['block_size'] = [w, h]
            for yy in range(y0, y1):
                for xx in range(w):
                    canvas[(py_ + yy - y0 - Y) * W + px_ + xx - X] = px[yy * w + xx]
            sources.append(src)
        if region(a.probe, screen, (X, Y, W, H)) != bytes(canvas):
            sys.exit(f'{key}：與證據畫面 {screen} 不同')
        items.append(emit(a.out, s, key, desc, (X, Y, W, H), screen, canvas, sources))
    for key, desc, (X, Y, W, H), screen, fname, bid in ANIM.get(a.batch, []):
        raw, src = source(a.orig, fname, bid, placed_at=[X, Y], block_size=[W, H])
        states = anim_states(raw, W, H)
        ev = region(a.probe, screen, (X, Y, W, H))
        if ev not in [st for _, st, _ in states]:
            sys.exit(f'{key}：證據畫面 {screen} 不等於任何動畫狀態')
        for k, st, mask in states:
            extra = {'animation': {'rule': 'XOR 疊在第 0 幀', 'frame': k, 'base': f'{key}@0',
                                   'evidence_state': st == ev}}
            if mask:
                mk = f'{key}@{k}-mask-x{s}.png'
                rgb_png(a.out / mk, mask, W, H, s, [(0, 0, 0), (255, 255, 255)])
                extra['animation']['mask'] = mk
            items.append(emit(a.out, s, f'{key}@{k}', desc, (X, Y, W, H), screen if st == ev else None,
                              st, [dict(src, frame=k)], **extra))
    for fname, bid in SOURCE_BATCHES.get(a.batch, []):
        X, Y, W, H = (8, 8, 304, 120) if fname.startswith('BIGPIC') else (24, 24, 88, 88)
        raw, src = source(a.orig, fname, bid, placed_at=[X, Y], block_size=[W, H])
        stem = f'{Path(fname).stem.lower()}-{bid:03d}'
        try:
            w, h, count, _, _, px = picture(raw)
            assert (w, h, count) == (W, H, 1), (fname, bid, w, h, count)
            states = [(0, bytes(px[:w*h]), None)]
            animated = False
        except ValueError:
            decoded_frames = frames(raw)
            assert all(f[1:4] == (W, H, 1) for f in decoded_frames), (fname, bid, '來源幀形狀不符')
            states = anim_states(raw, W, H)
            animated = True
        for k, px, mask in states:
            key = f'{stem}@{k}' if animated else stem
            extra = {'runtime_evidence': 'unobserved', 'verification': 'DAX 解碼與完整色號匹配契約；不是執行器畫面收據'}
            if animated:
                extra['animation'] = dict(rule='XOR 疊在第 0 幀', frame=k, base=f'{stem}@0', evidence_state=False)
                if mask:
                    mk = f'{key}-mask-x{s}.png'
                    rgb_png(a.out / mk, mask, W, H, s, [(0,0,0), (255,255,255)])
                    extra['animation']['mask'] = mk
            items.append(emit(a.out, s, key, f'{fname} 區塊 {bid} 狀態 {k}', (X,Y,W,H), None,
                              px, [dict(src, frame=k)], **extra))
    manifest = {
        'batch': a.batch, 'spec': 'docs/spec/010-hd-theme-draft.md',
        'issue': {'b12': '#24', 'b13': '#25', 'b14': '#26', 'b15': '#27'}.get(a.batch, '#8'),
        'palette': {'name': 'EGA 預設 16 色', 'rgb': [list(c) for c in EGA],
                    'evidence': 'probe 畫面 PNG 與色號陣列比對（g2-cold/d050m、g3-ingame/k1），16 色全吻合'},
        'scale': s, 'items': items,
    }
    (a.out / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'{a.batch}：{len(items)} 項，倍率 {s}')


if __name__ == '__main__':
    main()
