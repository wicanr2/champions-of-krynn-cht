# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

所有工作規則以 [`AGENTS.md`](AGENTS.md) 為單一權威。本檔只保留使用者最初需求與資料索引，
目前狀態見 [`CONTEXT.md`](CONTEXT.md)。

## 原始目標（2026-10-19）

1. 參考 `/home/anr2/cht/golden_box/death_knight_of_krynn/`（幽靈騎士），治理與範圍完全比照，
   用 dosgolem 執行原版 DOS《Champions of Krynn》，以輸出攔截覆繪繁體中文化。
2. 提供 HD 版本，HD 整條交 Codex（`AGENTS.md` 第 6 節）。
3. 建立 private GitHub 儲存庫 `champions-of-krynn-cht`，Issue 先行。

## 常用指令

G2 建立 dosgolem 基線後補上。

## 資料索引

| 項目 | 位置 |
|---|---|
| 原版遊戲 | `Champions of Krynn (1990).zip`（唯讀，不進 Git） |
| 軟體世界攻略掃描 | `walkthrough/`（待使用者提供，不進 Git） |
| 研究工作區 | `workplace/`（gitignore） |
| dosgolem 隔離 clone | `workplace/dosgolem/`，分支 `champions-of-krynn-cht`，推到本 repo 的 `dosgolem/cok`；adapter 在 `apps/cok/` |
| 第一參考 | `/home/anr2/cht/golden_box/death_knight_of_krynn/`（只讀） |
