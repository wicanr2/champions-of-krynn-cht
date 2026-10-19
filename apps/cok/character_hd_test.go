package cok

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// png.Encode 通常把全不透明圖改存 RGB；此處明確產出契約要求的 RGBA。
type characterRGBA struct{ *image.NRGBA }

func (characterRGBA) Opaque() bool { return false }

func writeCharacterFixture(t *testing.T, dir string, height int, alpha uint8) []byte {
	t.Helper()
	head := bytes.Repeat([]byte{8}, 24*height)
	for y := 2; y < 6; y++ {
		for x := 8; x < 14; x++ {
			head[y*24+x] = byte(4 + 8*(x%2))
		}
	}
	body := bytes.Repeat([]byte{8}, 576)
	for y := 10; y < 18; y++ {
		for x := 3; x < 21; x++ {
			body[y*24+x] = byte(1 + (x+y)%3)
		}
	}
	body[19*24+4] = 1 // 明確非對稱，避免原圖與鏡像重複。
	body[4*24+9] = 3  // 頭部前景必須覆於身體。
	body[1*24+3] = 2  // 頭部背景必須保留身體。
	write := func(name string, b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	makeItem := func(key string, ref []byte, h int, head bool) map[string]any {
		img := image.NewNRGBA(image.Rect(0, 0, 72, h*3))
		for y := 0; y < h*3; y++ {
			for x := 0; x < 72; x++ {
				p := (y*72 + x) * 4
				color := []byte{85, 85, 85, alpha}
				if ref[y/3*24+x/3] != 8 {
					color = []byte{byte(10 + x), byte(110 + y%24), 30, alpha}
					if head {
						color = []byte{byte(210 + x%24), byte(20 + y%24), byte(40 + x%3), alpha}
					}
				}
				copy(img.Pix[p:], color)
			}
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, characterRGBA{img}); err != nil {
			t.Fatal(err)
		}
		write(key+".idx", ref)
		write(key+".png", encoded.Bytes())
		return map[string]any{"key": key, "size": []int{24, h}, "indexed": key + ".idx",
			"indexed_sha256": actorTestSHA(ref), "png": key + ".png", "png_sha256": actorTestSHA(encoded.Bytes())}
	}
	m, _ := json.Marshal(map[string]any{"version": 1, "scale": 3, "kind": "combat-characters",
		"grid": []int{8, 8, 7, 7, 24}, "background_index": 8,
		"heads": []any{makeItem("head", head, height, true)}, "bodies": []any{makeItem("body", body, 24, false)}})
	write("characters.json", m)
	combined := append([]byte(nil), body...)
	for n, c := range head {
		if c != 8 {
			combined[n] = c
		}
	}
	return combined
}

func TestCharacterHD(t *testing.T) {
	for _, height := range []int{8, 9, 10} {
		t.Run(strconv.Itoa(height), func(t *testing.T) {
			dir, guards := writeCombatFixture(t, 255)
			ref := writeCharacterFixture(t, dir, height, 255)
			live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
			if err := live.AddHD(dir); err != nil {
				t.Fatal(err)
			}
			if live.HDItems != 5 || len(live.actors.patterns) != 2 {
				t.Fatal("組件計數或兩個方向不符")
			}
			var pal [256][3]uint8
			copy(pal[:], combatEGA[:])
			screen := bytes.Repeat([]byte{8}, 64000)
			put := func(x, y int, tile []byte) {
				for row := 0; row < 24; row++ {
					copy(screen[(y+row)*320+x:], tile[row*24:(row+1)*24])
				}
			}
			for n, r := range guards {
				put(8+n*24, 8, r)
			}
			put(80, 80, ref)
			before := sha256.Sum256(screen)
			live.Frame(screen, pal)
			at := func(x, y, col int) [3]byte {
				out := live.Compose(3, true, nil)
				p := (y*3*960 + x*3 + col) * 4
				return [3]byte{out[p], out[p+1], out[p+2]}
			}
			if at(88, 82, 0) != [3]byte{210, 26, 40} || at(88, 82, 2) != [3]byte{212, 26, 42} ||
				at(83, 81, 0) != [3]byte{19, 113, 30} || at(89, 84, 0) != [3]byte{213, 32, 40} {
				t.Fatal("頭部層、背景透出身體或子像素位置錯誤")
			}
			if sha256.Sum256(screen) != before || !bytes.Equal(screen, live.Indexed()) {
				t.Fatal("改動原版畫面")
			}
			hd := live.Compose(3, true, nil)
			plain := live.Compose(3, false, nil)
			p := (82*3*960 + 88*3) * 4
			if !bytes.Equal(plain[p:p+3], combatEGA[ref[2*24+8]][:]) ||
				!bytes.Equal(hd, live.Compose(3, true, nil)) ||
				!bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil)) {
				t.Fatal("F12或二倍回退錯誤")
			}
			live.Layer.Add(&xlate.Stamp{Key: "character-text", X: 88, Y: 82, Cells: 1, CellW: 8, CellH: 8,
				Font: live.Font, State: xlate.Shown, BG: [3]byte{9, 19, 29}})
			if at(88, 82, 0) != [3]byte{9, 19, 29} {
				t.Fatal("角色蓋住文字清除區")
			}
			live.Layer.Stamps = nil
			for _, offset := range []int{80*320 + 80, 82*320 + 88} {
				screen[offset] ^= 1
				live.Frame(screen, pal)
				if len(live.actors.matches) != 0 {
					t.Fatal("前景或背景一點變動未回退")
				}
				screen[offset] ^= 1
			}
			live.Frame(screen, pal)
			live.indexed[80*320+80] ^= 1
			if at(88, 82, 0) == [3]byte{210, 26, 40} {
				t.Fatal("繪製前未重新核對完整矩形")
			}
			live.indexed[80*320+80] ^= 1
			put(80, 80, bytes.Repeat([]byte{8}, 576))
			mirrored := make([]byte, 576)
			for y := 0; y < 24; y++ {
				for x := 0; x < 24; x++ {
					mirrored[y*24+x] = ref[y*24+23-x]
				}
			}
			put(128, 128, mirrored)
			live.Frame(screen, pal)
			if at(88, 82, 0) != combatEGA[8] || at(143, 130, 0) != [3]byte{212, 26, 42} ||
				at(143, 130, 2) != [3]byte{210, 26, 40} {
				t.Fatal("移動殘影或鏡像子像素錯誤")
			}
			changed := pal
			changed[2][0]++
			live.Frame(screen, changed)
			if len(live.actors.matches) != 0 {
				t.Fatal("未知色盤仍辨識頭身")
			}
			live.Frame(screen, pal)
			for n := range guards {
				put(8+n*24, 8, bytes.Repeat([]byte{8}, 576))
			}
			live.Frame(screen, pal)
			if len(live.actors.matches) != 0 {
				t.Fatal("地形守衛不足仍辨識頭身")
			}
		})
	}
}

