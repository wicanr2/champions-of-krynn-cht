// handoff 觀察 EOB1 從 START1.EXE 到主選單這一段，只讀不寫。
//
// 錨在「載入器開 EOB.EXE」那一刻：它接著就用 AH=4Ah 決定要把自己的區塊設多大，
// 而那個決定看的是環境區塊有幾段（PSP+2Ch 指到的區塊，大小在它前一段的 MCB）。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

// dumpEnv 印出目前行程的環境區塊與它的 MCB。
func dumpEnv(o *oracle.Oracle, tag string) {
	psp := uint16(0x0100)
	env := o.Word(oracle.Addr{Seg: psp, Off: 0x2C})
	mcb := o.Bytes(oracle.Addr{Seg: env - 1, Off: 0}, 16)
	fmt.Printf("  [%s] PSP+2Ch=%04X 它的 MCB(%04X)：簽章=%c 擁有者=%02X%02X 段數=%02X%02X\n",
		tag, env, env-1, mcb[0], mcb[2], mcb[1], mcb[4], mcb[3])
	fmt.Printf("       環境前 48 bytes：% X\n", o.Bytes(oracle.Addr{Seg: env, Off: 0}, 48))
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	budget := flag.Uint64("budget", 60_000_000, "指令預算")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	o.OnFileOpen(func(name string) {
		if strings.Contains(strings.ToUpper(name), "EOB.EXE") {
			dumpEnv(o, "開 "+name)
		}
	})
	fmt.Println("標誌:", eob1.ToWestwoodLogo(o), "步數:", o.Steps())
	dumpEnv(o, "標誌後")
	o.PressKey(oracle.KeyEscape)
	fmt.Println("主選單:", o.RunUntil(oracle.Opened("EOBDATA3.PAK"), oracle.Budget(*budget)) == nil,
		"步數:", o.Steps(), "非零:", nonZero(o))
	dumpEnv(o, "結束時")
	ticks, ifSet, masked := o.TimerHealth()
	fmt.Printf("計時器：送出 %d 次，IF=%v，IRQ0 被遮=%v，IRQ0Every=%d\n", ticks, ifSet, masked, o.StepsPerTick())
	stop := o.IP()
	fmt.Printf("停在 %s；前後 96 bytes：\n", stop)
	base := int(stop.Off) - 32
	if base < 0 {
		base = 0
	}
	for off := base; off < base+96; off += 16 {
		fmt.Printf("  %04X: % X\n", off, o.Bytes(oracle.Addr{Seg: stop.Seg, Off: uint16(off)}, 16))
	}
	fmt.Printf("BIOS tick 0040:006C = % X\n", o.Bytes(oracle.Addr{Seg: 0x0040, Off: 0x006C}, 4))
	fmt.Println("AH=4Ah 紀錄（含呼叫端）：")
	for _, r := range o.Resizes() {
		fmt.Printf("  seg=%04X want=%04X before=%04X after=%04X freeSeg=%04X OK=%v 呼叫端=%04X:%04X\n",
			r.Seg, r.Want, r.Before, r.After, r.FreeSeg, r.OK, r.CS, r.IP)
	}
	fmt.Println("overlay:")
	for _, ov := range o.Overlays() {
		fmt.Printf("  %-24s 載入段=%04X 參數區=%04X:%04X\n", ov.Name, ov.Seg, ov.PBSeg, ov.PBOff)
	}
}
