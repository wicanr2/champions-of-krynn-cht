#!/usr/bin/env bash
# 建 tools/go.sh 預設用的映像。只建這一個映像，不動任何共用資源。
#
#   tools/build_image.sh
#
# 要網路（apt 與拉基底映像），所以不走 tools/go.sh 的 --network none。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${DOSGOLEM_GO_IMAGE:-dosgolem-go-ebiten:1.24-r1}"
exec timeout 30m docker build -t "$IMAGE" -f "$ROOT/tools/docker/go-ebiten.Dockerfile" "$ROOT/tools/docker"
