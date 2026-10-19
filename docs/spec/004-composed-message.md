# 004 組句訊息（戰鬥、座標列）

狀態：**CONFORMED**（2026-10-19，收據 docs/re/008 demo 戰鬥重跑）。Issue：#6。證據：docs/re/008。

## 1. 契約

沿用幽靈騎士 `docs/spec/004-composed-message-draft.md` 第 2 節的契約，除下列差異外不變。

## 2. 本作差異

模板 `text/composed-templates.tsv` 繼承幽靈騎士，另補本作寫法：`Hitting for {n} points of damage`（單一空白）、`takes {n} points of damage from <元素>`、無尾端空白的移動力列、`Range = {n}  `、金錢列；商店清單與購買訊息由 `compose.go` 的 pricedRow、buys 規則處理。

## 3. 驗收

同狀態原文／繁中 A／B 收據與正常玩家路徑完成後升級 CONFORMED，收據登在 docs/re。
