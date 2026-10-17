// dkk-hd-check 以保存的原版畫面（probe -dump-at 的 320×200 色號 .bin）核對 HD 主題（主專案規格 010 §3）：
// 每張畫面餵給 dkk.Live.Frame，3× 合成時只有預期那一項的矩形換成 HD 且逐像素等於主題圖，
// 矩形外與不掛 HD 相同；2× 合成與原版相同。輸出每張畫面一列 JSON，任一不符結束碼 1。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/dosgolem/apps/dkk"
	"github.com/wicanr2/dosgolem/xlate"
)

// ega 是 EGA 預設 16 色（主專案規格 010 §1 色盤證據）。
var ega = [16][3]uint8{{0, 0, 0}, {0, 0, 170}, {0, 170, 0}, {0, 170, 170}, {170, 0, 0}, {170, 0, 170}, {170, 85, 0}, {170, 170, 170},
	{85, 85, 85}, {85, 85, 255}, {85, 255, 85}, {85, 255, 255}, {255, 85, 85}, {255, 85, 255}, {255, 255, 85}, {255, 255, 255}}

type result struct {
	Screen      string `json:"screen"`
	Want        string `json:"want"`
	HDPixels    int    `json:"hd_pixels"`
	HDMatch     int    `json:"hd_match"`
	DiffOutside int    `json:"diff_outside"`
	Same2x      bool   `json:"same_2x"`
	OK          bool   `json:"ok"`
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "dkk-hd-check:", err)
	os.Exit(2)
}

func main() {
	dir := flag.String("hd", "", "HD 主題目錄（theme.json）")
	screens := flag.String("screens", "", "`畫面.bin=預期 key`，逗號分隔；key 為 - 表示不得畫出任何 HD")
	flag.Parse()
	if *dir == "" || *screens == "" {
		flag.Usage()
		os.Exit(2)
	}
	items, err := dkk.LoadHDTheme(*dir)
	if err != nil {
		fail(err)
	}
	var pal [256][3]uint8
	for i := range pal {
		pal[i] = ega[i&15]
	}
	ok := true
	enc := json.NewEncoder(os.Stdout)
	for _, spec := range strings.Split(*screens, ",") {
		path, want, found := strings.Cut(spec, "=")
		if !found {
			fail(fmt.Errorf("畫面參數 %q 缺 =key", spec))
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		if len(raw) != 320*200 {
			fail(fmt.Errorf("%s：%d bytes，應為 64000", path, len(raw)))
		}
		idx := make([]byte, len(raw))
		for i, c := range raw {
			idx[i] = c & 15
		}
		// 每張畫面用新的 Live：前一張畫面的啟用狀態不帶過來
		live := dkk.NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
		if err := live.AddHD(*dir); err != nil {
			fail(err)
		}
		live.Frame(idx, pal)
		r := result{Screen: path, Want: want}
		var it *dkk.HDItem
		for i := range items {
			if items[i].Key == want {
				it = &items[i]
			}
		}
		if want != "-" && it == nil {
			fail(fmt.Errorf("主題沒有 %q", want))
		}
		hd, orig := live.Compose(3, true, nil), live.Compose(3, false, nil)
		const W = 320 * 3
		for y := 0; y < 600; y++ {
			for x := 0; x < W; x++ {
				o := (y*W + x) * 4
				inside := it != nil && x >= it.X*3 && x < (it.X+it.W)*3 && y >= it.Y*3 && y < (it.Y+it.H)*3
				if !inside {
					if !bytes.Equal(hd[o:o+3], orig[o:o+3]) {
						r.DiffOutside++
					}
					continue
				}
				r.HDPixels++
				p := ((y-it.Y*3)*it.W*3 + x - it.X*3) * 4
				if bytes.Equal(hd[o:o+3], it.Pix[p:p+3]) {
					r.HDMatch++
				}
			}
		}
		r.Same2x = bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil))
		r.OK = r.HDMatch == r.HDPixels && r.DiffOutside == 0 && r.Same2x
		ok = ok && r.OK
		if err := enc.Encode(r); err != nil {
			fail(err)
		}
	}
	if !ok {
		os.Exit(1)
	}
}
