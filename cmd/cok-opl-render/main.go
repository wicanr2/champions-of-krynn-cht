// cok-opl-render 把 OPL2 暫存器寫入序列合成成 WAV（主專案 Issue #16 遊戲中配樂）。
//
// 輸入每行 `時間 暫存器 值`（暫存器與值十六進位，`#` 開頭為註解）：
//   - 預設時間是秒（小數），給新編曲用；
//   - `-steps-per-second N` 時時間是 dosgolem 步數（probe -opl-log 的格式），換算成秒。
//
// 合成器與 cok-play 相同（audio/nukedopl，48 kHz 立體聲），所以離線輸出的音色就是遊戲內播放的音色。
// 只接受 OPL2 暫存器（0x01–0xF5）；計時器暫存器 02–04 不送合成器（同 audio/mixer）。
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/audio/nukedopl"
)

const rate = 48000

type write struct {
	at       float64
	reg, val uint8
}

func main() {
	in := flag.String("in", "", "OPL 寫入序列")
	out := flag.String("out", "", "輸出 WAV（48 kHz 16 位元立體聲）")
	sps := flag.Float64("steps-per-second", 0, "時間欄是 dosgolem 步數時的每秒步數（0 ＝ 時間欄是秒）")
	tail := flag.Float64("tail", 2, "最後一筆寫入後再合成幾秒（餘音）")
	gain := flag.Float64("gain", 1, "輸出增益")
	flag.Parse()
	if *in == "" || *out == "" {
		fail(fmt.Errorf("-in、-out 必填"))
	}
	ws, err := read(*in, *sps)
	if err != nil {
		fail(err)
	}
	if len(ws) == 0 {
		fail(fmt.Errorf("%s 沒有寫入", *in))
	}
	t0 := ws[0].at
	if *sps == 0 {
		t0 = 0 // 秒數格式從 0 秒起算，保留開頭的休止
	}
	end := ws[len(ws)-1].at - t0 + *tail
	n := int(math.Ceil(end * rate))
	chip := nukedopl.New(rate)
	pcm := make([]int16, 0, n*2)
	frame := make([]int16, 2)
	i := 0
	for s := 0; s < n; s++ {
		now := float64(s) / rate
		for i < len(ws) && ws[i].at-t0 <= now {
			if r := ws[i].reg; r < 0x02 || r > 0x04 {
				chip.WriteRegBuffered(uint16(r), ws[i].val)
			}
			i++
		}
		chip.GenerateStream(frame)
		for _, v := range frame {
			x := float64(v) * *gain
			x = math.Max(-32768, math.Min(32767, x))
			pcm = append(pcm, int16(x))
		}
	}
	if err := writeWAV(*out, pcm); err != nil {
		fail(err)
	}
	fmt.Printf("%d 筆寫入，%.2f 秒 → %s\n", len(ws), end, *out)
}

func read(path string, sps float64) ([]write, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ws []write
	sc := bufio.NewScanner(f)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fs := strings.Fields(line)
		if len(fs) != 3 {
			return nil, fmt.Errorf("%s:%d：要三欄", path, ln)
		}
		at, err := strconv.ParseFloat(fs[0], 64)
		if err != nil {
			return nil, fmt.Errorf("%s:%d：時間 %v", path, ln, err)
		}
		if sps > 0 {
			at /= sps
		}
		reg, err1 := strconv.ParseUint(fs[1], 16, 8)
		val, err2 := strconv.ParseUint(fs[2], 16, 8)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s:%d：暫存器或值不是十六進位", path, ln)
		}
		ws = append(ws, write{at, uint8(reg), uint8(val)})
	}
	sort.SliceStable(ws, func(a, b int) bool { return ws[a].at < ws[b].at })
	return ws, sc.Err()
}

func writeWAV(path string, pcm []int16) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data := uint32(len(pcm) * 2)
	hdr := []any{[4]byte{'R', 'I', 'F', 'F'}, 36 + data, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, data}
	for _, h := range hdr {
		if err := binary.Write(f, binary.LittleEndian, h); err != nil {
			return err
		}
	}
	return binary.Write(f, binary.LittleEndian, pcm)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cok-opl-render：", err)
	os.Exit(1)
}
