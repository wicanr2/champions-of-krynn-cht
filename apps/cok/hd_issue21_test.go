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

func exactTestPattern(w, h int) exactHDItem {
	p := exactHDItem{w: w, h: h, ref: make([]byte, w*h), pixels: make([]byte, w*h*36)}
	for n := range p.ref {
		p.ref[n] = byte((n*7+n/w)%15 + 1)
	}
	for n := 0; n < len(p.pixels); n += 4 {
		copy(p.pixels[n:n+4], []byte{1, 2, 3, 255})
	}
	p.makeAnchor()
	return p
}
func exactTestPut(idx []byte, p exactHDItem, x, y int) {
	for row := 0; row < p.h; row++ {
		copy(idx[(y+row)*320+x:], p.ref[row*p.w:(row+1)*p.w])
	}
}
func TestHDIssue21ExactTransientAndFallback(t *testing.T) {
	p := exactTestPattern(24, 24)
	h := &exactHD{items: []exactHDItem{p}}
	idx := make([]byte, 64000)
	exactTestPut(idx, p, 37, 59)
	var pal [256][3]byte
	copy(pal[:], combatEGA[:])
	before := append([]byte{}, idx...)
	h.frame(idx, pal)
	if len(h.matches) != 1 || h.matches[0].x != 37 || h.matches[0].y != 59 {
		t.Fatal("漏掉非格點完整來源")
	}
	out := make([]byte, 960*600*4)
	h.draw(out, idx, 2)
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Fatal("二倍畫了HD")
	}
	h.draw(out, idx, 3)
	if bytes.Equal(out, make([]byte, len(out))) || !bytes.Equal(idx, before) {
		t.Fatal("沒有畫HD或改了原色號")
	}
	// 畫前的一點變更須令整項失效，不能留下其餘格子。
	idx[59*320+37] ^= 1
	clear(out)
	h.draw(out, idx, 3)
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Fatal("畫前失配未整項回退")
	}
	h.frame(idx, pal)
	if len(h.matches) != 0 {
		t.Fatal("失配仍命中")
	}
	copy(idx, before)
	pal[2][1] ^= 1
	h.frame(idx, pal)
	if len(h.matches) != 0 {
		t.Fatal("未知色盤仍命中")
	}
	copy(pal[:], combatEGA[:])
	h.frame(nil, pal)
	if len(h.matches) != 0 {
		t.Fatal("清除後候選殘留")
	}
}
func TestHDIssue21ExactOverlapRejectsAll(t *testing.T) {
	a := exactTestPattern(8, 8)
	b := exactTestPattern(8, 8)
	for y := 0; y < 8; y++ {
		for x := 0; x < 4; x++ {
			b.ref[y*8+x] = a.ref[y*8+x+4]
		}
		for x := 4; x < 8; x++ {
			b.ref[y*8+x] = byte(1 + (y+x*3)%15)
		}
	}
	b.makeAnchor()
	h := &exactHD{items: []exactHDItem{a, b}}
	idx := make([]byte, 64000)
	exactTestPut(idx, a, 20, 20)
	exactTestPut(idx, b, 24, 20)
	var pal [256][3]byte
	copy(pal[:], combatEGA[:])
	h.frame(idx, pal)
	if len(h.matches) != 2 {
		t.Fatalf("預期兩個完整重疊來源，得到%d", len(h.matches))
	}
	for _, m := range h.matches {
		if !m.conflict {
			t.Fatal("重疊來源未全部拒絕")
		}
	}
	out := make([]byte, 960*600*4)
	h.draw(out, idx, 3)
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Fatal("歧義來源被畫出")
	}
}
func TestHDIssue21WallCompleteBackground(t *testing.T) {
	dir, refs := wallFixture(t)
	path := filepath.Join(dir, "walls.json")
	raw, _ := os.ReadFile(path)
	var m wallManifest
	json.Unmarshal(raw, &m)
	ref := append([]byte{}, refs[3]...)
	for n, v := range ref {
		if v == 13 {
			ref[n] = 4
		}
	}
	m.Cells = append(m.Cells, wallCellSpec{Key: "complete-unobserved", Base: 4, Layers: []wallLayerSpec{{Key: "d"}}, IndexSHA: wallHash(ref), Paint: true, CompleteBackground: true})
	write := func() {
		b, _ := json.Marshal(m)
		if err := os.WriteFile(path, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	h, err := loadWallHD(dir)
	if err != nil {
		t.Fatal(err)
	}
	idx := make([]byte, 64000)
	for n := 0; n < 3; n++ {
		putWallCell(idx, n, refs[n])
	}
	putWallCell(idx, 60, ref)
	var pal [256][3]byte
	copy(pal[:], combatEGA[:])
	h.frame(idx, pal)
	if !h.active {
		t.Fatal("完整來源加底色未命中")
	}
	idx[(24+60/11*8)*320+24+60%11*8] = 5
	h.frame(idx, pal)
	if !h.active || h.matches[60] != nil || h.matches[0] == nil {
		t.Fatal("留底區失配未限於單格")
	}
	m.Cells[len(m.Cells)-1].CompleteBackground = false
	write()
	if _, err = loadWallHD(dir); err == nil {
		t.Fatal("未授權的留底通用格被接受")
	}
}
func TestHDIssue21ThemeCoverage(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_COK_ISSUE21_THEME")
	if dir == "" {
		t.Skip("缺少本機HD主題，略過來源矩陣")
	}
	h, count, err := LoadHDLayers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if count < 1800 || h.exact == nil || len(h.exact.items) < 51 {
		t.Fatalf("來源未載入：%d", count)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "walls.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m wallManifest
	json.Unmarshal(raw, &m)
	checked := 0
	for _, a := range m.Assets {
		if a.Kind != "wall" {
			continue
		}
		ref, err := readChecked(filepath.Join(dir, a.Indexed), a.IndexSHA)
		if err != nil {
			t.Fatal(err)
		}
		bases := []byte{0}
		if bytes.IndexByte(ref, 13) >= 0 {
			bases = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
		}
		for _, base := range bases {
			var full [64]byte
			for n, v := range ref {
				if v == 13 {
					v = base
				}
				full[n] = v
			}
			found := h.walls.global[full] != nil
			for pos := 0; pos < 121 && !found; pos++ {
				found = h.walls.cells[pos][full] != nil
			}
			if !found {
				t.Fatalf("%s 底色%d沒有可達完整模板", a.Key, base)
			}
			checked++
		}
	}
	// 歧義模板的完整像素必須保留原版，不能只檢查它有登錄。
	for _, c := range m.Cells {
		if c.Paint {
			continue
		}
		var sum [64]byte
		// 載入器已核對 SHA；由 lookup 找回完整來源，核對所有三倍像素。
		var cell *wallCell
		for ref, v := range h.walls.global {
			if wallHash(ref[:]) == c.IndexSHA {
				sum = ref
				cell = v
				break
			}
		}
		if cell == nil && len(c.Positions) > 0 {
			for ref, v := range h.walls.cells[c.Positions[0]] {
				if wallHash(ref[:]) == c.IndexSHA {
					sum = ref
					cell = v
					break
				}
			}
		}
		if cell == nil {
			t.Fatal("歧義模板不存在")
		}
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				n := (y*24 + x) * 4
				color := combatEGA[sum[y/3*8+x/3]]
				if !bytes.Equal(cell.pix[n:n+3], color[:]) || cell.pix[n+3] != 255 {
					t.Fatal("歧義模板未保留原版")
				}
			}
		}
	}
	t.Logf("核對%d個完整牆面來源模板", checked)
}

type exactTestRGBA struct{ image.Image }

func (exactTestRGBA) Opaque() bool { return false }

func TestHDIssue21ExactLoadAtomic(t *testing.T) {
	dir, _ := wallFixture(t)
	raw, _ := os.ReadFile(filepath.Join(dir, "walls.json"))
	var walls wallManifest
	json.Unmarshal(raw, &walls)
	// Go 會將不透明 NRGBA 自動編成 RGB；此來源契約明訂 RGBA PNG。
	pngPath := filepath.Join(dir, walls.Assets[0].PNG)
	encoded, _ := os.ReadFile(pngPath)
	im, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	var rgba bytes.Buffer
	if err = png.Encode(&rgba, exactTestRGBA{im}); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(pngPath, rgba.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	walls.Assets[0].PNGSHA = wallHash(rgba.Bytes())
	wallBytes, _ := json.Marshal(walls)
	os.WriteFile(filepath.Join(dir, "walls.json"), wallBytes, 0644)
	type manifest struct {
		Version int             `json:"version"`
		Scale   int             `json:"scale"`
		Kind    string          `json:"kind"`
		Items   []wallAssetSpec `json:"items"`
	}
	m := manifest{1, 3, "complete-source-patterns", walls.Assets[:1]}
	write := func() {
		b, _ := json.Marshal(m)
		if err := os.WriteFile(filepath.Join(dir, "exact.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if _, err := loadExactHD(dir); err != nil {
		t.Fatal(err)
	}
	m.Items = append(m.Items, m.Items[0])
	write()
	if _, err := loadExactHD(dir); err == nil {
		t.Fatal("重複完整來源被接受")
	}
	m.Items = m.Items[:1]
	m.Items[0].Indexed = "../escape.idx"
	write()
	if _, err := loadExactHD(dir); err == nil {
		t.Fatal("來源路徑逃出主題")
	}
	// 載入錯誤不能留下已登記的其他家族。
	l := NewHDLive(nil, nil, nil)
	if err := l.AddHD(dir); err == nil || l.HDItems != 0 {
		t.Fatal("失敗後留下部分主題")
	}
	m.Items[0].Indexed = "a.idx"
	m.Items[0].PNG = "missing.png"
	write()
	if err := l.AddHD(dir); err == nil || l.HDItems != 0 {
		t.Fatal("撤除素材未原子拒絕")
	}
	if err := os.Remove(filepath.Join(dir, "exact.json")); err != nil {
		t.Fatal(err)
	}
	if h, err := loadExactHD(dir); err != nil || h != nil {
		t.Fatal("可選家族撤除後未回退")
	}
}

func TestHDIssue21ObservedMovingEffects(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_COK_ISSUE21_THEME")
	probe := os.Getenv("DOSGOLEM_COK_ISSUE21_PROBE")
	if dir == "" || probe == "" {
		t.Skip("缺少本機原版COMSPR畫格")
	}
	h, _, err := LoadHDLayers(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		file, key string
		x, y      int
	}{
		{"frame-001572780000.bin", "comspr-005-00-00", 64, 96},
		{"frame-001573110000.bin", "comspr-133-00-00", 56, 96},
	} {
		idx, err := os.ReadFile(filepath.Join(probe, test.file))
		if err != nil {
			t.Fatal(err)
		}
		var pal [256][3]byte
		copy(pal[:], combatEGA[:])
		h.Frame(idx, pal)
		found := false
		for _, m := range h.actors.matches {
			p := h.actors.patterns[m.pattern]
			if p.key == test.key && m.x == test.x && m.y == test.y && !m.conflict {
				found = true
			}
		}
		if !found {
			t.Fatalf("本作完整飛行物未命中：%s", test.file)
		}
	}
}
