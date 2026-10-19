# 工作歷程

## 2026-10-19 G0 啟動（#1）

- 目標：依使用者指示，參考幽靈騎士建立治理文件、private 儲存庫與初始 Issues。
- 讀取：幽靈騎士 `AGENTS.md`、`CLAUDE.md`、`CONTEXT.md`、`WORKLOG.md`、`LICENSE`、`.gitignore`。
- 使用者決定：完全比照幽靈騎士；沒有中文手冊，攻略日後提供軟體世界掃描；repo `champions-of-krynn-cht`、短名 `cok`；
  幽靈騎士成果當參考逐項重驗。
- 變更：新增 `AGENTS.md`、`CLAUDE.md`、`.gitignore`、`LICENSE`（由幽靈騎士 RRSAL-1.0 改填本作名稱、年份、網址，灰色地帶改為目前沒有）、
  `CONTEXT.md`、`WORKLOG.md`、`README.md`；`git init`，repo-local 身分 `wicanr2@gmail.com`。
- GitHub：`gh auth status` 為 `wicanr2`；同名 repo 不存在；建立 private repo，回讀 PRIVATE、Issues 啟用；建立 #1–#10（G0–G9）。
- 未做：未解壓原版、未跑遊戲、未建立 Docker 容器。

## 2026-10-19 G1 原始輸入盤點（#2）

- 新增 `tools/inventory.py`，在 `python:3.13-slim`（`--network none`）內計算雜湊、解壓到 `workplace/input/dos/`、產生 `workplace/input/inventory.json`。
- 結果見 `docs/re/001-input-baseline.md`：v1.2，`START.EXE`＋`GAME.OVR`（`FBOV`，與幽靈騎士同魔數），Borland 1983,91；附 pregenerated party 存檔；
  壓縮檔內沒有手冊或旅行手札，`GAME.OVR` 的查驗題字串以 journal entry 與 word # 出題。
- `.gitignore` 補原版副檔名；LICENSE 灰色地帶改列已收錄的雜湊、版本與查驗題字串、檔名清單。
- 清理：容器皆 `--rm`，`docker ps -a` 無殘留；`workplace/` 無 root 擁有檔；來源壓縮檔雜湊未變。
