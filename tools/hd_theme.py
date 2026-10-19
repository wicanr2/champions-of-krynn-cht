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
    ap.add_argument('--batches', default='', help='追加批次，以逗號分隔；須通過數值與目視閘門')
    ap.add_argument('--accepted-only', action='store_true', help='明列跳過未驗收項目，只接入已核准項目')
    a = ap.parse_args()
    a.out.mkdir(parents=True, exist_ok=True)
    items = []
    picks=list(PICKS)
    for batch in filter(None,a.batches.split(',')):
        man=json.loads((a.req/batch/'manifest.json').read_text())
        picks.extend((it['key'],batch) for it in man['items'] if not it.get('disposition'))
    seen=set()
    prepared=[]
    for key, batch in picks:
        if key in seen:sys.exit(f'重複 key：{key}')
        seen.add(key)
        man = json.loads((a.req / batch / 'manifest.json').read_text(encoding='utf-8'))
        it = next(i for i in man['items'] if i['key'] == key)
        if batch != 'b01':
            vp=a.hd/batch/'validation.json';rp=a.hd/batch/'visual-review.json'
            if not vp.exists() or not rp.exists():
                if a.accepted_only:
                    print(f'跳過 {batch}/{key}：尚無完整驗收紀錄');continue
                sys.exit(f'{key}：缺數值或目視驗收')
            validation = json.loads((a.hd / batch / 'validation.json').read_text(encoding='utf-8'))
            row = next(r for r in validation['items'] if r['key'] == key)
            review=next((r for r in json.loads(rp.read_text())['items'] if r['key']==key),{})
            accepted=row['target_met'] and review.get('approved') and it.get('integration_allowed',False)
            if not accepted:
                if a.accepted_only:
                    print(f'跳過 {batch}/{key}：數值、目視或來源位置尚未通過');continue
                sys.exit(f'{key}：個別美術或來源位置尚未通過驗收')
        X, Y, W, H = it['screen_rect']
        if man['scale'] != SCALE or X % 8 or Y % 8 or W % 8 or H % 8:
            sys.exit(f'{key}：倍率 {man["scale"]} 或矩形 {it["screen_rect"]} 不合')
        src = a.hd / batch / f'{key}.png'
        if batch != 'b01':
            png_sha=hashlib.sha256(src.read_bytes()).hexdigest()
            req_sha=hashlib.sha256((a.req/batch/'manifest.json').read_bytes()).hexdigest()
            if png_sha!=row['png_sha256'] or png_sha!=review.get('png_sha256') or req_sha!=validation.get('request_manifest_sha256'):
                sys.exit(f'{key}：驗收後圖片或請求已改變，須重跑驗收')
        ct, px = read_png(src)
        if ct != 6 or px.shape[:2] != (H * SCALE, W * SCALE) or (px[..., 3] != 255).any():
            sys.exit(f'{key}：{src} 不是 {W * SCALE}×{H * SCALE} 不透明 RGBA')
        idx = (a.req / batch / it['indexed']).read_bytes()
        if len(idx) != W * H or hashlib.sha256(idx).hexdigest() != it['indexed_sha256']:
            sys.exit(f'{key}：色號基準與 manifest 不符')
        prepared.append((src,key,idx))
        items.append({'key': key, 'batch': batch, 'screen_rect': [X, Y, W, H], 'png': f'{key}.png',
                      'png_sha256': hashlib.sha256(src.read_bytes()).hexdigest(), 'indexed': f'{key}.idx',
                      'indexed_sha256': it['indexed_sha256'], 'sources': it['sources'],
                      'runtime_evidence': 'observed' if it.get('evidence_screen') else 'unobserved',
                      'evidence_screen': it.get('evidence_screen')})
    # 先完成全部 preflight，再寫資產，最後更新 manifest。
    for src,key,idx in prepared:
        shutil.copyfile(src,a.out/f'{key}.png')
        (a.out/f'{key}.idx').write_bytes(idx)
    theme = {'spec': 'docs/spec/010-hd-theme.md', 'issue': '#12', 'scale': SCALE, 'items': items}
    (a.out / 'theme.json').write_text(json.dumps(theme, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'主題 {len(items)} 項，倍率 {SCALE}')


if __name__ == '__main__':
    main()
