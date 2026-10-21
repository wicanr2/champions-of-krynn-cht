import re
BASE = "/home/anr2/cht/golden_box/champions_of_krynn"
def rows(p):
    with open(p, encoding="utf-8") as f:
        data = f.read()
    assert data.endswith("\n") and "\r" not in data
    return [l.split("\t") for l in data.split("\n")[:-1]]

bad = re.compile(r"[　-〿㐀-鿿＀-￯\U0001F000-\U0001FFFF—–]")
def chk(name, zh, ko, ncol, textcols):
    A = rows(BASE + "/text/" + zh)
    B = rows(BASE + "/workplace/tr-jako/aux-ko/" + ko)
    assert len(A) == len(B), (len(A), len(B))
    assert A[0] == B[0], (A[0], B[0])
    print(name, "rows", len(B) - 1)
    problems = 0
    for i, (a, b) in enumerate(zip(A, B)):
        if len(b) != ncol:
            print("cols", i, len(b)); problems += 1; continue
        for c in range(ncol):
            if c not in textcols and i > 0 and a[c] != b[c] and not (name == "wt" and c == 3):
                print("untranslated col changed", i, c, a[c], b[c]); problems += 1
            if b[c] != b[c].strip():
                print("space", i, c); problems += 1
            if i > 0 and c in textcols and bad.search(b[c]):
                print("badchar", i, c, bad.findall(b[c])); problems += 1
            if i > 0 and c in textcols and not b[c]:
                print("empty", i, c); problems += 1
        if i == 0:
            continue
        # coordinate preservation
        ca = re.findall(r"[（(]\s*(\d+)\s*[，,]\s*(\d+)\s*[）)]", "|".join(a[1:]))
        cb = re.findall(r"\(\s*(\d+),\s*(\d+)\)", "|".join(b[1:]))
        if name == "mg":
            ca = re.findall(r"[（(]\s*(\d+)\s*[，,]\s*(\d+)\s*[）)]", a[5])
            cb = re.findall(r"\(\s*(\d+),\s*(\d+)\)", b[5])
        if name == "wt":
            ca = re.findall(r"[（(]\s*(\d+)\s*[，,]\s*(\d+)\s*[）)]", a[2])
            cb = re.findall(r"\(\s*(\d+),\s*(\d+)\)", b[2])
        if ca != cb:
            print("coord diff", i, ca, cb); problems += 1
        # journal/tale number sets
        if name == "wt":
            if a[2].count("\\n") != b[2].count("\\n"):
                print("nl count", i, a[2].count("\\n"), b[2].count("\\n")); problems += 1
            na = re.findall(r"手札\s*([\d、]+)", a[2]); nb = re.findall(r"여행 일지 ([\d, ]+)", b[2])
            fa = sorted(x for s in na for x in re.findall(r"\d+", s)); fb = sorted(x for s in nb for x in re.findall(r"\d+", s))
            if fa != fb:
                print("journal nums", i, fa, fb); problems += 1
            # all digit-tokens multiset
        if name == "mg":
            L = b[4]
            n = len(L)
            if n > 8:
                print("label too long", i, L, n); problems += 1
    print(name, "problems", problems)

chk("wt", "walkthrough.zh-TW.tsv", "walkthrough.tsv", 4, {1, 2})
chk("mg", "map-guide.zh-TW.tsv", "map-guide.tsv", 7, {4, 5})
