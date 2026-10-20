package cok

// 規格010 §9，Scale3x 官方九點規則。只比較及傳回來源色號。
func hdScale3x(ref []byte, w, h int) []byte {
	get := func(x, y int) byte {
		if x < 0 {
			x = 0
		}
		if x >= w {
			x = w - 1
		}
		if y < 0 {
			y = 0
		}
		if y >= h {
			y = h - 1
		}
		return ref[y*w+x]
	}
	out := make([]byte, w*h*9)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a, b, c := get(x-1, y-1), get(x, y-1), get(x+1, y-1)
			d, e, f := get(x-1, y), get(x, y), get(x+1, y)
			g, hh, i := get(x-1, y+1), get(x, y+1), get(x+1, y+1)
			v := [9]byte{e, e, e, e, e, e, e, e, e}
			if b != hh && d != f {
				if d == b {
					v[0] = d
				}
				if b == f {
					v[2] = f
				}
				if d == hh {
					v[6] = d
				}
				if hh == f {
					v[8] = f
				}
				if d == b && e != c || b == f && e != a {
					v[1] = b
				}
				if d == b && e != g || d == hh && e != a {
					v[3] = d
				}
				if b == f && e != i || hh == f && e != c {
					v[5] = f
				}
				if d == hh && e != i || hh == f && e != g {
					v[7] = hh
				}
			}
			for n, c := range v {
				out[(y*3+n/3)*w*3+x*3+n%3] = c
			}
		}
	}
	return out
}

func hdScale3xPixelsEqual(ref, pix []byte, w, h int) bool {
	scaled := hdScale3x(ref, w, h)
	if len(pix) != len(scaled)*4 {
		return false
	}
	for n, c := range scaled {
		if c > 15 || pix[n*4] != combatEGA[c][0] || pix[n*4+1] != combatEGA[c][1] || pix[n*4+2] != combatEGA[c][2] || pix[n*4+3] != 255 {
			return false
		}
	}
	return true
}
