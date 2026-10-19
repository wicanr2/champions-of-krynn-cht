# 011 視窗頂端切換列

狀態：**CONFORMED**（2026-10-19，無 HD 範圍；收據 `tools/g7-toolbar-receipt.sh`）。Issue：#9。

## 1. 契約

沿用幽靈騎士 `docs/spec/011-toolbar-draft.md` 的契約；`cmd/cok-play/toolbar.go` 程式未改，文字取自 `text/host-text.zh-TW.tsv` 的 `toolbar.*`。

## 2. 本作差異

本作尚無 HD 主題，切換列顯示「無HD素材」並停用 HD 切換。

## 3. 驗收

2 倍、3 倍冷開機到標題畫面，顯示切換列與不顯示的同一畫格相比：切換列外差異 0 像素，切換列內改變 4922／10455 像素；無缺字。
有 HD 主題後補驗 HD 切換。
