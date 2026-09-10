// handoff 觀察 EOB1 從 START1.EXE 到主選單這一段。只讀不寫。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func nonZero(o *oracle.Oracle) int {
	_, _, px := o.Screen()
	n := 0
	for _, p := range px {
		if p != 0 {
			n++
		}
	}
	return n
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	escape := flag.Bool("esc", true, "按 Esc 略過片頭")
	budget := flag.Uint64("budget", 300_000_000, "指令預算")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	o.Type(eob1.LauncherInput)
	fmt.Println("等 EOBDATA4.PAK:", o.RunUntil(oracle.Opened("EOBDATA4.PAK"), oracle.Budget(60_000_000)))
	fmt.Println("  畫面非零:", nonZero(o), "步數:", o.Steps())
	fmt.Println("等畫面靜止:", o.RunUntil(oracle.ScreenIdle(1_000_000), oracle.Budget(60_000_000)))
	fmt.Println("  畫面非零:", nonZero(o), "步數:", o.Steps())
	o.TraceInt21()
	before := len(o.Int21Trace())
	if *escape {
		for i := 0; i < 1; i++ {
			o.PressKey(oracle.KeyEscape)
			err := o.RunUntil(oracle.Opened("EOBDATA3.PAK"), oracle.Budget(*budget/6))
			fmt.Printf("  Esc#%d 之後：非零=%d 步數=%d err=%v\n", i+1, nonZero(o), o.Steps(), err != nil)
			if err == nil {
				break
			}
		}
	}
	fmt.Println("  畫面非零:", nonZero(o), "步數:", o.Steps())
	tr := o.Int21Trace()
	fmt.Printf("== Esc 之後的 int 21h（共 %d 筆，前 40 筆）==\n", len(tr)-before)
	for i, r := range tr[before:] {
		if i >= 40 {
			break
		}
		fmt.Printf("  step=%-10d AH=%02X AL=%02X\n", r.Step, r.AH, r.AL)
	}
	fmt.Println("port:", o.PortReads())
	tick := o.Bytes(oracle.Addr{Seg: 0x0040, Off: 0x006C}, 4)
	fmt.Printf("BIOS tick 0040:006C = % X\n", tick)
}
