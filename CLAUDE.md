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

全部在 Docker 內執行，前置為 G1 解壓的 `workplace/input/dos/ckrynn/` 與 dosgolem 隔離 clone `workplace/dosgolem/`。

```sh
# 原版冷啟動到建立隊伍主選單，重播兩次比對畫面雜湊（docs/re/003）
tools/g2-startup-receipt.sh
# 正常玩家路徑：建角 ANNA → 加入隊伍 → 開始冒險 → 查驗題（答案表自動作答）→ 開場；FROM=<段名> 續跑
tools/g3-party-receipt.sh
# 覆繪收據 7 場景（docs/re/007）
tools/g5-receipt.sh
# 字型（倚天，本機）、術語一致性、查驗題答案、攻略與地圖點資料
tools/font-dev.sh; tools/term-check.sh
# 發行包與冒煙（沒有 HD 主題時只出原版包；COK_WITH_DATA=1 出本機完整版）
tools/package.sh v.0.9.1-20261019 all
tools/release/smoke-linux.sh v.0.9.1-20261019; tools/release/smoke-windows.sh v.0.9.1-20261019; tools/release/smoke-macos.sh v.0.9.1-20261019
# 開視窗遊玩（Docker＋主機 X11）
tools/play.sh

# 任意探針：原版唯讀掛到 /orig；scratch 要先放 AdLib 版設定，否則卡在 MPU-401
tools/cfg-adlib.sh workplace/input/dos/ckrynn/KRYNN.CFG workplace/probe/x/scratch/KRYNN.CFG
cd workplace/dosgolem
DOSGOLEM_ORIG=../input/dos/ckrynn DOSGOLEM_EXTRA_MOUNT="$PWD/../probe/x:/out" \
  tools/go.sh run ./cmd/probe -exe /orig/START.EXE -root /orig -scratch-dir /out/scratch \
  -steps 300001000 -bios-keys 'p' -bios-key-from 205000000 -dump-at "299000000:/out/menu.png"
```

- 送鍵用 `-bios-keys`。`-dump-at` 多個時點用 `;` 分隔，步數要小於 `-steps`。

## 資料索引

| 項目 | 位置 |
|---|---|
| 原版遊戲 | `Champions of Krynn (1990).zip`（唯讀，不進 Git） |
| 軟體世界攻略掃描 | `walkthrough/`（待使用者提供，不進 Git） |
| 研究工作區 | `workplace/`（gitignore） |
| dosgolem 隔離 clone | `workplace/dosgolem/`，分支 `champions-of-krynn-cht`，推到本 repo 的 `dosgolem/cok`；adapter 在 `apps/cok/` |
| 第一參考 | `/home/anr2/cht/golden_box/death_knight_of_krynn/`（只讀） |
