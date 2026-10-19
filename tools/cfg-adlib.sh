#!/usr/bin/env bash
# 由原版 KRYNN.CFG 產生音效改選 AdLib 的副本（只換第 2 行 R→A，其餘位元組不變），放進 dosgolem scratch 層。
# 用法：tools/cfg-adlib.sh <原版 KRYNN.CFG> <輸出路徑>
set -euo pipefail
src="$1"; dst="$2"
test -f "$src" || { echo "缺 $src" >&2; exit 1; }
[[ "$(sed -n 2p "$src" | tr -d '\r')" == "R" ]] || { echo "$src 第 2 行不是 R，未知版本" >&2; exit 1; }
sed '2s/^R\r$/A\r/' "$src" > "$dst"
cmp -s "$src" "$dst" && { echo "替換失敗" >&2; exit 1; }
exit 0
