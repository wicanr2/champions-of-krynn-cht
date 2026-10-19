# 008 跨畫面盤點：商店、戰鬥、存讀檔

Issue：#6。以 `cok-overlay-shot -inventory` 記錄每次印字的路徑、呼叫端、列欄、是否命中（含原文的盤點只留在 `workplace/`）。
純數字、角色數值等不需翻譯的未命中不列。

## 大地圖與戰鬥（預建隊伍，`workplace/probe/overland1`、`combat3`）

| 發現 | 處理 | 結果 |
|---|---|---|
| 戰鬥選單 `Move View Aim Use Done` 由多個帶尾端空白的字串組成 | 路徑 C 改以連續片段查表 | 選單全部覆繪 |
| 瞄準選單 `Target Next Prev Manual Center Exit` | 同上 | 全部覆繪 |
| `Move/Attack, Move Left = N`（本作無尾端空白）、`Range = N  ` | 新增組句模板 | 命中 |
| 金錢列 `Steel            2`（總寬 18、靠右） | 每種位數一條模板 | 命中 |
| 物品名 `Ring Mail` 被逐詞拼成「戒指甲」 | 名稱 catalog 收術語表的物品、法術；物品詞組先查它 | 「環甲」 |

## demo 戰鬥（`workplace/probe/demolong`）

demo 第 600M–3,500M 步有自動戰鬥。未命中的是本作與幽靈騎士寫法不同的組句：
`Hitting for N points of damage`（幽靈騎士兩個空白）、`takes N points of damage from Magic／Electricity`（幽靈騎士 damage 與 from 相連）、
`Goes Down, and is Dying`，以及怪物名 `BOZAK`。補模板與術語後重跑驗證見下節。

## 商店（屯墾區武器庫，`workplace/probe/shop1`）

| 發現 | 處理 | 結果 |
|---|---|---|
| 清單列「品名＋補位＋價格」整串印出（`Long Sword                   7`） | `compose.go` pricedRow：品名走物品規則，補位以半形合併後的格數計算，價格靠右 | 15 列全部覆繪、價格對齊 |
| 數量前綴 `20 Arrows`（引擎加 s 成複數） | 數量置後、去 s 查單數 | 「箭 20」 |
| `SIR STRONGSWORD buys a Hoopak ` | buys 規則：名字（音譯）＋「買下」＋品名 | 命中 |

## 存讀檔（`workplace/probe/saveload`）

大地圖紮營存到遊戲 B（`savgamB.dat`），主選單讀回：讀檔後隊伍選單疊字 19/19、範圍外差異 0，沒有殘字。
隊伍選單的名字在第 1 欄、列 4–9（冒險畫面在第 17 欄），adapter 兩處都學隊員名。

### 補模板後重跑（`workplace/probe/demolong2`）

同一 demo（start-200m.state 起、無輸入、到 #3,500M）10 張截圖疊字全部顯示、範圍外差異 0；盤點中除純數字外沒有未命中。
戰鬥面板顯示「瑟·林沃爾德 攻擊 波札克龍人 造成 18 點傷害」「受到 5 點魔法傷害」「倒下了 並且垂死」、移動力列。

## 底列繼續提示

`Press <enter>/<return> to continue` 印成首字大寫，字串表只有全小寫、全大寫兩種；路徑 C 查表補試大小寫變體後覆繪（旅店場景 `workplace/probe/inn1`）。
