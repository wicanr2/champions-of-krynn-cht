背景：使用者擁有《Champions of Krynn》（克萊恩英豪，1990 DOS）原版與其英文 Adventurer's Journal，正在自己的本機做繁體中文化 patch。
遊戲進行中會要求玩家「去讀手札第 N 則／酒館傳聞第 N 則」，中文化版要在遊戲畫面內顯示這些條目的中文譯文（前作《幽靈騎士》中文化已用同樣做法）。
你的工作是翻譯，不是轉錄英文：輸出只含中文譯文，不輸出英文原文。

先 Read：
- /home/anr2/cht/golden_box/champions_of_krynn/text/TRANSLATION-GUIDE.md（文風與硬規則）
- /home/anr2/cht/golden_box/champions_of_krynn/text/glossary.tsv（專名一律照表；source 欄 dkk: 開頭是前作定案，最高優先）

資料（只讀）：
- 頁面圖 /home/anr2/cht/golden_box/champions_of_krynn/workplace/reference-src/journal-pages/p-NN.png（150 dpi，一張是左右兩頁）。用 Read 看圖。
- OCR 文字（多欄、有雜訊，只作輔助）：/home/anr2/cht/golden_box/champions_of_krynn/workplace/reference-src/txt/adventurers-journal-user.txt

輸出：/home/anr2/cht/golden_box/champions_of_krynn/workplace/journal/zh/<你的檔名>.tsv，UTF-8、tab 分隔，表頭：
entry	title	body	note
- 手札：entry 為編號 N，title 寫「旅行手札第 N 則」。酒館傳聞：entry 為 1000+N，title 寫「酒館傳聞第 N 則」。
- body 是完整譯文，段落之間用字面 `\n`（兩個字元）分隔。對白用「」。只用 Big5 可編碼的字，不含 tab。
- 只有地圖沒有文字的條目：body 寫「（本則為地圖，見原版手札。）」。地圖上的文字標籤不譯。
- 條目內若有「Tale of …」這類小標題，放在 body 第一段。
- 每則譯文以 700 個中文字為上限（遊戲面板 3 頁）；原文很長時保持完整語意，不得刪情節。
- 新專名在 note 寫「新詞：英文=中文」。

邊界：只寫你的輸出檔；不改其他檔，不執行 git、docker，不安裝套件。寫完用 python3 -I 檢查欄數、編號齊全、Big5 可編碼。
完成後回報：輸出了哪些編號、新詞清單、疑點。
