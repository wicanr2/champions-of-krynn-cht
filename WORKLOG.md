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

## 2026-10-19 參考文件盤點（#7、#8）

- 使用者提供英文 Adventurer's Journal PDF（根層）與軟體世界第 16–18 期攻略掃描（`walkthrough/`）。
- 另從 mocagh.org 下載 SSI 的 journal、manual、hintbook PDF 到 `workplace/reference-src/web/`（容器開網路一次）。
- 新增 `tools/docker/pdf.Dockerfile`（取自幽靈騎士，poppler-utils），建 `cok-pdf:bookworm-r1`；`pdftotext` 抽文字到 `workplace/reference-src/txt/`。
- 結果見 `docs/re/002-reference-sources.md`：使用者 PDF 32 頁、Journal Entry 1–88 全部可辨識；mocagh 版 OCR 漏第 76 則，作校對。
- `.gitignore` 補 `*.pdf`。容器皆 `--rm`。

## 2026-10-19 G2 dosgolem 基線（#3）

- 由幽靈騎士工作 clone（`d4cd48c`，等於遠端 `dosgolem/dkk` 頂端）本機 clone 到 `workplace/dosgolem/`，分支 `champions-of-krynn-cht`；module cache 由幽靈騎士 clone 複製，未開網路。
- 原版 `KRYNN.CFG` 選 Roland LAPC，冷啟動卡在 MPU-401 輪詢（`5252:02AC`，`DX=0331`）。新增 `tools/cfg-adlib.sh`，scratch 層放 AdLib 版設定後可到標題、製作群、PLAY DEMO。
- 新增 `tools/g2-startup-receipt.sh`：送 `p` 到建立隊伍主選單，重播 a、b 雜湊一致。結果見 `docs/re/003-dosgolem-startup.md`。
- 一次手動測試把 `/dev/stdout` 當 `cfg-adlib.sh` 的輸出，`cmp` 讀它卡住，已中止該行程；腳本用法只接一般檔案。
- 清理：dosgolem 容器皆 `--rm`，無殘留；`workplace/` 無 root 擁有檔。

## 2026-10-19 G3 輸出路徑盤點（#4）

- 以幽靈騎士 adapter 的位元組簽章在本作 `START.EXE` 找到三個 resident 印字程序（路徑 A `0631:0353`、逐字 `0631:01DF`、路徑 B `0631:04AC`），
  主選單 `-call-args` 證實參數格式相同。路徑 B 與幽靈騎士逐 byte 對齊得游標 `DS:6134／6135`。見 `docs/re/004-output-paths.md`。
- dosgolem 分支建 `apps/cok`（複製 `apps/dkk`，換簽章、預設 DS `0CD6`、游標位址）與 `cmd/cok-play`、`cok-overlay-shot`、`cok-explore`。
- 預建隊伍讀檔到查驗題；`tools/g3-party-receipt.sh` 走建角（ANNA）→ 加入隊伍 → 開始冒險 → 查驗題 → 開場劇情。
- 路徑 C 呼叫端 bytes 與逐字程序位移 `01DF` 是逐版本的，改用本作值；城鎮選單詞是獨立 ECL 字串，介面查表加轉大寫查 ECL catalog。
- 手札引用分兩則印（`… JOURNAL ENTRY ` ＋ `51.`），adapter 改成兩段式觸發；酒店傳聞用 1000+N。實機第 51 則自動彈出。

## 2026-10-19 G4 譯文（#5）

- 靜態抽取 ECL 2302 則（83,495 字元）、介面候選 1326 則。繼承幽靈騎士：介面 517 則（雜湊相同）、ECL 50 則、術語表 664 條。
- 試譯 1 批經使用者確認文風後，8 個 sonnet subagent 平行翻譯 34 批；合併驗證 0 錯誤，ECL 2284 則、介面 889 則。
- 譯名：人名以幽靈騎士術語表優先（使用者提醒）；表外依 zh.wikipedia「龍槍系列小說」通行譯名；新詞 48 條收斂進術語表。
- 已知待查：介面候選中 32 字元滑動視窗的重疊片段，譯者只在一列放全句，實際印出的是哪一列待盤點確認。

## 2026-10-19 G6 手札與查驗題（#7）

- 三個英文轉錄 subagent 以著作權為由拒絕逐字重製手札，一個完成第 60–88 則（只留本機 `workplace/journal/en/`）。改用作法：
  查驗題答案由原版題目表＋OCR 逐字座標算出（只存答案字），手札中文版由 subagent 看頁面圖直接翻譯。
- `START.EXE` 題目表 49 筆（`docs/re/005-quiz-table.md`），答案長度 49 題全部相符，第 67 則一題人工覆寫；實機答對 2 題。
- 手札 88 則、酒店傳聞 57 則中譯進 `text/journal.zh-TW.tsv`；Tavern Tale 依幽靈騎士術語表譯「酒店傳聞」。
- 清理：容器皆 `--rm`；新建映像 `cok-pdf:bookworm-r1`、`cok-img:r1`（Pillow 11.0.0、numpy 2.1.3）。

## 2026-10-19 G7 攻略面板與地圖疊層（#8）

- subagent 轉錄《軟體世界》第 16–18 期攻略 14 頁、19 章到 `reference/walkthrough-zh/`（private）；雜誌譯名對照 `text/walkthrough-terms.tsv` 27 列；面板 26 段。
- 地圖：GEO 15 塊；名稱依同號 ECL 區塊文字判讀（`text/map-titles.tsv`）；攻略座標產生 157 個攻略點（`tools/map_guide_build.py`），標籤由 subagent 改寫成 8 字內。
- 地圖位址 `DS:6D7C`／`6D85` 在 demo 3D 畫面以 GEO 2/40 雜湊與畫面座標證實（docs/re/006）；性別 `+0x109` 以建角對照證實。
- 術語一致性檢查改本作版（`tools/term-check.sh`），收斂城主、酒店、善良陣營的軍隊、火球等，結果 0 問題。

## 2026-10-19 G5 收據與路徑修正（#6）

- 路徑 C 改以連續片段查表（戰鬥選單 `Move `、`View Aim ` 等多字串組成）；物品名詞組先查名稱 catalog（`Ring Mail` 環甲）。
- 本作組句模板：無尾端空白的移動力列、`Range = {n}  `、靠右對齊的金錢列、丟錢提示。
- `tools/g5-receipt.sh` 7 場景通過（docs/re/007），規格 001–003、006–009、013 升級 CONFORMED。

## 2026-10-19 G9 試打包（#10）

- `tools/package.sh v.0.9.0-20261019 all`：三平台原版包（沒有 HD 主題時只出原版包），外洩掃描通過。
- 冒煙：Linux AppImage 與 Windows（Wine 9.0，映像 `psgolem-wine-verify:noble-r2`，只使用）從乾淨目錄開到標題畫面、兩者截圖相同；缺原版時以錯誤結束並留 `cok-error.log`；macOS 只驗結構。
- 清理：兩個背景容器因外層 timeout 只殺 client 而殘留（`kind_bose`、`bold_wilson`，皆本 session 建立），已 `docker stop`，`--rm` 自動移除。
