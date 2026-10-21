package cok

import "github.com/wicanr2/dosgolem/xlate"

// 規格 005：中文句中的可列印 ASCII 兩兩合成一個組合字，佔一個 8×8 字格。
const pairBase = 0xF0000

func isHalf(r rune) bool { return r >= 0x20 && r <= 0x7E }

func pairRune(a, b rune) rune { return pairBase + a<<7 + b }

func isPair(r rune) bool { return r >= pairBase+0x20<<7 && r < pairBase+0x80<<7 }

// PackHalfWidth 依規格 005 §2 合併連續 ASCII；譯文全是 ASCII 時原樣回傳。
func PackHalfWidth(rs []rune) []rune {
	wide := false
	for _, r := range rs {
		if r >= 0x80 {
			wide = true
			break
		}
	}
	if !wide {
		return rs
	}
	return PackHalfWidthAll(rs)
}

// PackHalfWidthAll 同 PackHalfWidth，但全是 ASCII 也合併（英文版的切換列，主專案規格 015 §3.2）。
func PackHalfWidthAll(rs []rune) []rune {
	out := make([]rune, 0, len(rs))
	for i := 0; i < len(rs); {
		if !isHalf(rs[i]) {
			out = append(out, rs[i])
			i++
			continue
		}
		j := i
		for j < len(rs) && isHalf(rs[j]) {
			j++
		}
		if j-i == 1 && rs[i] == ' ' {
			out = append(out, ' ')
			i = j
			continue
		}
		for k := i; k < j; k += 2 {
			b := rune(' ')
			if k+1 < j {
				b = rs[k+1]
			}
			out = append(out, pairRune(rs[k], b))
		}
		i = j
	}
	return out
}

// AddHalfWidthPairs 補上全部組合字（規格 005 §2）：半形字墨跡在中間半格（16 格 x=4..11、24 格 x=6..17），
// 組合字左半取 a、右半取 b。字型的 3 倍字模（Scaled[3]，24×24 外部字模）一併處理。
func AddHalfWidthPairs(f *xlate.Font) {
	if f == nil || f.W != f.H || f.W%8 != 0 {
		return
	}
	if s, ok := f.Scaled[3]; ok && s.Font != f {
		AddHalfWidthPairs(s.Font)
	}
	w, rb := f.W, f.W/8
	half, quarter := w/2, w/4
	bit := func(g []byte, x, y int) bool { return g[y*rb+x/8]&(0x80>>(x%8)) != 0 }
	for a := rune(0x20); a <= 0x7E; a++ {
		ga, okA := f.Glyphs[a]
		for b := rune(0x20); b <= 0x7E; b++ {
			gb, okB := f.Glyphs[b]
			if !okA || !okB {
				continue
			}
			g := make([]byte, len(ga))
			for y := 0; y < f.H; y++ {
				for x := 0; x < half; x++ {
					if bit(ga, x+quarter, y) {
						g[y*rb+x/8] |= 0x80 >> (x % 8)
					}
					if bit(gb, x+quarter, y) {
						g[y*rb+(x+half)/8] |= 0x80 >> ((x + half) % 8)
					}
				}
			}
			f.Glyphs[pairRune(a, b)] = g
		}
	}
}
