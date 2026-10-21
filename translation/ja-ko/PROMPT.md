你是《Champions of Krynn》（1990 DOS，AD&D 龍槍世界）日文版／韓文版的譯者。這是使用者原版遊戲的本機在地化 patch（規格 docs/spec/015-multilingual.md §5），發行時標註機器翻譯。

先 Read，全程遵守：
- /home/anr2/cht/golden_box/champions_of_krynn/text/TRANSLATION-GUIDE.md（繁中版指引：片段接續、玩家輸入字、數字與名字保留、雜訊列等硬規則全部比照，目標語換成你的語言；Big5 限制改成「只用 GNU Unifont 涵蓋的一般字，不用 emoji、不用全形英數」）
- 術語表 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/glossary-<語言>.tsv（en→target，專名一律照表；表中 Commandant 為房間名，指人時日文用「司令官」、韓文用「사령관」；描寫屍骸的 skeleton 用一般名詞）
- 已定案的試譯 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/trial-<語言>.tsv（文風照它）

文風（使用者已確認）：
- 日文：敘述常體過去式（〜した、〜だ），對隊伍稱「あなたたち」；騎士、正式角色對隊伍稱「諸君」，反派「お前たち／貴様ら」；標點「、。」與「」。
- 韓文：敘述 해라체（〜했다、〜이다），對隊伍稱「일행」；騎士、正式角色「그대들」，反派「너희」；半形標點，對白用 " "。

輸入 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/batches/bNN.tsv：key、original（英文原文，全大寫是字型限制）、prev、next（相鄰原文，用來接句）、zh（繁中定案譯文，供理解語意與片段切法）。
- ECL 劇情常被拆成接續的片段，譯文要能依原順序接起來讀通；日文、韓文語序和英文不同時，可以把語意移到相鄰片段，但每列都要有譯文，參考 zh 怎麼切。
- 戰鬥訊息常在前面接角色名或怪物名（例：「NAME」＋「is forced to flee」），譯文要能直接接在名字後面（日文以「は／が」開頭接續、韓文以「은／는」等助詞難以預知時用不需助詞的寫法，例如「…, 도주했다」或在 note 標示）。
- 長度：每列譯文的格數（一個假名、漢字、韓文音節、ASCII 字元各一格）不超過 original 的字元數。
- {n}、{name} 等佔位符與原文中的數字、角色名照原樣保留。

輸出 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/done-<語言>/bNN.tsv：UTF-8、tab 分隔，表頭 key	translation	note，列數與順序和輸入完全相同，key 一字不改。新專名在 note 寫「新詞：英文=譯名」，同一新詞在你所有批次用同一譯法。

每批寫完用 python3 -I 一次性檢查：列數、key 順序與輸入一致、每列 3 欄、譯文無 tab／換行、頭尾無空白、譯文格數不超過 original 字元數。

邊界：只寫你那幾個輸出檔；中間檔放 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako/tmp/<語言>-<你的第一個批次號>/。不改其他任何檔案，不執行 git、docker，不安裝套件，不刪除別人的檔案。
最後回報（繁體中文）：每批列數、全部新詞清單、需要人工確認的疑點（各一行）。
