#!/usr/bin/env bash
# HD 主題同步：Codex 在 workplace/hd/theme 交付並經 Claude 審查後，同步到 repo 的 hd/theme（正式版本，
# 打包與錄影都讀這裡）。只在兩邊不同時複製；之後以 git 提交。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; SRC="$ROOT/workplace/hd/theme"; DST="$ROOT/hd/theme"
test -f "$SRC/theme.json" || { echo "缺 $SRC/theme.json" >&2; exit 1; }
if diff -rq "$SRC" "$DST" >/dev/null 2>&1; then echo "hd/theme 已是最新"; exit 0; fi
rm -rf "$DST.new"; cp -r "$SRC" "$DST.new"; rm -rf "$DST"; mv "$DST.new" "$DST"
echo "已同步 $(find "$DST" -type f | wc -l) 個檔到 hd/theme"
