// handoff 觀察 START1.EXE 交棒之後、按 Esc 略過片頭那一段。只讀不寫。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	budget := flag.Uint64("budget", 200_000_000, "指令預算")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	fmt.Println("標誌:", eob1.ToWestwoodLogo(o))
	o.PressKey(oracle.KeyEscape)
	fmt.Println("主選單資產:", o.RunUntil(oracle.Opened("EOBDATA3.PAK"), oracle.Budget(*budget)))
	fmt.Println("開過的檔:", o.Opened())
	fmt.Println("找不到:", o.Missing())
	w, h, px := o.Screen()
	nz := 0
	for _, p := range px {
		if p != 0 {
			nz++
		}
	}
	fmt.Printf("畫面 %dx%d 非零 %d\n", w, h, nz)
	fmt.Println("沒實作:", o.Unimplemented())
	fmt.Printf("步數=%d 幀=%d 計時器已設定=%v %.2fHz 每tick步數=%d\n",
		o.Steps(), o.Frames(), o.TimerProgrammed(), o.TimerHz(), o.StepsPerTick())
	fmt.Println("port 讀取:", o.PortReads())
	fmt.Println("滑鼠呼叫:", o.MouseCalls())
}
