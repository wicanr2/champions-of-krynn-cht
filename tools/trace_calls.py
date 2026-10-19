#!/usr/bin/env python3
"""由 dosgolem probe `-trace-file` 軌跡推出呼叫與返回序列。

軌跡每行只有 CS:IP 與暫存器，沒有助記符；以「CS:IP 不連續且 SP 減 2／4」判為 near／far 呼叫，
「CS:IP 不連續且 SP 增加 2–12」判為返回。這是啟發式，中斷與手動改 SP 的程式碼會誤判。

用法：trace_calls.py <trace> [--last N] [--min-sp HEX] [--into SEG]
"""
import argparse
import re

LINE = re.compile(r'#(\d+) (\w+):(\w+) .*SP=(\w+)')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('trace')
    ap.add_argument('--last', type=int, default=60)
    ap.add_argument('--min-sp', type=lambda x: int(x, 16), default=0)
    ap.add_argument('--into', type=lambda x: int(x, 16))
    a = ap.parse_args()
    rows = []
    for l in open(a.trace):
        m = LINE.match(l)
        if m:
            rows.append((int(m[1]), int(m[2], 16), int(m[3], 16), int(m[4], 16)))
    ev = []
    for p, q in zip(rows, rows[1:]):
        d = p[3] - q[3]
        jump = not (q[1] == p[1] and 0 < q[2] - p[2] <= 6)
        if not jump or q[3] < a.min_sp:
            continue
        if a.into is not None and q[1] != a.into:
            continue
        if d in (2, 4):
            ev.append(f'CALL{d} {p[1]:04X}:{p[2]:04X} -> {q[1]:04X}:{q[2]:04X} SP={q[3]:04X} #{p[0]}')
        elif -12 <= d <= -2:
            ev.append(f'RET   {p[1]:04X}:{p[2]:04X} -> {q[1]:04X}:{q[2]:04X} SP={q[3]:04X} #{p[0]}')
    for e in ev[-a.last:]:
        print(e)


if __name__ == '__main__':
    main()
