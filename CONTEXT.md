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
| 2026-10-19 | 執行時音效設定用 AdLib（scratch 層的 `KRYNN.CFG` 副本），原版檔不改 | 原版 Roland 設定在 dosgolem 卡住（docs/re/003）；比照幽靈騎士 |
| 2026-10-19 | 旅行手札採英文 Adventurer's Journal：使用者下載的 PDF 為主要來源，mocagh 版校對（docs/re/002）；手札入遊戲與查驗題顯示答案兩條決策生效 | 使用者指示 |
| 2026-10-19 | 軟體世界第 16–18 期攻略掃描已提供，放 `walkthrough/`（docs/re/002） | 使用者提供 |
| 2026-10-19 | 可散布 patch 包維持全放（手札、酒店傳聞、攻略面板、地圖攻略點、查驗題答案表），只私下分享給持有原版的人，不公開上架 | 使用者指示 |
| 2026-10-19 | 跨區盤點只抽樣斯拉托（ECL1）、蓋加斯（ECL2）、桑克申（ECL3）；其餘 12 張地圖不實機進入 | 使用者指示：覆繪不改引擎，事件文字共用路徑 B |
| 2026-10-20 | 多語系（規格 015）：英文以新字型重繪原文並加英文手札面板；簡中由繁中 OpenCC 轉換加詞彙表；日文、韓文以英文為底 AI 翻譯並標註機器翻譯；簡中、日文、韓文字型用 Unifont；啟動器選擇並記住，切換列循環切換 | 使用者回答 |
| 2026-10-20 | HD 收尾（#21）；無敵作弊再擋麻痺、睡眠等效果；刪除遠端分支 `dosgolem/cok-ml` | 使用者指示 |
| 2026-10-20 | 增加作弊選項：我方 HP 不減少、不進入昏迷中毒等異常；README 放 HD 截圖與成果說明 | 使用者指示 |
| 2026-10-20 | 第一次執行的語言依系統語言偵測，不做選擇畫面；12 首新配樂試聽通過 | 使用者指示 |
| 2026-10-20 | 英文版字型 Courier Prime Bold 水平拉寬填滿字格；ECL 劇情維持原版全大寫（規格 015 §3.1） | 使用者回答 |
| 2026-10-20 | 日韓文風依試譯定案（日：常體過去式、隊伍「あなたたち」；韓：해라체、隊伍「일행」）；日韓手札從英文逐字稿翻譯；攻略面板與地圖攻略點從繁中翻譯（規格 015 §5） | 使用者回答 |

## 待定

目前沒有。

## dosgolem 基線

| 項目 | 值 |
|---|---|
| commit | `d4cd48c`（幽靈騎士 `dosgolem/dkk` 頂端；本作分支 `champions-of-krynn-cht`，推到本 repo 的 `dosgolem/cok`） |
| Go／image | go1.24.13；`dosgolem-go-ebiten:1.24-r1` `sha256:8ed6bdee2bce…` |
| 冷啟動 | 0 種缺服務；mode 0Dh；scratch 放 AdLib 版 `KRYNN.CFG`（原版選 Roland 會卡在 MPU-401）；送 `p` 到建立隊伍主選單，見 [docs/re/003](docs/re/003-dosgolem-startup.md) |

## 中文化進度

