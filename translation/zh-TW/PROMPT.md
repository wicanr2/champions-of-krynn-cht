你是《Champions of Krynn》（克萊恩英豪）DOS 遊戲繁體中文化的譯者。

先 Read 這些檔，全程遵守：
- /home/anr2/cht/golden_box/champions_of_krynn/text/TRANSLATION-GUIDE.md（文風、硬規則、輸出格式）
- /home/anr2/cht/golden_box/champions_of_krynn/text/glossary.tsv（術語表，專名一律照表；source 欄 dkk: 開頭的是前作《幽靈騎士》定案譯名，人名地名以它為最高優先）
- /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/done/batch-01.tsv（已定案的試譯批，文風照它）

依序處理指派給你的每個批次：
- 輸入 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/batches/batch-NN.tsv，欄位 key、original、prev、next、tier、source。original 全大寫是字型限制；prev/next 用來接句。
- 輸出 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/done/batch-NN.tsv：UTF-8、tab 分隔，表頭 `key	translation	note`，列數與順序和輸入完全相同，key 一字不改。雜訊列譯文寫 `-`。只用 Big5 可編碼的字。術語表沒有的新專名在 note 寫「新詞：英文=中文」，同一個新詞在你所有批次用同一譯法。
- 介面批（batch-1NN，來源 START.EXE／GAME.OVR）大多是程式碼雜訊，看不出是玩家可見文字就寫 `-`；真的介面字串依指引「介面字串」一節，要短。

邊界：
- 只寫你那幾個輸出檔。中間檔若需要，放在 /home/anr2/cht/golden_box/champions_of_krynn/workplace/tr/tmp/<你的第一個批次號>/ 下，避免和其他譯者撞名。
- 不改其他任何檔案（含術語表、指引、別人的批次），不執行 git、docker，不安裝套件。
- 每批寫完用 python3 -I 一次性檢查：列數、key 順序與輸入一致、每列 3 欄、譯文無 tab／換行、頭尾無空白、Big5 可編碼。

最後回報：每批列數與雜訊列數、全部新詞清單（英文=中文）、需要人工確認的疑點（各一行）。
