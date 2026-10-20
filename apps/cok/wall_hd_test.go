package cok

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

func wallHash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func wallFixture(t *testing.T) (string, [][]byte) {
	t.Helper()
	dir := t.TempDir()
	refs := [][]byte{}
	m := wallManifest{Version: 1, Scale: 3, Kind: "wall-grid", Viewport: [5]int{24, 24, 11, 11, 8}}
	write := func(name string, b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 4; n++ {
		key := string(rune('a' + n))
		ref := make([]byte, 64)
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				v := byte(1 + n*2 + (x+y)%2)
				if n == 3 && x < 4 {
					v = 13
				}
				ref[y*8+x] = v
			}
		}
		refs = append(refs, ref)
		im := image.NewNRGBA(image.Rect(0, 0, 24, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				i := (y*24 + x) * 4
				v := ref[y/3*8+x/3]
				c := [3]byte{17, 123, 231}
				if v == 13 {
					c = combatEGA[8]
				} else if x == 0 || x == 23 || y == 0 || y == 23 {
					c = combatEGA[v]
				}
				copy(im.Pix[i:i+3], c[:])
				im.Pix[i+3] = 255
			}
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, im); err != nil {
			t.Fatal(err)
		}
		write(key+".idx", ref)
		write(key+".png", encoded.Bytes())
		m.Assets = append(m.Assets, wallAssetSpec{Key: key, Kind: "wall", Size: [2]int{8, 8}, Indexed: key + ".idx", IndexSHA: wallHash(ref), PNG: key + ".png", PNGSHA: wallHash(encoded.Bytes())})
		if n < 3 {
			m.Cells = append(m.Cells, wallCellSpec{Key: key, Base: 0, Layers: []wallLayerSpec{{Key: key}}, IndexSHA: wallHash(ref), Paint: true})
		}
	}
	positions := make([]int, 121)
	for n := range positions {
		positions[n] = n
	}
	m.Cells = append(m.Cells, wallCellSpec{Key: "black-backdrop", Base: 0, Positions: positions, IndexSHA: wallHash(make([]byte, 64)), Paint: true})
	partial := append([]byte{}, refs[3]...)
	for n, v := range partial {
		if v == 13 {
			partial[n] = 11
		}
	}
	m.Cells = append(m.Cells, wallCellSpec{Key: "partial-over-sky", Base: 11, Positions: []int{0}, Layers: []wallLayerSpec{{Key: "d"}}, IndexSHA: wallHash(partial), Paint: true})
	b, _ := json.Marshal(m)
	write("walls.json", b)
	write("theme.json", []byte(`{"scale":3,"items":[]}`))
	return dir, refs
}
func putWallCell(screen []byte, pos int, ref []byte) {
	x, y := 24+pos%11*8, 24+pos/11*8
	for row := 0; row < 8; row++ {
		copy(screen[(y+row)*320+x:], ref[row*8:(row+1)*8])
	}
}
func wallLive(t *testing.T, dir string) *Live {
	t.Helper()
	l := NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := l.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	return l
}
func TestWallHDPerCellViewportAndFallback(t *testing.T) {
	dir, refs := wallFixture(t)
	l := wallLive(t, dir)
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := make([]byte, 64000)
	for n := 0; n < 3; n++ {
		putWallCell(screen, n+12, refs[n])
	}
	before := sha256.Sum256(screen)
	l.Frame(screen, pal)
	base, hd := l.Compose(3, false, nil), l.Compose(3, true, nil)
	if bytes.Equal(base, hd) || !l.walls.active {
		t.Fatal("完整視窗沒有啟用")
	}
	for y := 0; y < 600; y++ {
		for x := 0; x < 960; x++ {
			n := (y*960 + x) * 4
			if x < 72 || x >= 336 || y < 72 || y >= 336 {
				if !bytes.Equal(base[n:n+4], hd[n:n+4]) {
					t.Fatal("視窗外被改動")
				}
			}
		}
	}
	if before != sha256.Sum256(screen) || !bytes.Equal(screen, l.Indexed()) {
		t.Fatal("改動原版色號")
	}
	if !bytes.Equal(l.Compose(2, true, nil), l.Compose(2, false, nil)) {
		t.Fatal("二倍畫出HD")
	}
	if !bytes.Equal(hd, l.Compose(3, true, nil)) {
		t.Fatal("切換後結果不同")
	}
	// 未知格只撤除自己，不影響別處的牆面。
	screen[100*320+100] = 15
	l.Frame(screen, pal)
	if bytes.Equal(l.Compose(3, true, nil), l.Compose(3, false, nil)) || !l.walls.active || l.walls.matches[9*11+9] != nil {
		t.Fatal("未知格影響其他格，或自身未回退")
	}
	screen[100*320+100] = 0
	l.Frame(screen, pal)
	if !bytes.Equal(hd, l.Compose(3, true, nil)) {
		t.Fatal("恢復後結果不同")
	}
	changed := pal
	changed[7][0] ^= 1
	l.Frame(screen, changed)
	if l.walls.active {
		t.Fatal("未知色盤仍啟用")
	}
	l.Frame(screen, pal)
	for n := 0; n < 2; n++ {
		putWallCell(screen, n+12, make([]byte, 64))
	}
	l.Frame(screen, pal)
	if !l.walls.active || l.walls.matches[14] == nil {
		t.Fatal("一個完整牆面被其他格擋住")
	}
}
func TestWallHDMaskedCompositionAndStaleDraw(t *testing.T) {
	dir, refs := wallFixture(t)
	l := wallLive(t, dir)
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := make([]byte, 64000)
	for n := 0; n < 3; n++ {
		putWallCell(screen, n+12, refs[n])
	}
	partial := append([]byte{}, refs[3]...)
	for n, v := range partial {
		if v == 13 {
			partial[n] = 11
		}
	}
	putWallCell(screen, 0, partial)
	l.Frame(screen, pal)
	out := l.Compose(3, true, nil)
	left := ((24*3+5)*960 + 24*3 + 5) * 4
	right := left + 12*4
	if !bytes.Equal(out[left:left+3], combatEGA[11][:]) || !bytes.Equal(out[right:right+3], []byte{17, 123, 231}) {
		t.Fatal("留底與牆面遮擋不符")
	}
	// Frame後單格變動，只跳過過期格，其餘格仍須畫。
	altered := append([]byte{}, screen...)
	altered[100*320+100] ^= 1
	base := l.Compose(3, false, nil)
	want := append([]byte{}, base...)
	l.walls.draw(base, altered, 3)
	if bytes.Equal(base, want) || !bytes.Equal(base[left:left+3], out[left:left+3]) {
		t.Fatal("過期單格錯誤阻擋其他完整格")
	}
	putWallCell(screen, 0, make([]byte, 64))
	l.Frame(screen, pal)
	out = l.Compose(3, true, nil)
	if !bytes.Equal(out[left:left+3], combatEGA[0][:]) {
		t.Fatal("遮擋清除後殘影")
	}
}
func TestWallHDBadAssetsAreAtomic(t *testing.T) {
	for _, kind := range []string{"hash", "composition", "duplicate", "position", "alpha", "size", "mask", "seam", "non-ega"} {
		t.Run(kind, func(t *testing.T) {
			dir, _ := wallFixture(t)
			l := wallLive(t, dir)
			old, count := l.HDLayers, l.HDItems
			b, _ := os.ReadFile(filepath.Join(dir, "walls.json"))
			var m wallManifest
			_ = json.Unmarshal(b, &m)
			switch kind {
			case "hash":
				m.Assets[0].PNGSHA = "bad"
			case "composition":
				m.Cells[0].IndexSHA = "bad"
			case "duplicate":
				m.Cells = append(m.Cells, m.Cells[0])
				m.Cells[len(m.Cells)-1].Key = "different-key"
			case "position":
				m.Cells[len(m.Cells)-1].Positions = []int{121}
			case "non-ega":
				p := filepath.Join(dir, m.Assets[0].Indexed)
				data, _ := os.ReadFile(p)
				data[0] = 16
				_ = os.WriteFile(p, data, 0644)
				m.Assets[0].IndexSHA = wallHash(data)
			default:
				asset := 0
				if kind == "mask" {
					asset = 3
				}
				p := filepath.Join(dir, m.Assets[asset].PNG)
				data, _ := os.ReadFile(p)
				im, _ := png.Decode(bytes.NewReader(data))
				nr := image.NewNRGBA(im.Bounds())
				for y := 0; y < 24; y++ {
					for x := 0; x < 24; x++ {
						nr.Set(x, y, im.At(x, y))
					}
				}
				if kind == "alpha" {
					nr.Pix[3] = 0
				}
				if kind == "mask" || kind == "seam" {
					nr.Pix[0] ^= 1
				}
				if kind == "size" {
					nr = image.NewNRGBA(image.Rect(0, 0, 21, 24))
				}
				var encoded bytes.Buffer
				_ = png.Encode(&encoded, nr)
				_ = os.WriteFile(p, encoded.Bytes(), 0644)
				m.Assets[asset].PNGSHA = wallHash(encoded.Bytes())
			}
			b, _ = json.Marshal(m)
			_ = os.WriteFile(filepath.Join(dir, "walls.json"), b, 0644)
			if err := l.AddHD(dir); err == nil {
				t.Fatal("接受壞件")
			}
			if l.HDLayers != old || l.HDItems != count {
				t.Fatal("壞件留下部分載入")
			}
		})
	}
}
func TestWallHDMissingFamily(t *testing.T) {
	dir, _ := wallFixture(t)
	_ = os.Remove(filepath.Join(dir, "walls.json"))
	l := wallLive(t, dir)
	if l.walls != nil || l.HDItems != 0 {
		t.Fatal("撤除素材未保持原版")
	}
}
