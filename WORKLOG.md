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

## 2026-10-19 跨畫面盤點與 v.0.9.1（#6、#10）

- 盤點 demo 戰鬥、大地圖與手動戰鬥、屯墾區武器庫、旅店、紮營存讀檔（docs/re/008）。補本作戰鬥模板、Bozak 譯名；
  `compose.go` 新增商店價格列（補位按半形合併後格數）、購買訊息、數量置後與複數；路徑 C 補試大小寫變體（底列繼續提示）；
  建立隊伍選單第 1 欄也學隊員名。demo 重跑 10 張截圖全部通過，未命中只剩純數字。規格 004 升級 CONFORMED。
- G5 收據回歸 7 場景通過。
- `v.0.9.1-20261019`：三平台原版包（Linux、Wine 冒煙到標題且截圖相同，macOS 結構通過）與本機完整版（`COK_WITH_DATA=1`），都在 `dist-all/`，未公開。
- 清理：本輪背景容器都已結束；`workplace/`、`dist-all/` 無 root 擁有檔。

## 2026-10-19 HD 首批、城鎮服務與跨區盤點（#6、#9）

- HD b01：標題、營火、卡爾爵士、最後家園旅店、商隊五張由 Codex 重繪，3 倍顯示接入、2 倍不變；規格 010、011 CONFORMED。
- 城鎮服務盤點（酒店傳聞、神殿、金庫、訓練選單、法術、旅店）缺口 0。酒店傳聞是三段式引用，`.` 不關面板。
- 跨區：本作改存檔跳區不會載入 GEO（docs/re/009），改在大地圖改座標（存檔 +0x187、+0x189）正規進城；生命值（角色紀錄 +0x62、+0x197）改 250 防全滅。
- 抽斯拉托、蓋加斯（只到城門）、桑克申。補 8 個怪物名、本作背後攻擊模板、`compose.go` 數量＋物品整串；斯拉托重跑 10,415 筆、桑克申 2,366 筆，缺口 0。
- cok-explore：大地圖模式（`-overland`、`-pick`）、提問比對最近四段路徑 B、每 100 動作存盤點（外層 timeout 殺掉時保留結果）。
- 清理：中途停掉的探索容器都隨 `--rm` 移除，`docker ps` 無本專案容器。

## 2026-10-19 v.0.9.2（#10）

- 原 macOS 交叉編譯映像 `psychicwar-osxcross:go1.24.13-15.5-r2` 已不在本機；改用共用映像 `psgolem-osxcross:1.24-15.5-r1`（只使用；go1.24.13、MacOSX15.5 SDK，與原設定相同），`tools/package.sh` 鎖新 ID，可用 `COK_MAC_IMAGE` 覆寫。
- `tools/package.sh v.0.9.2-20261019 all`：原版包與原版＋HD 包各三平台，外洩掃描通過；`COK_WITH_DATA=1` 本機完整版三平台。
- 冒煙：Linux 與 Wine 的原版、HD 包都開到標題且截圖相同（冒煙以 2 倍執行，HD 只在 3 倍顯示）；macOS 結構檢查通過（HD 版含 `hd/theme.json`、不含原版檔）。規格 012 CONFORMED。
- 散布範圍：使用者決定 patch 包全放，只私下分享（CONTEXT 決策表）。
- 正常玩家路徑 `tools/g3-party-receipt.sh` 以 c7ab271 與本輪譯文重跑通過（建角到開場，查驗題自動作答）。

## 2026-10-19～20 HD 全量（#12、#13、#14）

- Codex 交付、Claude 審查提交：#12 PIC／BIGPIC／TITLE（主題 97 項）、#13 戰鬥地形、CPIC、頭身、COMSPR、SPRIT（490 項）、#14 3D 牆面與天空（1,841 項）。#13 的前端接線由 Claude 套用到 `live.go`；各輪 G5 七場景繁中雜湊不變。
- 未完成：COMSPR 原版動態語意、HEAD2/3、BODY2/3、TILES 保留原版；未在執行畫面觀測的來源狀態對不上時顯示原版。

