# 201 非鏈結 256 色（Mode X）

狀態：**READY**
日期：2026-10-04
前置：[`009-planar-vga.md`](009-planar-vga.md)、[`013-vga-planar.md`](013-vga-planar.md)（平面寫入與讀取）、
[`192-crtc-effects.md`](192-crtc-effects.md)（列距、顯示起點、分割線）、
[`005-oracle-api.md`](005-oracle-api.md) §3（對拍在色號空間做）
後續：[`200-eob3-oracle-adapter.md`](200-eob3-oracle-adapter.md)

---

## 1. 問題

模式 13h 的 A0000 是線性 64 KB：位元組 A 就是像素 A。EOB3 的 CINE 與主選單在
設完 13h 之後，把序列器與 CRTC 改成**非鏈結（unchained）256 色**，也就是 Mode X：

```
3C4:04 ← 04    ; Memory Mode：清掉 Chain-4（bit3），關閉 odd/even
3D4:14 ← 00    ; Underline Location：關閉 doubleword
3D4:17 ← E3    ; Mode Control：byte mode
```

此後 CPU 位址直接是平面內的位移，Map Mask（`3C4:02`）決定寫進哪幾個平面，
像素 `(x, y)` 在平面 `x & 3` 的 `y × 列距 + x / 4`。

模擬器原本把 mode 13h 當線性：畫面讀 `Mem` 的 A0000，程式的寫入走平面路徑，
兩邊不相交。結果是**畫面有東西，但每個平面的資料被當成線性位元組讀**，
圖被壓成原來的 1/4 寬、左右並排四份（EOB3 的主選單就是這樣）。
沒有任何錯誤訊息。

## 2. 規則

### 2.1 何時走平面路徑

`planarActive` 在視訊模式 13h 時回 `!chain4()`：

- 序列器 index 04 的 bit3 為 1（鏈結）：線性，走 `Mem`。
- bit3 為 0（非鏈結）：走四個平面（`Planes`）。
- 其他模式的判準不變（`planarMode(mode)` 為真就是平面；其餘看 Map Mask 訊號，假說，見該函式註解）。

### 2.2 BIOS 設 13h 時（`setMode13`）

`SetVideoMode(0x13)` 呼叫 `VGA.setMode13()`：

| 暫存器 | 值 |
|---|---|
| 序列器 04 | `0Eh`（鏈結位元立起來） |
| CRTC 0Ch、0Dh（顯示起點） | 0 |
| CRTC 13h（列距）、14h、17h | `28h`、`40h`、`A3h` |
| 四個平面 | 清 0 |

**不碰 `Mem`**：既有的線性路徑不清畫面。

### 2.3 組畫面（`Unchained256`）

`Indexed()` 在 `Unchained()`（模式 13h 且 A0000 走平面）時呼叫 `VGA.Unchained256(320, 200)`：

- 像素 `(x, y)` ＝ `Planes[x & 3][起點 + y × 列距 + x >> 2]`，**直接是 DAC 索引**
  （屬性控制器的 8 位元模式，不經 16 色模式的 4 位元對照）。
- 列距 ＝ CRTC 13h × 2（以位元組為單位）。**列距 0 當成沒設過**，退回 `320 / 4`。
- 顯示起點來自 CRTC 0Ch／0Dh（`DisplayStart`）。
- 分割線來自 CRTC 18h 等（`LineCompare`）。單位是掃描線；200 列模式每列掃兩次
  （Maximum Scan Line ＝ 1），所以換成像素列要除以每列的掃描數（`CRTC 09h & 1Fh + 1`）。
  超過分割線的列從平面偏移 0 重新開始。
- 平面位址取 `& (PlaneSize − 1)`。

### 2.4 寫入與讀取

寫入與讀取沿用 `009`、`013` 的平面路徑，Mode X 與 16 色模式共用：

- 寫入受 Map Mask 控制；每個平面各有自己的位址。
- 讀取依 Read Map Select（`3CE:04`）選平面，讀同時載入 latch。
- Write Mode 1（`3CE:05 ← 01`）把 latch 整組寫回，四個平面一次搬完（Mode X 的區塊複製靠這個）。

### 2.5 其他介面

- `PlanarSize()`：Mode X 不是 16 色平面模式，回 `0, 0`。
- `VideoSize()`：320×200。
- `oracle.Indexed()`、`oracle.WritePNG`：Mode X 時從平面組畫面。
- `SaveState`、`LoadState`、記憶體內快照、`Clone`／`Restore` 都要帶 CRTC 暫存器組、
  CRTC 索引、Misc Output 與是否寫過 CRTC；**漏抄的話還原之後非鏈結 256 色的畫面整個錯位**。

### 2.6 不模擬

- 水平像素平移（屬性控制器 13h）。
- 雙字組模式（CRTC 14h bit6）。兩者在標準 Mode X 設定裡都是關的。
- 逐週期的掃描與暫存器切換時機（`193-crtc-timing.md` 的時序模型不預設啟用）。

## 3. 驗收

`internal/machine/unchained_test.go`：

- `TestMode13hStaysLinearUntilUnchained`：剛設好的 13h 是線性，寫 `Mem`；鏈結位元清掉才走平面，立回來又回線性。
- `TestUnchainedPixelLayout`：像素位置 ＝ 平面 `x & 3` 的 `y × 80 + x >> 2`；其他像素為 0；`Indexed()` 回複本；`PlanarSize ＝ 0,0`、`VideoSize ＝ 320×200`。
- `TestUnchainedReadAndLatchCopy`：Read Map Select 讀回、Write Mode 1 的 latch 複製。
- `TestUnchainedCRTCPitchStartAndSplit`：列距、顯示起點、分割線。
- `TestSnapshotAndStateKeepCRTCAndHighHalves`：快照與狀態檔保留 CRTC。
- 全套 `internal/machine`、`oracle` 與各 `apps/*` 測試全綠（既有的線性 13h 程式不受影響）。

實跑：EOB3 的主選單畫面（320×200）完整、不並排，標題與選單文字可由人眼辨識
（`docs/re/eob3-dosgolem-oracle.md`）。

## 4. 證據等級

| 項目 | 等級 |
|---|---|
| EOB3 實際寫入的序列器與 CRTC 值（04h←04h、CRTC 14h←00h、17h←E3h） | 已證實（執行時觀測） |
| 像素與平面的對應、CRTC 列距與起點的語意 | 強推論（依 VGA 公開規格與 DOSBox-X 的行為，沒有實機對拍） |
| 畫面正確 | 已證實（人眼檢視：圖不再並排四份，文字可讀）；沒有與實機畫面逐像素對拍 |
| 分割線的像素換算 | 假說（只有單元測試涵蓋，EOB3 的選單沒有用分割線） |

## 5. 已知限制

- Mode X 的 400 列版本（max scan line ＝ 0）沒有專門測試。
- 其他程式如果在非鏈結模式下依賴屬性控制器的色盤對照或水平平移，畫面會不同。
