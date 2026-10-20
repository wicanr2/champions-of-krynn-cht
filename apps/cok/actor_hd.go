package cok

// CPIC 前景呈現：主專案規格 010 §6.2。來源背景區不畫，不推斷原版透明 driver。
import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type actorInk struct {
	x, y, offset int
	color        byte
}

type actorPattern struct {
	key            string
	w, h           int
	ink            []actorInk
	anchors        []actorInk
	pixels         []byte
	full           []byte // 頭身組合另核對完整24×24；CPIC維持逐前景契約。
	parts          *characterLayers
	freeFull       bool // COMSPR 完整模板可位於格子間；規格010 §8.4。
	fullAnchor     []byte
	fullAX, fullAY int
}

type actorMatch struct {
	pattern, x, y int
	conflict      bool
}

type actorHD struct {
	sources  int
	patterns []actorPattern
	matches  []actorMatch
	owner    []int
	full     map[[32]byte]int
}

func actorShape(w, h int) bool {
	switch [2]int{w, h} {
	case [2]int{24, 24}, [2]int{24, 48}, [2]int{48, 24}, [2]int{48, 48}:
		return true
	}
	return false
}

func loadActorHD(dir string) (*actorHD, error) {
	data, err := os.ReadFile(filepath.Join(dir, "actors.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Version    int    `json:"version"`
		Scale      int    `json:"scale"`
		Kind       string `json:"kind"`
		Grid       [5]int `json:"grid"`
		Background int    `json:"background_index"`
		Items      []struct {
			Key      string `json:"key"`
			Size     [2]int `json:"size"`
			PNG      string `json:"png"`
			PNGSHA   string `json:"png_sha256"`
			Indexed  string `json:"indexed"`
			IndexSHA string `json:"indexed_sha256"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version != 1 || manifest.Scale != 3 || manifest.Kind != "combat-actors" ||
		manifest.Grid != [5]int{8, 8, 7, 7, 24} || manifest.Background != 8 ||
		len(manifest.Items) == 0 || len(manifest.Items) > 256 {
		return nil, fmt.Errorf("actors.json：版本、倍率、種類、格線、背景或件數不符規格 010 §6.2")
	}
	result := &actorHD{sources: len(manifest.Items)}
	keys, signatures := map[string]bool{}, map[string]bool{}
	for _, item := range manifest.Items {
		w, h := item.Size[0], item.Size[1]
		if item.Key == "" || keys[item.Key] || !actorShape(w, h) {
			return nil, fmt.Errorf("actors.json：%q key 或尺寸不合法", item.Key)
		}
		keys[item.Key] = true
		ref, err := readChecked(filepath.Join(dir, item.Indexed), item.IndexSHA)
		if err != nil || len(ref) != w*h {
			return nil, fmt.Errorf("%s：色號長度或 SHA 不符：%v", item.Key, err)
		}
		var colors [16]bool
		count, foreground := 0, 0
		for _, c := range ref {
			if c > 15 {
				return nil, fmt.Errorf("%s：色號 %d 超過 15", item.Key, c)
			}
			if !colors[c] {
				colors[c] = true
				count++
			}
			if c != 8 {
				foreground++
			}
		}
		if count < 3 || foreground < 144 {
			return nil, fmt.Errorf("%s：前景資訊不足", item.Key)
		}
		encoded, err := readChecked(filepath.Join(dir, item.PNG), item.PNGSHA)
		if err != nil {
			return nil, err
		}
		decoded, err := png.Decode(bytes.NewReader(encoded))
		if err != nil || decoded.Bounds().Dx() != w*3 || decoded.Bounds().Dy() != h*3 {
			return nil, fmt.Errorf("%s：PNG 或三倍尺寸不符：%v", item.Key, err)
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
		draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
		for n := 3; n < len(rgba.Pix); n += 4 {
			if rgba.Pix[n] != 255 {
				return nil, fmt.Errorf("%s：PNG 含透明點", item.Key)
			}
		}
		for _, flip := range []bool{false, true} {
			pattern := actorPattern{key: item.Key, w: w, h: h, pixels: make([]byte, len(rgba.Pix))}
			transformed := make([]byte, len(ref))
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					sx := x
					if flip {
						sx = w - 1 - x
					}
					c := ref[y*w+sx]
					transformed[y*w+x] = c
					if c != 8 {
						pattern.ink = append(pattern.ink, actorInk{x, y, y*320 + x, c})
					}
				}
			}
			for y := 0; y < h*3; y++ {
				for x := 0; x < w*3; x++ {
					sx := x
					if flip {
						sx = w*3 - 1 - x
					}
					copy(pattern.pixels[(y*w*3+x)*4:], rgba.Pix[(y*w*3+sx)*4:(y*w*3+sx)*4+4])
				}
			}
			sum := sha256.Sum256(transformed)
			signature := fmt.Sprintf("%dx%d:%x", w, h, sum)
			if signatures[signature] {
				return nil, fmt.Errorf("%s：來源或鏡像基準重複", item.Key)
			}
			signatures[signature] = true
			for n := 0; n < 8; n++ {
				pattern.anchors = append(pattern.anchors, pattern.ink[n*(len(pattern.ink)-1)/7])
			}
			result.patterns = append(result.patterns, pattern)
		}
	}
	return result, nil
}

func (p *actorPattern) present(indexed []byte, x, y int) bool {
	start := y*320 + x
	if len(p.full) != 0 {
		for row := 0; row < 24; row++ {
			if !bytes.Equal(indexed[start+row*320:start+row*320+24], p.full[row*24:(row+1)*24]) {
				return false
			}
		}
		return true
	}
	for _, point := range p.anchors {
		if indexed[start+point.offset] != point.color {
			return false
		}
	}
	for _, point := range p.ink {
		if indexed[start+point.offset] != point.color {
			return false
		}
	}
	return true
}

func (a *actorHD) frame(indexed []byte, terrainGuard bool) {
	if a == nil {
		return
	}
	a.matches = a.matches[:0]
	if !terrainGuard || len(indexed) != 320*200 {
		return
	}
	for n := range a.patterns {
		pattern := &a.patterns[n]
		if len(pattern.full) != 0 {
			continue
		}
		for y := 8; y+pattern.h <= 176; y += 24 {
			for x := 8; x+pattern.w <= 176; x += 24 {
				if pattern.present(indexed, x, y) {
					a.matches = append(a.matches, actorMatch{pattern: n, x: x, y: y})
				}
			}
		}
	}
	if len(a.full) != 0 {
		var tile [24 * 24]byte
		for y := 8; y+24 <= 176; y += 24 {
			for x := 8; x+24 <= 176; x += 24 {
				for row := 0; row < 24; row++ {
					copy(tile[row*24:(row+1)*24], indexed[(y+row)*320+x:(y+row)*320+x+24])
				}
				if n, found := a.full[sha256.Sum256(tile[:])]; found && !a.patterns[n].freeFull {
					a.matches = append(a.matches, actorMatch{pattern: n, x: x, y: y})
				}
			}
		}
	}
	for n := range a.patterns {
		p := &a.patterns[n]
		if !p.freeFull {
			continue
		}
		for from := 0; from+len(p.fullAnchor) <= len(indexed); {
			off := bytes.Index(indexed[from:], p.fullAnchor)
			if off < 0 {
				break
			}
			at := from + off
			from = at + 1
			x, y := at%320-p.fullAX, at/320-p.fullAY
			if x >= 8 && y >= 8 && x+p.w <= 176 && y+p.h <= 176 && p.present(indexed, x, y) {
				a.matches = append(a.matches, actorMatch{pattern: n, x: x, y: y})
			}
		}
	}
	// 同一個前景點若有多個候選，全部回退；不猜測覆蓋順序。
	if len(a.matches) == 0 {
		return
	}
	if len(a.owner) != 320*200 {
		a.owner = make([]int, 320*200)
	} else {
		clear(a.owner)
	}
	for n := range a.matches {
		match := &a.matches[n]
		for _, point := range a.patterns[match.pattern].ink {
			offset := (match.y+point.y)*320 + match.x + point.x
			if previous := a.owner[offset]; previous != 0 {
				match.conflict = true
				a.matches[previous-1].conflict = true
			} else {
				a.owner[offset] = n + 1
			}
		}
	}
}

func (a *actorHD) draw(out, indexed []byte, scale int) {
	if a == nil || scale != 3 || len(out) != 960*600*4 || len(indexed) != 320*200 {
		return
	}
	for _, match := range a.matches {
		pattern := &a.patterns[match.pattern]
		if match.conflict || !pattern.present(indexed, match.x, match.y) {
			continue
		}
		for _, point := range pattern.ink {
			if pattern.parts != nil {
				pattern.drawCharacterPoint(out, point, match.x, match.y)
				continue
			}
			for row := 0; row < 3; row++ {
				src := ((point.y*3+row)*pattern.w*3 + point.x*3) * 4
				dst := (((match.y+point.y)*3+row)*960 + (match.x+point.x)*3) * 4
				copy(out[dst:dst+12], pattern.pixels[src:src+12])
			}
		}
	}
}
