# 200 EOB3 原版 oracle 轉接層

狀態：**READY**（主選單與選單輸入已驗證；主選單之後的遊戲內容未驗證）
日期：2026-10-04
前置：[`197-cpu386-real-mode-subset.md`](197-cpu386-real-mode-subset.md)、
[`198-dos-relative-directories.md`](198-dos-relative-directories.md)、
[`199-psp-block-resize-boundary.md`](199-psp-block-resize-boundary.md)、
[`201-vga-unchained-256.md`](201-vga-unchained-256.md)、
[`202-bios-int09-default-handler.md`](202-bios-int09-default-handler.md)、
[`005-oracle-api.md`](005-oracle-api.md)、[`194-stubseg-font-collision-and-program-path.md`](194-stubseg-font-collision-and-program-path.md)

---

## 1. 目的

讓 remake 專案能把原版《魔眼殺機三》當成對拍的 oracle：以 `AESOP.EXE eye start`
無頭、決定性地執行，在固定的指令數注入滑鼠與鍵盤，擷取 320×200 的畫面
（色號、調色盤與 PNG）。

套件 `apps/eob3` 不含任何原版檔案；`-exe` 與 `-root` 由使用者提供，唯讀。
輸出（PNG、色號、調色盤）是原版畫面的衍生物，只放在輸出目錄，不得提交或散布。

## 2. 介面

### 2.1 `apps/eob3`

| 符號 | 內容 |
|---|---|
| `ProgramPath` | `C:\EOB3\AESOP.EXE`。環境區塊的 argv[0]，`dos.New` 由它推出 `RootDir`（`EOB3`）。給錯目錄名，第二層以下的相對路徑找不到 |
| `CommandTail` | `" eye start"`（含前導空白，與 `EYE.BAT` 的 `aesop eye start` 相同） |
| `MarkerOffset` | `0x4F0`，DOS 低記憶體的階段標記位址（§4） |
| `Load(exe, root, scratch)` | 載入並設好環境：命令列、程式路徑、暫存層、AdLib 存在、`StrictPSPGrowth`、`StrictChdir` |
| `Keys`、`KeyByName` | IBM PC/AT Set 1 的 make 掃描碼表，名字小寫 |

`Load` 設定的環境開關：

| 開關 | 值 | 理由 |
|---|---|---|
| AdLib（OPL2）偵測存在、輸出丟棄 | 開 | `ADLIB.ADV` 偵測失敗時程式印「Music initialization failed」並等一個鍵，停在文字模式（`hardware-spec approximation`，不模擬音訊） |
| `StrictPSPGrowth` | 開 | Borland exec 的載入器 stub 不能蓋到環境區塊（`199` §3） |
| `StrictChdir` | 開 | `launch` 會 chdir 到不存在的目錄名（`198` §5） |

可寫暫存層 `scratch`：存檔、程式自建的檔落在這裡，`root` 永遠唯讀。
**每次執行要給一個空的目錄**，暫存層會蓋過 `root`，殘留檔案會改變結果。

### 2.2 指令列 `apps/eob3/cmd/oracle`

```
oracle -exe <AESOP.EXE> -root <EOB3 資料夾> -out <輸出目錄> -script '<腳本>' [-scratch <空目錄>] [-budget N]
```

| 旗標 | 內容 |
|---|---|
| `-exe`、`-root`、`-out` | 必填 |
| `-scratch` | 可寫暫存層，必須是空的，預設 `<out>/scratch`；不是空的就拒絕執行 |
| `-script` | 動作用 `;` 或換行分隔，見 §3 |
| `-budget` | 單一動作的指令數上限，預設 2,000,000,000 |

輸出目錄的內容：每個 `shot` 動作一組 `<名>.png`、`<名>.idx`、`<名>.pal`，以及 `receipt.json`。

### 2.3 腳本動作

| 動作 | 意義 |
|---|---|
| `run=N` | 再跑 N 道指令 |
| `until=N` | 跑到絕對步數 N（已經超過就報錯） |
| `idle=N` | 畫面連續 N 道指令不變。**這是猜**：片頭動畫有靜止的段落，不要拿它當載入完成的證據 |
| `opened=NAME` | 程式開了這個檔（比 basename，不分大小寫） |
| `key=NAME` | 以硬體 IRQ1 送一次按下加放開（名稱見 `Keys`） |
| `move=X,Y` | 游標移到像素座標（320×200 邏輯座標，水平倍率 2） |
| `click=X,Y[,r]` | 移動、按下、按住、放開、等畫面回應；`r` 為右鍵。點了畫面沒變不算錯誤。**一次點擊約消耗 600 萬道指令** |
| `marker=INTR` | 在 `0000:04F0` 寫階段標記（INTR、CINE、VICT、CHGN），見 §4 |
| `shot=NAME` | 存 `NAME.png`、`NAME.idx`（320×200 色號，64,000 bytes）、`NAME.pal`（768 bytes） |
| `text=NAME` | 存 `NAME.txt`（文字模式畫面，載入失敗時用） |
| `mark=LABEL` | 印一行目前步數、`CS:IP`、視訊模式 |

