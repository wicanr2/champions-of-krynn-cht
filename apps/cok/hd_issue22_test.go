package cok

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestHDIssue22OneCellSurvivesUnknownAndStaleNeighbor(t *testing.T) {
	dir, refs := wallFixture(t)
	h, err := loadWallHD(dir)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.Repeat([]byte{5}, 64000)
	putWallCell(idx, 60, refs[0])
	putWallCell(idx, 61, refs[1])
	var pal [256][3]byte
	copy(pal[:], combatEGA[:])
	h.frame(idx, pal)
	if !h.active || h.matches[60] == nil || h.matches[61] == nil {
		t.Fatal("未知鄰格擋住完整來源")
	}
	putWallCell(idx, 61, bytes.Repeat([]byte{5}, 64))
	out := make([]byte, 960*600*4)
	h.draw(out, idx, 3)
	for pos := 0; pos < 121; pos++ {
		x, y := (24+pos%11*8)*3, (24+pos/11*8)*3
		for row := 0; row < 24; row++ {
			part := out[((y+row)*960+x)*4 : ((y+row)*960+x+24)*4]
			if pos == 60 {
				if bytes.Equal(part, make([]byte, 96)) {
					t.Fatal("另一格過期令完整格失效")
				}
			} else if !bytes.Equal(part, make([]byte, 96)) {
				t.Fatal("未知或過期格被改動")
			}
		}
	}
	clear(out)
	h.draw(out, idx, 2)
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Fatal("二倍畫出HD")
	}
	h.frame(nil, pal)
	if h.active || h.matches[60] != nil {
		t.Fatal("清除後殘留候選")
	}
	h.frame(idx, pal)
	pal[0][0] ^= 1
	h.frame(idx, pal)
	if h.active || h.matches[60] != nil {
		t.Fatal("未知色盤未清候選")
	}
}

func TestHDIssue22Scale3xRejectsAlteredPixelsAndUnknownAlgorithm(t *testing.T) {
	dir, refs := wallFixture(t)
	p := filepath.Join(dir, "walls.json")
	raw, _ := os.ReadFile(p)
	var m wallManifest
	json.Unmarshal(raw, &m)
	ref := refs[0]
	scaled := hdScale3x(ref, 8, 8)
	im := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for n, c := range scaled {
		copy(im.Pix[n*4:n*4+3], combatEGA[c][:])
		im.Pix[n*4+3] = 255
	}
	m.Assets[0].Algorithm = "scale3x"
	write := func() {
		var enc bytes.Buffer
		png.Encode(&enc, im)
		os.WriteFile(filepath.Join(dir, "a.png"), enc.Bytes(), 0644)
		m.Assets[0].PNGSHA = wallHash(enc.Bytes())
		b, _ := json.Marshal(m)
		os.WriteFile(p, b, 0644)
	}
	write()
	if _, err := loadWallHD(dir); err != nil {
		t.Fatal(err)
	}
	im.Pix[(12*24+12)*4] ^= 1
	write()
	if _, err := loadWallHD(dir); err == nil {
		t.Fatal("接受壞Scale3x結果")
	}
	im.Pix[(12*24+12)*4] ^= 1
	m.Assets[0].Algorithm = "nearest"
	write()
	if _, err := loadWallHD(dir); err == nil {
		t.Fatal("接受未知演算法")
	}
}

func TestHDIssue22RealScale3xSources(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_COK_ISSUE22_THEME")
	batch := os.Getenv("DOSGOLEM_COK_ISSUE22_BATCH")
	if dir == "" || batch == "" {
		t.Skip("缺少本機b18來源")
	}
	h, _, err := LoadHDLayers(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(batch, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Items []struct {
			Key, Kind, Indexed string
			OutputSHA          string `json:"output_indexed_sha256"`
		}
	}
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Items) != 26 {
		t.Fatal("b18項數不符")
	}
	for _, it := range m.Items {
		var ref []byte
		if it.Kind == "cursor" {
			for _, p := range h.exact.items {
				if p.key == it.Key {
					ref = p.ref
				}
			}
		} else {
			b, err := os.ReadFile(filepath.Join(dir, "walls", it.Key+".idx"))
			if err != nil {
				t.Fatal(err)
			}
			ref = b
		}
		want, err := os.ReadFile(filepath.Join(batch, it.Indexed))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(hdScale3x(ref, 8, 8), want) || wallHash(want) != it.OutputSHA {
			t.Fatalf("%s 與獨立Python不符", it.Key)
		}
	}
}

func TestHDIssue22PortableCompleteCellsAndAmbiguity(t *testing.T) {
	dir, refs := wallFixture(t)
	p := filepath.Join(dir, "walls.json")
	raw, _ := os.ReadFile(p)
	var m wallManifest
	json.Unmarshal(raw, &m)
	m.Matching = "portable-complete-cells"
	write := func() {
		b, _ := json.Marshal(m)
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	h, err := loadWallHD(dir)
	if err != nil {
		t.Fatal(err)
	}
	ref := append([]byte{}, refs[3]...)
	for n, v := range ref {
		if v == 13 {
			ref[n] = 11
		}
	}
	idx := bytes.Repeat([]byte{5}, 64000)
	putWallCell(idx, 60, ref)
	var pal [256][3]byte
	copy(pal[:], combatEGA[:])
	h.frame(idx, pal)
	if h.matches[60] == nil {
		t.Fatal("完整合成模板仍被原格位限制")
	}
	if bytes.Equal(h.matches[60].pix[12*24*4+16*4:12*24*4+16*4+3], combatEGA[ref[4*8+5]][:]) {
		t.Fatal("跨位置未使用來源HD")
	}
	duplicate := m.Cells[len(m.Cells)-1]
	duplicate.Key = "conflicting-original"
	duplicate.Positions = []int{1}
	duplicate.Paint = false
	m.Cells = append(m.Cells, duplicate)
	write()
	h, err = loadWallHD(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.frame(idx, pal)
	c := h.matches[60]
	if c == nil {
		t.Fatal("歧義完整來源未保留可回查模板")
	}
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			n := (y*24 + x) * 4
			if !bytes.Equal(c.pix[n:n+3], combatEGA[ref[y/3*8+x/3]][:]) {
				t.Fatal("跨位置歧義未保留原版")
			}
		}
	}
	m.Matching = "guess-layout"
	write()
	if _, err := loadWallHD(dir); err == nil {
		t.Fatal("接受未知比對模式")
	}
}
