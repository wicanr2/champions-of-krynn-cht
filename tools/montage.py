"""把多張 PNG 拼成格狀總覽（除錯用）。用法：montage.py <輸出> <欄數> <png...>"""
import sys
from PIL import Image, ImageDraw
out, cols, files = sys.argv[1], int(sys.argv[2]), sys.argv[3:]
ims = [Image.open(f).convert('RGB') for f in files]
w, h = max(i.width for i in ims), max(i.height for i in ims)
rows = (len(ims) + cols - 1) // cols
m = Image.new('RGB', (cols * w, rows * (h + 10)), 'white')
for k, (f, im) in enumerate(zip(files, ims)):
    x, y = k % cols * w, k // cols * (h + 10)
    m.paste(im, (x, y + 10)); ImageDraw.Draw(m).text((x + 2, y), f.split('/')[-1], fill='black')
m.save(out)
