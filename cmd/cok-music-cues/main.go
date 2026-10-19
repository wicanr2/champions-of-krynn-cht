// cok-music-cues 以實機盤點的印字序列重播選曲（主專案規格 014 §4 收據）：
// 依步數順序把 inv.tsv（cok-explore、cok-overlay-shot -inventory 的輸出）的路徑 C 整列餵給 Menu、
// 路徑 A、B 原文餵給 Text，印出每次選曲改變的步數、觸發字串與曲目。短曲在觸發後 8 秒（步數換算）視為播完。
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/cok"
)

const stepsPerSecond = 11_580_882 // machine.StepsPerSecond()（未釘 IRQ0 間隔的探索執行）

func main() {
	cues := flag.String("cues", "", "text/music-cues.tsv")
	oneShot := flag.Float64("one-shot", 8, "短曲長度（秒）")
	flag.Parse()
	if *cues == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法：cok-music-cues -cues music-cues.tsv inv.tsv…")
		os.Exit(2)
	}
	b, err := os.ReadFile(*cues)
	if err != nil {
		fail(err)
	}
	for _, path := range flag.Args() {
		d, err := cok.LoadMusicCues(b)
		if err != nil {
			fail(err)
		}
		fmt.Printf("== %s\n", path)
		if err := replay(path, d, uint64(*oneShot*stepsPerSecond)); err != nil {
			fail(err)
		}
	}
}

func replay(path string, d *cok.MusicDirector, oneShotSteps uint64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	last, lastSerial := "\x00", -1
	var shotAt uint64
	counts := map[string]int{}
	for ln := 0; sc.Scan(); ln++ {
		fs := strings.Split(sc.Text(), "\t")
		if ln == 0 || len(fs) < 9 {
			continue
		}
		step, _ := strconv.ParseUint(fs[0], 10, 64)
		text := strings.Join(strings.Fields(fs[8]), " ")
		if shotAt != 0 && step >= shotAt+oneShotSteps {
			d.OneShotDone()
			shotAt = 0
		}
		switch fs[1] {
		case "C":
			d.Menu(text)
		case "A", "B":
			d.Text(fs[8])
		default:
			continue
		}
		track, one, serial := d.Want()
		if one && serial != lastSerial {
			shotAt, lastSerial = step, serial
		}
		if track != last || one && serial != lastSerial {
			fmt.Printf("#%d\t%s\t%-12s\t%q\n", step, fs[1], track, text)
			counts[track]++
			last = track
		}
	}
	fmt.Printf("換曲次數：%v\n", counts)
	return sc.Err()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cok-music-cues：", err)
	os.Exit(1)
}
