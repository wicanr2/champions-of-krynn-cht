# 克萊恩英豪（Champions of Krynn）繁體中文化

以 [dosgolem](https://github.com/wicanr2/dosgolem) 執行 SSI 1990 年 DOS 版《Champions of Krynn》v1.2，
在執行期攔截原版輸出並覆繪繁體中文。原版遊戲、規則與存檔保持原樣，需自備原版。

現況見 [CONTEXT.md](CONTEXT.md)，工作項目見
[GitHub Issues](https://github.com/wicanr2/champions-of-krynn-cht/issues)，協作規則見 [AGENTS.md](AGENTS.md)。

## 故事背景

本作的舞台是《龍槍》系列小說與 AD&D 共用的世界克萊恩（Krynn），也是 SSI 克萊恩三部曲的第一部，續作是《幽靈騎士》（Death Knights of Krynn）。
龍槍戰爭剛結束，黑暗之后塔克西絲的龍軍在奈拉卡潰敗，可是龍人殘軍、邪惡牧師與各地的怪物並沒有消失。
索蘭尼克騎士在邊境設下幾座屯墾區，招募冒險者守住戰後殘破的土地。

你們的隊伍來到第一屯墾區報到。卡爾爵士熱情接待，派下第一項任務：巡視東北方的斯拉托，有任何可疑的動靜立刻回報。
一出屯墾區，隊伍就撞見龍人屠殺商隊，倖存的龍人搶走一本書逃走。回到屯墾區時，指揮官已被冒名頂替，屍體現出西維克龍人的原形。
在斯拉托，隊伍聽見斷斷續續的回聲：「麥坦尼的計畫……神殿中的蛋……死亡騎士……」。麥坦尼這個名字從此一再出現，連科南都被他和新造出的大批龍人占領（手札）。
隊伍要走遍蓋加斯、傑勒克、奈拉卡與桑克申，追查他的陰謀。

遊戲中的長篇劇情不印在畫面上，而是以「參見手札第 N 則」指向隨盒附的《冒險者手札》。本中文化把 88 則手札與 57 則酒店傳聞譯成中文，引用時直接在遊戲內開啟面板，不必另翻手冊。

出處：遊戲開場劇情與手札（`text/ecl-text.zh-TW.tsv`、`text/journal.zh-TW.tsv`）、《軟體世界》第 16–18 期攻略（白怒火、南帝），內容經改寫。

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

## 語言

繁體中文為人工翻譯。簡體中文由繁體中文以 OpenCC 轉換。日文、韓文為機器翻譯，未經母語者校對。英文版保留原文，以 Courier Prime 重繪，並可在遊戲內閱讀英文手札。遊戲中按 F8 切換語言，規格見 [docs/spec/015](docs/spec/015-multilingual.md)。

## 授權

本專案自有的程式與文件採 [RRSAL-1.0](LICENSE)（source-available，非商業免費、商業另洽）。
授權不涵蓋原版遊戲素材，使用者須自備合法取得的原版。手札譯自 SSI 英文版 Adventurer's Journal，
攻略取自《軟體世界》雜誌第 16–18 期，著作權屬原權利人。
