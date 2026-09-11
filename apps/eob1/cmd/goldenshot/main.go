// goldenshot 把等 digest 的那幾條路徑各跑一次，存下 PNG 並印出目前的 digest。
//
// 用途是「重錄 golden 之前先看畫面對不對」。digest 對不上時，錯誤訊息只告訴你
// 兩個雜湊不一樣，看不出畫面是對是錯；沒有圖就只能猜，而猜出來的 golden 會把
// 錯誤永久釘住。這支不改任何 golden，只產生證據。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

type path struct {
	name string
	run  func(*oracle.Oracle) error
}

var paths = []path{
	{"protection-cast-on-ariel", eob1.ToSavedGameProtectionCastOnAriel},
	{"protection-targeting", eob1.ToSavedGameProtectionTargeting},
	{"protection-book-closed", eob1.ToSavedGameProtectionBookClosed},
	{"level1-first-pickup", eob1.ToLevel1FirstPickup},
	{"level1-first-drop", eob1.ToLevel1FirstDrop},
	{"level1-camp-save-cancel", eob1.ToLevel1CampSaveCancel},
	{"level1-memorize-spells", eob1.ToLevel1MemorizeSpells},
	{"level1-memorize-first-spell", eob1.ToLevel1MemorizeFirstSpell},
	{"level1-memorize-return", eob1.ToLevel1MemorizeReturn},
	{"level1-rest-after-memorize", eob1.ToLevel1RestAfterMemorize},
}

func digest(px []uint8) string {
	sum := sha256.Sum256(px)
	return hex.EncodeToString(sum[:])
}

func digestRegion(px []uint8, x, y, w, h int) string {
	sum := sha256.New()
	for row := y; row < y+h; row++ {
		sum.Write(px[row*320+x : row*320+x+w])
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// stability 在原地再跑 budget 步，每 100_000 步取樣一次，回報各候選取樣區出現幾種值。
// 只出現一種的才適合當 golden；出現多種代表那塊有東西在動。
func stability(o *oracle.Oracle, budget uint64) {
	type box struct {
		name       string
		x, y, w, h int
	}
	boxes := []box{
		{"全畫面", 0, 0, oracle.Width, oracle.Height},
		{"左panel 176x120", 0, 0, 176, 120},
		{"左panel 176x96", 0, 0, 176, 96},
		{"左panel 176x168", 0, 0, 176, 168},
		{"訊息列 320x32", 0, 168, 320, 32},
		{"訊息列 320x24", 0, 168, 320, 24},
		{"訊息列 320x16", 0, 168, 320, 16},
		{"右人像 144x120", 176, 0, 144, 120},
	}
	seen := make([]map[string]bool, len(boxes))
	for i := range seen {
		seen[i] = map[string]bool{}
	}
	for step := uint64(0); step < budget; step += 100_000 {
		if err := o.Run(100_000); err != nil {
			fmt.Fprintln(os.Stderr, "穩定度取樣:", err)
			return
		}
		px := o.Indexed()
		for i, b := range boxes {
			seen[i][digestRegion(px, b.x, b.y, b.w, b.h)] = true
		}
	}
	parts := make([]string, 0, len(boxes))
	for i, b := range boxes {
		parts = append(parts, fmt.Sprintf("%s=%d種", b.name, len(seen[i])))
	}
	fmt.Printf("  穩定度（再跑 %d 步）：%s\n", budget, strings.Join(parts, "　"))
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	out := flag.String("out", "", "PNG 輸出目錄")
	only := flag.String("only", "", "只跑名字含這段的路徑")
	settle := flag.Uint64("settle", 3_000_000, "路徑結束後再跑多久，用來量取樣區穩不穩")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "要 -out")
		os.Exit(2)
	}
	failed := 0
	for _, p := range paths {
		if *only != "" && !strings.Contains(p.name, *only) {
			continue
		}
		o, err := oracle.Load(*exe, *root)
		if err != nil {
			fmt.Fprintln(os.Stderr, p.name, "load:", err)
			os.Exit(1)
		}
		runErr := p.run(o)
		// 走不到也要存圖：卡住的那一刻畫面停在哪，正是要看的東西。
		px := o.Indexed()
		file := *out + "/" + p.name + ".png"
		if runErr != nil {
			file = *out + "/" + p.name + "-stuck.png"
		}
		if err := o.WritePNG(file); err != nil {
			fmt.Fprintln(os.Stderr, p.name, "存圖:", err)
			os.Exit(1)
		}
		// 停在那裡之後再跑一段，看哪些取樣區是靜的。full-screen golden 一旦壓到會動的
		// 東西（CAMP 的人像會眨眼），就只能在某個動畫相位對上，時序一變就永遠對不上。
		stability(o, *settle)
		state := "走到了"
		if runErr != nil {
			state = "走不到：" + runErr.Error()
			failed++
		}
		// 兩種取樣區都印：測試依狀態用 176×120（視窗）或 176×96（視窗扣掉訊息列）。
		fmt.Printf("%-26s 步=%d %s\n  全畫面    %s\n  176x120   %s\n  176x96    %s\n  176x168   %s\n  圖        %s\n",
			p.name, o.Steps(), state, digest(px),
			digestRegion(px, 0, 0, 176, 120), digestRegion(px, 0, 0, 176, 96),
			digestRegion(px, 0, 0, 176, 168), file)
		fmt.Printf("  訊息列32  %s\n  訊息列24  %s\n  訊息列16  %s\n",
			digestRegion(px, 0, 168, 320, 32), digestRegion(px, 0, 168, 320, 24),
			digestRegion(px, 0, 168, 320, 16))
		o.Close()
	}
	if failed != 0 {
		os.Exit(1)
	}
}
