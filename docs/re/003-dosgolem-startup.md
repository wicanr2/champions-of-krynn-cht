# 003 dosgolem 冷啟動基線

日期：2026-10-19。Issue：#3（G2）。輸入見 [001](001-input-baseline.md)。

## 固定版本

| 項目 | 值 |
|---|---|
| dosgolem | `d4cd48c86bf8f5ed334e008633caa0a5d7b2c936`（幽靈騎士 `dosgolem/dkk` 分支頂端），隔離 clone `workplace/dosgolem/`，分支 `champions-of-krynn-cht`，尚無本專案 commit |
| Go | go1.24.13 linux/amd64 |
| image | `dosgolem-go-ebiten:1.24-r1`（`sha256:8ed6bdee2bce1f32f804e888f942b25bbf62f2165a43344bb4587f3371de0581`） |
| 執行入口 | dosgolem `tools/go.sh`（`--rm`、`--network none`） |

## 音效設定（已證實）

- `START.EXE` offset `0x1d23` 起的設定選單：顯示卡 `[1] CGA [2] EGA [3] Tandy`；音效 `[1] C/MS [2] Roland LAPC [3] AdLib
  [4] Tandy 1000 [5] PC [6] Silent`；輸入 `[1] JOYSTICK [2] MOUSE [3] NEITHER`；存檔路徑。
- 原版附的 `KRYNN.CFG` 前三行 `E`、`R`、`M`，對應 EGA、Roland LAPC、MOUSE（強推論，依選單字首）。
- 用原版設定冷啟動，停在 `LOADING...PLEASE WAIT`：讀完 `sound.dax`（#27,676,636）後 CPU 在 `5252:02AC`，`IF=0`，
  `DX=0331`（MPU-401 狀態埠），讀回 `FF`，迴圈不結束。dosgolem 沒有 MPU-401。
- 對策：`tools/cfg-adlib.sh` 產生只把第 2 行改成 `A` 的副本放進 dosgolem scratch 層（scratch 蓋過唯讀原版目錄，
  見 dosgolem `009-scratch-writes` §2），等同玩家在安裝選單選 AdLib。幽靈騎士的 `DKK.CFG` 第 2 行也是 `A`。原版檔不改。

## 冷啟動觀察（已證實，dosgolem 收據）

| 步數 | 事件 |
|---|---|
| #801,529 | 開 `GAME.OVR` |
| #23,936,121 | 讀 `krynn.cfg`（要 128 得 71） |
| #23,936,342 | 切到視訊模式 0Dh（EGA 320×200 16 色 planar） |
| #100,000,000 | 標題畫面 Champions of Krynn |
| #150,000,000 | 製作群畫面 |
| #200,000,000 | 底列 `CHAMPIONS OF KRYNN V1.2 PLAY DEMO`，等選擇 |
| #299,000,000 | 送 `p` 後：建立隊伍主選單 CREATE NEW CHARACTER、ADD CHARACTER TO PARTY、LOAD SAVED GAME、INITIALIZE MOUSE/JOYSTICK、EXIT TO DOS，底列 `CHOOSE A FUNCTION SELECT` |

- 沒實作的服務：0 種。寫過的 I/O 埠：020、040、043、061、0C0、388、389、3C4、3C5、3CE、3CF、3D4、3D5。
- 送鍵用 `-bios-keys`（沿用幽靈騎士做法），`p` 被取走 1 個。
- 不送鍵時，PLAY DEMO 選單逾時後進 demo，#300M 已在冒險畫面（SIR RINGWALD 等預設角色）。
- `-dump-at` 多個時點以 `;` 分隔。

## 收據

[`tools/g2-startup-receipt.sh`](../../tools/g2-startup-receipt.sh)：冷啟動到 #200,001,000 存狀態，再從狀態送 `p`
重播兩次到 #299,000,000。

| 產物 | SHA-256 |
|---|---|
| `start-200m.state` | `c14e2751a0b3c1269f8e0558c68f38f8f3a346140b313e50a0088a51567c19ed` |
| 主選單色號陣列 `menu-*.bin` | `9ad0f68dd5010711e641e1986f1a1d13594c74dffae162f6b06cb7ff11e06383`（重播 a、b 相同） |

## 與幽靈騎士的對照

| 項目 | 本作 | 幽靈騎士 |
|---|---|---|
| 視訊模式 | 0Dh | 0Dh |
| overlay 魔數 | `FBOV` | `FBOV` |
| 送鍵 | `-bios-keys` | `-bios-keys` |
| 進主選單 | 一鍵 `p` | 兩鍵 `' p'`（先過製作群） |

## 未知

- `KRYNN.CFG` 第 5 行 `F` 與後續 `0` 欄位語意。
- AdLib 音樂路徑（埠 388／389 有寫入）。
