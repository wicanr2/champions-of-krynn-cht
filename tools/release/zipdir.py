"""（取自拯救地球 tools/release）把目錄壓成 zip，保留執行權限位元（容器內沒有 zip 指令）。

    python3 zipdir.py <來源目錄> <輸出.zip>

zip 內的頂層是來源目錄的名稱。時間戳固定，重跑產出相同。
"""
import os
import sys
import zipfile


def main():
    src, out = os.path.abspath(sys.argv[1]), sys.argv[2]
    base = os.path.dirname(src)
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for d, dirs, names in os.walk(src):
            dirs.sort()
            for n in sorted(names):
                p = os.path.join(d, n)
                info = zipfile.ZipInfo(os.path.relpath(p, base), date_time=(2026, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = (os.stat(p).st_mode & 0xFFFF) << 16
                with open(p, "rb") as f:
                    z.writestr(info, f.read())


if __name__ == "__main__":
    main()
