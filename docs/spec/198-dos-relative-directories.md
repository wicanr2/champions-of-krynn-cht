# 198 DOS 相對子目錄解析

狀態：**READY**
日期：2026-10-04
前置：[`004-dos-bios-services.md`](004-dos-bios-services.md)（`AH=3Bh／47h／4Eh`）、
[`009-scratch-writes.md`](009-scratch-writes.md)（可寫暫存層）、
[`194-stubseg-font-collision-and-program-path.md`](194-stubseg-font-collision-and-program-path.md)（`ProgramPath`）
後續：[`200-eob3-oracle-adapter.md`](200-eob3-oracle-adapter.md)

---

## 1. 問題

`resolve` 原本**只認 basename**。這條規則是為了多磁片殘留路徑（遊戲組出
`A:\<垃圾>\DATA.PAK`，目錄部分對不上硬碟安裝版）。

`AESOP.EXE` 用的是**真實子目錄**：`SAVEGAME\ITEMS_00.BIN`、`CHARGEN\CHGEN.EXE`、
`CINE\…`。只看 basename 時：

- 同名檔在不同子目錄會互相蓋掉（暫存層攤平成 `scratch/<basename>`）。
- `.BIN` 找不到，程式就從 `.TXT` 翻譯，翻譯寫檔也落在錯的位置，開機以離開碼 1 結束，
  主控台只印 `Translating ...`。
- 目前目錄預設是 `RICH2`（既有程式的硬編碼），與 `C:\EOB3\AESOP.EXE` 對不上。

## 2. 規則

### 2.1 兩個目錄變數

