# 手冊 PDF 轉文字與頁面圖（poppler-utils）。
FROM debian@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587
RUN apt-get update \
 && apt-get install -y --no-install-recommends poppler-utils \
 && rm -rf /var/lib/apt/lists/*
