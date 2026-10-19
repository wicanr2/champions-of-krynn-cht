# 010 HD 重繪主題

狀態：**CONFORMED**（2026-10-19，首批 b01 範圍；收據 `tools/g8-hd-receipt.sh`、`tools/g7-toolbar-receipt.sh`）。Issue：#9、#11。

## 1. 契約

沿用幽靈騎士 `docs/spec/010-hd-theme-draft.md` §2：請求（`tools/hd_request.py`，組圖後與證據畫面逐像素核對）、
Codex 重繪（3 倍、風格 B、配色照原版）、主題（`tools/hd_theme.py` → `workplace/hd/theme/theme.json`）、
接入（dosgolem `apps/cok/hd.go` 圖面 watcher，整塊原版色號吻合才畫；只在 3 倍顯示）。

## 2. 本作來源（`tools/hd_locate.py`，`workplace/hd/locate/locate.tsv`）

| key | 來源 | 畫面位置 | 證據畫面 |
|---|---|---|---|
| title | TITLE 區塊 2 全幅背景＋區塊 3 標題字樣貼在 (40,8) | (0,0) 320×200 | 冷啟動 #100M |
| camp | PIC1 區塊 59（PIC2、PIC3 同圖） | (24,24) 88×88 | 旅店紮營 |
| outpost | PIC1 區塊 57（卡爾爵士肖像，屯墾區畫面） | (24,24) 88×88 | 屯墾區選單 |
| solace | BIGPIC1 區塊 112（最後家園旅店） | (8,8) 304×120 | 開場 |
| caravan | BIGPIC1 區塊 114（龍人襲擊商隊） | (8,8) 304×120 | 大地圖事件 |

分工：請求與接入由 Claude（Docker），重繪由 Codex（`codex exec`，Issue #11 記錄）。Codex 可在 sandbox 內執行的範圍見其回報 `workplace/hd/out/b01/codex-last-message.md`。

## 3. 驗收

- `tools/g8-hd-receipt.sh`：卡爾爵士、營火、商隊、最後家園旅店在 3 倍掛主題時畫面改變（HD 圖面），同輸入 2 倍時疊字外差異 0。
- `tools/g7-toolbar-receipt.sh`：3 倍 HD 標題頁與無 HD 差異 527,184 像素；切換列在 HD 下正常。
- 起點狀態若在圖畫好之後才開始，watcher 看不到繪製，不會畫 HD（正常遊玩從冷開機起不受影響）。

## 4. 首批 b01 當時未做

DragonLance 版標題（TITLE 區塊 4）、其他 PIC／BIGPIC、戰鬥圖塊、3D 牆面。

## 5. PIC、BIGPIC、TITLE 全量來源（Issue #12）

### 5.1 來源契約與證據審查

