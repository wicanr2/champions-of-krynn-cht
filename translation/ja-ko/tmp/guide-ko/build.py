import re, sys, csv
BASE = "/home/anr2/cht/golden_box/champions_of_krynn"
TMP = BASE + "/workplace/tr-jako/tmp/guide-ko"
OUT = BASE + "/workplace/tr-jako/aux-ko"

def rd(p):
    with open(p, encoding="utf-8") as f:
        return [l.rstrip("\n") for l in f if l.strip()]

# walkthrough
src = rd(BASE + "/text/walkthrough.zh-TW.tsv")
dr = rd(TMP + "/wt.draft")
assert len(src) - 1 == len(dr), (len(src) - 1, len(dr))
out = [src[0]]
for s, d in zip(src[1:], dr):
    c, t, b, so = s.split("\t")
    k, nt, nb = d.split("|||")
    assert k == c, (k, c)
    m = re.fullmatch(r"軟體世界 (\d+) 期 (p\.[\d-]+)", so)
    assert m, so
    out.append("\t".join([c, nt, nb, "소프트월드 %s호 %s" % (m.group(1), m.group(2))]))
open(OUT + "/walkthrough.tsv", "w", encoding="utf-8").write("\n".join(out) + "\n")

# map guide
src = rd(BASE + "/text/map-guide.zh-TW.tsv")
dr = rd(TMP + "/mg.draft")
assert len(src) - 1 == len(dr), (len(src) - 1, len(dr))
out = [src[0]]
for s, d in zip(src[1:], dr):
    f = s.split("\t")
    k, lab, summ = d.split("|||")
    assert k == f[0] + "#" + f[1], (k, f[0], f[1])
    f[4] = lab
    f[5] = summ
    out.append("\t".join(f))
open(OUT + "/map-guide.tsv", "w", encoding="utf-8").write("\n".join(out) + "\n")
print("ok")
