先 Read /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/PROMPT.md 的「先 Read」與「文風」兩節並遵守。本任務是遊戲內手札面板的譯文。

輸入 /home/anr2/cht/golden_box/champions_of_krynn/workplace/journal/journal.en.tsv（entry、title、body、source；英文 Adventurer's Journal 逐字稿，段落以字面 \n 分隔；entry 1–88 是手札，1001–1057 是酒店傳聞）。參考 /home/anr2/cht/golden_box/champions_of_krynn/text/journal.zh-TW.tsv（同 entry 的繁中定案譯文，對照語意與專名）。

輸出 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/aux-<語言>/journal.tsv，表頭 entry	title	body	source，列數與順序同輸入：
- title：日文「旅の手記 第 N 項」／酒店傳聞「酒場の噂 第 N 項」；韓文「여행 수기 제N항」／「주점 소문 제N항」（N 為 1–88 或 1–57）。
- body：全文翻譯，段落仍以字面 \n 分隔，不含 tab。地圖條目（英文為 "(This entry is a map. See the printed journal.)"）譯成同義的一句。
- source 照抄。
- 這是印刷手冊上的書信與記錄，敘述文體可依原文（書信用書信語氣），不受「敘述常體」限制。

邊界：只寫這一個輸出檔；中間檔放 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/tmp/journal-<語言>/。不改其他檔，不執行 git、docker，不安裝套件。寫完用 python3 -I 檢查列數、entry 順序、欄數、無 tab。最後回報（繁體中文）：新詞與疑點。
