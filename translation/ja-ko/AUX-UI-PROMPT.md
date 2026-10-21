先 Read /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/PROMPT.md 的「先 Read」與「文風」兩節並遵守（術語表、試譯、指引）。本任務是介面輔助表，不是 bNN 批次。

翻譯四份表，輸出到 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/aux-<語言>/（UTF-8、tab 分隔，列數與順序同輸入）：

1. /home/anr2/cht/golden_box/champions_of_krynn/text/composed-templates.tsv（template 英文模板、translation 繁中、note）→ composed-templates.tsv，同三欄表頭，只改 translation 欄為目標語。{n}、{name} 等佔位符個數與名稱不變；模板是戰鬥與商店訊息的組句，譯文接在名字或數字之後要通順。note 欄照抄。
2. /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/aux-menu-masters.tsv（master、index、original_word、max_cells、zh、master_line）→ menu-masters.tsv，表頭 master	index	translation。這是底列選單逐詞的譯文：同一 master 的各詞依 index 由左到右排在原詞的位置，每詞格數（假名、漢字、韓文音節、ASCII 各一格）不得超過 max_cells；人名（例 Mark Johnson）照原文。
3. /home/anr2/cht/golden_box/champions_of_krynn/text/host-text.zh-TW.tsv（key、translation、note）→ host-text.tsv，同三欄表頭，只改 translation。{entry}、{page}、{x} 等佔位符保留。map.dirs 必須剛好四個字（北東南西）。toolbar.* 是 320 像素寬切換列上的標籤，每個 1–3 格；lang.name 日文寫「日本」、韓文寫「한국」。journal.ref、journal.refdot 這兩列譯文先照抄繁中，合併時另定。
4. 地圖名：/home/anr2/cht/golden_box/champions_of_krynn/text/map-titles.en.tsv（map、title 英文）→ map-titles.tsv，表頭 map	title。專名照術語表。

邊界：只寫 aux-<語言>/ 底下這四個檔；不改其他檔，不執行 git、docker，不安裝套件。寫完用 python3 -I 檢查列數、欄數、佔位符一致、menu-masters 的格數限制。最後回報（繁體中文）：疑點各一行。
