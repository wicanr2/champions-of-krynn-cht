任務：為《Champions of Krynn》（1990 DOS，SSI，AD&D 龍槍 Dragonlance 世界）的日文版／韓文版建立術語表。這是使用者自己原版遊戲的本機在地化 patch（規格 docs/spec/015-multilingual.md §5），日韓譯文標註為機器翻譯。

先 Read：
- /home/anr2/cht/golden_box/champions_of_krynn/text/glossary.tsv（en、zh、category、source、note；繁中定案譯名，供理解語意）
- /home/anr2/cht/golden_box/champions_of_krynn/text/TRANSLATION-GUIDE.md（繁中翻譯指引，文風與硬規則可比照）

規則：
- 每個 en 詞給一個目標語譯名，整個遊戲只用這一種。
- 專有名詞（category 為 person、place、race、monster、class、spell、item）：
  - 龍槍小說／TRPG 已有正式譯名（日文：富士見書房《ドラゴンランス戦記》等；韓文：正式出版譯本）且你能找到可引用的出處時採用，basis 寫 official，note 寫出處（書名或網址）。可以用 WebSearch／WebFetch 查證；查不到就不要憑記憶當成正式譯名。
  - AD&D 通用詞（Magic-User、Cleric、Fighter、Paladin、Ranger、Thief、各法術、怪物）若有日文或韓文 D&D 正式版常見譯法且查得到，同樣 basis=official。
  - 其餘人名、地名：日文用片假名音譯、韓文用韓文音譯，basis 寫 translit。有意義的地名（Outpost、Tomb 等）可意譯，basis 寫 meaning。
- 介面與機制詞（category 為 ui、attribute、other）：用該語言 RPG 慣用說法，簡短；AC、HP、THAC0 等縮寫保留英文。basis 寫 ui。
- 長度：介面詞要短，選單一個詞通常不超過原文字母數的一半（一個假名、漢字、韓文音節各佔一格，原文一個字母佔一格）。
- 只用 GNU Unifont 涵蓋的字（一般常用字都可以），不用 emoji、不用全形英數。

輸出 TSV（UTF-8、tab 分隔），表頭：
en	target	basis	note
列數、順序與 glossary.tsv 相同（en 欄照抄）。

邊界：只寫你被指定的一個輸出檔；不改其他檔，不執行 git、docker，不安裝套件，不刪除任何東西。完成後回報：official 採用了幾個與出處類型、translit／meaning 各幾個、沒把握的詞清單。
