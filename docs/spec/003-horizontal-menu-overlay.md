# 003 路徑 C：底列水平選單覆繪

狀態：**READY**（2026-10-19）。Issue：#4、#6。證據：docs/re/004。

## 1. 契約

沿用幽靈騎士 `docs/spec/003-horizontal-menu-overlay-draft.md` 第 2 節的契約，除下列差異外不變。

## 2. 本作差異

逐字程序 `0631:01DF`；呼叫端返回位址前 `9A DF 01`、後 bytes 見 docs/re/004 表；選單由多個帶尾端空白的 Pascal 字串接成，改以連續片段查表；ECL 選單詞轉大寫查 ECL catalog。

## 3. 驗收

同狀態原文／繁中 A／B 收據與正常玩家路徑完成後升級 CONFORMED，收據登在 docs/re。
