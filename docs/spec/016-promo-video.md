# 016 推廣影片

狀態：**READY**（2026-10-20 成品已產出，待使用者觀看確認）。Issue：#15。

使用者指示（2026-10-19）：60–90 秒精華，3 倍畫面加 HD，原版 AdLib 聲音，MP4（H.264），放 `workplace/promo/`，不進 Git。

## 1. 錄製

dosgolem `cmd/cok-promo` 從既有狀態檔續跑，逐格輸出覆繪後的 3 倍畫面（PNG）與音訊（48 kHz 立體聲 s16le），另寫 `receipt.json`。

- 畫面、HD 主題、語言包、手札、攻略、地圖疊層與配樂的接法同 cok-play；配樂播放器與 cok-play 共用 `apps/cok/bgm`。
- 時間基準同 cok-play：VGA 回掃每秒 70 次，每秒 `VGAFrameEvery×70` 步，從載入後的機器讀。狀態檔是 100% 時脈時每秒 11,550,000 步。影片畫格與音訊樣本都從這個量換算，收據的 `audio_seconds` 與 `video_seconds` 相等。
- 不用 `StepsPerSecondNow()`（IRQ0 間隔×PIT 頻率）：本作狀態下它約 890k 步／秒，影片會慢 13 倍，也與 cok-play 的實際節奏不同。
- 前端動作以 `-actions 步數:動作` 排程：手札、攻略翻頁、地圖疊層與全圖、換語言。送鍵用 `-keys-at` 或 `-keys`、`-key-from`、`-key-every`。
- 狀態檔載入前印出的文字沒有覆繪（攔截點在載入後才綁定），隊員名也要在載入後印過一次才會音譯。所以每段要從遊戲會重印隊伍欄的時點之前起跑：battle 從 demo 的 450M 狀態起跑；throtl 先紮營再離開讓遊戲重印隊伍欄。

## 2. 分鏡

`tools/promo.sh` 錄五段，`tools/promo-build.sh` 依剪輯表剪接。

| 段 | 狀態檔 | 步數 | 內容 |
|---|---|---|---|
| title | `probe/g2-receipt/start-200m.state` | 1.870e9–1.995e9 | SSI、HD 主標題、DragonLance、原版標題曲 |
| open | `probe/g3-party/5-opening.state` | 2.45e9–2.95e9，Enter 從 2.51e9 每 6e7 | 旅店、大地圖、卡爾爵士、手札第 51 則、屯墾區選單 |
| throtl | `probe/throtl1/end.state` | 1.985e10–2.023e10；E 三次（第一個被吃掉，第二個紮營、第三個離開）、M 回移動模式，再走六步 | 斯拉托 3D 牆面 HD、地圖疊層全圖、攻略第 2 章 |
| battle | `probe/demo3d/d450.state` | 錄 1.46e9–1.60e9 | demo 戰鬥，HD 戰鬥圖塊、音譯隊員名 |
| langs | 同 open | 2.64e9–2.72e9 | 卡爾爵士對話依序換成简中、日文、韓文、英文、繁中 |

原版開場劇情沒有音樂（規格 014 §1），open 的前四個片段以 `sanctuary`、langs 以 `outpost` 的渲染檔墊底，與遊戲音效混音，同一首跨片段接續。其餘片段是遊戲本身的 AdLib 與配樂。片尾 6 秒字卡配 `finale` 開頭。

## 3. 編碼

畫格 2 倍最近鄰放大到 1920×1200，30 fps，libx264 `-preset slow -crf 18`、yuv420p，AAC 192 kb/s。字卡用 Noto Sans CJK TC。片段頭尾 0.25 秒淡入淡出，以 concat 接成 `workplace/promo/cok-promo.mp4`。映像 `cok-ffmpeg:r1`（`tools/docker/ffmpeg.Dockerfile`，Debian bookworm ffmpeg 5.1.9）。

## 4. 收據（2026-10-20）

- 成品 66.0 秒，H.264 1920×1200 30 fps 加 AAC，SHA-256 `4ced5f7d7f16…`（langs 段以在地字形重錄後）。
- title、langs、battle、throtl 各重錄一次，畫格與音訊 SHA-256 完全相同。
- 全片的隊伍欄、戰鬥面板與劇情文字都是覆繪後的文字（分鏡檢查 `tools/promo-sheet.sh`）。
- 每段 `audio_seconds` 等於 `video_seconds`；各段沒有缺字。
- 片段平均音量 -24 到 -32 dB，沒有靜音片段。
- 配樂確實混入（以前一版 battle 段量測，該段由 issue13 的 battle.state 起跑）：battle 段與不給 `-music-dir` 的同段相減，差值每秒 -29 到 -31 dB，12 秒全程都有；throtl 段前 9 秒開配樂 -29.5 dB、不開 -35.9 dB。
