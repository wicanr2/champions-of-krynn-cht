#!/usr/bin/env bash
# 彙整盤點 inv.tsv 的未命中印字（docs/re/008 的盤點格式）：排除純數字、生命值與骰子表示法、空白、預建隊員名，
# 剩下的就是缺口。用法：tools/area-misses.sh <inv.tsv>…   輸出「次數  路徑  原文」。
set -euo pipefail
awk -F'\t' '
  FNR == 1 { next }
  $8 !~ /未命中/ { next }
  { t = $9; gsub(/^ +| +$/, "", t) }
  t == "" { next }
  t ~ /^[-+0-9()\/d ]+$/ { next }
  t ~ /^(SIR STRONGSWORD|ISTAN HORBIN|GARIN|MOLLY|KAL|SIRRION)$/ { next }
  { c[$2 "\t" t]++ }
  END { for (k in c) printf "%5d\t%s\n", c[k], k }
' "$@" | sort -rn