`NAME.pal` 是 8 位元 RGB（DAC 的 6 位元值以 `v<<2 | v>>4` 擴成 8 位元），不是 DAC 原始值。
`NAME.png` 用同一份調色盤上色。

`receipt.json` 不含任何路徑，只有：執行檔與 `EYE.RES` 的 SHA-256、腳本全文、總步數、
每張 `shot` 的步數與視訊模式與色號與調色盤的 SHA-256、開過的檔、`exec` 紀錄（`exit=255` 表示還沒結束）、
沒實作的服務清單、主控台輸出、最後的 `CS:IP` 與視訊模式。

## 3. 範例

```
# 自然路徑，在片頭按 Esc 跳過（約 1 億道指令內到主選單）
until=30000000;key=esc;until=100000000;shot=menu

# 不按任何鍵等片頭播完（約 15 億道指令）
until=1500000000;shot=menu

# 到主選單後，點「傳送二代隊員」並截圖
until=30000000;key=esc;until=100000000;shot=menu;move=150,134;run=2500000;click=150,134;run=3000000;shot=dialog
```

單機速度（容器限 2 CPU）：約每秒 1,000 萬至 2,000 萬道指令；不按鍵的自然路徑跑 20 億道指令約 1 分 50 秒，
按 Esc 跳過片頭到第 1 億道指令約 5 至 10 秒。

## 4. 啟動鏈與階段標記

`EYE.RES` 序號 2388 的 `start` 腳本一開頭 `peekmem(0x4F0)`，依 32 位元標記分支
（已證實：腳本反組譯 `work/eob3-p0/disasm/2388.txt`，與執行軌跡）：

| 標記 | 走向 |
|---|---|
| 0（預設） | 建初始檔、初始化音效、寫 `INTR`、`launch cine.exe` 播片頭 |
| `INTR` | 片頭播完的那一支，進主選單 |
| `CINE`、`VICT`、`CHGN` | 其他階段（結局、建角等），未走過 |

標記是 32 位元小端，`marker=INTR` 寫入的位元組是 `52 54 4E 49`。

### 4.1 自然路徑（已證實）

```
AESOP.EXE（標記 0）
  → launch cine.exe：chdir（失敗，見 198 §5）、AH=4Bh AL=03h 載入 CINE.EXE 覆蓋自己
  → CINE.EXE 播片頭（INTRO.GFF、XVA.DRV、INTRO.PAT）
  → 播完或按 Esc：CINE.EXE launch aesop.exe（同一個載入器，沿用環境區塊）
  → AESOP.EXE（標記 INTR）→ 主選單
```

`exec` 紀錄的名稱是 `C:\EOB3\CINE.EXE`、`C:\EOB3\AESOP.EXE`，兩個都是 `AH=4Bh AL=03h` 的覆蓋式載入
（`exit=255`＝沒有返回）。

### 4.2 捷徑 `marker=INTR`（已證實可到同一個畫面）

跳過 `launch cine.exe`，第 1,000 萬道指令時已是主選單（沒有量更早的步數）。
主選單畫面色號雜湊與自然路徑相同。

**兩條路徑的差異**：自然路徑讀過 `ITEMS_00.BIN`、`LVL01_00.BIN` 到 `LVL14_00.BIN`，
並在暫存層寫了 `INTRO.GFF`、`INTRO.PAT`、`TEXTPAT.FNT`；捷徑只寫 `TEXTPAT.FNT`。
捷徑會不會讓之後的遊戲狀態與自然路徑不同，沒有量到。要對拍主選單之後的內容時，
**優先用自然路徑加 Esc**（約 1 億道指令，與捷徑同樣是秒級）。

### 4.3 載入器 stub 與 exec 鏈

Borland C++ 的 exec 載入碼（AESOP 與 CINE 的 stub 逐 byte 相同；`_LoadProg` 是依 Borland 的
spawn 實作推定的名稱）：

