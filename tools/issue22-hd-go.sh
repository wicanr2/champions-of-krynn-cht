#!/usr/bin/env bash
# Issue22：沿用專案 Go 入口，限制資源並標記容器。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test -d "$ROOT/workplace/dosgolem"
bash "$ROOT/tools/issue22-hd-img.sh" sh -c '
 d=/src/workplace/hd/out/issue22-runner
 mkdir -p "$d"
 if ! cmp -s /src/tools/issue22-docker.sh "$d/docker";then
  t=$(mktemp "$d/docker.XXXXXX")
  cp /src/tools/issue22-docker.sh "$t"
  chmod 755 "$t"
  mv "$t" "$d/docker"
 fi'
export PATH="$ROOT/workplace/hd/out/issue22-runner:$PATH"
cd "$ROOT/workplace/dosgolem"
export DOSGOLEM_MEM=2g DOSGOLEM_CPUS=2 DOSGOLEM_NETWORK=none DOSGOLEM_TIMEOUT="${DOSGOLEM_TIMEOUT:-20m}"
export DOSGOLEM_GO_CMD="${DOSGOLEM_GO_CMD:-env}"
if [[ "$DOSGOLEM_GO_CMD" == env ]]; then
 exec tools/go.sh GOMAXPROCS=2 go "$@"
else
 exec tools/go.sh "$@"
fi
