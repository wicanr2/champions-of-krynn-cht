package cok

// SPRIT 使用獨立映射13→0，原始0區不畫。規格010 §6.3。
import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type encounterItem struct {
	key        string
	x, y, w, h int
	ink        []actorInk
	pixels     []byte
}
type encounterHD struct {
	items   []encounterItem
	matches []int
}

func loadEncounterHD(dir string) (*encounterHD, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "encounters.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m struct {
		Version int    `json:"version"`
		Scale   int    `json:"scale"`
		Kind    string `json:"kind"`
		Items   []struct {
			characterItem
			Position [2]int `json:"position"`
		} `json:"items"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || m.Scale != 3 || m.Kind != "encounter-sprites" || len(m.Items) < 1 || len(m.Items) > 108 {
		return nil, fmt.Errorf("encounters.json 不符規格010 §6.3")
	}
	result := &encounterHD{}
	keys := map[string]bool{}
	refs := map[string]bool{}
	for _, it := range m.Items {
		w, h := it.Size[0], it.Size[1]
		x, y := it.Position[0], it.Position[1]
		if it.Key == "" || keys[it.Key] || w < 8 || w > 88 || w%8 != 0 || h < 1 || h > 88 || x < 24 || y < 24 || x+w > 112 || y+h > 112 || x%8 != 0 || y%8 != 0 {
			return nil, fmt.Errorf("SPRIT key尺寸或位置不合法：%s", it.Key)
		}
		keys[it.Key] = true
		ref, err := readChecked(filepath.Join(dir, it.Indexed), it.IndexSHA)
		if err != nil {
			return nil, err
		}
		if len(ref) != w*h {
			return nil, fmt.Errorf("SPRIT基準長度不符")
		}
		signature := fmt.Sprintf("%d,%d,%d,%d:%s", x, y, w, h, string(ref))
		if refs[signature] {
			return nil, fmt.Errorf("SPRIT重複基準")
		}
		refs[signature] = true
		item := encounterItem{key: it.Key, x: x, y: y, w: w, h: h}
		colors := map[byte]bool{}
		for n, v := range ref {
			if v > 15 {
				return nil, fmt.Errorf("SPRIT非EGA色號")
			}
			if v == 0 {
				continue
			}
			if v == 13 {
				v = 0
			}
			colors[v] = true
			item.ink = append(item.ink, actorInk{n % w, n / w, (y+n/w)*320 + x + n%w, v})
		}
		if len(item.ink) < 16 || len(colors) < 3 {
			return nil, fmt.Errorf("SPRIT前景資訊不足")
		}
		encoded, err := readChecked(filepath.Join(dir, it.PNG), it.PNGSHA)
		if err != nil {
			return nil, err
		}
		im, err := png.Decode(bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		if im.Bounds().Size() != (image.Point{X: w * 3, Y: h * 3}) {
			return nil, fmt.Errorf("SPRIT三倍尺寸不符")
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
		draw.Draw(rgba, rgba.Bounds(), im, im.Bounds().Min, draw.Src)
		for n := 3; n < len(rgba.Pix); n += 4 {
			if rgba.Pix[n] != 255 {
				return nil, fmt.Errorf("SPRIT透明PNG")
			}
		}
		item.pixels = rgba.Pix
		result.items = append(result.items, item)
	}
	return result, nil
}
func (it *encounterItem) present(idx []byte) bool {
	for _, p := range it.ink {
		if idx[p.offset] != p.color {
			return false
		}
	}
	return true
}
func (h *encounterHD) frame(idx []byte, pal [256][3]uint8) {
	if h == nil {
		return
	}
	h.matches = h.matches[:0]
	if len(idx) != 64000 {
		return
	}
	for n, c := range combatEGA {
		if pal[n] != c {
			return
		}
	}
	candidates := []int{}
	for n := range h.items {
		if h.items[n].present(idx) {
			candidates = append(candidates, n)
		}
	}
	owners := map[int]int{}
	conflicts := map[int]bool{}
	for _, n := range candidates {
		for _, p := range h.items[n].ink {
			if old, ok := owners[p.offset]; ok {
				conflicts[n] = true
				conflicts[old] = true
			} else {
				owners[p.offset] = n
			}
		}
	}
	for _, n := range candidates {
		if !conflicts[n] {
			h.matches = append(h.matches, n)
		}
	}
}
func (h *encounterHD) draw(out, idx []byte, scale int) {
	if h == nil || scale != 3 || len(out) != 960*600*4 || len(idx) != 64000 {
		return
	}
	for _, n := range h.matches {
		it := &h.items[n]
		if !it.present(idx) {
			continue
		}
		for _, p := range it.ink {
			for row := 0; row < 3; row++ {
				src := ((p.y*3+row)*it.w*3 + p.x*3) * 4
				dst := (((it.y+p.y)*3+row)*960 + (it.x+p.x)*3) * 4
				copy(out[dst:dst+12], it.pixels[src:src+12])
			}
		}
	}
}
