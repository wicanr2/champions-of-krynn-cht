// rng 觀測 EOB1 原版的亂數狀態。
//
// EOB.EXE 的亂數就是 Borland Turbo C 的 `rand()`：
//
//	155A6  srand：RANDSEED = 參數（低字），高字清零
//	155B7  rand ：RANDSEED = RANDSEED*0x015A4E35 + 1；回傳高字 & 0x7FFF
//
// 兩個字在 DGROUP 的 8784h／8786h（IDA 39A64／39A66），初值 1。
//
// ⚠ **「有沒有被 srand 過」不能只看值。** 從 1 往前推 N 步就能判定：
// 推得到表示只是被消費了 N 次、從來沒重新播種；推不到才是有人播過種。
// 少了這一步，「被播過種」與「被叫了很多次」長得一模一樣。
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

// dgroupInModule 是 DGROUP 在 EOB.EXE 載入模組裡的偏移。
// IDA 線性 31 2E0h 減掉 IDA 的載入基底 10000h。
const dgroupInModule = 0x312E0 - 0x10000

// randSeedOff 是 RANDSEED 在 DGROUP 裡的偏移（低字）。
const randSeedOff = 0x8784

func lcg(s uint32) uint32 { return s*0x015A4E35 + 1 }

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	stage := flag.String("stage", "title", "停在哪：title／stats／alfa／level1")
	walk := flag.Uint64("walk", 50_000_000, "從初值 1 最多往前推幾步找對應")
	dump := flag.Int("dump", 0, "把前幾筆 rand 回傳值與 roll 呼叫印出來當 fixture")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()

	steps := map[string]func(*oracle.Oracle) error{
		"title":  eob1.ToTitleMenu,
		"stats":  eob1.ToFirstCharacterStats,
		"alfa":   eob1.ToFirstCharacterALFA,
		"level1": eob1.ToLevel1Entrance,
	}
	run, ok := steps[*stage]
	if !ok {
		fmt.Fprintln(os.Stderr, "不認得的 stage:", *stage)
		os.Exit(2)
	}
	// EOB.EXE 固定被 chain 載到 0110h（見 docs/re/eob1-dosgolem-handoff-255.md），
	// 所以位址算得出來；但 INTRO.EXE 也載在同一處，要等它換人才算數。
	const eobBase = 0x0110 * 16
	live := false
	o.OnFileOpen(func(name string) {
		if strings.Contains(strings.ToUpper(name), "EOB.EXE") {
			live = true
		}
	})
	ida := func(linear uint32) oracle.Addr {
		run := linear - 0x10000 + eobBase
		return oracle.Addr{Seg: uint16(run >> 4), Off: uint16(run & 0xF)}
	}
	rands := 0
	seed := uint32(1)
	seeded := false
	var outs []uint16
	type rollCall struct {
		times, pips, add int
		got              int
	}
	var rolls []rollCall
	var pending *rollCall
	type rangeCall struct {
		low, high int
		got       int
	}
	var ranges []rangeCall
	var pendingRange *rangeCall
	callers := map[uint32]int{}
	o.OnCall(ida(0x155A6), func(o *oracle.Oracle) {
		if !live {
			return
		}
		seed, seeded = uint32(o.Arg(0)), true
		// EOB.EXE 就位之前，INTRO.EXE 的碼會跑過同一批線性位址，
		// 攔到的都是雜訊。**播種是這一輪的起點**，在這裡把帳歸零。
		rands, outs, rolls, pending = 0, nil, nil, nil
		callers = map[uint32]int{}
		ranges, pendingRange = nil, nil
		c := o.Caller()
		fmt.Printf("srand(%04X)  呼叫端 %04X:%04X（IDA %05X）  步數=%d  "+
			"此前 rand 被叫 %d 次  BIOS tick=%d\n",
			o.Arg(0), c.Seg, c.Off, uint32(c.Linear())-eobBase+0x10000, o.Steps(), rands, o.Ticks())
	})
	o.OnCall(ida(0x155B7), func(o *oracle.Oracle) {
		if live {
			rands++
			c := o.Caller()
			callers[uint32(c.Linear())-eobBase+0x10000]++
		}
	})
	// 155DC 是 rand 的 retf，AX 已經是遮好的回傳值。
	o.OnCall(ida(0x155DC), func(o *oracle.Oracle) {
		if live {
			outs = append(outs, o.AX())
		}
	})
	// 18718 是骰子函式入口，參數是三個 char（各佔一個字）。
	o.OnCall(ida(0x18718), func(o *oracle.Oracle) {
		if !live {
			return
		}
		rolls = append(rolls, rollCall{
			times: int(int8(o.Arg(0))), pips: int(int8(o.Arg(1))), add: int(int8(o.Arg(2))),
		})
		pending = &rolls[len(rolls)-1]
	})
	// 2AAA5 是範圍取值 rnd(low, high)，2AADF 之後是它的收尾。
	o.OnCall(ida(0x2AAA5), func(o *oracle.Oracle) {
		if !live {
			return
		}
		ranges = append(ranges, rangeCall{low: int(int16(o.Arg(0))), high: int(int16(o.Arg(1)))})
		pendingRange = &ranges[len(ranges)-1]
	})
	o.OnCall(ida(0x18780), func(o *oracle.Oracle) {
		if live && pending != nil {
			pending.got = int(int16(o.AX()))
			pending = nil
		}
	})

	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, *stage+":", err)
		os.Exit(1)
	}

	var base uint32
	for _, ov := range o.Overlays() {
		if strings.Contains(strings.ToUpper(ov.Name), "EOB.EXE") {
			base = uint32(ov.Seg) * 16
		}
	}
	if base == 0 {
		fmt.Fprintln(os.Stderr, "沒看到 EOB.EXE 的 overlay 載入紀錄")
		os.Exit(1)
	}
	at := base + dgroupInModule + randSeedOff
	seg, off := uint16(at>>4), uint16(at&0xF)
	lo := uint32(o.Word(oracle.Addr{Seg: seg, Off: off}))
	hi := uint32(o.Word(oracle.Addr{Seg: seg, Off: off + 2}))
	state := hi<<16 | lo

	fmt.Printf("stage=%s 步數=%d\n", *stage, o.Steps())
	fmt.Printf("EOB.EXE 載入段=%04X  DGROUP 線性=%05X  DS 暫存器=%04X\n",
		base/16, base+dgroupInModule, o.DSReg())
	fmt.Printf("RANDSEED = %08X（低字 %04X 高字 %04X）  rand 被叫 %d 次\n", state, lo, hi, rands)

	from := "初值 1（沒人播過種）"
	if seeded {
		from = fmt.Sprintf("srand 播下的 %08X", seed)
	}
	// 逐次驗證輸出：從播下的種子遞推，每一步的 (s>>16)&0x7FFF 要等於攔到的回傳值。
	if len(outs) > 0 {
		v, bad := seed, -1
		for i, got := range outs {
			v = lcg(v)
			if want := uint16((v >> 16) & 0x7FFF); want != got {
				bad = i
				fmt.Printf("   第 %d 次 rand 回 %04X，遞推算出來是 %04X\n", i, got, want)
				break
			}
		}
		if bad < 0 {
			fmt.Printf("rand 回傳值逐次核對：%d 次全部相符\n", len(outs))
		}
	}
	fmt.Println("rand 的呼叫端（IDA 位址 → 次數）：")
	keys := make([]uint32, 0, len(callers))
	for k := range callers {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return callers[keys[i]] > callers[keys[j]] })
	for _, k := range keys {
		fmt.Printf("   %05X → %d 次\n", k, callers[k])
	}
	if len(ranges) > 0 {
		fmt.Printf("rnd(low,high) 被叫 %d 次：", len(ranges))
		for i, r := range ranges {
			if i < 12 {
				fmt.Printf(" (%d,%d)", r.low, r.high)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("rnd(low,high) 一次都沒被叫到")
	}
	_ = pendingRange
	// 逐次驗證骰子映射。
	if len(rolls) > 0 {
		v := seed
		ok, checked := true, 0
		for _, r := range rolls {
			total := 0
			if r.pips > 0 && r.times > 0 {
				for i := 0; i < r.times; i++ {
					v = lcg(v)
					x := int((v >> 16) & 0x7FFF)
					d := r.pips*x/32767 + 1
					if d > r.pips {
						d -= r.pips
					}
					total += d
				}
			}
			checked++
			if total+r.add != r.got {
				fmt.Printf("   roll(%d,%d,%d) 原版回 %d，公式算出 %d\n",
					r.times, r.pips, r.add, r.got, total+r.add)
				ok = false
				break
			}
		}
		if ok {
			fmt.Printf("骰子映射逐次核對：%d 次呼叫全部相符\n", checked)
		}
	}

	if *dump > 0 {
		n := *dump
		if n > len(outs) {
			n = len(outs)
		}
		fmt.Printf("前 %d 次 rand 回傳值（種子 %04X）：\n", n, seed)
		for i := 0; i < n; i += 8 {
			end := i + 8
			if end > n {
				end = n
			}
			line := ""
			for _, v := range outs[i:end] {
				line += fmt.Sprintf(" 0x%04X,", v)
			}
			fmt.Println("   " + line)
		}
		m := *dump
		if m > len(rolls) {
			m = len(rolls)
		}
		fmt.Printf("前 %d 次 roll(times,pips,add) → 回傳：\n", m)
		for _, r := range rolls[:m] {
			fmt.Printf("    {%d, %d, %d, %d},\n", r.times, r.pips, r.add, r.got)
		}
	}

	s := seed
	for n := uint64(0); n <= *walk; n++ {
		if s == state {
			fmt.Printf("→ 從%s往前推 %d 步就是現在的狀態；攔到的 rand 呼叫是 %d 次。\n",
				from, n, rands)
			if uint64(rands) != n {
				fmt.Printf("   ⚠ 兩個數字不一致：有 rand 沒被攔到，或狀態被別的東西改過。\n")
			}
			fmt.Printf("   下一個 rand() 會回 %04X\n", (lcg(s)>>16)&0x7FFF)
			return
		}
		s = lcg(s)
	}
	fmt.Printf("→ 從%s推了 %d 步都對不上。\n", from, *walk)
	_ = outs
	_ = rolls
}
