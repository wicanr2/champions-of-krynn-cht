// party 在原版記憶體裡定位隊伍記錄。
//
// 錨點是角色名字：存檔的版面已知（remake 的 savegame parser），
// 名字之後 +27 是目前 HP、+29 是最大 HP、+37 是食物。
// 拿存檔裡已知的 HP 值交叉驗證，對得上才算找到——
// 光是「有一串看起來像名字的位元組」不算，記憶體裡到處都是字串。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob1"
	"github.com/wicanr2/dosgolem/oracle"
)

// 角色記錄裡相對於**名字起點**的位移。
const (
	offHitPointsCur = 27
	offHitPointsMax = 29
	offFood         = 37
)

func main() {
	exe := flag.String("exe", "", "START1.EXE 路徑")
	root := flag.String("root", "", "原版資料目錄")
	names := flag.String("names", "ALLABAR,ARIEL,VALANAU,TENMIYANA", "要找的角色名字")
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
	fmt.Println("已載入存檔，步數:", o.Steps())

	ram := make([]byte, 0xA0000)
	for i := range ram {
		ram[i] = o.Byte(oracle.Addr{Seg: uint16(i >> 4), Off: uint16(i & 0xF)})
	}
	// 四個名字的位置：等距就是隊伍陣列。
	var base []int
	for _, name := range strings.Split(*names, ",") {
		idx := bytes.Index(ram, append([]byte(name), 0))
		if idx < 0 {
			fmt.Fprintln(os.Stderr, "找不到", name)
			os.Exit(1)
		}
		base = append(base, idx)
		fmt.Printf("  %-10s 線性 %05X\n", name, idx)
	}
	stride := base[1] - base[0]
	fmt.Printf("record stride = %d (0x%X)\n", stride, stride)

	// 相對名字起點的位移，由 ALLABAR 那筆逐欄對出來（見下面的 dump）。
	const (
		offHP    = 0x19 // 目前 HP，1 byte
		offHPMax = 0x1A // 最大 HP，1 byte
		offClass = 0x1E
		offFood  = 0x21
	)
	fmt.Println("逐人核對（HP／職業取自存檔解析）：")
	wantHP := []int{33, 37, 12, 15}
	wantClass := []int{0, 1, 14, 6}
	bad := false
	for i, name := range strings.Split(*names, ",") {
		hp := int(ram[base[i]+offHP])
		hpMax := int(ram[base[i]+offHPMax])
		class := int(ram[base[i]+offClass])
		food := int(ram[base[i]+offFood])
		ok := hp == wantHP[i] && hpMax == wantHP[i] && class == wantClass[i] && food <= 100
		mark := "✗"
		if ok {
			mark = "✓"
		} else {
			bad = true
		}
		fmt.Printf("  %s %-10s 線性 %05X  HP=%d/%d（預期 %d）職業=%d（預期 %d）食物=%d\n",
			mark, name, base[i], hp, hpMax, wantHP[i], class, wantClass[i], food)
	}
	if bad {
		fmt.Println("有欄位對不上——版面還沒解對，不要拿去改記憶體")
		os.Exit(1)
	}
	fmt.Printf("隊伍陣列起於線性 %05X，每筆 %d bytes\n", base[0], stride)
	fmt.Println("第一個角色記錄的前 64 bytes：")
	for i := 0; i < 64; i += 16 {
		fmt.Printf("   +%02X: % X\n", i, ram[base[0]+i:base[0]+i+16])
	}
}
