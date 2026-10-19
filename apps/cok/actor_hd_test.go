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

func actorTestSHA(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func writeActorFixture(t *testing.T, dir string, w, h int, overlap bool, alpha uint8) []byte {
	t.Helper()
	ref := bytes.Repeat([]byte{8}, w*h)
	for y := 4; y < 21; y++ {
		for x := 5; x < 17; x++ {
			ref[y*w+x] = byte(1 + (x+2*y)%3)
		}
	}
	ref[6*w+19] = 4
	write := func(name string, b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	items := []map[string]any{}
	for n := 0; n < 1+boolInt(overlap); n++ {
		r := append([]byte(nil), ref...)
		if n == 1 {
			r[20*w+20] = 5
		}
		img := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
		for y := 0; y < h*3; y++ {
			for x := 0; x < w*3; x++ {
				o := (y*w*3 + x) * 4
				color := []byte{85, 85, 85, alpha}
				if r[(y/3)*w+x/3] != 8 {
					color = []byte{byte(100 + x), 113, 42, alpha}
				}
				copy(img.Pix[o:], color)
			}
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			t.Fatal(err)
		}
		key := "actor-" + string(rune('a'+n))
		write(key+".idx", r)
		write(key+".png", encoded.Bytes())
		items = append(items, map[string]any{"key": key, "size": []int{w, h}, "indexed": key + ".idx",
			"indexed_sha256": actorTestSHA(r), "png": key + ".png", "png_sha256": actorTestSHA(encoded.Bytes())})
	}
	m, _ := json.Marshal(map[string]any{"version": 1, "scale": 3, "kind": "combat-actors",
		"grid": []int{8, 8, 7, 7, 24}, "background_index": 8, "items": items})
	write("actors.json", m)
	return ref
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestActorHD(t *testing.T) {
	dir, guards := writeCombatFixture(t, 255)
	ref := writeActorFixture(t, dir, 24, 24, false, 255)
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	if live.HDItems != 4 {
		t.Fatalf("資產數 %d", live.HDItems)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{8}, 320*200)
	put := func(x, y, w, h int, pixels []byte) {
		for row := 0; row < h; row++ {
			copy(screen[(y+row)*320+x:], pixels[row*w:(row+1)*w])
		}
	}
	for n, r := range guards {
		put(8+n*24, 8, 24, 24, r)
	}
	put(80, 80, 24, 24, ref)
	screen[80*320+80] = 15 // 來源背景中的異物必須保留。
	before := sha256.Sum256(screen)
	live.Frame(screen, pal)
	at := func(x, y int) [3]byte {
		out := live.Compose(3, true, nil)
		p := (y*3*960 + x*3) * 4
		return [3]byte{out[p], out[p+1], out[p+2]}
	}
	if at(85, 84) != [3]byte{115, 113, 42} || at(80, 80) != combatEGA[15] {
		t.Fatal("前景、來源背景異物或像素位置錯誤")
	}
	if sha256.Sum256(screen) != before || !bytes.Equal(screen, live.Indexed()) {
		t.Fatal("改動原版色號")
	}
	if !bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil)) {
		t.Fatal("二倍畫出角色")
	}
	hdBeforeToggle := live.Compose(3, true, nil)
	plain := live.Compose(3, false, nil)
	p := (84*3*960 + 85*3) * 4
	if !bytes.Equal(plain[p:p+3], combatEGA[ref[4*24+5]][:]) {
		t.Fatal("F12 原版檢視被覆蓋")
	}
	if !bytes.Equal(hdBeforeToggle, live.Compose(3, true, nil)) {
		t.Fatal("切回不穩定")
	}
	live.Layer.Add(&xlate.Stamp{Key: "cover-actor", X: 84, Y: 84, Cells: 1, CellW: 8, CellH: 8,
		Font: live.Font, State: xlate.Shown, BG: [3]byte{9, 19, 29}})
	if at(85, 84) != [3]byte{9, 19, 29} {
		t.Fatal("角色蓋住文字的背景清除區")
	}
	live.Layer.Stamps = nil
	screen[84*320+85] ^= 1
	live.Frame(screen, pal)
	if at(85, 84) == [3]byte{115, 113, 42} {
		t.Fatal("前景一點失配未回退")
	}
	put(80, 80, 24, 24, bytes.Repeat([]byte{8}, 576))
	mirrored := make([]byte, 576)
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			mirrored[y*24+x] = ref[y*24+23-x]
		}
	}
	put(128, 128, 24, 24, mirrored)
	live.Frame(screen, pal)
	if at(85, 84) != combatEGA[8] || at(146, 132) != [3]byte{117, 113, 42} {
		t.Fatal("移動殘影或鏡像錯誤")
	}
	changed := pal
	changed[1][0]++
	live.Frame(screen, changed)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("未知色盤未回退")
	}
	live.Frame(screen, pal)
	if at(146, 132) != [3]byte{117, 113, 42} {
		t.Fatal("恢復後未重新識別")
	}
	for n := range guards {
		put(8+n*24, 8, 24, 24, bytes.Repeat([]byte{8}, 576))
	}
	live.Frame(screen, pal)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("缺地形守衛仍畫角色")
	}
	live.Frame(screen[:10], pal)
	if len(live.actors.matches) != 0 || live.Compose(3, true, nil) != nil {
		t.Fatal("短畫面沿用舊角色")
	}
}