## 2026-10-20 多語系（#17–#20）

- 架構：單一 `Live` 加可替換語言包，換語言時以保留的原文就地重譯（路徑 A、B、C、查驗題、面板）。`tools/g9-lang-receipt.sh` 六項通過：繁中與改動前逐位元組相同，中途換語言與一開始就是該語言的畫面相同。
- 簡中：`tools/zhcn-convert.sh`（OpenCC 1.4.1 tw2sp），專有名詞只轉字形；修正兩處 OpenCC 誤轉。
- 英文：Courier Prime Bold 拉寬字型（使用者選定），劇情維持全大寫；英文手札 145 則由三個子代理從頁面圖逐字轉錄（第 1–30 則查驗題答案 15 題吻合），逐字稿只在本機與發行包。
- 日韓：術語表、試譯經使用者確認文風後，sonnet 子代理全量翻譯（3,173 列加輔助表），`tools/jako_merge.py` 合併檢查。
- 收據：G5 支援 `G5_LANG`；繁中、簡中、日文、韓文七場景、英文六場景通過。過程中修掉：先 SetLang 再綁定時手札編號格式退回繁中、英文換行行首空白、韓文半形標點落在行首、韓文底列提示詞數與原文相同導致逐詞對應失敗、日文「No.」奇數半形字補白。
- 第一次執行改依系統語言選擇（使用者指示，不做選擇畫面），#17 關閉。
- 簡中改用 Unifont 預設版（文泉驛，大陸字形）、日文改用 `unifont_jp`；G5 兩種語言 7/7 通過。同時發現發行包的簡中字型漏收玩家名音譯用字 77 字（開發用字型有收，G5 測不到），`package.sh` 補收後出 v.0.9.4-20261020。

## 2026-10-20 配樂驗收與推廣影片（#15、#16）

- 使用者試聽 12 首新配樂通過，規格 014 CONFORMED。播放器搬到 dosgolem `apps/cok/bgm`，cok-play 與新的 cok-promo 共用。
- 推廣影片（規格 016）：dosgolem `cmd/cok-promo` 從狀態檔續跑逐格輸出畫面與音訊，`tools/promo.sh` 錄五段、`tools/promo-build.sh` 剪成 66 秒 1920×1200 H.264。三段重錄雜湊相同；配樂以有無 `-music-dir` 相減確認混入。
- 試錄時以 cok-play 預設 50% 時脈換算步數，但狀態檔是 100% 時脈，影片快兩倍；改從載入後的機器讀 `VGAFrameEvery`。原版開場劇情沒有音樂，這幾段以新編配樂渲染檔墊底。

## 2026-10-20 作弊與 README

- 我方無敵（規格 017）：逆向隊伍串列 `DS:609A`、紀錄 `+0x17F` 下一位、`+0x197` HP、`+0x188` 狀態、`+0x18A` 陣營（docs/re/010）。dosgolem session 加選用擴充 `WriteGuard`，寫入落地後可寫回舊值；`machine.WatchWrites` 在寫入前通知，回呼寫回會被接著的寫入蓋掉，所以改用寫入後通知的 `WatchWrite`。
- 切換列多一項後無 HD 素材時超寬，繁中、简中「無HD素材」縮為「無HD」。
- README 加成果表與 3 倍 HD 截圖（取自推廣影片錄影畫格）。發行包 v.0.9.5-20261020，三平台冒煙通過。

## 2026-10-20 HD 收尾（#21）與作弊效果串列

