# 199 PSP 區塊調整的邊界

狀態：**READY**
日期：2026-10-04
前置：[`004-dos-bios-services.md`](004-dos-bios-services.md) §1.2（`AH=4Ah` 記憶體探測）、
[`009-memory-allocator.md`](009-memory-allocator.md)（arena）、
[`010-exec-memory-reclaim-and-planar-write-modes.md`](010-exec-memory-reclaim-and-planar-write-modes.md)
後續：[`200-eob3-oracle-adapter.md`](200-eob3-oracle-adapter.md)

---

## 1. 問題

Borland C++ 的 `exec`／`spawn`（AESOP 的 `launch`）在換程式時：

1. `AH=4Ah BX=FFFFh` 探出可用上限 X。
2. 把自己的區塊調成 `X − (參數段數 + 1)`（或直接調成 X，見 §3）。
3. 若需要，`AH=48h` 要「參數段數」那麼多段放環境區塊。
4. 把載入器 stub（0x160 bytes）複製到**區塊尾端往下 0x17 段**，用 `AH=4Bh AL=03h` 載入新程式，
   再由 stub 換堆疊、跳進去。

這個版面對 DOS 的記憶體語意有兩個要求，`dos` 層原本各差一點：

- §2：自由區塊的 MCB 緊接在 `PSP + 新大小`（原本多算一段）。
- §3：探測與放大不能越過已配出的區塊（原本無視）。

少了任何一條，`launch` 都會印 `Secondary process launch failure 8`（ENOMEM），
或是 stub 被環境字串蓋掉而跑到記憶體上緣之外（`跑出可用記憶體：9FE8:0181`）。

## 2. 剩餘空間少一段（全域修正）

`setBlock` 在行程調整自己的 PSP 區塊後，把可配置區起點設成 `blk + want + 1`。
資料區塊是 `[blk, blk+want)`，DOS 的 `AH=4Ah` 把自由區塊的 MCB 放在**緊接著的 `blk+want`**
（DOSBox-X `src/dos/dos_memory.cpp` `DOS_ResizeMemory`，429 行 `DOS_MCB mcb_new_next(segment+(*blocks))`，
437 行自由區塊大小 `total-*blocks-1`）。

舊寫法讓剩餘空間少一段。Borland 的算法是「撐到最大值減 5，再要 4 段」，少一段就 ENOMEM：

```
探測 → X
撐到 X − 5          ; 自由區塊的 MCB 在 PSP + X − 5，到 MemTop 之間有 5 段，扣掉 MCB 本身剩 4 段資料
AH=48h BX=4         ; 剛好 4 段
```

修正：`d.setPSPBlock(blk + want)`。**這是全域行為變更，所有程式都吃到**；既有套件全綠，
沒有逐支既有遊戲重跑對拍（見 §5）。

驗收：`TestPSPBlockResizeLeavesExactTail`（要 4 段成功、要 5 段失敗）、
`TestPSPBlockResizeMCBMatches`（程式自己的 MCB 大小等於它要的大小、下一個 MCB 在 `PSP+大小`）。

## 3. PSP 區塊不能長過已配出的區塊（opt-in）

### 3.1 現象

AESOP 第一次 `launch cine.exe`：

- 探測 X = `9EFFh`，把 PSP 區塊（PSP 段 `0100h`）調成 `9EFAh`（`X − 5`），
  `AH=48h BX=4` 得到資料段 `9FFBh`（MCB 在 `9FFAh`）。這 4 段是環境區塊，
  之後 `CINE.EXE` 的 `PSP:2Ch` 指向它。stub 在 `9FE3h`（區塊尾端 `9FFAh` 往下 `17h`），
  到環境資料 `9FFBh` 有 `18h` 段，放得下 `16h` 段的 stub。
- `CINE.EXE` 按 Esc 或播完後 `launch aesop.exe`：它的載入碼（`0110:1B24` 一帶，與 AESOP 同為
  Borland 的 `_LoadProg`）**沿用繼承來的環境區塊**，不再 `AH=48h`，探測 X 之後直接把區塊調成 X。
  舊 `dos` 層回 `X = 9EFFh`，區塊尾端變成 `9FFFh`，stub 放在 `9FE8h`；
  環境資料仍在 `9FFBh`，`9FFBh − 9FE8h = 13h` 段，只有 `130h` bytes，放不下 `160h` bytes 的 stub。
  把環境字串（`COMSPEC=C:\COMMAND.COM`、`C:\EOB3\CINE.EXE\AESOP.EXE`）複製到 `9FFBh:0`，
  就覆寫了 stub 在偏移 `130h` 之後的尾端（`xor ax,ax; push ax; ...; jmp far`）。
  stub 跑到 `9FE8:012Eh` 之後踩進字串，一路執行到記憶體上緣 `A0001h`，模擬器回報
  `跑出可用記憶體：9FE8:0181`。

