#!/usr/bin/env python3
"""由原版 ECL*.DAX 靜態抽取敘事字串（移植自拯救地球 tools/ecl_text_catalog.py，ECL 檔數改依實際檔案），產生不含原文的 catalog。

輸出：
  --events   text/ecl-text-events.tsv（key、長度、SHA-256、來源位置；不含原文）
  --source   ignored 工作區的 key→原文對照（只供翻譯，不得入版控）

DAX 容器與 6-bit 解碼同 golden-box-remake-engine 的 dax、ecl/text.go，
但不去頭尾空白：原版把整串（含尾隨空白）交給印字程序，雜湊要逐位元相同。
候選判準：0x80 長度前綴、解碼後只含允許字元、90% 以上 token 像英文詞、
首尾 token 都像詞、長度 > 3 且含空白（第 1 層）；規格 027 §3.1 修訂另收第 2 層：
第 1 層不成立，但長度 ≥ 2、含字母、不要求空白。這是啟發式，會漏字串也會收雜訊；
執行期以整串雜湊比對，漏掉的字串顯示英文，雜訊永遠不會命中。
"""
import argparse, hashlib, re, struct, sys
from pathlib import Path

CLEAN = re.compile(r"[A-Z0-9 .,'!?\-:;\"()/&%$#*+<>=]+")
EDGE = re.compile(r"^[\"'(<\-.,!?;:)>=*]+|[\"'(<\-.,!?;:)>=*]+$")
CORE = re.compile(r"^(?:[A-Z]+(?:[.'\-/][A-Z]+)*(?:\(S\))?|[0-9]+(?:[.,:][0-9]+)*(?:ST|ND|RD|TH|S|%|CR)?)$")


def word_ok(tok):
    core = EDGE.sub('', tok)
    return core == '' or bool(CORE.match(core))


def dax_blocks(data):
    head = struct.unpack('<H', data[:2])[0] + 2
    for p in range(2, head, 9):
        bid = data[p]
        off, dec, packed = struct.unpack('<IHH', data[p + 1:p + 9])
        src = data[head + off:head + off + packed]
        out = bytearray()
        q = 0
        while q < len(src) and len(out) < dec:
            c = src[q]
            q += 1
            if c < 128:
                out += src[q:q + c + 1]
                q += c + 1
            else:
                out += bytes([src[q]]) * (256 - c)
                q += 1
        if len(out) != dec:
            raise ValueError(f'block {bid} decoded {len(out)} != {dec}')
        yield bid, bytes(out)


def decode6(payload):
    s, state, prev = [], 1, 0

    def put(v):
        if v:
            s.append(chr(v + 0x40 if v <= 0x1f else v))
    for c in payload:
        if state == 1:
            put((c >> 2) & 63)
            state = 2
        elif state == 2:
            put(((prev << 4) | (c >> 4)) & 63)
            state = 3
        else:
            put(((prev << 2) | (c >> 6)) & 63)
            put(c & 63)
            state = 1
        prev = c
    return ''.join(s)


def wordy(t):
    toks = [x for x in re.split(r' |\.\.\.+|--+', t) if x]
    if not toks or not word_ok(toks[0]) or not word_ok(toks[-1]):
        return False
    return sum(1 for x in toks if word_ok(x)) / len(toks) >= 0.9


def tier(t):
    """規格 027 §3.1：1、2，或 0（不收）。只看字串本身。"""
    v = t.strip()
    if not (any(c.isalpha() for c in v) and CLEAN.fullmatch(v) and wordy(v)):
        return 0
    if len(v) > 3 and ' ' in v:
        return 1
    return 2 if len(v) >= 2 else 0


def extract(orig):
    found = {}
    for path in sorted(orig.glob('ECL*.DAX')):
        n = int(path.stem[3:])
        for bid, blk in dax_blocks(path.read_bytes()):
            for i in range(len(blk) - 2):
                if blk[i] != 0x80:
                    continue
                ln = blk[i + 1]
                if i + 2 + ln > len(blk):
                    continue
                t = decode6(blk[i + 2:i + 2 + ln])
                if tier(t):
                    found.setdefault(t, []).append(f'ECL{n}:{bid}:{i}')
    return found


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--orig', required=True, type=Path)
    ap.add_argument('--events', required=True, type=Path)
    ap.add_argument('--source', required=True, type=Path)
    a = ap.parse_args()
    found = extract(a.orig)
    ev = ['event_key\toriginal_length\toriginal_sha256\tsources']
    src = ['key\toriginal\tsources\ttier']
    for t, locs in found.items():
        n, b, o = locs[0].split(':')
        key = f'ecl.{n[3:]}.{int(b):02d}.{int(o):05d}'
        raw = t.encode('ascii')
        if len(raw) > 255:
            continue
        ev.append(f'{key}\t{len(raw)}\t{hashlib.sha256(raw).hexdigest()}\t{";".join(locs)}')
        src.append(f'{key}\t{t}\t{";".join(locs)}\t{tier(t)}')
    keys = [l.split('\t')[0] for l in ev[1:]]
    if len(keys) != len(set(keys)):
        sys.exit('duplicate key')
    a.events.write_text('\n'.join(ev) + '\n', encoding='utf-8')
    a.source.write_text('\n'.join(src) + '\n', encoding='utf-8')
    print(f'{len(ev) - 1} strings, {sum(len(t) for t in found)} chars', file=sys.stderr)


if __name__ == '__main__':
    main()