- 本節的來源級接入契約為 READY。個別美術通過數值與目視審查後才接入；不擴張首批 b01 的 CONFORMED 範圍。
- 入口為 [Issue #12](https://github.com/wicanr2/champions-of-krynn-cht/issues/12)。本輪報告與逐項狀態保存在 [`workplace/hd/out/issue12-report.md`](../../workplace/hd/out/issue12-report.md)。本輪不 commit、不 push。
- `tools/hd_coverage.py` 從本作 DAX 重解碼，按尺寸與完整色號 SHA-256 去重。來源檔雜湊、區塊 offset、解碼雜湊、原幀號、別名、保存畫面與主題對應保存在 `workplace/hd/inventory/coverage.json`。
- PIC1–3 為 67 個來源區塊、125 個原始幀記錄、85 個獨立 XOR 候選狀態；BIGPIC1–3 為 9 個 304×120 單幀來源；TITLE 為 4 個來源區塊，尺寸分別為 320×200、320×200、240×88、320×88。尺寸、色號長度與範圍已由本作 bytes 核對。
- 保存的 `.bin` 直接比色號；`*-base.png` 必須是原版最近鄰整數倍圖，每個 RGB 顏色都須精確對應 EGA 色號。遇到非 EGA 或非整數倍圖即失敗，不以近似量化補齊。
- 每個動畫狀態各登記一個 `HDItem.Watcher`，以整張狀態色號為 `Want`。本作 `apps/cok/hd.go` 與前作對應部分相同，已有多狀態能力，沒有移植前作的戰鬥或角色專用程式。
- XOR 規則為第 0 幀與各原始後續幀逐點 XOR。只有與本作保存畫面完整吻合的狀態可稱為 observed；其餘相位的時序、順序與正常玩家路徑仍為未知。來源衍生對照不作原版動畫收據。
- PIC 沿用本作已觀測的 (24,24)、88×88 家族位置；BIGPIC 沿用 (8,8)、304×120。未觀測來源只做條件式覆蓋，必須實際在該位置完整吻合才啟用，不以人名、地點或推測狀態選圖。
- 前作美術只在兩作完整色號逐位元組相同時列為可沿用候選。對應、來源 PNG 雜湊及生成紀錄保存在各批 `reuse.json` 與 `reused/`，仍須在本作重新量測與目視。

### 5.2 批次與美術閘門

- `tools/hd_full_request.py` 產生 b02–b06，每批 11–12 個去重來源區塊，動畫組不拆批。批次入口為 `workplace/hd/inventory/request-plan.json`，請求位於 `workplace/hd/requests/<批次>/manifest.json`。
- 請求保存本作來源定位、原版 PNG、3 倍參考圖、`.idx` 與動畫遮罩。未觀測項目保留 `runtime_evidence=unobserved`、`evidence_screen=null`。
- 輸出為 3 倍、風格 B、原版配色、全不透明 RGBA。PIC 為 264×264，BIGPIC 為 912×360；TITLE 按請求尺寸。提示詞與每次生成路徑記在各批 `jobs/`；生成原圖以 `*-raw.png` 保存。
- `tools/hd_prepare.py` 只做整幅 LANCZOS 縮放與 RGBA 轉換，不裁切或自動挪動構圖。動畫 @k 以 @0 為底，只取原始 XOR 非零遮罩向外 6 個輸出像素的範圍。遮罩以邊界截斷，不循環到對邊。
- `tools/hd_validate.py` 以 `hd_compare.structural_score` 量測雙側對稱 `[1,2,1]/4` 低通後的原位結構分數，門檻 ≥0.40。色相與未濾波原版相比，中位須 <30 度；另核對尺寸、RGBA、alpha 與遮罩外差異 0。
- 數值通過後仍須逐張並列目視。目視紀錄綁定 PNG SHA-256；未完成目視或圖片雜湊改變時不得接入。純色來源只允許與原色完全一致，不以純色特例放寬一般圖片的結構門檻。
- TITLE 的原始零件與完整組圖分開列記。已由完整組圖覆蓋的 TITLE 零件保留來源索引，不另登記互相競爭的 watcher。

### 5.3 驗證邊界

- `apps/cok/hd_animation_test.go` 測來源衍生多相位切換、過渡失配、清除與恢復、原版色號不變、2 倍原版及撤除素材回退。
- `apps/cok/hd_receipt_test.go` 與 `tools/issue12-hd-receipt.sh` 將來源衍生正負對照及既有原版畫面回歸分開記錄。收據為 `workplace/hd/out/issue12-receipts.json`，不把 fixture 列為遊戲執行畫面。
- 保存畫面回歸驗證呈現層的 3 倍覆繪範圍與 2 倍完全不變。它不驗證原版記憶體、暫存器或存檔。需要執行器狀態對照的項目另列在本輪報告，未量測者不宣稱完成。
- 戰鬥圖塊與 3D 牆面不屬本 Issue。封包不重打，其他規格與有效決策不修改。

### 5.4 本輪結果

- b02–b06 共 95 筆請求，其中 92 筆是新增主題候選，另 3 筆為 TITLE 原始零件索引。92 筆全部通過數值與逐張目視，接入後主題為 97 項，包含 b01 的 5 項。
- PIC 的 85 個獨立狀態與 BIGPIC 的 9 個來源全部有主題對應。TITLE 1 直接對應 SSI 頁；TITLE 2+3 對應既有主標題，TITLE 2+4 對應新增 DragonLance 交替標題。完整色號相同的前作重繪沿用 15 項，其餘 77 項本輪生成；含重畫共保存 79 次生成紀錄。
- 新增候選的結構分數最低 0.408，未濾波色相差中位最大 29.4 度。37 個非基底動畫狀態的遮罩外差異皆為 0。BIGPIC3 區塊 120 的初版結構分數 0.369，重畫後為 0.480；PIC3 區塊 24 的 @1 初版色相差 36.2 度，重畫後為 29.4 度。初版與提示詞均保留。
- DragonLance 冷啟動畫面為 `workplace/hd/screens/title750.bin`，TITLE 4 貼在 (0,8)，320×88。與 TITLE 2 組成的 320×200 完整圖逐色號相同，來源與擷取方式見 `title750-receipt.json`。
- 請求重播的 327 個檔案及處理重播的 92 張 PNG 逐位元組相同，見 `workplace/hd/out/issue12-replay-control.json`。`tools/hd_gate_checks.py` 另驗證撤銷目視、目視雜湊過期及請求雜湊過期時，主題工具拒絕產生新 manifest。
- `bash tools/issue12-hd-receipt.sh` 通過：`apps/cok` 全套單元測試、97 項來源正負對照共 194 例、185 張原版側保存畫面回歸。其中 71 張有完整來源匹配；全部 2 倍無 HD 差異，3 倍範圍外差異為 0。此處的保存畫面不是本輪重新走過全部劇情。
- `bash tools/issue12-hd-runtime.sh` 通過：相同前端二進位與冷啟動輸入，主標題及 DragonLance 在 3 倍分別改變 560,714、512,814 像素；DragonLance 的 2 倍差異為 0。各組執行步數、原版記憶體 SHA-256 與色號 SHA-256 完全相同。未逐項比較暫存器，也未做存讀檔 A/B。
- 最終盤點為 PIC observed 7、unobserved 78；BIGPIC observed 2、unobserved 7；TITLE 原始區塊 observed 3、unobserved 1。未單獨觀測的是 TITLE 2 背景，其兩種組圖均有畫面證據。所有未知來源保留 `unobserved`，不宣稱本輪取得其正常玩家路徑或動畫時序。
- 交付索引為各批 `manifest.json`、`visual-review.json`、`validation.json`、`jobs/`、`reuse.json` 與 `reused/`；`tools/hd_batch_manifest.py` 可重建逐項索引。來源、數值、主題、重播及執行收據統一由本輪報告連結，不重打發行包。
