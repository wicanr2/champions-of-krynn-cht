#!/usr/bin/env bash
# Issue 21：沿用既有 Go Docker 入口，限制 CPU 並標記容器。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test -d "$ROOT/workplace/dosgolem"
docker() {
 local args=("$@")
 if [[ "${args[0]:-}" == run ]]; then args=(run --label cok.issue=21 "${args[@]:1}"); fi
 command docker "${args[@]}"
}
export -f docker
cd "$ROOT/workplace/dosgolem"
export DOSGOLEM_MEM=2g DOSGOLEM_CPUS=2 DOSGOLEM_TIMEOUT="${DOSGOLEM_TIMEOUT:-20m}"
export DOSGOLEM_GO_CMD="${DOSGOLEM_GO_CMD:-env}"
if [[ "$DOSGOLEM_GO_CMD" == env ]]; then
 exec tools/go.sh GOMAXPROCS=2 go "$@"
else
 exec tools/go.sh "$@"
fi
