// stall 追 EOB.EXE 起來之後停住的那一段：分段取樣 CS:IP 與旗標，
// 找出 IF 是在哪一步被關掉之後就再也沒開回來的。
//
// 「程式在等 tick」和「tick 送不出去」從外面看都是空轉；分段取樣把時間軸攤開，
// 才分得出是程式不想跑，還是中斷送不到它手上。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func ivt(o *oracle.Oracle, n int) oracle.Addr {
	off := o.Word(oracle.Addr{Seg: 0, Off: uint16(n * 4)})
	seg := o.Word(oracle.Addr{Seg: 0, Off: uint16(n*4 + 2)})
	return oracle.Addr{Seg: seg, Off: off}
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	chunk := flag.Uint64("chunk", 2_000_000, "每段步數")
	segs := flag.Int("segs", 30, "段數")
	from := flag.Uint64("from", 0, "這個步數之後每步都印")
	to := flag.Uint64("to", 0, "印到這個步數")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	fmt.Println("標誌:", eob1.ToWestwoodLogo(o), "步數:", o.Steps())
	o.PressKey(oracle.KeyEscape)
	o.OnTick(func(o *oracle.Oracle) {
		r := o.Regs()
		fmt.Printf("  tick #%d 步=%d 被打斷於 %04X:%04X FLAGS=%04X\n",
			o.Ticks(), o.Steps(), r.CS, r.IP, r.Flags)
	})

	// 粗掃：每 step 步取樣一次，只在 IF 或所在段改變時印，找出 IF 掉落的視窗。
	last := true
	lastCS := uint16(0)
	for i := 0; i < *segs; i++ {
		if err := o.Run(*chunk); err != nil {
			fmt.Println("run:", err)
			break
		}
		r := o.Regs()
		now := r.Flags&0x200 != 0
		if (*from > 0 && o.Steps() >= *from && o.Steps() <= *to) || now != last || r.CS != lastCS {
			fmt.Printf("  步=%d %04X:%04X FLAGS=%04X IF=%v SP=%04X ticks=%d\n",
				o.Steps(), r.CS, r.IP, r.Flags, now, r.SP, o.Ticks())
			last, lastCS = now, r.CS
		}
	}

	prevTicks := uint64(0)
	for i := 0; i < *segs; i++ {
		if err := o.Run(*chunk); err != nil {
			fmt.Println("run:", err)
			break
		}
		r := o.Regs()
		ticks, ifSet, masked := o.TimerHealth()
		fmt.Printf("步 %10d  CS:IP=%04X:%04X  FLAGS=%04X IF=%v  SS:BP=%04X:%04X  ticks=%d(+%d) 遮=%v\n",
			o.Steps(), r.CS, r.IP, r.Flags, ifSet, r.SS, r.BP, ticks, ticks-prevTicks, masked)
		prevTicks = ticks
	}

	r := o.Regs()
	fmt.Println("--- 停住時 ---")
	for _, n := range []int{0x08, 0x09, 0x1C, 0x21, 0x23, 0x24} {
		fmt.Printf("  INT %02X → %s\n", n, ivt(o, n))
	}
	fmt.Printf("  BIOS tick 0040:006C = % X\n", o.Bytes(oracle.Addr{Seg: 0x40, Off: 0x6C}, 4))
	if r.BP != 0 {
		ptrOff := o.Word(oracle.Addr{Seg: r.SS, Off: r.BP - 4})
		ptrSeg := o.Word(oracle.Addr{Seg: r.SS, Off: r.BP - 2})
		fmt.Printf("  [bp-4] 遠指標 = %04X:%04X\n", ptrSeg, ptrOff)
		fmt.Printf("  它指到的 4 bytes = % X\n", o.Bytes(oracle.Addr{Seg: ptrSeg, Off: ptrOff}, 4))
		fmt.Printf("  [bp-6]=%04X [bp-8]=%04X\n",
			o.Word(oracle.Addr{Seg: r.SS, Off: r.BP - 6}), o.Word(oracle.Addr{Seg: r.SS, Off: r.BP - 8}))
		fmt.Printf("  堆疊 %04X:%04X 起 24 bytes = % X\n", r.SS, r.SP, o.Bytes(oracle.Addr{Seg: r.SS, Off: r.SP}, 24))
	}
	fmt.Println("  未實作:", o.Unimplemented())
	h := ivt(o, 0x1C)
	fmt.Printf("  int 1Ch 常式 %s 起 64 bytes：\n", h)
	for i := 0; i < 64; i += 16 {
		fmt.Printf("    %04X: % X\n", int(h.Off)+i, o.Bytes(oracle.Addr{Seg: h.Seg, Off: h.Off + uint16(i)}, 16))
	}
	fmt.Println("  1B75:0000..0040：")
	for i := 0; i < 0x40; i += 16 {
		fmt.Printf("    %04X: % X\n", i, o.Bytes(oracle.Addr{Seg: 0x1B75, Off: uint16(i)}, 16))
	}
}
