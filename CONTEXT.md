# 目前真相

更新：2026-10-19（Asia/Taipei）。規則見 [AGENTS.md](AGENTS.md)，歷程見 [WORKLOG.md](WORKLOG.md)。

## 儲存庫

| 項目 | 值 | 回讀方式 |
|---|---|---|
| 網址 | <https://github.com/wicanr2/champions-of-krynn-cht> | `gh repo view --json url` |
| owner | `wicanr2` | `gh auth status`、`gh repo view --json owner` |
| 可見性 | PRIVATE | `gh repo view --json visibility` |
| Issues | 啟用；初始 G0–G9 為 #1–#10 | `gh repo view --json hasIssuesEnabled` |
| git 身分 | repo-local `wicanr2` <wicanr2@gmail.com> | `git config user.email` |
| 授權 | RRSAL-1.0（[LICENSE](LICENSE)） | |

## 有效決策

| 日期 | 決策 | 來源 |
|---|---|---|
| 2026-10-19 | 治理與範圍完全比照幽靈騎士（`/home/anr2/cht/golden_box/death_knight_of_krynn/`），沿用項目見 AGENTS.md 第 1 節 | 使用者回答 |
| 2026-10-19 | 本作沒有中文手冊；攻略由使用者日後提供軟體世界掃描圖 | 使用者回答 |
| 2026-10-19 | repo `champions-of-krynn-cht`；遊戲短名 `cok`，adapter 在 dosgolem `apps/cok/`，分支推到本 repo 的 `dosgolem/cok` | 使用者回答 |
| 2026-10-19 | 幽靈騎士的 adapter、譯名與工具當參考逐項重驗；dosgolem clone 從 `dosgolem/dkk` 分支起；共用譯名直接繼承 | 使用者回答 |
| 2026-10-19 | 中文題名「克萊恩英豪」 | 使用者指定 |

## 待定

| 項目 | 決定時機 |
|---|---|
| 旅行手札原文來源 | 壓縮檔內沒有（docs/re/001）；查驗題以手札條目出題，需使用者提供手冊或手札來源，G6（#7）依此生效 |
| 軟體世界攻略掃描 | 使用者提供後開 G7（#8） |

## dosgolem 基線

G2（#3）建立後填入 commit、Go 版本與 image ID。

## 原始輸入

| 檔案 | 狀態 |
|---|---|
| `Champions of Krynn (1990).zip` | 原版 DOS v1.2，入口 `START.EXE`＋`GAME.OVR`（`FBOV`），附 pregenerated party 存檔；見 [docs/re/001](docs/re/001-input-baseline.md) |

## Glossary

- 克萊恩英豪：本作中文題名，使用者指定。
- 幽靈騎士：參考專案《Death Knights of Krynn》繁中化，本機 `/home/anr2/cht/golden_box/death_knight_of_krynn/`。
- cok：本作遊戲短名，用於 dosgolem `apps/cok/` 與程式識別。
- dkk：幽靈騎士的遊戲短名；`dosgolem/dkk` 是本作 dosgolem clone 的起點分支。
- 覆繪：在原版已輸出的畫面上清除原文墨跡、重畫中文。_Avoid_：替換、打字串補丁
- 收據：dosgolem 以固定輸入重播產生、可重現的執行證據。_Avoid_：截圖證明
- HD 批次：一組交給 Codex 重繪的原版圖塊，含請求與輸出兩個目錄。

### 待定譯名

尚無。

### Flagged ambiguities

- 本機目錄名 `champions_of_krynn` 與 repo 名 `champions-of-krynn-cht` 分隔符不同，兩者都指本專案。
