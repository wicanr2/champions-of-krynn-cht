# HD 主題

`theme/` 是發行包 HD 版使用的完整主題（規格 010），共 1,984 項：PNG 是 3 倍重繪成品（Codex 生成，或簡單符號以 Scale3x 演算法放大），JSON 是各家族的比對與繪製規則，`.idx` 是比對用的原版圖色號。

- 打包（`tools/package.sh`）與推廣影片錄製（`tools/promo-rec.sh`）都讀這裡。
- Codex 的工作區是 `workplace/hd/`（請求、生成原圖、盤點、收據都在那裡，不進版控）；交付並經審查後以 `tools/hd-sync.sh` 同步到這裡再提交。
- `.idx` 衍生自原版圖，依使用者 2026-10-21 決定與成品一起進本 private repo；內容與發行包 HD 版的 `hd/` 相同。
