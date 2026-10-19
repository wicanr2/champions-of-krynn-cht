package cok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// 執行入口：tools/issue12-hd-receipt.sh。此測試只讀已產生的主題及畫面，
// 來源衍生正負對照與原版保存畫面在收據分開列出。
func TestHDIssue12Receipts(t *testing.T) {
	root := os.Getenv("DOSGOLEM_COK_HD_RECEIPT")
	if root == "" {
		t.Skip("未指定本機 HD 素材")
	}
	items, err := LoadHDTheme(filepath.Join(root, "theme"))
	if err != nil {
		t.Fatal(err)
	}
	var pal [256][3]uint8
	colors := [16][3]uint8{{0, 0, 0}, {0, 0, 170}, {0, 170, 0}, {0, 170, 170}, {170, 0, 0}, {170, 0, 170}, {170, 85, 0}, {170, 170, 170}, {85, 85, 85}, {85, 85, 255}, {85, 255, 85}, {85, 255, 255}, {255, 85, 85}, {255, 85, 255}, {255, 255, 85}, {255, 255, 255}}
	copy(pal[:], colors[:])
	newLive := func() *Live {
		l := NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
		for _, it := range items {
			l.Layer.Watch(it.Watcher())
		}
		l.HDItems = len(items)
		return l
	}
	rows := []map[string]interface{}{}
	for _, it := range items {
		screen := make([]byte, 64000)
		for i := range screen {
			screen[i] = byte((i/320*3 + i) % 16)
		}
		for y := 0; y < it.H; y++ {
			copy(screen[(it.Y+y)*320+it.X:], it.Reference[y*it.W:(y+1)*it.W])
		}
		before := append([]byte(nil), screen...)
		l := newLive()
		l.Frame(screen, pal)
		out := l.Compose(3, true, nil)
		ok := true
		for y := 0; y < it.H*3; y++ {
			start := ((it.Y*3+y)*960 + it.X*3) * 4
			if !bytes.Equal(out[start:start+it.W*3*4], it.Pix[y*it.W*3*4:(y+1)*it.W*3*4]) {
				ok = false
				break
			}
		}
		ok = ok && bytes.Equal(l.Compose(2, true, nil), l.Compose(2, false, nil)) && bytes.Equal(before, screen)
		rows = append(rows, map[string]interface{}{"key": it.Key, "class": "source-derived positive fixture, NOT oracle capture", "scale3_exact_hd": ok, "scale2_native": bytes.Equal(l.Compose(2, true, nil), l.Compose(2, false, nil)), "ok": ok})
		if !ok {
			t.Errorf("%s 正對照失敗", it.Key)
		}
		screen[it.Y*320+it.X] ^= 1
		l.Frame(screen, pal)
		ok = bytes.Equal(l.Compose(3, true, nil), l.Compose(3, false, nil))
		rows = append(rows, map[string]interface{}{"key": it.Key, "class": "source-derived mismatch fixture, NOT oracle capture", "ok": ok})
		if !ok {
			t.Errorf("%s 負對照未回退", it.Key)
		}
	}
	var frames struct {
		Items []struct {
			File   string `json:"file"`
			Source string `json:"source"`
		} `json:"items"`
	}
	b, err := os.ReadFile(filepath.Join(root, "screens/normal/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &frames); err != nil {
		t.Fatal(err)
	}
	for _, frame := range frames.Items {
		screen, err := os.ReadFile(filepath.Join(root, "screens/normal", frame.File))
		if err != nil {
			t.Fatal(err)
		}
		if len(screen) != 64000 {
			t.Fatalf("畫面長度 %s", frame.File)
		}
		l := newLive()
		before := append([]byte(nil), screen...)
		l.Frame(screen, pal)
		a, b := l.Compose(3, true, nil), l.Compose(3, false, nil)
		allowed := make([]bool, 64000)
		matches := []string{}
		for _, it := range items {
			match := true
			for y := 0; y < it.H; y++ {
				if !bytes.Equal(screen[(it.Y+y)*320+it.X:(it.Y+y)*320+it.X+it.W], it.Reference[y*it.W:(y+1)*it.W]) {
					match = false
					break
				}
			}
			if match {
				matches = append(matches, it.Key)
				for y := 0; y < it.H; y++ {
					for x := 0; x < it.W; x++ {
						allowed[(it.Y+y)*320+it.X+x] = true
					}
				}
			}
		}
		diff, outside := 0, 0
		for y := 0; y < 600; y++ {
			for x := 0; x < 960; x++ {
				p := (y*960 + x) * 4
				if !bytes.Equal(a[p:p+4], b[p:p+4]) {
					diff++
					if !allowed[(y/3)*320+x/3] {
						outside++
					}
				}
			}
		}
		ok := outside == 0 && (len(matches) == 0 || diff > 0) && bytes.Equal(before, screen) && bytes.Equal(l.Compose(2, true, nil), l.Compose(2, false, nil))
		rows = append(rows, map[string]interface{}{"class": "saved original screen regression", "source": frame.Source, "matches": matches, "scale3_difference": diff, "outside_difference": outside, "ok": ok})
		if !ok {
			t.Errorf("保存畫面回歸失敗 %s", frame.Source)
		}
	}
	b, err = json.MarshalIndent(map[string]interface{}{"theme_items": len(items), "normal_frames": len(frames.Items), "items": rows, "all_pass": !t.Failed()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "out/issue12-receipts.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("HD %d 項，來源正負對照 %d，既有畫面 %d\n", len(items), len(items)*2, len(frames.Items))
}
