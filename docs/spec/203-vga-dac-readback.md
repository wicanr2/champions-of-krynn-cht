# 203 VGA DAC 埠讀回與 EOB3 墓園色盤

狀態：**READY**
日期：2026-10-05
前置：`011-bios-palette-and-vector-stubs.md`、`201-vga-unchained-256.md`、`200-eob3-oracle-adapter.md`

## 問題與證據

EOB3 `AESOP.EXE`（SHA-256 `fb8900a114ad6c468be2e01737b9c3735ff8a9ec69126c9e3126293577670593`）
由正常片頭及主選單載入 `ITEMS_01.BIN` 的 Quick Start Party 後，dosgolem 在墓園能畫出
與 remake 相同的色號，但 77 項 DAC 顏色仍為白色。原版在主選單前已讀 `3C9h` 774 次，
進存檔清單前累計讀 1,215 次，對應寫 `3C7h` 258 次、405 次；現行 `Machine.In8`
對 `3C9h` 回 `FFh`，`Out8` 對 `3C7h` 沒有設定讀取位址。

可丟棄 `go -overlay` 探針只補這兩個埠：`3C7h` 寫值選讀取索引並歸零 RGB 分量；
`3C9h` 讀目前 6-bit 分量、每三次往下一色。相同原版腳本的起點與前進一步
`.idx` SHA-256 保持 `a365b12b…f5f42`、`e681aa8a…4d4b7`，而 `.pal` 改為
`0d09ca68fa27903ac504bc271a729df16071c8966a50dbd71f75e131d6b05d84`；
其 768 個 6-bit 分量全部等於 remake `dungeon.dac`
（SHA-256 `092280dd0b257ba8ff4a5dc2e8c0744bdc86e5b20acfdc9c493ef30df652ee13`）。
未修時的墓園 `.pal` SHA-256 為 `188ffa2bf7c6d85f850919393ecdae50cbc6e51e5c597e82fdcb48cca9aca336`，
與 remake 有 126 個分量差。測量方法、遊戲輸入與 oracle 限制見
`~/cht/eob_123_remake/docs/re/eob3-dosgolem-oracle.md` 第 7、8 節。

VGA 埠的行為交叉參照 DOSBox-X 原始碼 `src/hardware/vga_dac.cpp`：
`write_p3c7` 選讀取索引、清分量相位；`read_p3c9` 依 R、G、B 回值，第三次後
索引遞增。這是模擬器契約，未量真實 VGA 硬體週期。

## 規則

- `Out8(3C7h, i)` 選 DAC 讀取索引 `i`，分量相位設為 0。
- `In8(3C9h)` 回 DAC 當前索引的 R、G、B 中下一個 6-bit 分量；第三次後
  相位回 0，索引以 8-bit 遞增。`In8` 原有讀取計數照常增加。
- 寫入用的 `3C8h` 索引與 `3C9h` 寫入相位維持獨立；本切片不新增 `3C7h` 的
  讀取狀態查詢，也不處理 DAC 8-bit 擴展模式。
- 記憶體內 `Snapshot` 與 gob 狀態檔都保存讀取索引及分量相位。新增欄位
  不改既有版本號。新檔有 `DACReadKnown=true`；舊檔若已讀過 `3C9h`
  卻沒有這組欄位，載入前拒絕，避免從錯誤分量接續；未讀過的舊檔可載入。

## 驗收

1. 機器層測試直接寫兩筆 DAC，再寫 `3C7h`，連續讀六次 `3C9h`，核對
   R/G/B 順序、索引遞增與寫入索引不受影響。
2. 在讀取一個分量後分別做記憶體快照與 gob 狀態檔往返，下一次讀取仍是
   同一項的 G；舊檔已有 `3C9h` 讀取紀錄而缺讀取狀態時，明確拒絕且
   不更動現行機器。
3. `internal/machine`、`internal/dos`、`oracle` 與 `apps/eob3` 針對性測試通過。
   正常路徑重跑 EOB3 墓園兩張收據，`.idx` 不變；兩張 `.pal` 與 remake
   DAC 的 768 個分量全同。下方文字差異另查，不由本規格推論已解。

證據等級：EOB3 的埠存取數與修後色盤對拍是**已證實**的執行觀測；
`3C7h`／`3C9h` 的通用 VGA 語意取自成熟模擬器原始碼，為**強推論**；
本切片不宣稱實機逐週期一致。
