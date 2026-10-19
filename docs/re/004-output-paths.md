# 004 輸出路徑盤點

Issue：#4（G3）。基線見 [003](003-dosgolem-startup.md)。位址為 dosgolem 執行期 `段:位移`，由 `start-200m.state`
（`c14e2751…`）展開。幽靈騎士的對應結論引自其 `docs/re/003-output-paths.md`，在本作重驗前只當假說。

## resident 印字程序

三個進入點都在 `START.EXE`，以幽靈騎士 adapter 的位元組簽章找到（`tools/re/sigscan.py`），再用 dosgolem `-find`
與 `-ip-log` 取得執行期位址。

| 程序 | 本作檔案位置 | 執行期 | 幽靈騎士 | 等級 |
|---|---|---|---|---|
| 路徑 A Pascal 字串印字 | `START.EXE` 0x552A | `0631:0353` | 0x51C2／`05F4:036A` | 已證實：主選單 5 行、反白行、`Choose a FUNCTION ` 經過此處 |
| 逐字程序 | 0x53B6 | `0631:01DF` | 0x504D／`05F4:01F5` | 已證實（`-ip-log` 進入 154 次） |
| 路徑 B 文字視窗 | 0x5683 | `0631:04AC` | 0x5303／`05F4:04AB` | 位置已證實；呼叫參數待讀 |

- 路徑 A 簽章前 23 bytes 在檔內有兩處（0x552A、0x69DB），第 28 byte 分辨：0x552A 為 `9A 51 00`（遠呼叫），0x69DB 為 `8A 86`。
  幽靈騎士為 `9A 58 00`，adapter 的簽章已改用本作位移。
- 路徑 A 參數與幽靈騎士相同：`offset seg bg fg 列 欄`。例：`0 19535 0 10 12 9` → `Create New Character`。
- 路徑 B 函式與幽靈騎士逐 byte 對齊（`tools/re/fn_diff.py`），游標欄、列 `DS:B43A／B43B` 的每一處在本作都是 `DS:6134／6135`
  （強推論，靜態對齊）。函式後段（+0x22A 起）有結構差異：幽靈騎士讀 `DS:43F1`–`43F5` 的視窗參數，本作在同位置用常數，
  視窗與斷行行為要另驗。
- 資料段：冷啟動讀 `krynn.cfg` 時 `DS=0CD6`（幽靈騎士 `0CDE`）。

## 覆繪初驗

`apps/cok`（由 `apps/dkk` 複製，只換上述常數）載入繼承自幽靈騎士的介面譯文，主選單 5 則全部覆繪，範圍外差異 0
（`cok-overlay-shot`，#299,000,000）。`Choose a FUNCTION ` 未命中，屬底列，歸路徑 C 待查。

## 診斷路徑：預建隊伍到手冊查驗題

正式收據仍要走建角。這條路徑只供盤點：

1. scratch 層放入原版 `SAVE/` 的預建隊伍（比照 `COPYSAVE.BAT` 安裝到存檔目錄）。dosgolem 只依檔名在根目錄找，原版放在子目錄時讀不到。
2. 主選單 `Down Down Enter` → `LOAD WHICH GAME:` → `A` 讀入 `SAVGAMA.DAT` 與 `CHRDATA1–6`，6 人隊伍。
3. 隊伍選單移到 `BEGIN ADVENTURING`，Enter 後第一個畫面是手冊查驗題。

| 觀察 | 等級 |
|---|---|
| 查驗題以路徑 A 分段印出：` what is word # `、`6    `、` of journal entry # 30     `、` (journal, page 32)?      `；`Enter ` 由 resident `0631:077A` 印出。呼叫端執行期段 `2DC9`，結構同幽靈騎士 | 已證實 |
| 答案 `TO`：手札第 30 則正文第 6 個字（Stage two is currently coming to fruition），字序從正文第一個字起算、不含標題。作答後進入開場劇情 | 已證實（1 題） |
| 開場劇情畫面 `AT THE INN OF THE LAST HOME IN SOLACE, …` 底列 `PRESS <ENTER>/<RETURN> TO CONTINUE` | 已證實（畫面） |
| 遊戲開檔時以讀寫模式開部分 DAX，dosgolem 依規格 009 把它們複製到 scratch；收據每次要清空 scratch | 已證實（scratch 出現 `ECL2.dax` 等副本） |

## 路徑 C：底列水平選單（已證實）

營地選單繪製時呼叫逐字程序 `0631:01DF` 的 overlay 呼叫端（執行期段 `33E7`），返回位址前是 `9A DF 01 <段>`（幽靈騎士 `9A F5 01`），
返回位址後的 bytes 分辨四種呼叫：

| 種類 | 呼叫端 | 返回位址後 | 幽靈騎士 |
|---|---|---|---|
| selected | `33E7:028F` | `E9 95 00` | `E9 96 00` |
| hotkey | `33E7:02E9` | `EB 3C` | `EB 3C` |
| normal | `33E7:0327` | `8A 46 FF 3A 46 FE` | `FE 46 FF` |
| pad | `33E7:0397` | `9A 30 00 <段>`（只比前 3 bytes） | `C4 7E FC 26 8B 45 2A` |

城鎮選單 `INN HALL COMMANDANT ARMOURY NEXT LEAVE` 的每個詞是獨立的 ECL 字串（`ecl.1.17.01735`–`01768`），
印出時改成首字大寫；介面查表找不到時轉大寫查 ECL catalog。營地與城鎮選單 6 詞全部覆繪。

## 手札引用（已證實）

開場劇情以兩則路徑 B 字串印出 `… AND YOU RECORD IT IN JOURNAL ENTRY ` 與 `51.`。ECL 原文另有整句含編號的寫法
（`RECORD IT AS JOURNAL ENTRY 29.`）與酒店傳聞（`YOU HEAR TAVERN TALE `、`YOU OVERHEAR TAVERN TALE NUMBER `）。
adapter 依「上一則結尾＋這一則開頭的編號」打開手札面板（酒店傳聞用 1000+N），單獨的 `51.` 補成「 51 則。」。
實機：第 51 則自動彈出中文面板。

## 下一步

- 路徑 B 呼叫參數、視窗矩形與換頁。
- 路徑 C 入口（底列）。
- 戰鬥、營地、商店等畫面的呼叫端盤點（用 `cok-play` 的 inventory 輸出）。
- 查驗題再驗數題，建立手札字表（G6）。
