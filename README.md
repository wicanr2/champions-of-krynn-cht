# 克萊恩英豪（Champions of Krynn）繁體中文化

以 [dosgolem](https://github.com/wicanr2/dosgolem) 執行 SSI 1990 年 DOS 版《Champions of Krynn》v1.2，
在執行期攔截原版輸出並覆繪繁體中文。原版遊戲、規則與存檔保持原樣，需自備原版。

現況見 [CONTEXT.md](CONTEXT.md)，工作項目見
[GitHub Issues](https://github.com/wicanr2/champions-of-krynn-cht/issues)，協作規則見 [AGENTS.md](AGENTS.md)。

## 故事背景

本作是克萊恩三部曲的第一部，續作是《幽靈騎士》（Death Knights of Krynn）。龍槍戰爭結束後，
黑暗之后的龍人殘軍仍在索蘭尼亞邊境活動。你們的隊伍與年長的玫瑰騎士卡爾爵士同行，前往屯墾區報到，
第一項任務是巡視斯拉托並回報可疑的動靜。追查之下，龍人的屠殺、被奪走的龍蛋與牧師麥坦尼的陰謀逐一浮現。

出處：遊戲開場劇情（本作 ECL 譯文 `text/ecl-text.zh-TW.tsv`），內容經改寫。

## 畫面

以下是 dosgolem 執行原版並覆繪中文後的畫面（2026-10-19，收據 [docs/re/007](docs/re/007-g5-receipt.md)）。

| | |
|---|---|
| ![主選單](docs/screenshots/01-menu.png) | ![城市](docs/screenshots/02-city.png) |
| 建立隊伍主選單 | 3D 城市、隊伍欄與隊員名音譯 |
| ![查驗題](docs/screenshots/03-quiz.png) | ![手札](docs/screenshots/04-journal.png) |
| 手冊查驗題，問句後顯示答案（F9 切換） | 旅行手札自動彈出（F5 可翻閱） |
| ![城鎮選單](docs/screenshots/05-town.png) | ![戰鬥](docs/screenshots/08-combat.png) |
| 屯墾區底列選單 | 戰鬥選單與角色欄 |
| ![攻略](docs/screenshots/06-guide.png) | ![地圖](docs/screenshots/07-map.png) |
| 遊戲內攻略（F6，軟體世界雜誌） | 地圖疊層與攻略點（F7） |

## 遊玩

```sh
tools/play.sh   # Docker＋主機 X11；F2 切換 2 倍／3 倍，F12 中文／原版
```

需先依 [docs/re/001](docs/re/001-input-baseline.md) 準備原版，依 `tools/font-dev.sh` 建立字型。
原版附的 `KRYNN.CFG` 音效選 Roland，程式啟動時改成 AdLib；原版的預建隊伍會放進存檔目錄，主選單 LOAD SAVED GAME 選 A 即可載入。

## 授權

本專案自有的程式與文件採 [RRSAL-1.0](LICENSE)（source-available，非商業免費、商業另洽）。
授權不涵蓋原版遊戲素材，使用者須自備合法取得的原版。手札譯自 SSI 英文版 Adventurer's Journal，
攻略取自《軟體世界》雜誌第 16–18 期，著作權屬原權利人。
