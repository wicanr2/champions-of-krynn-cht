package cok

// 規格010 §8：未知位置的來源只以當幀完整色號矩形辨識。
import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type exactHDItem struct {
	key         string
	w, h        int
	ref, pixels []byte
	ax, ay      int
	anchor      []byte
}
type exactHDMatch struct {
	item, x, y int
	conflict   bool
}
type exactHD struct {
	items   []exactHDItem
	matches []exactHDMatch
}

func loadExactHD(dir string) (*exactHD, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "exact.json"))
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
			Algorithm string `json:"algorithm,omitempty"`
		} `json:"items"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || m.Scale != 3 || m.Kind != "complete-source-patterns" || len(m.Items) < 1 || len(m.Items) > 512 {
		return nil, fmt.Errorf("exact.json：不符規格010 §8")
	}
	h := &exactHD{}
	keys, refs := map[string]bool{}, map[string]bool{}
	checked := func(name, hash string) ([]byte, error) {
		full, ok := themeFile(dir, name)
		if !ok {
			return nil, fmt.Errorf("完整來源資產路徑不合法")
		}
		return readChecked(full, hash)
	}
	for _, it := range m.Items {
		if it.Algorithm != "" && (it.Algorithm != "scale3x" || it.Size != [2]int{8, 8}) {
			return nil, fmt.Errorf("完整來源演算法不合法")
		}
		w, ht := it.Size[0], it.Size[1]
		if it.Key == "" || keys[it.Key] || w < 8 || ht < 8 || w > 320 || ht > 200 {
			return nil, fmt.Errorf("%s：完整來源 key 或尺寸不合法", it.Key)
		}
		keys[it.Key] = true
		ref, err := checked(it.Indexed, it.IndexSHA)
		if err != nil || len(ref) != w*ht {
			return nil, fmt.Errorf("%s：來源長度或 SHA 不符：%v", it.Key, err)
		}
		var counts [16]int
		maximum := 0
		for _, c := range ref {
			if c > 15 {
				return nil, fmt.Errorf("%s：非 EGA 色號", it.Key)
			}
			counts[c]++
			if counts[c] > maximum {
				maximum = counts[c]
			}
		}
		if len(ref)-maximum < 16 {
			return nil, fmt.Errorf("%s：來源資訊不足", it.Key)
		}
		sig := fmt.Sprintf("%dx%d:%x", w, ht, sha256.Sum256(ref))
		if refs[sig] {
			return nil, fmt.Errorf("%s：重複完整來源", it.Key)
		}
		refs[sig] = true
		encoded, err := checked(it.PNG, it.PNGSHA)
		if err != nil {
			return nil, err
		}
		if len(encoded) < 26 || encoded[24] != 8 || encoded[25] != 6 {
			return nil, fmt.Errorf("%s：需要八位元 RGBA PNG", it.Key)
		}
		im, err := png.Decode(bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		if im.Bounds().Size() != (image.Point{X: w * 3, Y: ht * 3}) {
			return nil, fmt.Errorf("%s：三倍 PNG 尺寸不符", it.Key)
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, w*3, ht*3))
		draw.Draw(rgba, rgba.Bounds(), im, im.Bounds().Min, draw.Src)
		for n := 3; n < len(rgba.Pix); n += 4 {
			if rgba.Pix[n] != 255 {
				return nil, fmt.Errorf("%s：PNG 含透明點", it.Key)
			}
		}
		if it.Algorithm == "scale3x" && !hdScale3xPixelsEqual(ref, rgba.Pix, w, ht) {
			return nil, fmt.Errorf("Scale3x完整來源結果不符")
		}
		p := exactHDItem{key: it.Key, w: w, h: ht, ref: ref, pixels: rgba.Pix}
		p.makeAnchor()
		h.items = append(h.items, p)
	}
	return h, nil
}

// themeFile 把主題清單裡的相對路徑（一律以 / 分隔）轉成 dir 底下的檔案路徑。空字串、絕對路徑、反斜線、
// 未正規化或跳出主題目錄的路徑回 false。用 path 而非 filepath 檢查：Windows 上 filepath.Clean 會把 / 換成 \，
// 合法路徑也會被判為未正規化（v.1.0.3 以前 Windows HD 版牆面與完整來源因此無法載入）。
func themeFile(dir, name string) (string, bool) {
	if name == "" || path.IsAbs(name) || strings.ContainsAny(name, `\:`) || path.Clean(name) != name ||
		name == ".." || strings.HasPrefix(name, "../") {
		return "", false
	}
	return filepath.Join(dir, filepath.FromSlash(name)), true
}

func (p *exactHDItem) makeAnchor() {
	// 最多八點、同一橫列，偏好色彩轉折多的片段。錨點不取代完整比對。
	best := -1
	for y := 0; y < p.h; y++ {
		for x := 0; x+8 <= p.w; x++ {
			a := p.ref[y*p.w+x : y*p.w+x+8]
			seen := [16]bool{}
			score := 0
			for n, c := range a {
				if !seen[c] {
					score += 8
					seen[c] = true
				}
				if n > 0 && c != a[n-1] {
					score++
				}
			}
			if score > best {
				best = score
				p.ax = x
				p.ay = y
				p.anchor = a
			}
		}
	}
}
func (p *exactHDItem) present(idx []byte, x, y int) bool {
	if len(idx) != 64000 || x < 0 || y < 0 || x+p.w > 320 || y+p.h > 200 {
		return false
	}
	for row := 0; row < p.h; row++ {
		if !bytes.Equal(idx[(y+row)*320+x:(y+row)*320+x+p.w], p.ref[row*p.w:(row+1)*p.w]) {
			return false
		}
	}
	return true
}
func (h *exactHD) frame(idx []byte, pal [256][3]uint8) {
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
	for n := range h.items {
		p := &h.items[n]
		for from := 0; from+len(p.anchor) <= len(idx); {
			off := bytes.Index(idx[from:], p.anchor)
			if off < 0 {
				break
			}
			at := from + off
			from = at + 1
			x, y := at%320-p.ax, at/320-p.ay
			if p.present(idx, x, y) {
				h.matches = append(h.matches, exactHDMatch{item: n, x: x, y: y})
			}
		}
	}
	for i := range h.matches {
		for j := 0; j < i; j++ {
			a, b := &h.matches[i], &h.matches[j]
			pa, pb := h.items[a.item], h.items[b.item]
			if a.x < b.x+pb.w && b.x < a.x+pa.w && a.y < b.y+pb.h && b.y < a.y+pa.h {
				a.conflict = true
				b.conflict = true
			}
		}
	}
}
func (h *exactHD) draw(out, idx []byte, scale int) {
	if h == nil || scale != 3 || len(out) != 960*600*4 {
		return
	}
	for _, m := range h.matches {
		p := &h.items[m.item]
		if m.conflict || !p.present(idx, m.x, m.y) {
			continue
		}
		for row := 0; row < p.h*3; row++ {
			dst := ((m.y*3+row)*960 + m.x*3) * 4
			src := row * p.w * 3 * 4
			copy(out[dst:dst+p.w*12], p.pixels[src:src+p.w*12])
		}
	}
}
