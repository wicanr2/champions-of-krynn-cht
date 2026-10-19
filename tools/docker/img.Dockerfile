# 影像拼圖與比對（Pillow）。版本鎖定。
FROM python:3.13-slim
RUN pip install --no-cache-dir pillow==11.0.0 numpy==2.1.3
