// tolevel2 把原版帶到第二層。
//
// 走法：照 `docs/spec/014`「固定狀態，不要靠亂數」，直接把隊伍位置寫到樓梯前，
// 再用原版自己的移動走進轉場——不注入亂數，也不跳過轉場本身的程式碼。
// 之後在第二層 CAMP 存檔，那個存檔就是原版與 remake 共同的起點。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

// 三個同步的「目前所在方塊」全域，由「移動之後跟著變」加上「移動時各有不同的
// 常式寫它們」量出來。三個都要寫——只寫一個的話，其餘兩個會把它蓋回去。
var blockVars = []uint32{0xA9E8, 0xFBCA, 0xFC58}

const dgroupInModule = 0x312E0 - 0x10000

func dgroup(o *oracle.Oracle) (uint32, error) {
	for _, ov := range o.Overlays() {
		if ov.Name != "" && (ov.Name == "EOB.EXE" || len(ov.Name) >= 7 && ov.Name[len(ov.Name)-7:] == "EOB.EXE") {
			return uint32(ov.Seg)*16 + dgroupInModule, nil
		}
	}
	return 0, fmt.Errorf("找不到 EOB.EXE 的 overlay 載入紀錄")
}

func addr(base, off uint32) oracle.Addr {
	at := base + off
	return oracle.Addr{Seg: uint16(at >> 4), Off: uint16(at & 0xF)}
}

func readBlock(o *oracle.Oracle, base uint32) []uint16 {
	out := make([]uint16, 0, len(blockVars))
	for _, off := range blockVars {
		out = append(out, o.Word(addr(base, off)))
	}
	return out
}

func setBlock(o *oracle.Oracle, base uint32, block uint16) {
	for _, off := range blockVars {
		o.SetWord(addr(base, off), block)
	}
}

// campSave 走 CAMP → Game Options → Save Game → Yes。
// 按鍵序列取自既有的 ToLevel1CampSaveWritten 那一串。
func campSave(o *oracle.Oracle) error {
	o.PressKey(oracle.KeyEscape) // 關掉可能開著的物品欄
	if err := o.Run(1_500_000); err != nil {
		return err
	}
	if err := o.Click(304, 187, oracle.Hover(0), oracle.Hold(200_000), oracle.Settle(2_000_000)); err != nil && !eob1.Blocked(err) {
		return fmt.Errorf("點 CAMP：%w", err)
	}
	for i := 0; i < 5; i++ {
		o.PressKey(oracle.KeyDown)
		if err := o.Run(300_000); err != nil {
			return err
		}
	}
	o.PressKey(oracle.KeyEnter)
	if err := o.Run(3_000_000); err != nil {
		return err
	}
	o.PressKey(oracle.KeyDown)
	if err := o.Run(300_000); err != nil {
		return err
	}
	o.PressKey(oracle.KeyEnter)
	if err := o.Run(3_000_000); err != nil {
		return err
	}
	o.PressKey(oracle.KeyEnter) // 確認視窗的 Yes
	return o.Run(6_000_000)
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	start := flag.Int("start", 752, "先把隊伍放到哪一格")
	steps := flag.Int("steps", 2, "接著往前走幾格")
	turns := flag.Int("turns", 1, "先右轉幾次（存檔是朝北）")
	shots := flag.String("shots", "", "PNG 輸出目錄")
	answers := flag.String("answers", "", "手冊查字要試的答案，逗號分隔")
	scratch := flag.String("scratch", "", "可寫的暫存層目錄；設了就在第二層 CAMP 存檔")
	flag.Parse()
	o, err := eob1.Load(*exe, *root)
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
	// ⚠ 可寫層要**載入之後**才開。開在前面的話遊戲會在載入途中寫 LEVELS.TMP，
	// 走的是另一條路徑，而 ToSavedGameEntrance 的畫面 digest 是唯讀跑出來的，
	// 於是那個閘門會超時——症狀看起來像遊戲當掉。
	if *scratch != "" {
		o.SetScratch(*scratch)
		if err := o.AllowFileWrites("EOBDATA.SAV", "LEVELS.TMP"); err != nil {
			fmt.Fprintln(os.Stderr, "開放寫入:", err)
			os.Exit(1)
		}
	}
	base, err := dgroup(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lock, err := eob1.LockPartyVitals(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "上鎖:", err)
		os.Exit(1)
	}
	seedBefore, _ := eob1.RandomState(o)
	fmt.Printf("起點 block=%v  RANDSEED=%08X\n", readBlock(o, base), seedBefore)

	setBlock(o, base, uint16(*start))
	for i := 0; i < *turns; i++ {
		if err := eob1.Move(o, "turnRight"); err != nil && !eob1.Blocked(err) {
			fmt.Fprintln(os.Stderr, "右轉:", err)
		}
	}
	fmt.Printf("放到 %d 並轉向之後 block=%v\n", *start, readBlock(o, base))
	if *shots != "" {
		_ = o.WritePNG(*shots + "/tolevel2-start.png")
	}
	for i := 0; i < *steps; i++ {
		err := eob1.Move(o, "forward")
		blocked := err != nil && eob1.Blocked(err)
		if err != nil && !blocked {
			fmt.Fprintln(os.Stderr, "前進:", err)
			break
		}
		fmt.Printf("第 %d 步%s → block=%v\n", i+1,
			map[bool]string{true: "（畫面沒動）", false: ""}[blocked], readBlock(o, base))
		if *shots != "" {
			_ = o.WritePNG(fmt.Sprintf("%s/tolevel2-step%d.png", *shots, i+1))
		}
	}
	// 手冊查字。題目是亂數選的，(行,字) 可能對應到不只一個符號，
	// 所以候選逐個試——原版本來就允許重試，答錯不會直接關掉。
	if *answers != "" {
		for _, word := range strings.Split(*answers, ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}
			o.TypeBoth(word)
			o.PressKey(oracle.KeyEnter)
			if err := o.Run(6_000_000); err != nil {
				fmt.Fprintln(os.Stderr, "送答案:", err)
			}
			fmt.Printf("試 %-12s → block=%v\n", word, readBlock(o, base))
			if *shots != "" {
				_ = o.WritePNG(fmt.Sprintf("%s/tolevel2-answer-%s.png", *shots, word))
			}
		}
	}

	if *scratch != "" {
		if err := campSave(o); err != nil {
			fmt.Fprintln(os.Stderr, "存檔:", err)
		} else {
			fmt.Println("已在目前所在樓層 CAMP 存檔")
		}
		if *shots != "" {
			_ = o.WritePNG(*shots + "/tolevel2-saved.png")
		}
	}

	seedAfter, _ := eob1.RandomState(o)
	fmt.Println("收據：")
	fmt.Printf("  寫入隊伍位置 %d（三個同步全域）\n", *start)
	fmt.Printf("  鎖 %d 人的 HP 與食物，寫入 %d 次\n", lock.Members, lock.Writes)
	fmt.Printf("  RANDSEED %08X → %08X\n", seedBefore, seedAfter)
	fmt.Printf("  結束步數 %d\n", o.Steps())
}
