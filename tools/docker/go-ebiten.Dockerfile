# dosgolem 的建置與測試映像（`dosgolem-go-ebiten`）。用 tools/build_image.sh 建。
#
# frontend/ebiten、cmd/buckrogers-play、apps/phantasie/cmd/phantasie-play 走 Ebiten，
# 在 Linux 上要 cgo 連 X11／OpenGL／ALSA，官方 golang 映像沒有那些標頭檔。
# 它們的測試在 init 就開 GLFW，沒有 DISPLAY 會 panic，所以另裝 xvfb 給 xvfb-run。
FROM golang:1.24-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
        libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev \
        libgl1-mesa-dev libasound2-dev pkg-config xvfb xauth libgl1 libglx-mesa0 \
    && rm -rf /var/lib/apt/lists/*
