"""在檔案中找帶遮罩的位元組簽章。用法：sigscan.py <檔案> <名稱>=<hex，?? 表示不比對> ..."""
import sys
data = open(sys.argv[1], 'rb').read()
for spec in sys.argv[2:]:
    name, hexs = spec.split('=', 1)
    pat = [None if t == '??' else int(t, 16) for t in hexs.split()]
    hits = [i for i in range(len(data) - len(pat) + 1)
            if all(p is None or data[i + k] == p for k, p in enumerate(pat))]
    print(name, len(pat), [hex(h) for h in hits])
