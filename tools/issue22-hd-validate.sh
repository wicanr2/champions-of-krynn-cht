#!/usr/bin/env bash
# 最終二進位的三倍／二倍、無主題／有主題同狀態矩陣。各組依序執行。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
case "${1:-core-a}" in
 build) exec bash "$ROOT/tools/issue22-hd-runtime.sh" build;;
 core-a) scenes=(demo throtl);;
 core-b) scenes=(throtl2 sanction);;
 gargath) scenes=(gargath3d-a gargath3d-b gargath2-reenter);;
 gargath-a) scenes=(gargath3d-a);;
 gargath-b) scenes=(gargath3d-b);;
 gargath-reenter) scenes=(gargath2-reenter);;
 *) echo '使用 build、core-a、core-b 或 gargath' >&2;exit 1;;
esac
for scene in "${scenes[@]}";do
 case "$scene" in
 demo) until=850000000;from=0;keys='';;
 throtl) until=20342000000;from=19862000000;keys='Right,Up,Left,Left,Up,Right,Up,Right,Up,Left,Up,Right,Up,Up,Left,Up,Up,Right,Up,Up,Left,Up';;
 throtl2) until=12508000000;from=12228000000;keys='Right,Up,Left,Left,Up,Right,Up,Right,Up,Left,Up,Right';;
 sanction) until=17482000000;from=17002000000;keys='Right,Up,Left,Left,Up,Right,Up,Right,Up,Left,Up,Right,Up,Up,Left,Up,Up,Right,Up,Up,Left,Up';;
 gargath3d-a) until=4920000000;from=4640000000;keys='Right,Up,Left,Up,Up,Right,Up,Left,Up,Right,Right,Up';;
 gargath3d-b) until=5040000000;from=4640000000;keys='Left,Up,Right,Up,Up,Left,Up,Right,Up,Left,Left,Up,Return,Return,Return,Return,Return,Return';;
 gargath2-reenter) until=10466000000;from=9986000000;keys='Y,Return,Up,Up,Right,Up,Up,Left,Up,Up,Right,Up,Right,Up,Left,Up,Left,Up,Right,Up';;
 esac
 for scale in 3 2;do
  for mode in base hd;do
   bash "$ROOT/tools/issue22-hd-runtime.sh" "$scene" "$mode" "$scale" "$until" "$from" "$keys"
  done
 done
done