| 路徑 | 狀態 |
|---|---|
| A Pascal 字串（選單、清單、隊伍欄、查驗題） | 規格 001 CONFORMED（docs/re/007） |
| B 文字視窗（劇情） | 規格 002 CONFORMED |
| C 底列水平選單 | 規格 003 CONFORMED；城鎮、營地、戰鬥選單全部覆繪（多字串選單以連續片段查表） |
| 組句訊息 | 規格 004 CONFORMED；demo 戰鬥、商店清單、購買訊息、金錢列（docs/re/008） |
| 手冊查驗題 | 規格 006 CONFORMED；題目表 49 筆（docs/re/005），實機 2 題 |
| 旅行手札、酒店傳聞 | 規格 007 CONFORMED；88＋57 則中譯，兩段式引用觸發 |
| 攻略面板 | 規格 008 CONFORMED；軟體世界 16–18 期 19 章 |
| 地圖疊層 | 規格 009 CONFORMED；15 張地圖、157 個攻略點 |
| 玩家名音譯 | 規格 013 CONFORMED；性別 +0x109（docs/re/006） |
| 發行包 | 規格 012 CONFORMED；`v.1.0.0-20261020`（正式版）原版包、原版＋HD 包（三平台，五種語言、配樂）與本機完整版在 `dist-all/`；Linux、Wine 冒煙到標題、macOS 驗結構；只私下分享，不公開上架 |
| 存讀檔不殘字 | 紮營存遊戲 B、主選單讀回，隊伍選單 19/19、範圍外差異 0（docs/re/008） |
| HD（G8） | 規格 010 CONFORMED；Codex 重繪 1,984 項（PIC／BIGPIC／TITLE、戰鬥、3D 牆面與天空、HEAD/BODY 立繪、TILES、COMSPR，簡單符號與游標以 Scale3x），3 倍顯示接入、2 倍不變；3D 視窗逐格替換，`tools/hd-3d-sample.sh` 抽樣覆蓋率 97–100%；報告 `workplace/hd/out/issue21-report.md`、`issue22-report.md` |
| 配樂 | 規格 014 CONFORMED；12 首新編曲，使用者試聽通過 |
| 多語系 | 規格 015 CONFORMED；繁中、简中、日文、韓文、英文，第一次執行依系統語言；简中大陸字形、日文日本字形 |
| 作弊 | 規格 017 CONFORMED；我方無敵（F1，預設關），`tools/g10-cheat-receipt.sh` |
| 推廣影片 | 規格 016 READY（待使用者觀看）；`workplace/promo/cok-promo.mp4` 66 秒（`tools/promo.sh`） |
| 跨區盤點 | 斯拉托、蓋加斯（只到城門）、桑克申抽樣，補 8 個怪物名、背後攻擊模板、數量＋物品規則後缺口 0（docs/re/008、009）；其餘 12 張地圖沒有實機進入 |
| 城鎮服務 | 酒店傳聞、神殿、金庫、訓練選單、法術記憶與抄寫、旅店盤點缺口 0；切換列規格 011 CONFORMED。訓練只到選單，沒有實際升級 |

## 原始輸入

| 檔案 | 狀態 |
|---|---|
| `Champions_of_Krynn_Adventurers_Journal.pdf` | 英文手札 32 頁，Journal Entry 1–88；見 [docs/re/002](docs/re/002-reference-sources.md) |
| `walkthrough/` | 軟體世界第 16、17、18 期攻略掃描 14 張；見 [docs/re/002](docs/re/002-reference-sources.md) |
| `Champions of Krynn (1990).zip` | 原版 DOS v1.2，入口 `START.EXE`＋`GAME.OVR`（`FBOV`），附 pregenerated party 存檔；見 [docs/re/001](docs/re/001-input-baseline.md) |

## Glossary

- 克萊恩英豪：本作中文題名，使用者指定。
- 手札：英文 Adventurer's Journal 的 Journal Entries（1–88），遊戲內以「參見手札 N」引用。_Avoid_：日誌、日記
- 幽靈騎士：參考專案《Death Knights of Krynn》繁中化，本機 `/home/anr2/cht/golden_box/death_knight_of_krynn/`。
- cok：本作遊戲短名，用於 dosgolem `apps/cok/` 與程式識別。
- dkk：幽靈騎士的遊戲短名；`dosgolem/dkk` 是本作 dosgolem clone 的起點分支。
- 覆繪：在原版已輸出的畫面上清除原文墨跡、重畫中文。_Avoid_：替換、打字串補丁
- scratch 層：dosgolem 的可寫暫存目錄，開檔先找它再找唯讀原版目錄；執行時設定與存檔放這裡。
- 收據：dosgolem 以固定輸入重播產生、可重現的執行證據。_Avoid_：截圖證明
- HD 批次：一組交給 Codex 重繪的原版圖塊，含請求與輸出兩個目錄。

### 待定譯名

尚無。

### Flagged ambiguities

- 本機目錄名 `champions_of_krynn` 與 repo 名 `champions-of-krynn-cht` 分隔符不同，兩者都指本專案。
