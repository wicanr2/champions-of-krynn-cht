"""對齊比對兩個執行檔中的同一函式，列出不同的 byte 位置（找出逐版本位址差異）。
用法：fn_diff.py <檔A> <起點A hex> <檔B> <起點B hex> <長度 hex>"""
import sys
a = open(sys.argv[1], 'rb').read(); oa = int(sys.argv[2], 16)
b = open(sys.argv[3], 'rb').read(); ob = int(sys.argv[4], 16)
n = int(sys.argv[5], 16)
i = 0
while i < n:
    if a[oa + i] != b[ob + i]:
        j = i
        while j < n and (a[oa + j] != b[ob + j] or j - i < 2):
            j += 1
        print(f'+{i:04X}  A {a[oa+i-2:oa+j].hex(" ")}  |  B {b[ob+i-2:ob+j].hex(" ")}')
        i = j
    else:
        i += 1
