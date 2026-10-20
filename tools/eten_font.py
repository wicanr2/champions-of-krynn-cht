#!/usr/bin/env python3
"""建立 16×16 GOLEMFNT 子集：本機倚天 15 點字型（開發與本機完整版用，不可散布），或 --unifont 的 GNU Unifont（發行用，規格 012）。

分區、對齊與失敗條件依拯救地球 docs/spec/008-eten-top-pad-local-font-builder-draft.md：
- 非 ASCII 字以 CPython `big5` codec 編成兩 bytes，raw=(hi-0xA1)*157+column(lo)。
- ASCII U+0000..U+00FF 取 ASCFONT.15（每格 15 bytes），左移 4 bits，墨跡在 x=4..11。
- raw 0..407 取 SPCFONT.15；471..5871 取 STDFONT.15[raw-471]；6280..13972 取 STDFONT.15[5401+raw-6280]。
- top-pad：輸出第 0 列為零，來源 15 列放在第 1..15 列。
- 其他分區、codec 失敗、全零字模（空白除外）一律失敗，不 fallback。

輸出：GOLEMFNT magic、LE W=16、H=16、字數，每字 u32 碼點＋u8 來源標記＋32 bytes，碼點遞增。
字元集合取自一或多份 TSV 中表頭為 translation、title、body、label、summary 的欄（譯文、手札、攻略、地圖攻略點），另加 --extra 字元。
"""
import argparse
import csv
import hashlib
import struct
import sys
from pathlib import Path

SOURCES = {
    'asc': ('ASCFONT.15', 3840, '1d0cf09d0a319a9e7039190688c6a905ba4370bd369fbfcfa43b2078480d6918'),
    'spc': ('SPCFONT.15', 12240, 'f32049ba2a7a21db908878a488a2c1d93c389d17398cf46db1390ba89e247605'),
    'std': ('STDFONT.15', 392820, '39ba9c8519d75fe11d5988a8a27e6daa5794ad2ea215108390b0d7e9e53ff701'),
}


def load_source(kind, path):
    name, size, sha = SOURCES[kind]
    b = Path(path).read_bytes()
    if Path(path).name.upper() != name or len(b) != size or hashlib.sha256(b).hexdigest() != sha:
        sys.exit(f'{kind}: 來源身份不符（需 {name}、{size} bytes、{sha[:12]}…）')
    return b


def column(lo):
    if 0x40 <= lo <= 0x7E:
        return lo - 0x40
    if 0xA1 <= lo <= 0xFE:
        return lo - 0x62
    raise ValueError(f'非法 trail byte {lo:#x}')


def glyph(ch, asc, spc, std):
    cp = ord(ch)
    if cp <= 0xFF and ch.isascii():
        rows = asc[cp * 15:(cp + 1) * 15]
        out = b'\x00\x00' + b''.join(struct.pack('>H', r << 4) for r in rows)
        return 1, out
    enc = ch.encode('big5')
    if len(enc) != 2:
        raise ValueError(f'{ch!r} 不是兩 bytes Big5')
    raw = (enc[0] - 0xA1) * 157 + column(enc[1])
    if 0 <= raw <= 407:
        tag, src = 2, spc[raw * 30:(raw + 1) * 30]
    elif 471 <= raw <= 5871:
        tag, src = 3, std[(raw - 471) * 30:(raw - 470) * 30]
    elif 6280 <= raw <= 13972:
        i = 5401 + raw - 6280
        tag, src = 4, std[i * 30:(i + 1) * 30]
    else:
        raise ValueError(f'{ch!r} 落在不支援的 Big5 分區 raw={raw}')
    if len(src) != 30:
        raise ValueError(f'{ch!r} 來源索引超界')
    return tag, b'\x00\x00' + src


# 會畫到畫面上的欄：譯文、手札與攻略的標題與正文、地圖攻略點的名稱與說明
SHOWN = ('translation', 'title', 'body', 'label', 'summary')


