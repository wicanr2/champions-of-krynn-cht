# 《克萊恩英豪》（Champions of Krynn）dosgolem 繁體中文化與 HD 版本

本檔是本專案的協作入口，Claude 與 Codex 共用。面向使用者的回覆、文件與提交訊息使用
繁體中文。所有研究與實作先建立 GitHub Issue 再開工，遠端 Issue 是工作狀態的唯一權威。

## 1. 使用者授權與專案範圍

依使用者 2026-10-19 的指示，本專案參考 `/home/anr2/cht/golden_box/death_knight_of_krynn/`
（以下稱幽靈騎士），治理與範圍完全比照。本次工作已授權：

- 用 dosgolem 執行原版 DOS《Champions of Krynn》，以輸出攔截覆繪做繁體中文化，並提供 HD 版本。
- 建立 **private** GitHub 儲存庫 `champions-of-krynn-cht`，啟用 Issues。owner 以主機
  `gh auth status` 回讀的登入帳號為準。建立前檢查同名儲存庫，不接管既有專案。
- 初始化本目錄的 Git，設定 repo-local 身分 `wicanr2@gmail.com`，提交並推送本範圍內的文件、
  原創程式與建置腳本。推送前檢查明確的檔案清單，不使用 `git add .`。
- 建立、更新、留言及關閉本專案範圍內的 Issue。

下表是沿用幽靈騎士的定案，使用者 2026-10-19 以「完全比照」確認，不重複詢問：

| 項目 | 定案 |
|---|---|
| 工作順序 | 先完成中文化，再做 HD。HD 階段在中文化與手冊題完成前不開工 |
| 顯示語言 | 第一階段只交付繁體中文；架構預留語言切換，多語系日後另開 Issue |
| HD 規格 | HD 圖以 3 倍（960×600）為準，風格 B（奇幻插畫），配色照原版 |
| HD 顯示 | 預設 2 倍顯示原版，F2 切 3 倍，HD 圖只在 3 倍顯示。3 倍中文由 16×16 最近鄰放成 22×22，置於 24 像素格 |
| HD 分工 | HD 整條（抽樣、來源辨識、請求、重繪、驗收、接入）交 Codex，以 Issue 工作項目追蹤；使用者另行指派時 Claude 才接手。介面見第 6 節 |
| 切換列 | 視窗最上方的切換列平常隱藏，滑鼠移到頂端時出現；HD 切換與日後擴充功能放這裡。沒有 HD 素材時停用 HD 切換並顯示原因 |
| 旅行手札 | 手札全部內容放進遊戲，觸發時自動彈出，另以前端鍵翻閱已觸發過的手札 |
| 手冊查驗題 | 問句後顯示答案，預設開啟、可關閉；玩家仍要作答，判定照原版 |
| 攻略面板 | 攻略做成遊戲內可翻頁面板，另做地圖疊層攻略點 |
| 發行包 | 三平台（Linux AppImage、Windows zip、macOS universal zip）；可散布 patch 分原版、原版＋HD 兩包；本機另出完整版，不散布 |
| 截圖入 repo | README 展示用的中文化截圖放 `docs/screenshots/`（private repo）；其他截圖只留 `workplace/`。轉公開前與其他原版素材一併審查 |

手札與查驗題兩條要等 G1 確認原文來源後才生效（第 3 節）。

儲存庫轉公開、公開 Release、上傳原版素材、加入協作者、改動其他專案、推送 dosgolem 上游
及破壞性操作，另行取得授權。

啟動順序：**本 AGENTS.md → `.gitignore` → 私人儲存庫 → 初始 Issues → 研究與實作**。
建立 Issues 前，不解壓原版、不跑遊戲、不寫產品程式。

### 初始 Issue 範圍