func TestCharacterHDCollisionWithCPIC(t *testing.T) {
	dir := t.TempDir()
	ref := writeCharacterFixture(t, dir, 8, 255)
	characters, err := loadCharacterHD(dir)
	if err != nil {
		t.Fatal(err)
	}
	// 單獨的前景候選只佔同一角色的一部分，完整圖樣可不同仍必須拒絕。
	point := actorInk{8, 2, 2*320 + 8, ref[2*24+8]}
	cpic := &actorHD{sources: 1, patterns: []actorPattern{{key: "cpic-conflict", w: 24, h: 24,
		ink: []actorInk{point}, anchors: []actorInk{point}, pixels: make([]byte, 72*72*4)}}}
	merged := mergeCharacterHD(cpic, characters)
	frame := bytes.Repeat([]byte{8}, 64000)
	for row := 0; row < 24; row++ {
		copy(frame[(80+row)*320+80:], ref[row*24:(row+1)*24])
	}
	merged.frame(frame, true)
	if merged.sources != 3 || len(merged.matches) != 2 {
		t.Fatal("混合候選索引或計數錯誤")
	}
	for _, match := range merged.matches {
		if !match.conflict {
			t.Fatal("混合來源前景衝突未標記")
		}
	}
	out := bytes.Repeat([]byte{71}, 960*600*4)
	before := append([]byte(nil), out...)
	merged.draw(out, frame, 3)
	if !bytes.Equal(out, before) {
		t.Fatal("歧義候選仍繪製")
	}
	merged.frame(frame, false)
	if len(merged.matches) != 0 {
		t.Fatal("失去守衛後仍沿用舊候選")
	}
}

func TestCharacterHDRejectsAssetsAtomically(t *testing.T) {
	for _, kind := range []string{"version", "scale", "grid", "background", "kind", "empty", "limit", "size",
		"hash", "png-hash", "missing", "duplicate", "duplicate-source", "alpha", "foreground", "color"} {
		t.Run(kind, func(t *testing.T) {
			dir, _ := writeCombatFixture(t, 255)
			writeActorFixture(t, dir, 24, 24, false, 255)
			alpha := uint8(255)
			if kind == "alpha" {
				alpha = 254
			}
			writeCharacterFixture(t, dir, 8, alpha)
			path := filepath.Join(dir, "characters.json")
			data, _ := os.ReadFile(path)
			var manifest map[string]any
			_ = json.Unmarshal(data, &manifest)
			heads := manifest["heads"].([]any)
			item := heads[0].(map[string]any)
			switch kind {
			case "version":
				manifest["version"] = 2
			case "scale":
				manifest["scale"] = 2
			case "grid":
				manifest["grid"] = []int{0, 0, 7, 7, 24}
			case "background":
				manifest["background_index"] = 0
			case "kind":
				manifest["kind"] = "unknown"
			case "empty":
				manifest["heads"] = []any{}
			case "limit":
				for len(heads) < 26 {
					heads = append(heads, item)
				}
				manifest["heads"] = heads
			case "size":
				item["size"] = []int{24, 7}
			case "hash":
				item["indexed_sha256"] = "bad"
			case "png-hash":
				item["png_sha256"] = "bad"
			case "missing":
				_ = os.Remove(filepath.Join(dir, "head.png"))
			case "duplicate":
				manifest["heads"] = append(heads, item)
			case "duplicate-source":
				copyItem := map[string]any{}
				for k, v := range item {
					copyItem[k] = v
				}
				copyItem["key"] = "head-copy"
				manifest["heads"] = append(heads, copyItem)
			case "foreground", "color":
				refPath := filepath.Join(dir, "head.idx")
				ref, _ := os.ReadFile(refPath)
				if kind == "foreground" {
					ref = bytes.Repeat([]byte{8}, len(ref))
				} else {
					ref[0] = 16
				}
				_ = os.WriteFile(refPath, ref, 0o644)
				item["indexed_sha256"] = actorTestSHA(ref)
			}
			data, _ = json.Marshal(manifest)
			_ = os.WriteFile(path, data, 0o644)
			live := NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
			if err := live.AddHD(dir); err == nil {
				t.Fatal("壞頭身資產未拒絕")
			}
			if live.HDItems != 0 || live.combat != nil || live.actors != nil || live.Layer.Watchers() != 0 {
				t.Fatal("失敗留下部分主題")
			}
		})
	}
}
