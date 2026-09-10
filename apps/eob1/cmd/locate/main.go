// locate 找出「目前所在方塊」與「朝向」這兩個全域。
//
// 錨點是存檔已知的 block 490；但**光是有個地方寫著 490 不算找到**，
// 記憶體裡剛好等於 490 的字多得是。判準是移動之後它要跟著變成對的值：
// 32×32 的地圖往北一格是 -32。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func snapshot(o *oracle.Oracle) []byte {
	ram := make([]byte, 0xA0000)
	for seg := 0; seg < 0xA000; seg += 0x800 {
		copy(ram[seg*16:], o.Bytes(oracle.Addr{Seg: uint16(seg), Off: 0}, 0x8000))
	}
	return ram
}

func word(ram []byte, at int) uint16 { return uint16(ram[at]) | uint16(ram[at+1])<<8 }

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	o.SetClock(0, 0, 0, 0)
	if err := eob1.ToSavedGameEntrance(o); err != nil {
		fmt.Fprintln(os.Stderr, "載入存檔:", err)
		os.Exit(1)
	}
	// 第一人稱視窗的樣子＝「隊伍在哪、朝哪」的外部訊號。
	view := func() string {
		px := o.Indexed()
		sum := 0
		for y := 0; y < 120; y++ {
			for x := 0; x < 176; x++ {
				sum = sum*31 + int(px[y*320+x])
			}
		}
		return fmt.Sprintf("%08X", uint32(sum))
	}
	at490 := view()
	fmt.Println("在 490 的視窗指紋:", at490)

	before := snapshot(o)
	var candidates []int
	for at := 0; at+1 < len(before); at++ {
		if word(before, at) == 490 {
			candidates = append(candidates, at)
		}
	}
	fmt.Printf("等於 490 的位置有 %d 處\n", len(candidates))

	o.PressKey(oracle.KeyUp)
	if err := o.Run(4_000_000); err != nil {
		fmt.Fprintln(os.Stderr, "前進:", err)
	}
	at458 := view()
	fmt.Println("在 458 的視窗指紋:", at458)
	if at458 == at490 {
		fmt.Println("視窗沒變——沒走成，後面的判斷都不算數")
		os.Exit(1)
	}
	after := snapshot(o)
	var moved []int
	for _, at := range candidates {
		if word(after, at) == 458 {
			moved = append(moved, at)
		}
	}
	fmt.Printf("移動之後變成 458 的有 %d 處：\n", len(moved))

	// 畫面指紋不可靠（火把在閃、怪物會動）。改看移動時是誰寫這幾個位址：
	// 真正的那一個由移動常式寫，副本由別的地方抄過去，呼叫端不一樣。
	type hit struct {
		from  uint32
		count int
	}
	writers := map[int]map[uint32]int{}
	for _, at := range moved {
		writers[at] = map[uint32]int{}
	}
	lo, hi := uint32(moved[0]), uint32(moved[len(moved)-1])+1
	o.OnWrite(lo, hi, func(o *oracle.Oracle, w oracle.WriteHit) {
		base := int(w.Addr)
		if _, ok := writers[base]; !ok {
			base--
			if _, ok := writers[base]; !ok {
				return
			}
		}
		writers[base][w.At]++
	})
	if err := eob1.Move(o, "forward"); err != nil && !eob1.Blocked(err) {
		fmt.Fprintln(os.Stderr, "前進:", err)
	}
	o.StopWrites()
	fmt.Println("往前走一格時，誰寫了這幾個位址：")
	for _, at := range moved {
		fmt.Printf("   DGROUP+%04X（線性 %05X）→ 值 %d\n", at-0x223E0, at, word(snapshot(o), at))
		if len(writers[at]) == 0 {
			fmt.Println("      沒有人寫——這一格是移動時不會動的副本")
			continue
		}
		for from, n := range writers[at] {
			fmt.Printf("      呼叫端 IDA %05X 寫了 %d 次\n", from, n)
		}
	}
	_ = hit{}
}
