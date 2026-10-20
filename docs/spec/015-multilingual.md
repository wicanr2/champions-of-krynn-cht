# 015 多語系：簡體中文、日文、韓文、英文

狀態：**READY**（2026-10-20：§2–§5 已實作，G5 七場景繁中、簡中、日文、韓文、英文（六場景，無攻略）通過；啟動器第一次執行時選語言的畫面待做）。Issue：#17–#20。

使用者指示（2026-10-20）：規劃簡體中文、日文、韓文、英文的支援；英文版把原版英文整個換成更好看的英文字，並把手札放進遊戲。

使用者決策（2026-10-20）：

| 項目 | 決策 |
|---|---|
| 英文字型 | Courier Prime Bold（OFL）水平拉寬填滿原版方形字格（對照圖候選 G）；劇情維持原版全大寫（2026-10-20） |
| 英文範圍 | 原版英文內容不改，改用新字型重繪；手札與酒店傳聞以英文 Adventurer's Journal 原文在遊戲內面板顯示 |
| 譯文來源 | 簡中由繁中以 OpenCC 轉換，再套兩岸詞彙表；日文、韓文以英文原文為底，參考繁中與詞彙表，由 AI 批次翻譯；發行時標註為機器翻譯 |
| 字型 | 簡中、日文、韓文用 GNU Unifont（發行包現有的 `cok-unifont.golemfnt` 產線，OFL-1.1／GPL 雙授權） |
| 切換 | 啟動器首次選擇並記住；切換列「中文」鈕改成循環切換各語言與原版，另有快捷鍵 |

## 1. 現況

| 項目 | 現況 | 出處 |
|---|---|---|
| 譯文表 | `text/*.zh-TW.tsv`：ECL 2,284、UI 889、名稱 1,040、手札 145、地圖攻略點 157、攻略 26、宿主文字 33、組句模板 94 | `wc -l` |
| 比對鍵 | 事件表（`*-events.tsv`）只存英文原文的長度與 SHA-256，原文不進 Git；英文原文抽取在 `workplace/text/*-source.tsv` | AGENTS 第 2 節 |
| 字型 | 本機用倚天（`workplace/font/cok-eten.golemfnt`）；發行包由 `tools/eten_font.py --unifont` 以正式譯文的字集從 Unifont 產生 `cok-unifont.golemfnt` | `tools/package.sh:90` |
| 宿主讀檔 | `cok-play` 固定讀 `host-text.zh-TW.tsv`；切換列只有「中文／原版」兩態（`toolbarState.chinese`） | `cmd/cok-play/main.go:508`、`toolbar.go` |
| 英文手札原文 | `Champions_of_Krynn_Adventurers_Journal.pdf`（使用者提供）與轉出的 `workplace/reference-src/txt/champions-journal.txt`，不進 Git | docs/re/002 |

## 2. 語系架構（#17）

狀態：實作完成（dosgolem 分支 `ml-17`），收據 `tools/g9-lang-receipt.sh`。

### 2.1 檔案

| 內容 | 檔名 | 繁中 |
|---|---|---|
| 介面、ECL、名稱、手札、攻略、地圖攻略點、前端文字 | `<表>.<代碼>.tsv` | 沿用 `*.zh-TW.tsv` |
| 組句模板、選單母串 | `composed-templates.<代碼>.tsv`、`menu-masters.<代碼>.tsv` | 沿用無後綴的舊檔名，輸出與多語系前相同 |
| 地圖名稱 | `map-titles.<代碼>.tsv`（map、title），取代 `map-geo.tsv` 的繁中名；沒有的地圖顯示 `map.unknown` | 用 `map-geo.tsv` |
| 玩家名音譯（規格 013） | `translit-<代碼>/` 子目錄，檔名同繁中；沒有就不音譯 | `text/` 本身 |
| 切換列語言名稱 | `host-text.<代碼>.tsv` 的 `lang.name`（載入兩種以上語言時必填） | 同 |
| 手札編號補字（「 51 則。」） | `host-text` 的 `journal.ref`（一個 `%d`、一個 `%s`）與 `journal.refdot` | 缺省為「 %d 則%s」「。」 |
| 字型 | 發行包 `font/cok-<代碼>.golemfnt`，由該語言的全部 `*.<代碼>.tsv` 字集從 Unifont 建（`tools/package.sh`）；開發用 `cok-play -lang-fonts 代碼=路徑` | `-font` 或發行包預設字型 |

語言清單取自 `ui-text.<代碼>.tsv`，繁中排第一。沒有字型的語言略過；表不完整、切換列放不進 320 像素時拒絕啟動。

