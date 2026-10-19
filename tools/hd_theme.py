#!/usr/bin/env python3
"""組 HD 主題目錄（G7，#8；規格 010 §2.2）：從各批次挑定版圖，配上原版色號基準，寫 theme.json。

每項來源：Codex 輸出 `<hd>/<批次>/<key>.png` 與請求 `<req>/<批次>/<key>.idx`、manifest 的 screen_rect。
檢查：PNG 為 RGBA、alpha 全 255、尺寸 = screen_rect × 倍率；矩形對齊 8 像素格（spec 204 §2.2 切 8×8 格）。
輸出 `<out>/<key>.png`、`<out>/<key>.idx`、`<out>/theme.json`（dosgolem apps/cok LoadHDTheme 讀取）。
"""
import argparse
import hashlib
import json
import shutil
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from hd_compare import read_png  # noqa: E402

# 本作接入版本（#11）：b01 首批 5 項
PICKS = [(k, 'b01') for k in ('title', 'camp', 'outpost', 'solace', 'caravan')]
SCALE = 3  # HD 圖只做 3 倍（幽靈騎士定案，本作沿用）


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--req', required=True, type=Path, help='workplace/hd/requests')
    ap.add_argument('--hd', required=True, type=Path, help='workplace/hd/out')
    ap.add_argument('--out', required=True, type=Path)
    a = ap.parse_args()
    a.out.mkdir(parents=True, exist_ok=True)
    items = []
    for key, batch in PICKS:
        man = json.loads((a.req / batch / 'manifest.json').read_text(encoding='utf-8'))
        it = next(i for i in man['items'] if i['key'] == key)
        if batch in ('b12', 'b13', 'b14', 'b15'):
            validation = json.loads((a.hd / batch / 'validation.json').read_text(encoding='utf-8'))
            row = next(r for r in validation['items'] if r['key'] == key)
            if not row['target_met']:
                sys.exit(f'{key}：個別美術尚未通過驗收')
        X, Y, W, H = it['screen_rect']
        if man['scale'] != SCALE or X % 8 or Y % 8 or W % 8 or H % 8:
            sys.exit(f'{key}：倍率 {man["scale"]} 或矩形 {it["screen_rect"]} 不合')
        src = a.hd / batch / f'{key}.png'
        if batch in ('b12', 'b13', 'b14', 'b15') and hashlib.sha256(src.read_bytes()).hexdigest() != row['png_sha256']:
            sys.exit(f'{key}：驗收後圖片已改變，須重跑驗收')
        ct, px = read_png(src)
        if ct != 6 or px.shape[:2] != (H * SCALE, W * SCALE) or (px[..., 3] != 255).any():
            sys.exit(f'{key}：{src} 不是 {W * SCALE}×{H * SCALE} 不透明 RGBA')
        idx = (a.req / batch / it['indexed']).read_bytes()
        if len(idx) != W * H or hashlib.sha256(idx).hexdigest() != it['indexed_sha256']:
            sys.exit(f'{key}：色號基準與 manifest 不符')
        shutil.copyfile(src, a.out / f'{key}.png')
        (a.out / f'{key}.idx').write_bytes(idx)
        items.append({'key': key, 'batch': batch, 'screen_rect': [X, Y, W, H], 'png': f'{key}.png',
                      'png_sha256': hashlib.sha256(src.read_bytes()).hexdigest(), 'indexed': f'{key}.idx',
                      'indexed_sha256': it['indexed_sha256'], 'sources': it['sources'],
                      'runtime_evidence': 'observed' if it.get('evidence_screen') else 'unobserved',
                      'evidence_screen': it.get('evidence_screen')})
    theme = {'spec': 'docs/spec/010-hd-theme-draft.md', 'issue': '#8', 'scale': SCALE, 'items': items}
    (a.out / 'theme.json').write_text(json.dumps(theme, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'主題 {len(items)} 項，倍率 {SCALE}')


if __name__ == '__main__':
    main()
