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

`DS` 線性 `13635` 起的遠指標陣列是戰鬥名單（隊員加怪物），戰鬥外內容是舊的，不能當隊伍名單。

## 未知

- 6、10 兩種死亡狀態值的差別。
- 昏迷、中毒在本作的實際狀態值（demo 戰鬥我方沒人倒下）。規格 017 擋的是「由 0 變成任何非 0」，不依賴具體值。