證據：第二次 exec 的 stub 記憶體傾印（`9FE8:0130h` 起是字串）與第一次（`9FE3:0130h` 起是
`33 C0 50 8B 85 ...` 的程式碼）逐 byte 比對；兩份 stub 在 `0000h` 到 `012Dh` 完全相同；
`CINE` 載入碼反組譯。這三點為已證實。

### 3.2 真 DOS 的行為

真 DOS 的 `AH=4Ah` 放大只能吃**相鄰的自由區塊**。第一次 exec 配出的環境區塊（MCB `9FFAh`）
擋在 PSP 區塊之上，探測會回 `9FFAh − 0100h = 9EFAh`，`CINE` 把自己調成 `9EFAh`，
stub 在 `9FE3h`，與第一次一模一樣，版面對得上。

### 3.3 規則

`DOS.StrictPSPGrowth`（預設 **false**）為真時：

- `AH=4Ah` 對「目前行程的 PSP 區塊」的可用上限，改為
  `min(MemTop − blk, 第一個已配出（非自由）的 arena 區塊的 MCB 段 − blk)`。
- 探測（`BX=FFFFh`）回這個上限，放大超過上限失敗，AX=8、BX＝上限。
- 縮小不受影響。
- 只動 `blk == curPSP` 的路徑。arena 內的區塊走原本的 resize。

預設關閉的理由：`setBlock` 內原本就有一段註解記載，早先試過「放大時上限取第一個已配置區塊的起點」，
讓智冠《三國演義》更早死（探測回報上限之後，程式判定記憶體不足）。
那邊兩種行為哪一個才是真 DOS **沒有證據**，所以不改預設，
只讓需要真 DOS 語意的程式由轉接層打開（`oracle.StrictPSPGrowth(true)`，
`apps/eob3` 的 `Load` 會呼叫）。

### 3.4 驗收

- `TestStrictPSPGrowthStopsAtAllocatedBlock`：預設探測回 `MemTop − PSP`（舊行為）；
  打開後回「環境區塊的 MCB − PSP」；撐到上限成功、多一段失敗；stub 到環境資料段至少 `16h` 段。
- 實跑：`apps/eob3` 的 `AESOP.EXE eye start`，第 30,000,000 道指令時按 Esc，第 100,000,000 道指令時
  畫面已是主選單（沒有量更早的步數）；不按任何鍵，第 1,400,000,000 道指令時還在片頭，
  第 1,500,000,000 道指令時已是主選單；兩條路徑的主選單畫面色號雜湊相同（`docs/spec/200` §5）。
  加入 `StrictPSPGrowth` 之前，同一個 Esc 腳本在第 30,795,640 道指令停在 `9FE8:0181`
  （旗標預設關閉時的程式路徑與當時相同，但沒有另外用關閉的旗標重跑一次）。

## 4. 與 DOSBox-X 的對照

| 項目 | DOSBox-X | dosgolem |
|---|---|---|
| `AH=4Ah` 縮小後自由區塊的 MCB 位置 | `segment + blocks` | §2 修正後一致 |
| `AH=4Ah` 放大 | 只吃相鄰自由區塊，最大值 = 到下一個已配出區塊 | 預設不一致，`StrictPSPGrowth` 打開後一致 |
| 區塊鏈 | 真實 MCB 鏈 | arena 記錄表，`syncMCB` 在需要時寫回記憶體 |

## 5. 已知限制

- §2 的全域修正沒有拿其他遊戲的原版畫面對拍；`apps/*` 的測試套件全綠是間接證據。
- `StrictPSPGrowth` 只看 arena 裡的非自由區塊。arena 在第一次 `AH=48h` 才建立；
  沒有 `AH=48h` 就沒有已配出區塊可擋，行為與舊的相同。
- 沒有模擬 `AH=49h` 釋放與相鄰自由區塊合併之外的 MCB 細節（例如 owner 欄位、`AH=4Bh` 子行程配置）。
