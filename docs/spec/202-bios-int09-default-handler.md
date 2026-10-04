# 202 BIOS 的 int 09h 預設處理

狀態：**READY**
日期：2026-10-04
前置：[`014-hardware-keyboard.md`](014-hardware-keyboard.md)（IRQ1、埠 0x60）、
[`008-bios-keyboard-injection.md`](008-bios-keyboard-injection.md)（`int 16h` 與 BDA 鍵盤緩衝）、
[`185-keyboard-trace-and-keypad-names.md`](185-keyboard-trace-and-keypad-names.md)
後續：[`200-eob3-oracle-adapter.md`](200-eob3-oracle-adapter.md)

---

## 1. 問題

硬體鍵盤（`machine.keyTick`）把掃描碼放在埠 0x60，再呼叫 `int 09h`。程式有兩種接法：

1. 自己的 ISR 讀 `0x60`、自己記帳，不 chain（智冠《三國演義》的主選單，`014`）。
2. 自己的 ISR 只是**包裝**：做完記帳（`AESOP.EXE` 在 `1779:0008` 遞增再遞減一個重入計數），
   中間 `pushf / call far [舊向量]` chain 回舊向量，由 BIOS 把掃描碼翻成 `(掃描碼, ASCII)`
   放進 BDA 的環形緩衝，程式之後用 `int 16h` 或直接讀緩衝取鍵。

第二種的舊向量是模擬器的 `CD 09 / CF` stub（`int 09h; iret`），原本 `int 09h` 在 `dos` 層
**什麼都不做**。於是鍵送到了 ISR、ISR 也 chain 了，緩衝卻一個鍵都沒有，
程式對鍵完全沒反應，而且不會有任何錯誤或未實作記錄（滑鼠走另一條路，照常反應）。

## 2. 規則

向量 9 指向模擬器的 stub 時（程式沒有接管，或接管後 chain 回來），`dos.handle` 的 `0x09`
分支呼叫 `biosInt9`，做真機 BIOS 做的那部分：

1. 讀 `0x60`，高位元是放開（break），低 7 位元是掃描碼。
2. 修飾鍵（左 Shift 2Ah、右 Shift 36h、Ctrl 1Dh、Alt 38h）：只更新 BDA `0040:0017` 的
   bit1、bit0、bit2、bit3，不產生按鍵。
3. 放開鍵不進緩衝。
4. 通碼翻成 ASCII（美式鍵盤）：
   - 沒按 Shift：`unshiftedASCII`；按了 Shift（左或右）：`shiftedASCII`。
   - Ctrl 加字母：ASCII 變成 `1` 到 `26` 的控制碼。
   - F1 到 F10（3Bh 到 44h）、Home／上／PgUp／左／右／End／下／PgDn／Ins／Del
     （47h 到 53h，不含 4Ah、4Ch、4Eh）：ASCII 為 0，掃描碼照原樣。
   - 其餘沒有字元的鍵（例如 Caps Lock 本身）不放進緩衝。
5. 放進 `PushKey`（BDA 環形緩衝與 `int 16h` 共用的那份）。

`SetKeyPort(code)` 是給測試用的入口：直接設埠 0x60 的值，不經佇列也不送中斷。

向量被程式接管（不指向 stub）時，`handle` 回 `false`，讓 CPU 真的跳進程式的 ISR，
分派不再插手，否則一個鍵會被處理兩次。

## 3. 不做的

- Pause、PrintScreen、Ctrl-Alt-Del、Num Lock 與小鍵盤的數字翻譯。
- 延伸前綴 `E0`、`E1` 與其後的碼（右 Ctrl、右 Alt、獨立方向鍵區等）：
  前綴位元組的 bit7 是立的，被當成放開鍵丟掉；其後的碼照一般碼處理，
  所以獨立方向鍵會與小鍵盤方向鍵同碼，右 Ctrl 與 Alt 會被當成左邊的修飾鍵。
- Alt、Ctrl 與功能鍵的組合碼（例如 Alt+F1 的掃描碼 68h）：掃描碼 3Bh 到 44h 一律照原樣。
- Caps Lock、Scroll Lock 的鎖定狀態與 LED。

## 4. 影響範圍

這條改動是**全域**的：任何一支程式，只要它的 ISR chain 回向量 9 的 stub，
現在都會在 BDA 緩衝多出按鍵（原本是沒有）。

- 只讀埠 0x60、不 chain 的程式不受影響（`TestBiosInt9OnlyForOwnStub`）。
- 同時讀埠 0x60 並 chain 的程式，現在會在 `int 16h` 看到同一個鍵。真機也是這樣，
  但既有程式的對拍（FD2、rich2、wolong、eob1）當時是在沒有這條行為下建立的，
  既有 `apps/*` 測試全綠，沒有逐支重跑 GUI 對拍。

## 5. 驗收

`internal/dos/int9_test.go`：

- `TestBiosInt9TranslatesMakeCodes`：Enter、Esc、空白、字母、數字、退格、Tab、標點、F1、F10、四個方向鍵、Home、Del
  的 `(掃描碼 ＜＜ 8) | ASCII`；放開鍵不進緩衝。
- `TestBiosInt9ModifierKeys`：Shift 立與放開對 BDA 旗標、Shift+a、Shift+1、Ctrl+C、Caps Lock 不進緩衝。
- `TestBiosInt9OnlyForOwnStub`：向量不指向 stub 時回 `false`，不碰緩衝。

實跑：EOB3 主選單用鍵盤操作：Enter 可關閉選單對話框，Esc 可退出
（`docs/re/eob3-dosgolem-oracle.md` 的鍵盤章節）。

## 6. 證據等級

| 項目 | 等級 |
|---|---|
| `AESOP.EXE` 的 ISR 只是包裝、chain 回舊向量（`1779:0008` 與 `pushf / call far`） | 已證實（反組譯與執行軌跡） |
| 翻譯表（美式鍵盤） | 強推論（依 IBM PC/AT Set 1 掃描碼與 BIOS 慣例，沒有實機比對） |
| 延伸鍵、小鍵盤、Alt 組合碼的行為 | 未知（刻意不做） |