1. `AH=4Ah BX=FFFFh` 探測，把自己的區塊調到探出的上限。AESOP 要自己配環境區塊，
   所以扣掉環境區塊需要的段數（探出 `9EFFh`，調到 `9EFAh`）；CINE 沿用繼承來的環境區塊，直接調到上限。
2. 需要時 `AH=48h` 要環境區塊（AESOP 要，CINE 不要）。
3. 把 `160h` bytes 的 stub 複製到區塊尾端往下 `17h` 段。
4. `AH=4Bh AL=03h` 把新程式載到 `0110:0`（覆蓋式，PSP 仍是 `0100`），stub 換成新程式的堆疊，
   `jmp far` 進去。

三個 `dosgolem` 的缺口（`197`、`198`、`199` 與 §2.1 的開關）都在這條鏈上被踩到。

## 5. 驗證

### 5.1 單元與回歸

- `internal/cpu`（含 SingleStepTests 8088 語料）、`internal/dos`、`internal/machine`、`oracle`、
  `cmd/probe`、`apps/*` 全部通過。指令與命令見 `docs/re/eob3-dosgolem-oracle.md`。

### 5.2 實跑

| 項目 | 結果 | 等級 |
|---|---|---|
| CPU 偵測、開機、`launch` 鏈、片頭到主選單 | 自然路徑（不按鍵）第 14 億至 15 億道指令之間到主選單；按 Esc 跳過片頭則第 1 億道指令時已在主選單 | 已證實 |
| 主選單畫面 | 320×200、模式 13h 非鏈結；標題「魔眼殺機 III」、副標「血戰札諾爾」、五個選項（片頭故事介紹、繼續冒險任務、創建新隊伍、傳送二代隊員、結束遊戲）與「中文版」標誌，人眼檢視正常 | 已證實 |
| 兩條路徑的主選單畫面色號雜湊與調色盤雜湊 | 相同（色號 `2cbeca95…`、調色盤 `d2e087fc…`） | 已證實 |
| 滑鼠 | 游標移到選項上，畫面變化（選項高亮）；`click=150,134` 開啟「傳送二代隊員」的說明框 | 已證實 |
| 鍵盤 | 說明框中 Enter 關閉、主選單 Esc 離開（回文字模式）；片頭中 Esc 跳過。其他鍵（上下、空白、數字、F1、Tab）沒有逐一驗證畫面效果 | 部分證實 |
| 決定性 | 同一份腳本跑兩次，`receipt.json` 與每個 `.idx`、`.pal` 逐 byte 相同 | 已證實（同一台主機、同一個建置） |

### 5.3 決定性的來源

腳本是步數驅動，不讀牆上時鐘：

- 計時器中斷（IRQ0）、鍵盤中斷（IRQ1）與滑鼠事件都由指令數排程。
- DOS 的時間與日期固定為零（`Time{}`）。
- 模擬器沒有使用主機亂數或主機時間；程式自己若有亂數，狀態來自它的種子與指令流
  （沒有逐一查證 AESOP 的亂數來源）。
- 輸入注入的時機是腳本裡的絕對或相對步數。

**沒有驗證的**：不同主機、不同 Go 版本、不同 CPU 之間的逐位元一致（程式不含浮點，預期一致）。
IRQ 與指令數的比值（`190`、`191`）是校準值，不是量到的原版時序：
遊戲內經過的「秒」與真機不同，對拍要以畫面內容比對，不要拿步數換算成時間。

## 6. 證據等級與已知限制

| 項目 | 等級 |
|---|---|
| 啟動鏈、階段標記的分支結構 | 已證實 |
| 兩條路徑到達相同主選單畫面 | 已證實 |
| 主選單之後的行為（新隊伍、載入存檔、建角 `CHGEN.EXE`、遊戲內畫面） | 未驗證，沒有走過 |
| `StrictChdir` 的推論（原版預期 chdir 失敗） | 強推論（輸出變正確，沒有在真 DOS 上量過） |
| 音訊 | 不模擬：AdLib 偵測回「存在」，寫入被丟棄（`hardware-spec approximation`），也不模擬 SB／PIT／DMA 的逐週期行為 |
| `int 10h AH=08 AL=18` | 被叫到但沒實作（每次啟動一次，不阻塞，主選單畫面正確） |
| `EOB3FIX.EXE` 與 `PLAY.BAT` | 沒有測，只測 `aesop eye start` |

主選單之後的流程若 AESOP 走到沒實作的 `int`、記憶體語意或 `0F xx`，
`receipt.json` 的「沒實作的服務清單」與錯誤訊息會指出（CPU 未支援的指令一律失敗即關閉，見 `197` §2.2）。
