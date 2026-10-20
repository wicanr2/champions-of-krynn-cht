#!/usr/bin/env bash
# tools/go.sh 經 timeout 啟動外部 docker，因此須用 PATH 中的實體入口加標籤。
set -euo pipefail
if [[ "${1:-}" == run ]];then
 shift
 exec /usr/bin/docker run --label cok.issue=22 "$@"
fi
exec /usr/bin/docker "$@"
