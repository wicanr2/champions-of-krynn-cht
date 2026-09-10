// clickprobe 看「點完之後那塊區域的 digest 怎麼變」。
//
// 等 digest 的測試失敗時，「畫面定住在另一個值」和「畫面還在動、golden 只是
// 其中一相」從錯誤訊息上看一模一樣——兩者的處置完全不同（前者是行為變了，
// 後者是取樣點選錯）。這支把整段時間軸的 digest 攤開，才分得出來。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func region(px []uint8, x, y, w, h int) string {
	sum := sha256.New()
	for row := y; row < y+h; row++ {
		sum.Write(px[row*320+x : row*320+x+w])
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	x := flag.Int("x", 44, "點擊 X")
	y := flag.Int("y", 110, "點擊 Y")
	rw := flag.Int("w", 176, "取樣區寬")
	rh := flag.Int("h", 120, "取樣區高")
	want := flag.String("want", "", "要等的 digest")
	step := flag.Uint64("step", 200_000, "取樣間隔")
	total := flag.Uint64("total", 10_000_000, "點完之後總共跑多久")
	out := flag.String("out", "", "PNG 輸出目錄前綴")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	if err := eob1.ToLevel1Entrance(o); err != nil {
		fmt.Fprintln(os.Stderr, "到第一層入口:", err)
		os.Exit(1)
	}
	fmt.Printf("入口 digest = %s（步 %d）\n", region(o.Indexed(), 0, 0, *rw, *rh), o.Steps())
	if *out != "" {
		if err := o.WritePNG(*out + "before.png"); err != nil {
			fmt.Fprintln(os.Stderr, "存圖:", err)
		}
	}
	if err := o.Click(*x, *y, oracle.Hover(0), oracle.Hold(200_000), oracle.Settle(1_000_000)); err != nil {
		fmt.Fprintln(os.Stderr, "點擊:", err)
		os.Exit(1)
	}
	fmt.Printf("滑鼠被輪詢 %d 次\n", len(o.MousePollSteps()))
	seen := map[string]int{}
	order := []string{}
	for done := uint64(0); done < *total; done += *step {
		if err := o.Run(*step); err != nil {
			fmt.Println("run:", err)
			break
		}
		d := region(o.Indexed(), 0, 0, *rw, *rh)
		if seen[d] == 0 {
			order = append(order, fmt.Sprintf("步 %d  %s", o.Steps(), d))
		}
		seen[d]++
	}
	fmt.Printf("出現過 %d 種 digest：\n", len(seen))
	for _, line := range order {
		fmt.Println(" ", line)
	}
	if *out != "" {
		if err := o.WritePNG(*out + "after.png"); err != nil {
			fmt.Fprintln(os.Stderr, "存圖:", err)
		}
	}
	if *want != "" {
		fmt.Printf("要等的 %s 出現 %d 次\n", *want, seen[*want])
	}
}
