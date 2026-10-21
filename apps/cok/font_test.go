package cok

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// 24×24 字模的半形字墨跡在 x=6..17；組合字左半取 a 的中間半格、右半取 b 的中間半格。
func TestHalfWidthPairs24(t *testing.T) {
	g := func(x int) []byte { // 只有第 0 列第 x 點
		b := make([]byte, 3*24)
		b[x/8] = 0x80 >> (x % 8)
		return b
	}
	f3 := &xlate.Font{W: 24, H: 24, Glyphs: map[rune][]byte{'A': g(6), 'B': g(17)}}
	f := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{'A': make([]byte, 32), 'B': make([]byte, 32)},
		Scaled: map[int]xlate.ScaledFont{3: {Font: f3}}}
	AddHalfWidthPairs(f)
	p, ok := f3.Glyphs[pairRune('A', 'B')]
	if !ok {
		t.Fatal("3 倍字模沒有組合字")
	}
	on := func(x int) bool { return p[x/8]&(0x80>>(x%8)) != 0 }
	if !on(0) || !on(23) || on(6) || on(17) {
		t.Fatalf("組合字點位錯：%08b", p[:3])
	}
}

// 純 ASCII 譯文：中文等語言兩字併一格，英文版一字一格。
func TestPathAPackASCII(t *testing.T) {
	p := &PathA{}
	if got := len(p.pack("Jim Ward")); got != 4 {
		t.Fatalf("中文版純 ASCII 要併成 4 格，得 %d", got)
	}
	p.ASCIIWide = true
	if got := len(p.pack("Jim Ward")); got != 8 {
		t.Fatalf("英文版要一字一格 8 格，得 %d", got)
	}
}

// 主題清單路徑以 / 分隔；檢查不能依作業系統的分隔符（Windows 上 filepath.Clean 會改寫 /）。
func TestThemeFile(t *testing.T) {
	if p, ok := themeFile("d", "walls/a.idx"); !ok || p != filepath.Join("d", "walls", "a.idx") {
		t.Fatalf("合法路徑被拒：%q %v", p, ok)
	}
	for _, bad := range []string{"", "/abs.png", "../x.png", "..", "a/../b.png", "a/./b.png", `walls\a.idx`, "C:a.png"} {
		if _, ok := themeFile("d", bad); ok {
			t.Errorf("%q 應被拒", bad)
		}
	}
}