func TestActorHDRejectsAmbiguity(t *testing.T) {
	dir, guards := writeCombatFixture(t, 255)
	ref := writeActorFixture(t, dir, 24, 24, true, 255)
	ref[20*24+20] = 5 // 同時符合原始前景及增加一點的候選。
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{8}, 64000)
	for n, r := range guards {
		for y := 0; y < 24; y++ {
			copy(screen[(8+y)*320+8+n*24:], r[y*24:(y+1)*24])
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
			t.Fatal("歧義候選未標示")
		}
	}
	hd, base := live.Compose(3, true, nil), live.Compose(3, false, nil)
	for y := 80 * 3; y < 104*3; y++ {
		if !bytes.Equal(hd[(y*960+80*3)*4:(y*960+104*3)*4], base[(y*960+80*3)*4:(y*960+104*3)*4]) {
			t.Fatal("歧義區仍被繪製")
		}
	}
}

func TestActorHDRejectsAssets(t *testing.T) {
	for _, kind := range []string{"version", "scale", "grid", "background", "size", "hash", "duplicate", "alpha", "foreground-colors"} {
		t.Run(kind, func(t *testing.T) {
			dir, _ := writeCombatFixture(t, 255)
			alpha := uint8(255)
			if kind == "alpha" {
				alpha = 254
			}
			writeActorFixture(t, dir, 24, 24, false, alpha)
			path := filepath.Join(dir, "actors.json")
			b, _ := os.ReadFile(path)
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			items := m["items"].([]any)
			switch kind {
			case "version":
				m["version"] = 2
			case "scale":
				m["scale"] = 2
			case "grid":
				m["grid"] = []int{0, 0, 7, 7, 24}
			case "background":
				m["background_index"] = 0
			case "size":
				items[0].(map[string]any)["size"] = []int{25, 24}
			case "hash":
				items[0].(map[string]any)["indexed_sha256"] = "bad"
			case "duplicate":
				m["items"] = append(items, items[0])
			case "foreground-colors":
				refPath := filepath.Join(dir, "actor-a.idx")
				ref, _ := os.ReadFile(refPath)
				for n, c := range ref {
					if c != 8 {
						ref[n] = 15
					}
				}
				if err := os.WriteFile(refPath, ref, 0o644); err != nil {
					t.Fatal(err)
				}
				items[0].(map[string]any)["indexed_sha256"] = actorTestSHA(ref)
			}
			b, _ = json.Marshal(m)
			if err := os.WriteFile(path, b, 0o644); err != nil {
				t.Fatal(err)
			}
			live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
			if err := live.AddHD(dir); err == nil {
				t.Fatal("壞資產未拒絕")
			}
			if live.HDItems != 0 || live.combat != nil || live.actors != nil || live.Layer.Watchers() != 0 {
				t.Fatal("載入失敗留下部分主題")
			}
		})
	}
}

func TestActorHDMonochromeSource(t *testing.T) {
	dir, _ := writeCombatFixture(t, 255)
	ref := writeActorFixture(t, dir, 24, 24, false, 255)
	for n, c := range ref {
		if c != 8 {
			ref[n] = 7 + 8*(c%2)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "actor-a.idx"), ref, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "actors.json")
	b, _ := os.ReadFile(path)
	var manifest map[string]any
	_ = json.Unmarshal(b, &manifest)
	manifest["items"].([]any)[0].(map[string]any)["indexed_sha256"] = actorTestSHA(ref)
	b, _ = json.Marshal(manifest)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatalf("原版白銀兩色加背景來源遭拒：%v", err)
	}
}