- 無敵作弊：效果串列在紀錄 `+0xE3`（9 bytes 一筆，`+5` 下一筆），麻痺 `0x34` 實機證實；參考《詛咒藍符》反組譯（同引擎，health_status、combat_team 同差 0xE）。大地圖戰鬥重播：原版三人昏迷（狀態 4），開啟時全員正常。g10 收據的檢查用 docker 原本缺 `-i`，heredoc 沒傳入卻回 0，已修正。
- HD 收尾由 Codex 交付（報告 `workplace/hd/out/issue21-report.md`）：新增 117 項，主題 1,958 項；HEAD/BODY 立繪、TILES 以完整來源矩形條件式接入，COMSPR 改為可在格子間比對，牆面補留底區模板。Claude 審查：Go 測試、G5 繁中七場景雜湊不變、切換列收據通過。
- 發行包漏洞：`package.sh` 只複製 `theme.json` 與根目錄圖片，#13 以來的戰鬥、人物、效果、牆面主題檔都沒進包（冒煙只驗標題）。改為複製整個主題並逐檔核對；v.0.9.6-20261020 HD 包 3,924 個主題檔，以發行包的 `hd/` 跑 demo 3D 城市，3 倍 3D 視窗與無 HD 差 22,964 像素、文字區 0、記憶體雜湊相同。

## 2026-10-20 HD 逐格替換（#22）

- 使用者指示：牆面配置不用管，素材全部轉 HD，抽樣驗證 3D 視窗；簡單符號與游標也要 HD。
- Codex 交付：24 個簡單符號、全留底圖樣、CURSOR 以 Scale3x 轉 3 倍（不用影像生成），主題 1,984 項；`wall_hd` 改為逐格替換，一格失配只跳過該格。
- 獨立驗證 `tools/hd-3d-sample.sh`（cok-promo `-indexed`，逐格比較 HD 與原版最近鄰放大）：demo 53.1%→99.6%、斯拉托 56.7%→99.5%、72.1%→100%、桑克申 0%→97.0%、蓋加斯二 100%。蓋加斯那組存檔停在戰鬥畫面，不算 3D 樣本，Codex 改用進城前分支補樣 100%。G5 繁中七場景雜湊不變。

## 2026-10-21 圓體字型與製作群英文名

- 使用者回答：源泉圓體、2 倍與 3 倍都換、英文名兩字併一格。`tools/rounded_font.py` 由 GenSenRounded2 TW v2.100（OFL）產生 `cok-rounded.golemfnt`（16×16 Medium）與 `-3x`（24×24 Bold），繁中字集 2,403 字全部有字形。Bold 在 16 像素筆畫密的字會糊，所以 2 倍用 Medium。
- 源泉圓體只有台灣、日本字形，简中若改用會倒回台灣寫法，與先前「简中用大陸字形」衝突，简中維持文泉驛。
- 程式：`LoadFontScaled` 配對 3 倍字模；組合字支援 24×24；發行版優先圓體。製作群英文名原本是純 ASCII 譯文，半形併字規則對全 ASCII 不併，倚天的半形字只畫在格子中間，每個字母兩側各空 1/4 格；非英文版改為兩字併一格。
- 繁中 G5（倚天基準）雜湊不變；日文 demo、town 兩場景因「AC HP」等純英文標題併字而改變，目視正常。

## 2026-10-21 地圖清單版面與推廣影片第二版

- 地圖攻略清單（規格 009 §2.1）：3 倍時改用 16×16 原尺寸字，22 欄 × 36 列；編號固定兩格對齊名稱；所在格說明依長度占底部。2 倍版面相同但編號格式改變，所以繁中 G5 地圖場景雜湊由 `b43868123abc` 變成 `1f0b6f8eb9ad`（其餘六場景不變）。`tools/issue21-g5.sh` 內寫死的是 #21 當時的雜湊，屬歷史收據，不改。
- 推廣影片（規格 016）：加故事背景字卡四張、3D 與戰鬥的 HD 切換擦除示範（cok-promo `-hd-off` 錄同時間軸原版段）、F12 中英切換示範，82.8 秒。
- 發行包 v.1.0.2-20261021，三平台冒煙通過。
