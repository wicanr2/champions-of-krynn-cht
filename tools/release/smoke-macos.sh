#!/usr/bin/env bash
# macOS zip 結構檢查（規格 012 §4 第 3 項；沒有 macOS 實機，只驗結構）：.app 的執行檔是 universal Mach-O
# （x86_64＋arm64）、Info.plist 的 CFBundleExecutable 與版號、Resources 內的 text／font／授權，HD 版另有 hd/theme.json。
#
#   tools/release/smoke-macos.sh <版本>
# 輸出：dist-all/<版本>/smoke/macos.txt。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VER="${1:?版本}"
mkdir -p "$ROOT/dist-all/$VER/smoke"
docker run -i --rm --network none --memory 1g --cpus 1 --pids-limit 32 --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -e VER="$VER" -v "$ROOT/dist-all/$VER/patch":/p:ro python:3.13-slim python -I - <<'PY' | tee "$ROOT/dist-all/$VER/smoke/macos.txt"
import os, plistlib, struct, sys, zipfile
ver, ok = os.environ['VER'], True
for ed, sfx in (('base', ''), ('hd', '-hd')):
    if not os.path.exists(f'/p/COKCHT-{ver}{sfx}-macos.zip') and ed == 'hd':
        print('略過 hd（沒有 HD 包）'); continue
    z = zipfile.ZipFile(f'/p/COKCHT-{ver}{sfx}-macos.zip')
    names = set(z.namelist())
    app = 'COKCHT/COKCHT.app/Contents/'
    plist = plistlib.loads(z.read(app + 'Info.plist'))
    exe = z.read(app + 'MacOS/' + plist['CFBundleExecutable'])
    magic, n = struct.unpack('>II', exe[:8])
    cpus = {struct.unpack('>I', exe[8 + 20 * i:12 + 20 * i])[0] for i in range(n)} if magic == 0xCAFEBABE else set()
    mode = z.getinfo(app + 'MacOS/' + plist['CFBundleExecutable']).external_attr >> 16
    need = [app + 'Resources/' + p for p in ('text/host-text.zh-TW.tsv', 'font/cok-unifont.golemfnt', 'LICENSE', 'THIRD-PARTY.md', 'cok-cht.icns')]
    need += ['COKCHT/讀我.txt', 'COKCHT/LICENSE']
    hd = app + 'Resources/hd/theme.json' in names
    checks = {
        'universal（x86_64 0x1000007、arm64 0x100000C）': cpus == {0x1000007, 0x100000C},
        '執行權限': bool(mode & 0o111),
        f'版號 {ver}': plist['CFBundleVersion'] == ver,
        '必要檔': all(p in names for p in need),
        'hd/theme.json 只在 HD 版': hd == (ed == 'hd'),
        '不含 original/': not any('/original/' in p for p in names),
    }
    for k, v in checks.items():
        print(f'{ed}：{k} {"通過" if v else "不通過"}')
        ok &= v
sys.exit(0 if ok else 1)
PY
