# 012 發行包

狀態：**CONFORMED**（2026-10-19）。Issue：#10。證據：`v.0.9.2-20261019` 三平台原版包、原版＋HD 包與本機完整版；Linux、Wine 冒煙到標題且截圖相同，macOS 驗結構（WORKLOG）；G5 收據 7 場景與正常玩家路徑（`tools/g3-party-receipt.sh`）通過。

## 1. 契約

沿用幽靈騎士 `docs/spec/012-release-packaging-draft.md` 第 2 節的契約，除下列差異外不變。

## 2. 本作差異

`tools/package.sh`；沒有 HD 主題時只出原版包；KRYNN.CFG 改 AdLib 與預建隊伍複製在 `cmd/cok-play/install.go`。

## 3. 驗收

同狀態原文／繁中 A／B 收據與正常玩家路徑完成後升級 CONFORMED，收據登在 docs/re。