初始工作已登錄為 [#1–#10](https://github.com/wicanr2/champions-of-krynn-cht/issues)。下表只定義範圍與依賴，狀態只在遠端維護。規格編號以幽靈騎士
`docs/spec/001`–`013` 為對照，本作規格另行編號。

| 階段 | 依賴 | 完成條件 |
|---|---|---|
| [G0 協作治理／#1](https://github.com/wicanr2/champions-of-krynn-cht/issues/1) | 本檔 | 回讀 private 與 Issues；`.gitignore`、`LICENSE`、`CONTEXT.md`、`WORKLOG.md`、`CLAUDE.md` 就位；提交本檔 |
| [G1 原始輸入／#2](https://github.com/wicanr2/champions-of-krynn-cht/issues/2) | G0 | 壓縮檔與成員的 SHA-256、檔案清單、版本證據、EXE 與啟動參數；英文手冊、旅行手札、查驗題題庫是否在壓縮檔內 |
| [G2 dosgolem 基線／#3](https://github.com/wicanr2/champions-of-krynn-cht/issues/3) | G1 | 固定 dosgolem commit；從真實入口到第一個可操作畫面；輸入重播與缺服務紀錄，缺口逐項開 Issue |
| [G3 輸出路徑盤點／#4](https://github.com/wicanr2/champions-of-krynn-cht/issues/4) | G2 | 逐條印字路徑的原始定位與證據等級；驗證第 4 節的幽靈騎士對應假說 |
| [G4 譯文與字型／#5](https://github.com/wicanr2/champions-of-krynn-cht/issues/5) | G3 | TSV catalog、術語表、字型子集與檢查工具；先試作一批給使用者看文風再全量 |
| [G5 中文覆繪接入／#6](https://github.com/wicanr2/champions-of-krynn-cht/issues/6) | G3、G4 | 每條路徑 READY 規格、覆繪、A/B 收據與正常玩家路徑；含組句、英數半形、玩家名音譯 |
| [G6 手冊題與旅行手札／#7](https://github.com/wicanr2/champions-of-krynn-cht/issues/7) | G3、G4、G1 來源 | 查驗題與手札的輸出證據、中文段落來源與覆繪收據 |
| [G7 攻略面板與地圖疊層／#8](https://github.com/wicanr2/champions-of-krynn-cht/issues/8) | G5、使用者提供掃描 | 軟體世界攻略轉錄、術語對齊、面板與地圖疊層收據 |
| [G8 HD 與切換列／#9](https://github.com/wicanr2/champions-of-krynn-cht/issues/9) | G5、G6 | 交 Codex：來源與遮擋證據、批次重繪、接入、F2 與切換列收據 |
| [G9 封包驗收／#10](https://github.com/wicanr2/champions-of-krynn-cht/issues/10) | G5、G8 | 乾淨目錄啟動、manifest 與 SHA-256、存檔位置；明列未驗平台 |

## 2. 定位與邊界

- 原版 EXE、資料檔、規則、亂數、存檔格式、條件比較、檔名與腳本鍵值保持原樣。本專案不是
  remake，也不翻譯 EXE 二進位。
- 中文化只在已證實的玩家可見輸出路徑，以原文、呼叫位置與畫面座標辨識字串，清除原文墨跡後
  用繁中點陣字覆繪。英文原文是穩定比對鍵，譯文只影響顯示，不進入比較、查找、序列化、
  檔案路徑或存檔。
- 原版要求查閱手冊時，辨識該題目的輸出事件，覆繪對應的中文題幹與手冊段落。不繞過驗證、
  不改判定結果。
- HD 層只改變呈現。素材來源、繪製時機、遮擋、清除及動畫依本作證據辨識；未辨識或失配時
  顯示原版，不用推測圖像覆蓋畫面。
- 中文字依 `~/.claude/rulebook/81-retro-cjk-hires-canvas.md`：拉高輸出畫布，原版底圖整數倍
  最近鄰放大，中文用正常尺寸點陣字，不縮小中文去塞原字位。
- 不因幽靈騎士或其他 Gold Box 遊戲能在 dosgolem 執行，就宣稱本作已受支援。先跑本作的正常
  啟動探針。

## 3. 原始輸入

| 檔案 | 用途 | 狀態 |
|---|---|---|
| `Champions of Krynn (1990).zip` | 原版 DOS 遊戲 | 內容、EXE 名、版本待 G1 盤點 |
| 軟體世界攻略掃描圖 | 攻略面板與地圖疊層 | 待使用者提供，放 `walkthrough/` |

- 本作沒有中文手冊。手冊、旅行手札與查驗題的英文原文來源由 G1 確認；壓縮檔內沒有時向
  使用者索取，不從網路自行補抓後當成正式來源。
- `git init` 前先寫 `.gitignore` 排除壓縮檔名、`walkthrough/`、`workplace/` 與 `dist-all/`，
  G1 盤點後再補原版檔案的副檔名。每次 commit 前檢查 staged 清單。
- 原始壓縮檔保留原位、不覆寫，只在 Docker 內唯讀掛載。解壓、盤點、雜湊與 OCR 產物寫入
  `workplace/`。
- 原版遊戲、手冊、掃描、存檔、擷取素材及可還原原作的大量文字，不進 Git、GitHub 或可散布
  發行包。需要原版的測試在檔案缺席時明確 skip。
- 例外（沿用幽靈騎士）：攻略掃描的轉錄放 `reference/walkthrough-zh/` 並提交到 private repo
  保存，因為 `workplace/` 可能被刪除。repo 轉公開前必須移除此目錄並清理歷史，掃描原圖仍不進 Git。
  雜誌攻略有缺時，比照幽靈騎士以 SSI 官方英文提示書補，譯文放 `reference/hintbook-zh/`，同樣
  轉公開前移除；來源與取得方式記入對應 Issue。

### 譯名與文風

- 譯名優先序：本作遊戲譯文 > 幽靈騎士 `text/glossary.tsv` 的共用譯名 > 本作自譯。克萊恩世界設定、
  種族、職業、神祇、索蘭尼亞騎士團階級、怪物與共通物品直接繼承幽靈騎士譯名；本作新詞自譯並在
  術語表註明。
- 繼承幽靈騎士的譯名規則：通行用法優先（Paladin 聖騎士、Ranger 遊俠等）；物品名逐詞替換、加值置後
  （例「長弓 +3」）；英文雙關謎語附英文原句與答案。
- 文風忠實正式、奇幻古典語氣。敘述對隊伍用「你們」，對白依語境；玩家須輸入的字寫成
  「中文（ENGLISH）」。
- 字型比照幽靈騎士：開發用本機倚天 16×15，發行用 GNU Unifont。
- catalog 政策：repo 只放 key、長度、雜湊、譯文；key 對原文的對照只在 `workplace/`。
- 中文題名「克萊恩英豪」由使用者指定。遊戲內其他專名未定者先登錄在 `CONTEXT.md` 的 glossary 待定區。

## 4. 參考專案與按需載入

每個工作階段先讀本檔，再讀 `CONTEXT.md`、本次遠端 Issue 與直接相關規格。

| 來源 | 本專案採用 | 不直接移植 |
|---|---|---|
| 幽靈騎士（`AGENTS.md`、`CONTEXT.md`、`docs/spec/`、`docs/re/`、`tools/`、`text/glossary.tsv`） | 同系列同引擎的第一參考：輸出路徑家族、規格 001–013 的架構、收據腳本、術語表、HD 與切換列、發行包流程 | 位址、座標、字串、callsite、重播輸入、地圖雜湊、完成數字 |
| `/home/anr2/cht/golden_box/拯救地球/` | 輸出攔截覆繪架構、ECL 文字視窗、水平選單、引擎訊息模板、手冊事件、字型管線、A/B 驗收 | 位址、座標、字串、重播、完成數字 |
| `/home/anr2/cht/taipei/`（`AGENTS.md`、`docs/spec/014-live-theme-switch.md`） | HD 分層、來源失配回退、F12 原版／HD 即時切換、Issue 先行治理、封包驗收 | 美術定案、解析度、`workspace/` 目錄名 |
| `/home/anr2/cht/golden_box/curse_of_the_azure_bonds/`、`golden-box-remake-engine/` | Gold Box 的 DAX／ECL 格式解析、文字安全矩形、TPOV overlay 逆向方法 | 遊戲規則、素材、舊交付慣例 |
| `/home/anr2/cht/dosgolem/`（`CLAUDE.md`、`README.md`、`docs/spec/000-index.md`） | 執行器能力、`cmd/probe`、Docker 入口 `tools/go.sh` | 不直接改動其工作樹（第 5 節） |

**待驗假說**：本作與幽靈騎士同屬克萊恩系列 Gold Box 引擎，幽靈騎士的輸出路徑家族（A Pascal
字串、B 文字視窗、C 底列水平選單、組句訊息、手冊查驗題、旅行手札）很可能有對應。這是 G3 的
起點；對到本作的 callsite 與 bytes 之前不當成已知。本作年份較早，EXE 結構、overlay 配置與畫面
模式也要獨立確認。

其他按需入口：

| 觸發 | 入口 |
|---|---|
| 老遊戲逆向與中文化規則路由 | skill `re-retro-cht-rulebook` |
| Gold Box `.OVR`／TPOV | `~/.claude/knowledge-base/retro/borland-tpov-overlay-re.md` |
| IDA Pro 9.4 | `~/.claude/knowledge-base/retro/ida-pro-9.4.md` |
| 翻譯與顯示隔離 | `~/.codex/knowledge-base/local/localization-display-semantic-isolation.md` |
| 建立文件或新目錄 | `~/.codex/knowledge-base/local/project-document-responsibilities.md` |
| 交付目錄 | `~/.codex/knowledge-base/local/retro-remake-dist-all-output.md` |
| 授權 | `~/.codex/knowledge-base/local/rrsal-1.0.md`（RRSAL-1.0，已定案） |
| 正式版號 | `~/.codex/knowledge-base/local/retro-remake-release-versioning.md` |

## 5. dosgolem 分層

- 本作位址、畫面狀態、輸出攔截點、重播與 HD 素材對應屬遊戲專屬層，放在 dosgolem 的
  `apps/cok/`。通用 CPU、DOS、BIOS、VGA、輸入與音訊能力屬 dosgolem 通用層，不寫入本作名稱
  或位址。判準見 dosgolem `docs/spec/006-layering.md`。
- 遊戲短名 `cok`（使用者 2026-10-19 定案）。研究與開發在 `workplace/dosgolem/` 的隔離 clone
  進行，分支 `champions-of-krynn-cht`。不改動 `/home/anr2/cht/dosgolem` 工作樹，不推送 dosgolem
  遠端，也不推送幽靈騎士的儲存庫。
- clone 起點是幽靈騎士 private repo 的 `dosgolem/dkk` 分支，以取得已修正的通用層（例如
  `FramePalette`、`xlate` 部分覆蓋、字型倍率、HD 主題載入、切換列、發行模式）。`apps/dkk/` 只當
  參考實作，其中的位址、座標與字串在本作全部視為假說，不整包複製成 `apps/cok/` 的事實。
- 從 `dosgolem/dkk` 帶過來、但其實屬於通用層的程式，留在通用層，不寫入 `cok` 名稱。
- Go 不允許外部 module 引用 dosgolem `internal/`，adapter 必須在 dosgolem module 樹內。clone 分支
  推到本專案 private repo 的 `dosgolem/cok` 分支保存；本專案 `main` 只放文件、譯文、字型來源與
  工具。通用層修正要回饋 dosgolem 上游時另行取得授權。
- 固定使用的 dosgolem commit（含基底的 `dosgolem/dkk` commit）、Go 版本與 Docker image ID 記入
  `CONTEXT.md`。
- dosgolem 是原版行為、畫面與虛擬時間的權威執行器，正式收據由它產生。DOSBox／DOSBox-X
  只作診斷與交叉基準，擷取明標輔助，不能單獨宣稱完成。

## 6. HD：Codex 的工作介面

HD 整條由 Codex 負責（第 1 節）。每個批次先開 Issue，寫明請求目錄、規格、風格定案、完成條件，
以及 Codex 是否可以 commit、push。Issue 沒寫明時不 commit、不 push。

| 階段 | 產出 |
|---|---|
| 來源辨識 | 本作繪圖來源、繪製時機、遮擋與動畫相位證據；DRAFT 規格 |
| 匯出原版圖塊 | `workplace/hd/requests/<批次>/`：原版 PNG 與 `manifest.json`（來源檔、offset、SHA-256、尺寸、色盤、透明色、規格編號） |
| 重繪 | `workplace/hd/out/<批次>/`：重繪 PNG 與 `manifest.json`（對應來源 key、輸出尺寸、風格批次、生成方式、權利註記） |
| 驗收與接入 | 張數、尺寸、alpha、對位檢查；原版／HD 同狀態比較；接入 dosgolem 呈現層 |

- Claude 委派 Codex 時以主機 `codex exec` 執行，這是繼 `gh` 之後第二個主機工具例外。只用已核對
  過的旗標：`-C <本專案根目錄>`、`-s workspace-write`、`-o <輸出目錄>/codex-last-message.md`，必要時
  `--add-dir`。prompt 明寫可讀與可寫的目錄。
- Codex 只改該 Issue 指定的目錄與檔案，不改其他規格、`CONTEXT.md` 的有效決策或其他 Issue。
- 轉檔、縮放、建置與批次腳本一律在 Docker 內執行，遵守第 8 節。
- 輸出尺寸、透明色與構圖依請求 manifest；無法符合時在 `codex-last-message.md` 或 Issue 說明，不自行改規格。
- Codex 產出的素材視為可放進原版＋HD patch（沿用幽靈騎士），權利分類仍記在 manifest 與 Issue。

## 7. 規格與驗證閘門

正式行為採：`原版證據 → DRAFT → 證據審查 → READY → 實作 → 同狀態驗證 → CONFORMED`。
沒有 READY 的探針與原型不進產品路徑。

- 每項研究保留輸入檔名與 SHA-256、工具版本、dosgolem 位址空間、callsite、暫存器、字串位址、
  字型、座標、顏色與清除時機。證據等級使用「已證實、強推論、假說、未知」。反組譯名稱只供
  導覽，不取代原始定位。引用幽靈騎士的結論時註明出處，等級最高為「假說」，直到本作重驗。
- 所有玩家可見譯文以 UTF-8 TSV 為唯一正式來源，檢查 key 唯一、欄數、控制碼、placeholder、
  漏譯、孤兒 key、行寬與字型覆蓋。字型子集由正式譯文重建，不為缺字改譯。
- 術語表、手札、攻略面板與地圖點的譯名一致性用檢查腳本驗證（比照幽靈騎士 `tools/term-check.sh`），
  不一致時結束碼非 0。
- 每條已接通的中文路徑做原文／繁中 A/B：同資料、同初始狀態、同輸入、固定 seed，差異只能
  出現在批准的覆繪像素。另走正常玩家路徑驗證進入、捲動、清除、返回與存讀檔後不殘字。
- HD 驗證原版記憶體、暫存器、原版畫面緩衝與存檔不受影響；切換不得重開遊戲或重設亂數。
  視覺驗收另驗來源辨識、遮擋、背景恢復與讀檔殘影，並用失配來源與撤除素材建立負對照。
- 快照、direct-entry 或狀態注入只供診斷，不取代正常玩家路徑。
- 只有被 dosgolem 同狀態收據覆蓋的輸出路徑，才能稱為「已中文化」。尚未量到的路徑標示未知。

## 8. Docker、Git 與交付衛生

- 分析、解壓、搜尋、OCR、建置、測試、執行遊戲與 GUI 自動化只在 Docker 內進行。使用 `--rm`、
  `--memory`、`--cpus`、`--pids-limit`、外層逾時、`-u "$(id -u):$(id -g)"`、
  `--log-opt max-size=10m --log-opt max-file=3`，預設 `--network none`。
- 掛載前確認 host 來源存在且型態正確。每批工作後在容器內執行
  `find <工作根> -user root -o -type d -name '*.md'`，並確認本次容器已清理。不碰其他專案的
  image、volume 與 container，禁止任何 prune 與 `rmi`。
- 主機例外只有 `/home/anr2/.local/bin/gh` 與 `codex`。第一次使用 gh 先執行 `gh auth status`。
- `workplace/` 是唯一可寫研究工作區，保存解壓輸入、探針輸出、IDA 資料庫、dosgolem clone 與
  HD 批次，整個目錄 gitignore。不另建 `work/`、`research/` 等同義目錄。
- 幽靈騎士的目錄只讀，不在其中寫檔、建分支或執行會產生輸出的腳本。
- 自有程式與文件依 RRSAL-1.0 範本建立 `LICENSE`，第一個 commit 前就位。
- 正式交付根為 `dist-all/<版本>/`，子目錄依第 4 節交付契約。完整版只留本機。

## 9. 文件職責

| 文件 | 內容 |
|---|---|
| `AGENTS.md` | 本檔，專案規則 |
| `CLAUDE.md` | 指回本檔的薄入口、常用指令與資料索引 |
| `CONTEXT.md` | 目前真相、有效決策、glossary 與證據入口 |
| `WORKLOG.md` | 按時間追加的歷程、驗證結果與 Docker 清理狀態 |
| `docs/re/` | 輸入雜湊、位址、實驗與推論等級；`docs/re/README.md` 為索引 |
| `docs/spec/` | DRAFT、READY、CONFORMED 規格，回鏈 Issue |
| `text/`、`font/` | 譯文 catalog、術語表、手札與攻略索引；字型來源與字元清單 |
| `reference/` | 攻略與提示書轉錄（private，轉公開前移除） |
| `README.md` | 用途、穩定入口與現況摘要 |

GitHub Issue 是唯一可執行工作清單，本機不建 `WORKLIST.md`。

## 10. 共同決策

- 已定案事項（第 1 節、第 3 節譯名與文風）不重複詢問。
- HD 美術與操作取捨先展示並列原型，每次只問目前最早會阻塞的決定。
- 實質改變規則、存檔格式、原版忠實度、版面、平台範圍或散布方式時先確認。可由 bytes、
  程式、規格或測試查出的事實自行查證。
