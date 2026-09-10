// l2steps 從第二層存檔載入，走一串固定操作，逐步記下亂數消費。
//
// 這是同流對拍的原版那一側：每一步之後的方塊與 RANDSEED 都記下來，
// remake 走同一串操作時要逐步對得上。
//
// ⚠ **擋住的那幾步才是重點。** 撞牆與關著的門在畫面上都是「沒動」，
// 但它們可能各自消費不同次數的亂數；只比對能走通的步驟，
// 會漏掉整條流真正錯開的地方。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

func draws(from, to uint32) int {
	s := from
	for n := 0; n < 5_000_000; n++ {
		if s == to {
			return n
		}
		s = s*0x015A4E35 + 1
	}
	return -1
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "含第二層存檔的資料目錄")
	steps := flag.String("steps", "forward,forward,turnRight,forward,turnRight,back,turnRight",
		"要走的操作，逗號分隔")
	shots := flag.String("shots", "", "PNG 輸出目錄")
	flag.Parse()
	o, err := oracle.Load(*exe, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	defer o.Close()
	o.SetClock(0, 0, 0, 0)

	if err := eob1.ToTitleMenu(o); err != nil {
		fmt.Fprintln(os.Stderr, "到標題:", err)
		os.Exit(1)
	}
	// 載入存檔。畫面是第二層，不能用第一層那個 digest 當閘門；
	// 改用「畫面安定下來」，並用方塊編號確認真的進了地城。
	o.PressKey(oracle.KeyEnter)
	// 等到三個方塊副本一致且非零——那才是「地城真的載好了」。
	// 用「畫面安定」當閘門會在中途某個靜止畫面就放行，讀到 [0 755 0]。
	var block []uint16
	for round := 0; round < 60; round++ {
		if err := o.Run(1_000_000); err != nil {
			fmt.Fprintln(os.Stderr, "載入途中:", err)
			break
		}
		block, err = eob1.CurrentBlock(o)
		if err != nil {
			continue
		}
		if block[0] != 0 && block[0] == block[1] && block[1] == block[2] {
			break
		}
	}
	if len(block) == 0 || block[0] == 0 || block[0] != block[1] || block[1] != block[2] {
		if *shots != "" {
			_ = o.WritePNG(*shots + "/l2-load-failed.png")
		}
		fmt.Fprintf(os.Stderr, "等不到地城載入完成，方塊副本 %v\n", block)
		os.Exit(1)
	}
	seed, err := eob1.RandomState(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	party, err := eob1.Party(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lock, lockErr := eob1.LockPartyVitals(o)
	if lockErr != nil {
		fmt.Fprintln(os.Stderr, "上鎖:", lockErr)
		os.Exit(1)
	}
	fmt.Printf("載入完成：block=%d，隊伍 %d 人，RANDSEED=%08X，步數=%d（已鎖 %d 人的 HP 與食物）\n",
		block[0], len(party), seed, o.Steps(), lock.Members)
	if *shots != "" {
		_ = o.WritePNG(*shots + "/l2-000-start.png")
	}

	fmt.Println()
	fmt.Printf("%-4s %-12s %-8s %-10s %s\n", "#", "操作", "方塊", "RANDSEED", "這一步消費")
	previous := seed
	for index, what := range strings.Split(*steps, ",") {
		what = strings.TrimSpace(what)
		if what == "" {
			continue
		}
		var err error
		if strings.HasPrefix(what, "attack") {
			character := 0
			if len(what) > 6 {
				character = int(what[6] - '0')
			}
			err = eob1.AttackSlot(o, character, 0)
		} else {
			err = eob1.Move(o, what)
		}
		blocked := err != nil && eob1.Blocked(err)
		if err != nil && !blocked {
			fmt.Fprintln(os.Stderr, what+":", err)
			break
		}
		now, _ := eob1.CurrentBlock(o)
		seedNow, _ := eob1.RandomState(o)
		note := ""
		if blocked {
			note = "（畫面沒動）"
		}
		fmt.Printf("%-4d %-12s %-8d %08X   %d 次 %s\n",
			index+1, what, now[0], seedNow, draws(previous, seedNow), note)
		previous = seedNow
		if *shots != "" {
			_ = o.WritePNG(fmt.Sprintf("%s/l2-%03d-%s.png", *shots, index+1, what))
		}
	}
	fmt.Printf("\n總計：RANDSEED %08X → %08X，共消費 %d 次；結束步數 %d\n",
		seed, previous, draws(seed, previous), o.Steps())
	fmt.Printf("收據：鎖 %d 人的 HP 與食物，共寫入 %d 次\n", lock.Members, lock.Writes)
}
