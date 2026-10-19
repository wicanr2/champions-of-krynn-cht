"""G1 原始輸入盤點：壓縮檔雜湊、成員清單、解壓與逐檔雜湊。"""
import hashlib, json, sys, zipfile
from pathlib import Path

src, out = Path(sys.argv[1]), Path(sys.argv[2])
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
rows = []
with zipfile.ZipFile(src) as z:
    for i in z.infolist():
        rows.append({"name": i.filename, "size": i.file_size, "crc": f"{i.CRC:08x}", "date": "%04d-%02d-%02d %02d:%02d" % i.date_time[:5]})
    z.extractall(out / "dos")
files = [{"path": str(p.relative_to(out / "dos")), "size": p.stat().st_size, "sha256": sha(p)}
         for p in sorted((out / "dos").rglob("*")) if p.is_file()]
json.dump({"archive": src.name, "sha256": sha(src), "members": rows, "files": files},
          open(out / "inventory.json", "w"), ensure_ascii=False, indent=1)
print(src.name, sha(src))
for f in files: print(f"{f['size']:>8} {f['sha256'][:16]} {f['path']}")
