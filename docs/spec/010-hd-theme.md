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

## 4. 未做

DragonLance 版標題（TITLE 區塊 4）、其他 PIC／BIGPIC、戰鬥圖塊、3D 牆面。
