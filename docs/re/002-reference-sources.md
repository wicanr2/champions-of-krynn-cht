# 002 參考文件來源

2026-10-19 盤點。原始檔不進 Git；雜湊在 Docker（`cok-pdf:bookworm-r1`，`--network none`）內以 `sha256sum` 計算。
PDF 文字用 `pdftotext -layout` 抽到 `workplace/reference-src/txt/`。

## 英文 Adventurer's Journal（G6，#7）

| 來源 | 檔案 | SHA-256 | 頁數 | 文字層 | 手札條目 |
|---|---|---|---|---|---|
| 使用者下載（主要） | `Champions_of_Krynn_Adventurers_Journal.pdf`（本目錄根層） | `a1b41e85f5f325f9eb73bee5b19b1782abbbee095f7de958dee6e7c89c5803b9` | 32 | OmniPage 12 OCR | Journal Entry 1–88 全部可辨識 |
| mocagh.org `ssi/champions-journal.pdf`（校對） | `workplace/reference-src/web/` | `af106dc375a40ca89540b390156d3fc35a2815f848247dba7a085946ae282167` | 33 | Acrobat Paper Capture OCR | OCR 漏辨第 76 則 |

- 目錄頁：Journal Entries 起於原書第 46 頁，另有 Tavern Tales（原書第 44 頁）。已證實（OCR 文字）。
- 版面為多欄，OCR 有雜訊（條目標題前的方框字元被辨成 `0`、`J`、`U` 等），正式轉錄要逐則對照頁面圖。
- 查驗題（`GAME.OVR` 的 `word # … of journal entry #`）是否以這 88 則為題庫，待 G3、G6 對輸出事件確認。強推論。

## SSI 官方文件（mocagh.org，校對與攻略補充）

| 檔案 | SHA-256 | 頁數 | 用途 |
|---|---|---|---|
| `champions-manual.pdf` | `255aad131a602ef739d2f6d0385f0340763f91fb08a3011c92cfb6005c93adc2` | 9 | 規則卡／操作手冊 |
| `champions-hintbook.pdf` | `40e451421020a87851fcd64f7c44a99570c4d6a5c987a526830a20dc52c16478` | 33 | 官方提示書；含「Journal Entries: True or False?」，可判別哪些手札條目是假情報 |

下載網址：`https://www.mocagh.org/ssi/<檔名>`，頁面 `https://www.mocagh.org/loadpage.php?getgame=champions`。
同頁另有 `champions-refcard.pdf` 未下載。

## 軟體世界攻略掃描（G7，#8）

使用者提供，放 `walkthrough/`。第 16 期第 39 頁標題「克萊恩英豪 攻略（一）」，作者署名「白怒火＆南帝」，
內文以座標與「參見手札 N」引用英文手札條目編號。

| 檔案 | SHA-256 |
|---|---|
| `軟體世界16期/39.jpeg` | `1294fbefbe1478128cabe5f7e07757d37ab1737a3cde36036c214bef690a043d` |
| `軟體世界16期/40.jpeg` | `68d33422478b517c947ca5cd659419b4bdd018d7cd9954dd57dc8ba14c21ff66` |
| `軟體世界16期/41.jpeg` | `543135e48ec9d78226f4ce3dcad76728ee843c5b12172f6c32d97b2e6515e3a6` |
| `軟體世界16期/42.jpeg` | `5c1a02098446102d6104a905d4ad8d07b973e01c2d7f29056422660c9be1424b` |
| `軟體世界17期/47.jpeg` | `bd916aff9ad082914dbdf6bde08d106aee464b1f3318c77b93ad0fd583f8c461` |
| `軟體世界17期/48.jpeg` | `b9ccd998f36467d0efae033456ef7161ce8cbd1e1fa54f4b58369498e6196cb9` |
| `軟體世界17期/49.jpeg` | `8f90e0e4f577bc751e842896f29e82cf2559d85b6b64188efe45270be43aba05` |
| `軟體世界17期/50.jpeg` | `a60e65b23f1bcf94c662e8cc0162803b7c2d4fe661719cb0516b2188ea460fd1` |
| `軟體世界17期/51.jpeg` | `a7f4a93d786c08a46309611e30a3fe137bdde35352149d3c5f65e029d335c542` |
| `軟體世界18期/37.jpeg` | `bf56eb1ace4185ac00370778e9e4a07e96bfc27c773581f4c53125b107a87269` |
| `軟體世界18期/38.jpeg` | `52bbe73d0fadccf4557fe65e82487590cf74d74186d6557980c6d068bf524bf4` |
| `軟體世界18期/39.jpeg` | `8ac2e4eae8432d63bd840a5bd3cd01f4c619798469cbc7cf69c7d550a531b3ac` |
| `軟體世界18期/40.jpeg` | `3b7bfa0ebae53480b4d3cba63c4b63520df362d45f322533a1861b1d5e65d09e` |
| `軟體世界18期/41.jpeg` | `f87741489b879092d593f7559a734d63ed7c916655f31889d530cc589ff6a450` |
