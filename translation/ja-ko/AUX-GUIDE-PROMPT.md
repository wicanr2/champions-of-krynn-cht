先 Read /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/PROMPT.md 的「先 Read」與「文風」兩節並遵守。本任務是遊戲內攻略面板與地圖攻略點，從繁中翻譯（使用者指示）。

1. /home/anr2/cht/golden_box/champions_of_krynn/text/walkthrough.zh-TW.tsv（chapter、title、body、source；《軟體世界》雜誌攻略的繁中整理，段落以字面 \n 分隔）→ walkthrough.tsv，同四欄表頭，列數與順序相同。title、body 譯成目標語；source 欄日文寫「ソフトワールド誌 16号 p.38」這種格式，韓文寫「소프트월드 16호 p.38」，頁碼照抄。攻略的口語語氣可保留，專名照術語表；括號裡的英文原名照留。
2. /home/anr2/cht/golden_box/champions_of_krynn/text/map-guide.zh-TW.tsv（map、n、x、y、label、summary、source）→ map-guide.tsv，同七欄表頭，列數與順序相同，只譯 label 與 summary。label 是地圖上的短標籤，最多 8 格（假名、漢字、韓文音節各一格）。

輸出放 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/aux-<語言>/。
邊界：只寫這兩個檔；中間檔放 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/tmp/guide-<語言>/。不改其他檔，不執行 git、docker，不安裝套件。寫完用 python3 -I 檢查列數、欄數、無 tab、label 長度。最後回報（繁體中文）：疑點各一行。
