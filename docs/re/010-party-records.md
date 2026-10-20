# 010 隊伍紀錄：串列、HP、狀態、陣營

Issue：規格 017。位址以 DS=`0CD6` 表示（docs/re/003）。

## 方法

`cmd/probe` 從三個狀態續跑，以 `-dump-mem-at` 傾印線性 0–0x9FFFF：

| 傾印 | 狀態檔 | 隊伍 |
|---|---|---|
| `workplace/probe/cheat1/m<步數>.bin` | `probe/demo3d/d450.state`，1.40e9–1.70e9 共 7 份 | demo 三人：SIR RINGWALD、VARIANNA、LYN SWIFTFOOT（戰鬥中） |
| `workplace/probe/cheat-throtl1/m.bin` | `probe/throtl1/end.state` +1000 步 | SIR STRONGSWORD、ISTAN HORBIN |
| `workplace/probe/cheat-g3-party/mt.bin` | `probe/g3-party/5-opening.state`，2.925e9（屯墾區選單） | ANNA |

## 結果

| 欄位 | 位置 | 證據 |
|---|---|---|
| 隊伍串列頭 | `DS:609A` 遠指標 | 三份傾印中指向第一位隊員紀錄的位置，唯一共同者 |
| 下一位 | 紀錄 `+0x17F` 遠指標 | demo：RINGWALD→VARIANNA→LYN→GREEN DRAGON→…→ZOMBIE；斯拉托：STRONGSWORD→ISTAN |
| 目前 HP | `+0x197` | RINGWALD 七個時點 61、57、50、50、46、46、44 |
| 健康狀態 | `+0x188` | BOZAK 四隻 HP 歸 0 的同一時點由 0 變 6 或 10，`+0x189` 由 1 變 0、`+0xE3..E6` 指標清 0 |
| 陣營 | `+0x18A` | 6 名隊員皆 0、13 隻怪物皆 1；所有比對位移中唯一完全分開兩組且值固定者 |

| 效果串列頭 | `+0xE3` 遠指標；每筆 9 bytes：`+0` 類型、`+1..2` 持續時間、`+3` 資料、`+4` 旗標、`+5..8` 下一筆遠指標 | 預建隊伍大地圖戰鬥（下節）伊斯坦被麻痺後串列尾端多一筆 `34 63 00 0C 00 …`；怪物被殺時 `+0xE3..E6` 清 0 |

`DS` 線性 `13635` 起的遠指標陣列是戰鬥名單（隊員加怪物），戰鬥外內容是舊的，不能當隊伍名單。

## 效果串列（麻痺）

`cmd/cok-promo` 從 `probe/explore2/end.state`（4.318e9）重播 `probe/explore3/explore.log` 的送鍵（戰鬥中 Q 交給電腦），對手為山巨人與卡帕克。
4.700e9、4.7236e9 至 4.7237e9 間七份與 4.795e9 的傾印比對：麻痺前後角色紀錄 0–0x198 除 HP 外不變；沿 `+0xE3` 走訪，伊斯坦的串列由 `12、69` 變成 `12、69、34`。

| 類型 | 意義 | 等級 |
|---|---|---|
| `0x34` | 麻痺 | 已證實（本作，上述） |
| `0x1F`、`0x21`、`0x2A`、`0x33`、`0x35` | 失能、失明、緩慢、蛇咒、睡眠 | 假說（《詛咒藍符》反組譯 `curse_of_the_azure_bonds/docs`，同引擎；`0x34` 在兩作一致） |
| `0x12`、`0x69`、`0x5C`、`0x5E`、`0x07`、`0x5F` | 預建隊伍開戰前就有的效果（種族、裝備等） | 未知，視為非有害 |

與《詛咒藍符》的對照：該作 `health_status +0x196`、`combat_team +0x198`，本作 `+0x188`、`+0x18A`，同差 0xE，支持兩作結構相同的推論。

## 未知

- 6、10 兩種死亡狀態值的差別。
- 中毒在本作的實際狀態值。昏迷為 4（大地圖戰鬥原版 5.0e9 步時 SIR STRONGSWORD、ISTAN、SIRRION 皆 HP 0、狀態 4）。規格 017 擋的是「由 0 變成任何非 0」，不依賴具體值。
- 除 `0x34` 外各效果類型在本作的實際意義；恐懼、魅惑以外的其他心智效果、疾病、能量吸取的代碼。
