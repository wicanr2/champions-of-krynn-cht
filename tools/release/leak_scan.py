"""發行包外洩掃描（規格 012 §3.2；取自拯救地球 tools/release）：包內任何檔案的 SHA-256 或檔名（大小寫不拘）
與禁止來源相同即失敗。

    python3 leak_scan.py <包的解開目錄> <禁止來源目錄>...

只印統計與命中的包內路徑，不印禁止來源的內容。
"""
import hashlib
import os
import sys


def files(root):
    for d, _, names in os.walk(root):
        for n in names:
            p = os.path.join(d, n)
            if os.path.isfile(p) and not os.path.islink(p):
                yield p


def sha(p):
    h = hashlib.sha256()
    with open(p, "rb") as f:
        for b in iter(lambda: f.read(1 << 20), b""):
            h.update(b)
    return h.hexdigest()


def main():
    pkg, sources = sys.argv[1], sys.argv[2:]
    bad_sha, bad_name = set(), set()
    for s in sources:
        if not os.path.isdir(s):
            raise SystemExit(f"禁止來源不存在：{s}")
        for p in files(s):
            if os.path.getsize(p) > 0:
                bad_sha.add(sha(p))
            bad_name.add(os.path.basename(p).upper())
    hits, n = [], 0
    for p in files(pkg):
        n += 1
        if sha(p) in bad_sha or os.path.basename(p).upper() in bad_name:
            hits.append(os.path.relpath(p, pkg))
    print(f"掃描 {n} 個檔案，比對 {len(bad_sha)} 個雜湊、{len(bad_name)} 個檔名")
    if hits:
        raise SystemExit("外洩：" + "、".join(hits))


if __name__ == "__main__":
    main()
