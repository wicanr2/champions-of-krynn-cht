# 簡中轉換（規格 015 §4）：OpenCC 的 Python 綁定，另帶與 img.Dockerfile 相同的 Pillow、NumPy。版本鎖定。
FROM python:3.13-slim
RUN pip install --no-cache-dir pillow==11.0.0 numpy==2.1.3 opencc==1.4.1
