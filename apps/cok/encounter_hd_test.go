package cok

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/wicanr2/dosgolem/xlate"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func hdFixtureItem(t *testing.T, dir, key string, w, h int, ref []byte) map[string]any {
	t.Helper()
	hash := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	im := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
	for n := 0; n < len(im.Pix); n += 4 {
		copy(im.Pix[n:n+4], []byte{19, 77, 133, 255})
	}
	var b bytes.Buffer
	if err := png.Encode(&b, characterRGBA{im}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, key+".png"), b.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, key+".idx"), ref, 0644); err != nil {
		t.Fatal(err)
	}
	return map[string]any{"key": key, "size": []int{w, h}, "png": key + ".png", "png_sha256": hash(b.Bytes()), "indexed": key + ".idx", "indexed_sha256": hash(ref)}
}
func hdFixtureManifest(t *testing.T, dir, name string, m any) {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEncounterHDMappingAndFallback(t *testing.T) {
	dir, _ := writeCombatFixture(t, 255)
	ref := make([]byte, 64)
	for y := 1; y < 7; y++ {
		for x := 1; x < 7; x++ {
			ref[y*8+x] = byte(1 + (x+y)%3)
			if x == 1 {
				ref[y*8+x] = 13
			}
		}
	}
	it := hdFixtureItem(t, dir, "enc", 8, 8, ref)
	it["position"] = []int{80, 80}
	m := map[string]any{"version": 1, "scale": 3, "kind": "encounter-sprites", "items": []any{it}}
	hdFixtureManifest(t, dir, "encounters.json", m)
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{3}, 64000)
	for n, v := range ref {
		if v == 0 {
			continue
		}
		if v == 13 {
			v = 0
		}
		screen[(80+n/8)*320+80+n%8] = v
	}
	before := append([]byte(nil), screen...)
	live.Frame(screen, pal)
	if len(live.encounters.matches) != 1 {
		t.Fatal("映射13→0未命中")
	}
	hd := live.Compose(3, true, nil)
	native := live.Compose(3, false, nil)
	at := func(x, y int) int { return (y*3*960 + x*3) * 4 }
	if !bytes.Equal(hd[at(81, 81):at(81, 81)+4], []byte{19, 77, 133, 255}) {
		t.Fatal("前景未繪製")
	}
	if !bytes.Equal(hd[at(80, 80):at(80, 80)+4], native[at(80, 80):at(80, 80)+4]) {
		t.Fatal("來源0區被覆蓋")
	}
	if !bytes.Equal(before, screen) || !bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil)) {
		t.Fatal("改動原版或二倍")
	}
	screen[80*320+80] = 12
	live.Frame(screen, pal)
	if len(live.encounters.matches) != 1 {
		t.Fatal("背景改變使前景失效")
	}
	screen[81*320+81] = 1
	live.Frame(screen, pal)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("一點前景失配沒有回退")
	}
	screen[81*320+81] = 0
	live.Frame(screen, pal)
	if len(live.encounters.matches) != 1 {
		t.Fatal("前景恢復失敗")
	}
	pal[1][0]++
	live.Frame(screen, pal)
	if len(live.encounters.matches) != 0 {
		t.Fatal("未知色盤仍畫")
	}
	pal[1] = combatEGA[1]
	live.Frame(screen[:2], pal)
	if len(live.encounters.matches) != 0 {
		t.Fatal("短畫面沿用舊圖")
	}
	// 另一來源只多一個與背景同色的前景點，兩者都命中時必須全回退。
	second := append([]byte(nil), ref...)
	second[0] = 3
	other := hdFixtureItem(t, dir, "enc2", 8, 8, second)
	other["position"] = []int{80, 80}
	m["items"] = []any{it, other}
	hdFixtureManifest(t, dir, "encounters.json", m)
	fresh := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := fresh.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	screen[80*320+80] = 3
	fresh.Frame(screen, pal)
	if len(fresh.encounters.matches) != 0 {
		t.Fatal("候選交集未全回退")
	}
	// 任一可選家族載入失敗，不登記其他有效家族。
	m["scale"] = 2
	hdFixtureManifest(t, dir, "encounters.json", m)
	bad := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := bad.AddHD(dir); err == nil || bad.HDItems != 0 || bad.Layer.Watchers() != 0 {
		t.Fatal("壞家族沒有原子拒絕")
	}
}

func TestEffectHDRequiresFullRectangle(t *testing.T) {
	dir, refs := writeCombatFixture(t, 255)
	ref := bytes.Repeat([]byte{8}, 576)
	for x := 6; x < 18; x++ {
		ref[12*24+x] = byte(11 + 4*(x%2))
	}
	it := hdFixtureItem(t, dir, "effect", 24, 24, ref)
	hdFixtureManifest(t, dir, "effects.json", map[string]any{"version": 1, "scale": 3, "kind": "combat-effects", "grid": []int{8, 8, 7, 7, 24}, "background_index": 8, "items": []any{it}})
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{8}, 64000)
	for n, p := range refs {
		for y := 0; y < 24; y++ {
			copy(screen[(8+y)*320+8+n*24:], p[y*24:(y+1)*24])
		}
	}
	for y := 0; y < 24; y++ {
		copy(screen[(80+y)*320+80:], ref[y*24:(y+1)*24])
	}
	live.Frame(screen, pal)
	hits := func() int {
		count := 0
		for _, m := range live.actors.matches {
			if live.actors.patterns[m.pattern].key == "effect" && !m.conflict {
				count++
			}
		}
		return count
	}
	if hits() != 1 {
		t.Fatal("完整特效未匹配")
	}
	screen[80*320+80] = 3
	live.Frame(screen, pal)
	if hits() != 0 {
		t.Fatal("背景失配仍畫特效")
	}
	screen[80*320+80] = 8
	live.Frame(screen, pal)
	if hits() != 1 {
		t.Fatal("恢復失敗")
	}
	for n := range screen {
		screen[n] = 8
	}
	live.Frame(screen, pal)
	if hits() != 0 {
		t.Fatal("沒有地形守衛仍畫")
	}
}

func TestEffectHDConflictsWithCPIC(t *testing.T) {
	dir, guards := writeCombatFixture(t, 255)
	ref := writeActorFixture(t, dir, 24, 24, false, 255)
	it := hdFixtureItem(t, dir, "effect", 24, 24, ref)
	hdFixtureManifest(t, dir, "effects.json", map[string]any{"version": 1, "scale": 3, "kind": "combat-effects", "grid": []int{8, 8, 7, 7, 24}, "background_index": 8, "items": []any{it}})
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{8}, 64000)
	for n, p := range guards {
		for y := 0; y < 24; y++ {
			copy(screen[(8+y)*320+8+n*24:], p[y*24:(y+1)*24])
		}
	}
	for y := 0; y < 24; y++ {
		copy(screen[(80+y)*320+80:], ref[y*24:(y+1)*24])
	}
	live.Frame(screen, pal)
	if len(live.actors.matches) != 2 {
		t.Fatalf("候選數 %d", len(live.actors.matches))
	}
	for _, m := range live.actors.matches {
		if !m.conflict {
			t.Fatal("CPIC與COMSPR未共同回退")
		}
	}
	out := live.Compose(3, true, nil)
	at := ((90*3)*960 + 90*3) * 4
	color := combatEGA[ref[10*24+10]]
	if !bytes.Equal(out[at:at+3], color[:]) {
		t.Fatal("衝突後仍繪製前景")
	}
}