### 2.2 執行期切換

- 單一 `Live` 搭配可替換的語言包（`apps/cok/lang.go` 的 `LangPack`、`Live.SetLang`）。不為每種語言各開一個 `Live`：配樂選曲、手札自動開啟、地圖取樣都掛在攔截事件上，重複送會讓短曲重播。
- 換語言時，畫面上已有的疊字以保留的原文就地重譯：路徑 A 每筆疊字記住查表用的原文；路徑 B 每頁記住依序排入的原文與換行條件，以新語言重排，已被原版蓋掉的列維持移除；路徑 C 以保留的原文格子重組，不觸發配樂的選單事件。新譯文放不下的疊字移除，露出原版。查驗題三列以新格式重寫。
- 隊伍欄學過的隊員名交給新語言重學（換語言時隊伍欄不一定重印）。
- 面板（手札、攻略、地圖）只換內容與格式；開關、頁碼、已讀、走過的格子、攻略位置不變，狀態檔語言無關。
- 原版記憶體、畫面緩衝與存檔不受影響。

### 2.3 操作

- `cok-play -lang 代碼` 指定起始語言；沒指定時用上次的選擇（前端資料目錄 `cok-lang.txt`），再沒有就是繁中。
- 載入兩種以上語言時：F8 換下一種語言；切換列的語言項顯示目前語言名稱，按下依序換下一種，最後一種之後回原版，原版時再按回第一種。F12 照舊在譯文與原版之間切換。只有一種語言時不保留 F8，切換列與多語系前相同。
- 自動模式腳本動作 `lang` 等同 F8（收據用）。
- 啟動器第一次執行時選語言的畫面待做（目前以切換列與記住選擇代替）。

### 2.4 驗收

`tools/g9-lang-receipt.sh`：cok-play 自動模式冷開機到建立隊伍主選單，比對改動前與多語系版本，並以只在 workplace 產生的假語言 `zz` 驗證切換。要求六組記憶體與原版畫面相同；多語系版本與改動前畫面相同；`zz` 與繁中不同；主選單出現後換到 `zz` 與一開始就是 `zz` 相同；再換回繁中與繁中相同。

每種正式語言另跑 G5 七場景同狀態 A/B（範圍外差異 0）；`tools/term-check.sh`、行寬與字型覆蓋檢查逐語言執行（隨 #19、#20 補上）。

### 2.5 字形

發行包的 Unifont 來源目前只有台灣字形版 `unifont_t`，涵蓋簡體字、假名與韓文，但漢字是台灣寫法。簡中、日文要改用各地字形時，從 Unifont 原始碼建對應的 hex，隨 #19、#20 處理。

## 3. 英文版（#18）

### 3.1 字型

- 原版文字格 8×8，欄 c、列 r 的左上角在 (8c, 8r)；字型只有大寫字模，畫面全是大寫（本作 G2 主選單截圖 `workplace/probe/g2-receipt/menu-a.png`，已證實）。
- 字串本身的大小寫分兩類（實機盤點 `workplace/probe/throtl1/inv.tsv`，已證實）：介面與戰鬥訊息是大小寫混排（`Move/Attack, Move Left = 11`、`View Take Pool Exit`）；ECL 劇情文字在原始資料裡就是全大寫（`THIS IS A WELL FURNISHED STUDY.`）。使用者決定（2026-10-20）保留全大寫，只換字型；介面文字照原本的大小寫。
- 原版字格是正方形，一般等寬字型字寬約為字高六成，放進格子後字距偏鬆；候選包含原寬、水平拉寬填滿字格、方形點陣字與比例字四種排法。對照圖 `workplace/fonts-en/compare.png`（產生腳本 `workplace/fonts-en/compare.py`，字型取自 google/fonts 的 OFL 目錄，授權檔同目錄）。
- 新字型維持一字一格的等寬排版，原版的欄位對齊與游標位置才不會跑掉。
- 字型檔：`tools/en_font.py` 由 Courier Prime Bold（google/fonts `ofl/courierprime`，SHA-256 前綴 `ff1f38786c849d1c`）產生可列印 ASCII 95 字。每字先畫在 (格寬×2/3)×格高的窄格，上伸到下伸整段置中，再水平拉成方格，以 50% 門檻轉 1 bit。`cok-en.golemfnt` 16×16 給 2 倍，`cok-en-3x.golemfnt` 24×24 給 3 倍（cok-play、cok-overlay-shot 在同目錄找 `<名稱>-3x.golemfnt` 當 `Scaled[3]`，不從 16×16 放大）。
- 原版字串若出現可列印 ASCII 以外的字元，會以缺字回報，待實機盤點補字模。

