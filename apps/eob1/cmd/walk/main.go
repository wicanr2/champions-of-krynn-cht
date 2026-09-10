// walk 帶著「鎖 HP／食物」與「強制攻擊」在原版裡走動，收據印在最後。
//
// 目的是把原版帶到第二層做同流對拍。亂數保持自然——這裡只固定狀態
// （`docs/spec/014`），不改任何亂數回傳值。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

// draws 反推兩個狀態之間消費了幾次亂數。推不到就回 -1，
// 不要回 0——「沒消費」與「推不到」是兩件事。
func draws(from, to uint32) int {
	s := from
	for n := 0; n < 2_000_000; n++ {
		if s == to {
			return n
		}
		s = s*0x015A4E35 + 1
	}
	return -1
}

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	swings := flag.Int("swings", 3, "全隊各揮幾輪")
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

	seedBefore, err := eob1.RandomState(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "讀亂數狀態:", err)
		os.Exit(1)
	}
	before, err := eob1.Party(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "讀隊伍:", err)
		os.Exit(1)
	}
	fmt.Println("鎖之前：")
	for _, m := range before {
		fmt.Printf("  %-10s HP=%d/%d 食物=%d @ %s\n", m.Name, m.HP, m.Max, m.Food, m.At)
	}

	// 先扣血扣食物，證明鎖真的有作用——不扣的話「鎖住了」與「本來就滿的」
	// 長得一模一樣。
	for _, m := range before {
		o.SetByte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + 0x19}, 1)
		o.SetByte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + 0x21}, 5)
	}
	hurt, _ := eob1.Party(o)
	fmt.Println("人為扣到剩 1 滴血、食物 5 之後：")
	for _, m := range hurt {
		fmt.Printf("  %-10s HP=%d/%d 食物=%d\n", m.Name, m.HP, m.Max, m.Food)
	}

	lock, err := eob1.LockPartyVitals(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "上鎖:", err)
		os.Exit(1)
	}
	if err := o.Run(3_000_000); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
	}
	after, _ := eob1.Party(o)
	fmt.Printf("上鎖並跑 3,000,000 道指令之後（寫入 %d 次）：\n", lock.Writes)
	for _, m := range after {
		fmt.Printf("  %-10s HP=%d/%d 食物=%d\n", m.Name, m.HP, m.Max, m.Food)
	}

	n, err := eob1.AttackAll(o, *swings)
	if err != nil {
		fmt.Fprintln(os.Stderr, "攻擊:", err)
		os.Exit(1)
	}
	fmt.Printf("強制攻擊：%d 輪 × 4 人，有反應 %d 次\n", *swings, n)
	seedAfter, _ := eob1.RandomState(o)
	fmt.Println("收據：")
	fmt.Printf("  鎖 %d 人的 HP 與食物，從第 %d 步起，共寫入 %d 次\n", lock.Members, lock.From, lock.Writes)
	fmt.Printf("  強制攻擊 %d 輪 × 4 人\n", *swings)
	fmt.Printf("  RANDSEED %08X → %08X（消費 %d 次）\n", seedBefore, seedAfter, draws(seedBefore, seedAfter))
	fmt.Printf("  結束步數 %d\n", o.Steps())
}
