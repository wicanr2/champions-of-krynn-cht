任務：《Champions of Krynn》（1990 DOS，AD&D 龍槍世界）日文版／韓文版的試譯批次，給使用者確認文風後才全量翻譯（規格 docs/spec/015-multilingual.md §5）。這是使用者原版遊戲的本機在地化 patch，發行時標註機器翻譯。

先 Read：
- /home/anr2/cht/golden_box/champions_of_krynn/text/TRANSLATION-GUIDE.md（繁中版的翻譯指引；文風、片段接續、玩家輸入字、數字與名字保留等硬規則一律比照，只是目標語換成你的語言）
- 術語表 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/glossary-<語言>.tsv（en→target，專名一律照表）
- 輸入 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/trial-input.tsv（key、original 英文原文、zh 繁中定案譯文）。原文是遊戲畫面的字串，ECL 劇情常被拆成接續的片段，繁中譯文示範了片段怎麼接；原文全大寫是字型限制，不代表喊叫。
- 另譯一則手札：/home/anr2/cht/golden_box/champions_of_krynn/workplace/journal/journal.en.tsv 的 entry 51（英文原文，段落以字面 \n 分隔）。

文風：
- 日文：敘述用である調／常體（〜した、〜だ），對隊伍稱「あなたたち」；NPC 對白依角色自然說話；奇幻正式語氣，不用網路語。漢字假名比例照一般小說。
- 韓文：敘述用 해라체 平敘（〜했다、〜이다），對隊伍稱「너희」或「일행」擇一並全批一致；NPC 對白依角色；奇幻正式語氣。
- 長度：每列譯文的字數（一個假名、漢字、韓文音節、ASCII 字元各算一格）不超過英文原文的字元數；介面字串越短越好。
- 標點：日文用「、。」與「」；韓文用半形標點與 " "，句末用 . ? !。

輸出 TSV（UTF-8、tab 分隔），表頭 key	original	target	note；列順序同輸入，最後加一列 key 為 journal.51、target 為手札全文（段落用字面 \n）。note 寫沒把握處或新專名（新詞：英文=譯名）。

邊界：只寫你被指定的一個輸出檔；不改其他檔，不執行 git、docker，不安裝套件，不刪除任何東西。完成後回報：沒把握的地方、建議使用者特別看哪幾列。
