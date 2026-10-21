任務：逐字轉錄《Champions of Krynn》英文 Adventurer's Journal 的手札（Journal Entry）與酒館傳聞（Tavern Tale）。
遊戲的防拷查驗題會問「第 M 則手札的第 N 個字」，所以轉錄必須和印刷原文逐字相同：字序、拼字、標點、連字號都不能錯，不改寫、不修正原文錯字。

資料（只讀）：
- 頁面圖：/home/anr2/cht/golden_box/champions_of_krynn/workplace/reference-src/journal-pages/p-NN.png（NN 為兩位數 PDF 頁碼，150 dpi，一張是左右兩頁）。用 Read 工具看圖，以圖為準。
- OCR 文字 A（同一份 PDF）：/home/anr2/cht/golden_box/champions_of_krynn/workplace/reference-src/txt/adventurers-journal-user.txt
- OCR 文字 B（另一份掃描）：/home/anr2/cht/golden_box/champions_of_krynn/workplace/reference-src/txt/champions-journal.txt
OCR 是多欄版面、有雜訊（條目標題前的方框被辨成 0、J、U 等），只當打字底稿，有疑問以圖為準；圖看不清再比對兩份 OCR。
看圖時可以用 Read 讀整張；字太小就用 python3 -I 搭配標準庫以外的工具都不要裝，必要時只能靠兩份 OCR 交叉判斷，並在 note 標「圖不清」。

輸出：/home/anr2/cht/golden_box/champions_of_krynn/workplace/journal/en/page-NN.tsv，UTF-8、tab 分隔，表頭：
kind	n	text	note
- kind 為 journal 或 tavern；n 為編號。
- text 是該則全文（不含「Journal Entry N」標題本身），段落之間用字面 `\n` 兩個字元分隔，其餘空白收成單一空格。
  斜體或引號照原樣保留引號字元（統一用直引號 ' 和 "）。
- 一則跨頁或跨欄時，在它開頭所在的那一頁輸出完整全文，另一頁不重複。
- note 寫不確定處（哪個字、為什麼），沒有就空白。

邊界：只寫你負責頁面的輸出檔；不改其他檔，不執行 git、docker，不安裝套件。
完成後回報：每頁輸出了哪些編號、不確定處清單。
