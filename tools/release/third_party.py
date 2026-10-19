"""由 `go version -m` 的輸出產生 THIRD-PARTY.md（規格 012；取自拯救地球 tools/release，附註改為本作內容）。

    python3 third_party.py <modinfo.txt> <模組快取> <輸出目錄>

逐一複製實際連結模組的授權檔到 <輸出目錄>/licenses/，缺授權檔即失敗。
"""
import pathlib
import shutil
import sys

SKIP = {"github.com/wicanr2/dosgolem"}  # 主模組，授權另附 LICENSE-dosgolem


def escape(path):
    return "".join("!" + c.lower() if c.isupper() else c for c in path)


def kind(text):
    t = text[:4000]
    for key, name in [("Apache License", "Apache-2.0"), ("Redistribution and use in source and binary forms", "BSD 類"),
                      ("Permission is hereby granted, free of charge", "MIT"), ("zlib", "zlib")]:
        if key in t:
            return name
    return "見授權檔"


def main():
    info, cache, out = sys.argv[1], pathlib.Path(sys.argv[2]), pathlib.Path(sys.argv[3])
    mods = []
    for line in open(info, encoding="utf-8"):
        f = line.split()
        if len(f) >= 3 and f[0] == "dep":
            mods.append((f[1], f[2]))
    (out / "licenses").mkdir(parents=True, exist_ok=True)
    rows, missing = [], []
    for path, ver in mods:
        if path in SKIP:
            continue
        d = cache / f"{escape(path)}@{ver}"
        lic = sorted(p for p in d.glob("*") if p.is_file() and p.name.upper().startswith(("LICENSE", "COPYING", "LICENCE")))
        if not lic:
            missing.append(f"{path}@{ver}")
            continue
        names = []
        for p in lic:
            dst = f"{path.replace('/', '_')}-{p.name}"
            shutil.copyfile(p, out / "licenses" / dst)
            names.append(f"licenses/{dst}")
        rows.append(f"| {path} | {ver} | {kind(lic[0].read_text(errors='replace'))} | {'、'.join(names)} |")
    if missing:
        raise SystemExit("缺授權檔：" + "、".join(missing))
    body = ["# 第三方元件", "", "由 `go version -m` 列出的實際連結模組（不含 dosgolem 本身，見 LICENSE-dosgolem）。", "",
            "| 模組 | 版本 | 授權 | 授權檔 |", "|---|---|---|---|", *rows, "",
            "另含：GNU Unifont 17.0.05 字型子集（font/OFL-1.1.txt、font/COPYING-unifont），取其台灣字形來源（unifont_t）；",
            "Nuked OPL3 的 Go 移植（dosgolem audio/nukedopl，LGPL-2.1-or-later，COPYING.LGPL、nukedopl-SOURCE.md）。", ""]
    (out / "THIRD-PARTY.md").write_text("\n".join(body), encoding="utf-8")
    print(f"{len(rows)} 個模組")


if __name__ == "__main__":
    main()