| 變數 | 意義 |
|---|---|
| `Dir` | DOS 的目前目錄（`AH=47h` 回的），從磁碟機根起算，不含磁碟機與開頭的 `\` |
| `RootDir` | `Root`（主機上的原版目錄）在 DOS 樹上的位置，`C:\EOB3` 對應 `EOB3` |

預設兩者相同。`dos.New` 看到 `Machine.ProgramPath`（例如 `C:\EOB3\AESOP.EXE`）時，
兩者都設成程式所在目錄（`EOB3`）。沒給 `ProgramPath` 時維持原預設，舊程式不受影響。
`chdir` 之後 `Dir` 離開 `RootDir`，相對路徑才跟著換目錄。

### 2.2 路徑解析

1. 去掉磁碟機字母。`\` 開頭是絕對路徑，否則接在 `Dir` 後面。`.` 丟掉，`..` 退一層，
   退過根目錄停在根目錄（與 DOS 同）。
2. 目錄元件大寫。去掉 `RootDir` 前綴，得到「從 Root 起算」的元件。
3. 若有目錄元件，**而且那串目錄在 `Root` 或 `Scratch` 裡真的存在**，就走這一條：
   `Scratch` 先、`Root` 後（暫存層蓋過原版），逐層大小寫不分地找子目錄，
   再用 8.3 截斷後的名字比一次，最後找檔案。
4. 其他情況（沒有目錄元件、目錄不存在、檔案不在其中）退回 **basename 規則**。
   junk 路徑因此不受影響。

### 2.3 寫入

- 路徑帶的目錄在 `Root` 或 `Scratch` 真的存在時，寫進暫存層的同名子目錄
  （`SAVEGAME\X.BIN` 寫成 `scratch/SAVEGAME/X.BIN`），必要時建立中間目錄。
- 否則照舊攤平成 `scratch/<basename>`。
- 讀與寫用同一套判準，寫進去的檔下一次一定讀得回來。
- `mkdir`／`rmdir` 同理：父目錄存在時用巢狀位置，否則攤平成大寫 basename。

### 2.4 其他呼叫

- `AH=43h`（取屬性）：存在的目錄回 `0x10`，檔案回 `0x20`，不存在回 CF。
- `AH=3Bh`（chdir）：**預設一律成功**（既有行為：回失敗會讓程式以為安裝不完整），不驗證目錄存在；
  `StrictChdir` 打開時改為目標不存在就失敗（§5）。
  新規則只改它怎麼算新的 `Dir`：相對路徑接在目前目錄後面，`.`、`..` 照 DOS 處理；
  絕對路徑去掉磁碟機字母與開頭的 `\`。仍記一筆 `int 21h AH=3Bh` 的註記。
- `AH=4Eh／4Fh`（find first／next）：樣式帶真實存在的子目錄（`SAVEGAME\*.BIN`）時掃那個目錄
  （`Root` 與 `Scratch` 各掃一次，同名檔取暫存層），回傳磁碟上的完整檔名，
  包含 `LONGFILE.BIN`、`LOWERCASE.BIN` 這類超過 8.3 與小寫的名字。
  樣式沒有真實子目錄時維持舊行為，只掃最上層。
- `AH=39h／3Ah／56h`（mkdir、rmdir、rename）：見 §2.3。
- `RootDir` 隨 `SaveState`／`LoadState` 存取。舊狀態檔沒有它，沿用目前這一台的設定。

## 3. 驗收

`internal/dos/paths_test.go`：

- `TestProgramDir`：由 `ProgramPath` 取出目錄。
- `TestResolveRealSubdirectories`：真實子目錄的相對、絕對、`..`、大小寫、8.3 名；暫存層蓋過原版。
- `TestChdirMovesRelativeLookups`、`TestDirectoryAttributes`、`TestFindFirstInSubdirectory`。
- `TestScratchWritesFollowDirectories`：寫入落在同名子目錄，讀得回來。
- `TestLegacyDefaultKeepsBasenameRule`：沒給 `ProgramPath`（預設 `RICH2`）時，根目錄的檔與
  `C:\RICH2\...` 絕對路徑照舊走 basename；預設下真實子目錄也能解析。
  **junk 路徑（目錄部分不存在）退回 basename 規則由 `resolve` 的流程保證**
  （`resolveDir` 回 `""` 後落到原本的程式碼），這份測試沒有單獨涵蓋它。既有程式的測試套件
  （`internal/dos`、`apps/*`）全綠是這一條的間接證據。
- `TestStateKeepsRootDir`。

整個 `internal/dos`、`internal/machine`、`oracle` 與各 `apps/*` 測試全綠
（既有程式的對拍測試沒有被這條改動影響）。

實跑：`AESOP.EXE eye start` 開得到 `SAVEGAME\ITEMS_00.BIN` 與 `LVL01_00.BIN` 到 `LVL14_00.BIN`
（收據的 `opened` 清單），開機走過去（見 `200`）。修正前這些找不到，程式退回 `.TXT` 翻譯，
翻譯也失敗，開機以離開碼 1 結束。

## 4. 已知限制與取捨

- **預設的 `Dir` 與 `RootDir` 仍是 `RICH2`，只有給了 `ProgramPath` 才改。** 這是為了不動既有程式，
  代價是兩種預設並存。要統一需要用既有程式的對拍重新驗收。
- **新規則對既有程式也生效**（只要它的資料夾有子目錄，路徑又帶那個子目錄名）。這是行為變更，
  不只是新增能力；既有套件全綠，但沒有逐支既有遊戲重跑對拍。
- 8.3 截斷比對（`dosName`）只在名字沒有完全相同時用，不處理 `~1` 這類序號衝突。
- 目錄比對與檔名比對一律大小寫不分。
- 沒有多磁碟機的模型：磁碟機字母直接丟掉。

## 5. `StrictChdir`（opt-in）

### 5.1 現象

`chdir`（`AH=3Bh`）原本一律成功（舊註解：回失敗會讓既有程式判定安裝不完整）。
EOB3 的 `launch cine.exe` 會 `chdir` 到名為 `CINE.EXE` 的目錄（原版根本沒有這個目錄），
真 DOS 回失敗，目前目錄不變，`exec` 的路徑是 `C:\EOB3\CINE.EXE`。
舊的一律成功讓目前目錄漂到不存在的 `EOB3\CINE.EXE`：

- 環境區塊與 `AH=4Bh AL=03h` 的程式路徑變成 `C:\EOB3\CINE.EXE\CINE.EXE`
  （由收據的 `exec` 欄與環境區塊傾印看到）；
- `CINE.EXE` 回頭 exec 時又變成 `C:\EOB3\CINE.EXE\AESOP.EXE`，而且**目前目錄會一路帶進
  重新啟動的 `AESOP.EXE`**，之後帶子目錄的相對路徑（`SAVEGAME\...`、`CHARGEN\...`）
  是接在 `EOB3\CINE.EXE` 後面解析，只靠 basename 退路才找得到。

### 5.2 規則

`DOS.StrictChdir`（預設 **false**）為真時，`AH=3Bh` 的目標必須是存在的目錄
（`dirExists`：磁碟機根目錄、`RootDir` 與其祖先、`Root`／`Scratch` 底下真實存在的子目錄），
否則 CF、AX=3（path not found）、目前目錄不變。`oracle.StrictChdir(true)` 打開，
`apps/eob3` 的 `Load` 會呼叫。預設關閉，既有程式不受影響。

### 5.3 驗收與證據

- `TestStrictChdir`：預設不存在的目錄也成功；嚴格模式下失敗且 `Dir` 不變、AX=3；
  存在的子目錄、絕對路徑、磁碟機根、`..` 都成功。
- 實跑：打開之後同一個 Esc 腳本的收據 `exec` 欄是 `C:\EOB3\CINE.EXE`、`C:\EOB3\AESOP.EXE`，
  不再有 `\CINE.EXE\` 的重複；第 100,000,000 道指令的主選單畫面色號雜湊與打開之前相同。
  這支持「真 DOS 的 `chdir` 失敗是原版預期行為」的推論（強推論：沒有在真 DOS 上量過）。
