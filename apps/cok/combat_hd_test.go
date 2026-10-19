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

func writeCombatFixture(t *testing.T, alpha uint8) (string, [][]byte) {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sha := func(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
	img := image.NewNRGBA(image.Rect(0, 0, 72, 72))
	for n := 0; n < len(img.Pix); n += 4 {
		copy(img.Pix[n:n+4], []byte{17, 123, 231, alpha})
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	write("terrain.png", encoded.Bytes())
	refs := make([][]byte, 3)
	items := make([]map[string]any, 3)
	for n := range refs {
		ref := make([]byte, 24*24)
		for i := range ref {
			ref[i] = byte(1 + n*3 + (i/24+i%24+n)%3)
		}
		refs[n] = ref
		name := string(rune('a'+n)) + ".idx"
		write(name, ref)
		items[n] = map[string]any{"key": name, "size": []int{24, 24}, "png": "terrain.png",
			"png_sha256": sha(encoded.Bytes()), "indexed": name, "indexed_sha256": sha(ref)}
	}
	manifest, _ := json.Marshal(map[string]any{"version": 1, "scale": 3, "kind": "combat-terrain",
		"grid": []int{8, 8, 7, 7, 24}, "items": items})
	write("combat.json", manifest)
	write("theme.json", []byte(`{"scale":3,"items":[]}`))
	return dir, refs
}

func TestCombatHD(t *testing.T) {
	dir, refs := writeCombatFixture(t, 255)
	live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(dir); err != nil {
		t.Fatal(err)
	}
	if live.HDItems != 3 {
		t.Fatalf("地形資產數 %d", live.HDItems)
	}
	var pal [256][3]uint8
	copy(pal[:], combatEGA[:])
	screen := bytes.Repeat([]byte{8}, 320*200)
	put := func(x, y, n int) {
		for row := 0; row < 24; row++ {
			copy(screen[(y+row)*320+x:], refs[n][row*24:(row+1)*24])
		}
	}
	clear := func(x, y int) {
		for row := 0; row < 24; row++ {
			copy(screen[(y+row)*320+x:], bytes.Repeat([]byte{8}, 24))
		}
	}
	put(8, 8, 0)
	put(56, 32, 1)
	put(128, 104, 2)
	put(152, 152, 0)
	before := sha256.Sum256(screen)
	live.Frame(screen, pal)
	at := func(x, y int) [3]byte {
		out := live.Compose(3, true, nil)
		n := ((y*3)*960 + x*3) * 4
		return [3]byte{out[n], out[n+1], out[n+2]}
	}
	art := [3]byte{17, 123, 231}
	if at(8, 8) != art || at(175, 175) != art || at(7, 8) != combatEGA[8] {
		t.Fatal("地形範圍或邊界不符")
	}
	if before != sha256.Sum256(screen) || !bytes.Equal(live.Indexed(), screen) {
		t.Fatal("改動原版色號")
	}
	if !bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil)) {
		t.Fatal("二倍模式畫出地形")
	}
	original := live.Compose(3, false, nil)
	n := ((8*3)*960 + 8*3) * 4
	if !bytes.Equal(original[n:n+3], combatEGA[refs[0][0]][:]) {
		t.Fatal("原版檢視被覆蓋")
	}
	clear(8, 8)
	put(32, 128, 0)
	live.Frame(screen, pal)
	if at(8, 8) != combatEGA[8] || at(32, 128) != art {
		t.Fatal("移動後殘影或新位置未顯示")
	}
	screen[(32+12)*320+56+12] ^= 1
	live.Frame(screen, pal)
	if at(56, 32) == art || at(128, 104) != art {
		t.Fatal("一像素失配未回退或影響其他格")
	}
	put(56, 32, 1)
	live.Frame(screen, pal)
	if at(56, 32) != art {
		t.Fatal("遮擋恢復後未重新顯示")
	}
	changed := pal
	changed[1][0]++
	live.Frame(screen, changed)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("未證實色盤仍覆蓋")
	}
	live.Frame(screen, pal)
	if at(32, 128) != art {
		t.Fatal("色盤恢復後未顯示")
	}
	for i := range screen {
		screen[i] = 8
	}
	live.Frame(screen, pal)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("清除後殘影或純色誤啟用")
	}
	put(8, 8, 0)
	put(56, 32, 1)
	live.Frame(screen, pal)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("不足三個辨識地形仍啟用")
	}
	put(128, 104, 2)
	live.Frame(screen, pal)
	live.Frame(screen[:10], pal)
	if len(live.combat.matches) != 0 || live.Compose(3, true, nil) != nil {
		t.Fatal("短畫面沿用舊地形")
	}
}

func TestCombatHDRejects(t *testing.T) {
	for _, kind := range []string{"scale", "grid", "kind", "hash", "size", "duplicate", "alpha"} {
		t.Run(kind, func(t *testing.T) {
			alpha := uint8(255)
			if kind == "alpha" {
				alpha = 254
			}
			dir, _ := writeCombatFixture(t, alpha)
			path := filepath.Join(dir, "combat.json")
			data, _ := os.ReadFile(path)
			var m map[string]any
			_ = json.Unmarshal(data, &m)
			items := m["items"].([]any)
			switch kind {
			case "scale":
				m["scale"] = 2
			case "grid":
				m["grid"] = []int{8, 8, 8, 7, 24}
			case "kind":
				m["kind"] = "unknown"
			case "hash":
				items[0].(map[string]any)["indexed_sha256"] = "bad"
			case "size":
				items[0].(map[string]any)["size"] = []int{24, 25}
			case "duplicate":
				m["items"] = append(items, items[0])
			}
			data, _ = json.Marshal(m)
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
			if err := live.AddHD(dir); err == nil {
				t.Fatal("壞地形資產未拒絕")
			}
			if live.combat != nil || live.HDItems != 0 || live.Layer.Watchers() != 0 {
				t.Fatal("失敗留下部分主題")
			}
		})
	}
}
