#!/usr/bin/env bash
# 規格 010 §6：沿用 dosgolem tools/go.sh，只收緊本輪資源與 PID 上限。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DG="$ROOT/workplace/dosgolem"
test -d "$DG"; test -d "$ROOT/workplace/hd"
docker() {
  local args=("$@") i
  if [[ "${args[0]:-}" == run ]]; then
    for ((i=0;i<${#args[@]};i++)); do
      if [[ "${args[i]}" == --pids-limit ]]; then args[i+1]=64; fi
    done
  fi
  command docker "${args[@]}"
}
export -f docker
cd "$DG"
export DOSGOLEM_MEM=2g DOSGOLEM_CPUS=2 DOSGOLEM_TIMEOUT=6m
export DOSGOLEM_EXTRA_MOUNTS="${DOSGOLEM_EXTRA_MOUNTS:-$ROOT/workplace/hd:/hd}"
export DOSGOLEM_GO_CMD="${DOSGOLEM_GO_CMD:-env}"
if [[ "$DOSGOLEM_GO_CMD" == env ]]; then
  exec timeout 370s tools/go.sh GOMAXPROCS=2 go "$@"
else
  exec timeout 370s tools/go.sh "$@"
fi