def charset(paths, extra):
    chars = set(extra)
    for p in paths:
        rows = list(csv.reader(open(p, encoding='utf-8'), delimiter='\t', quoting=csv.QUOTE_NONE))
        cols = [i for i, h in enumerate(rows[0] if rows else []) if h in SHOWN]
        if not cols:
            sys.exit(f'{p}: 表頭沒有要顯示的欄（{"、".join(SHOWN)}）')
        for r in rows[1:]:
            for c in cols:
                if c < len(r):
                    chars.update(r[c])
    chars.discard('\t')
    return sorted(chars)


# 發行用 GNU Unifont 17.0.05（同一個 tarball 的 precompiled/）。繁中、韓文用台灣字源 unifont_t（規格 012；
# 拯救地球規格 050 同一檔），簡中用預設版 unifont（漢字取自文泉驛，大陸字形），日文用 unifont_jp（規格 015 §4.1）。
UNIFONT = {
    'unifont_t-17.0.05.hex': '169634258e4037b507beaafad5d72edc2e44b3faeaa856d9669e4657d1eee454',
    'unifont-17.0.05.hex': 'fd79af3613ec1b984a98d33428fdd43fcf06018d18059960d78edeb63d958622',
    'unifont_jp-17.0.05.hex': '3e88e5e98470e7547555202cebb13bcd4c393d417e2296a0f0f8adade7d37f43',
}


def load_unifont(path):
    """回 {碼點: 32 bytes}。8×16 字左移 4 bits（墨跡在 x=4..11，同倚天 ASCII 對齊），16×16 照用。"""
    b = Path(path).read_bytes()
    if UNIFONT.get(Path(path).name) != hashlib.sha256(b).hexdigest():
        sys.exit(f'unifont: 來源身份不符（需 {"、".join(UNIFONT)} 之一，雜湊相符）')
    out = {}
    for line in b.decode('ascii').splitlines():
        cp, _, hx = line.partition(':')
        raw = bytes.fromhex(hx)
        if len(raw) == 16:
            raw = b''.join(struct.pack('>H', r << 4) for r in raw)
        if len(raw) == 32:
            out[int(cp, 16)] = raw
    return out


def unifont_glyph(ch, uni):
    bm = uni.get(ord(ch))
    if bm is None:
        raise ValueError('Unifont 沒有這個字')
    return 5, bm


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('catalogs', nargs='+')
    ap.add_argument('--asc')
    ap.add_argument('--spc')
    ap.add_argument('--std')
    ap.add_argument('--unifont', help='改用 Unifont hex（發行用，規格 012、015 §4.1）；與 --asc／--spc／--std 擇一')
    ap.add_argument('--out', required=True)
    ap.add_argument('--extra', default=''.join(chr(c) for c in range(0x20, 0x7F)),
                    help='額外必收字元（預設全部可列印 ASCII：選單熱鍵字母與照原文接入的名字要用）')
    a = ap.parse_args()
    if a.unifont:
        uni = load_unifont(a.unifont)
        source = lambda ch: unifont_glyph(ch, uni)  # noqa: E731
    elif a.asc and a.spc and a.std:
        asc, spc, std = load_source('asc', a.asc), load_source('spc', a.spc), load_source('std', a.std)
        source = lambda ch: glyph(ch, asc, spc, std)  # noqa: E731
    else:
        sys.exit('需要 --unifont，或 --asc、--spc、--std 三個倚天來源')
    chars = charset(a.catalogs, a.extra)
    body, errors = [], []
    for ch in chars:
        try:
            tag, bm = source(ch)
        except (ValueError, UnicodeEncodeError) as e:
            errors.append(f'U+{ord(ch):04X} {ch}：{e}')
            continue
        if ch != ' ' and not any(bm):
            errors.append(f'U+{ord(ch):04X} {ch}：字模全零')
            continue
        body.append(struct.pack('<IB', ord(ch), tag) + bm)
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        sys.exit(f'{len(errors)} 字失敗，未輸出')
    data = b'GOLEMFNT' + struct.pack('<HHI', 16, 16, len(body)) + b''.join(body)
    out = Path(a.out)
    tmp = out.with_suffix(out.suffix + '.tmp')
    tmp.write_bytes(data)
    tmp.replace(out)
    print(f'{len(body)} 字，{len(data)} bytes，SHA-256 {hashlib.sha256(data).hexdigest()}')


if __name__ == '__main__':
    main()
