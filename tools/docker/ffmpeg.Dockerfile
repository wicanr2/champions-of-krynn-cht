# 推廣影片組裝（規格 016）：Debian bookworm 的 ffmpeg（libx264、aac）。
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg && rm -rf /var/lib/apt/lists/*
