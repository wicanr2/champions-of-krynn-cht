#!/usr/bin/env bash
# 規格010 §7：仍由dosgolem tools/go.sh執行，限定本輪資源與容器標籤。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
test -d "$DG"
docker() {
 local args=("$@") i
 if [[ "${args[0]:-}" == run ]];then
  for ((i=0;i<${#args[@]};i++));do
   if [[ "${args[i]}" == --pids-limit ]];then args[i+1]=64;fi
  done
  args=(run --label cok.issue=14 "${args[@]:1}")
 fi
 command docker "${args[@]}"
}
export -f docker
cd "$DG"
export DOSGOLEM_MEM=2g DOSGOLEM_CPUS=2 DOSGOLEM_TIMEOUT=10m
if [[ -z "${DOSGOLEM_EXTRA_MOUNTS:-}" && -d "$ROOT/workplace/hd" ]];then
 export DOSGOLEM_EXTRA_MOUNTS="$ROOT/workplace/hd:/hd"
fi
export DOSGOLEM_GO_CMD="${DOSGOLEM_GO_CMD:-env}"
if [[ "$DOSGOLEM_GO_CMD" == env ]];then
 exec timeout 610s tools/go.sh GOMAXPROCS=2 go "$@"
else
 exec timeout 610s tools/go.sh "$@"
fi