### 3.2 覆繪方式

- 英文原文就是比對鍵，所以 `en` 不需要譯文表：語言包的查表把原文當譯文（`identityLookup`），三條路徑照常產生疊字，清除與座標不變（規格 001–004）。路徑 B 給空的 ECL catalog，讓綁定流程照常建立路徑 B。
- 全 ASCII 字串本來就不做半形併字（規格 005），一字一格；組句與音譯（規格 004、013）不作用。
- 路徑 C 的熱鍵字母：原版選單詞的熱鍵字母顏色不同。英文詞整詞重畫後，熱鍵字母在原位置以原熱鍵色另畫一格。
- 切換列：英文字型沒有半形組合字，英文版的切換列改用預設字型（`-font`），ASCII 也兩字併一格，標籤取 `host-text.en.tsv` 的縮寫。
- 地圖名稱 `map-titles.en.tsv`：8 張由術語表反查英文；其餘 7 張的中文名是本專案自訂，英文照中文意譯，Southern Outpost、Ogre Base 依手冊製作名單的寫法。地圖攻略點出自中文攻略，英文版不顯示。
- 字寬與跨行：英文與原文逐字相同，疊字範圍與原版墨跡一致。路徑 B 依原版接續規則重排，若排法與原版不同，疊字仍遮滿起始列到視窗底列（規格 002 §2.4）。

### 3.3 手札與酒店傳聞

- 原版提示「請讀手札第 N 則」時，開啟與規格 007 相同的面板，內容是英文 Adventurer's Journal 第 N 則原文。
- 逐字稿依 `workplace/journal/PROMPT-en.md` 從頁面圖轉錄（PDF 為主，兩份 OCR 為輔），每頁或每批一個 TSV 放 `workplace/journal/en/`。`tools/journal_en_build.py` 合併成 `journal.en.tsv`（手札 1–88、酒店傳聞 1001–1057，缺漏或重複就失敗），放 `workplace/journal/`，不進 Git。本機開發另複製到 `text/journal.en.tsv`（.gitignore）；打包時 `tools/package.sh` 放進發行包（CONTEXT「可散布 patch 包維持全放，只私下分享」）。
- 英文段落斷行：拉丁詞不拆，換行後的行首空白略過（只作用於全 ASCII 段落，繁中排版不變）。
- 手札編號補字：英文原文已有空白，`journal.ref` 為「%d%s」、`journal.refdot` 為「.」。
- 攻略面板出自《軟體世界》中文攻略，英文版沒有攻略表時面板停用，切換列的攻略項顯示停用；換回有攻略的語言時還原同一個面板與位置。

## 4. 簡體中文（#19）

狀態：譯文表已產生；待 G5 簡中 A/B 與實機目視。

- 入口 `tools/zhcn-convert.sh`（映像 `cok-opencc:r1`，`tools/docker/opencc.Dockerfile`，OpenCC 1.4.1）；`CHECK=1` 檢查 `text/` 裡的簡中表是否與繁中同步。工具 `tools/zhcn_convert.py`。
- 一般文字用 OpenCC `tw2sp`（台灣正體到大陸簡體，含詞彙轉換，例：檢視→查看、搖桿開啟→摇杆打开）。
- 專有名詞（`glossary.tsv` 的 person、place、race、monster、class，及 `translit-names.tsv` 的中文名，共 377 個）先換成佔位符，單獨用 `t2s` 逐字轉簡體再放回，譯名用字只轉字形。
- 轉換後套 `text/zhcn-fixes.tsv`（from、to、note）逐條修正。目前一條：OpenCC 把「防寫保護」轉成「写保护保护」。
- 每列 `{…}` 佔位符與 `\n` 數量不變，否則工具中止；轉換後變長的列寫到 `workplace/zhcn/longer.tsv` 審查（目前 1 列：「新檔名：」→「新文件名：」，原文格數足夠）。
- 產出：`*.zh-CN.tsv`（7 份正式表）、`composed-templates.zh-CN.tsv`、`menu-masters.zh-CN.tsv`、`map-titles.zh-CN.tsv`、`translit-zh-CN/`（音譯表逐字轉簡體，發音詞典沿用上一層 `cmudict/`）。`host-text.zh-CN.tsv` 的 `lang.name`、`journal.ref`、`journal.refdot` 由工具指定（简中、「 %d 则%s」、「。」）。
- 字型：發行包由 `tools/package.sh` 以簡中表字集從 `unifont_t` 建 `font/cok-zh-CN.golemfnt`；2,341 字全部有字模。開發用同一指令輸出到 `workplace/font/cok-zh-CN.golemfnt`。
- 引號沿用「」。

