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
	tail := flag.Int("tail", 25, "印最後幾筆 int 21h")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	o.TraceInt21()
	if err := eob1.ToWestwoodLogo(o); err != nil {
		fmt.Println("標誌:", err)
		return
	}
	fmt.Println("標誌 OK")
	o.PressKey(oracle.KeyEscape)
	err = o.RunUntil(oracle.Opened("EOBDATA3.PAK"), oracle.Budget(20_000_000))
	fmt.Println("主選單:", err)
	fmt.Println("== EXEC ==")
	for _, r := range o.ExecLog() {
		fmt.Printf("  %+v\n", r)
	}
	fmt.Println("== 開過的檔 ==", o.Opened())
	trace := o.Int21Trace()
	start := len(trace) - *tail
	if start < 0 {
		start = 0
	}
	fmt.Printf("== int 21h 共 %d 筆，最後 %d 筆 ==\n", len(trace), *tail)
	for _, r := range trace[start:] {
		fmt.Printf("  step=%-10d AH=%02X AL=%02X ES:BX in=%04X:%04X out=%04X:%04X\n",
			r.Step, r.AH, r.AL, r.ESIn, r.BXIn, r.ESOut, r.BXOut)
	}
}