## 5. 日文與韓文（#20）

狀態：全量譯文已合併（`text/*.ja.tsv`、`text/*.ko.tsv`）；G5 七場景日韓各自通過。

- 術語表：`workplace/tr-jako/glossary-{ja,ko}.tsv`（en、target、basis、note，736 詞）。basis 為 official 的需有可引用出處：日文 57 詞（日文維基百科引用的富士見書房／KADOKAWA 譯名、WotC 日文頁），韓文 208 詞（D&D 韓文官方網站 dndkr.com 的世界觀專欄、5 版法術卡與基本規則）；其餘為音譯、意譯或介面用語。
- 試譯 `workplace/tr-jako/trial-{ja,ko}.tsv` 經使用者確認文風（§6）後全量翻譯。
- 批次：`workplace/tr-jako/batches/bNN.tsv` 由原文抽取檔建立（key、original、prev、next、zh），只收繁中有正式譯文的 3,173 列，130 列一批共 25 批；子代理（sonnet）依 `workplace/tr-jako/PROMPT.md` 輸出 `done-{ja,ko}/`。組句模板、選單母串、前端文字、地圖名、手札（英文逐字稿為底）、攻略與地圖攻略點（繁中為底）另由輔助批次產出 `aux-{ja,ko}/`。
- 合併 `tools/jako_merge.py --lang {ja,ko}`：每批 key 與順序同輸入；佔位符與繁中一致；介面字串格數不超過原文（硬限制），ECL 劇情超過只列警告（路徑 B 在視窗內換行）；選單母串每詞不超過原詞格數；韓文譯文不得混入漢字或假名、日文不得混入韓文；名稱表依 `name_catalog.py` 的規則由術語表產生。
- 手札編號補字：日文譯文收在「手記第」，`journal.ref`「%d項%s」；韓文收在「…제」，`journal.ref`「%d항%s」。
- 玩家名不音譯（沒有 `translit-{ja,ko}/`），隊伍欄名字顯示原版英文。
- 版面：韓文用半形標點，接在韓文音節後的 . , ! ?（含半形併字的組合字）不放行首（路徑 B 排版與面板斷行共用 `closeAfter`）。底列提示「Press <enter>/<return> to continue」的譯文詞數必須和原文不同，才會走整列對應（規格 003 §2.2），韓文寫成三個詞。
- 字型：Unifont `unifont_t` 子集，日文 1,693 字、韓文 1,097 字，全部有字模；漢字是台灣字形（§2.5）。
- 發行包與 README 註明日文、韓文為機器翻譯、未經母語者校對。

## 6. 已定案的範圍

| 項目 | 決定（2026-10-20，使用者回答） |
|---|---|
| 攻略面板、地圖攻略點 | 簡中由轉換產生；日文、韓文從繁中 AI 翻譯；英文不提供 |
| 手札、酒店傳聞 | 英文顯示逐字稿；日文、韓文從英文逐字稿 AI 翻譯，譯文與繁中手札一樣進 `text/journal.<代碼>.tsv`；英文逐字稿本身不進 Git |
| 日韓文風 | 依試譯（`workplace/tr-jako/trial-*.tsv`）：日文敘述常體過去式、隊伍稱「あなたたち」；韓文敘述 해라체、隊伍稱「일행」；NPC 依角色 |

## 7. 排程

| 順序 | Issue | 內容 | 依賴 |
|---|---|---|---|
| 1 | #17 | 語系架構、啟動器與切換列、逐語言檢查腳本 | 無 |
| 2 | #18 | 英文字型選定、`en` 覆繪、英文手札面板 | #17 |
| 3 | #19 | 簡中轉換與詞彙表 | #17 |
| 4 | #20 | 日文、韓文術語表、試譯、全量、版面檢查 | #17、#18（英文原文面板與行寬工具） |

HD（#14）與推廣影片（#15）照原排程進行；多語系完成後重打三平台完整版。

## 8. 驗收

- 每種語言：譯文表檢查（key、欄數、控制碼、placeholder、漏譯、孤兒、行寬、字型覆蓋）通過；G5 七場景 A/B 通過；正常玩家路徑 g3 走一次。
- `zh-TW` 回歸：G5 雜湊與改動前相同。
- 切換：同一狀態循環切換全部語言與原版，原版記憶體與畫面緩衝不變。
