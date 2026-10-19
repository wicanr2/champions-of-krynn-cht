package cok

// 主專案規格 010 §6.2：完整矩形辨識頭身組合，只改輸出。
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

type characterComponent struct {
	key    string
	w, h   int
	ref    []byte
	pixels []byte
}

type characterLayers struct {
	head, body *characterComponent
	flip       bool
}

type characterItem struct {
	Key      string `json:"key"`
	Size     [2]int `json:"size"`
	PNG      string `json:"png"`
	PNGSHA   string `json:"png_sha256"`
	Indexed  string `json:"indexed"`
	IndexSHA string `json:"indexed_sha256"`
}

func loadCharacterHD(dir string) (*actorHD, error) {
	data, err := os.ReadFile(filepath.Join(dir, "characters.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Version    int             `json:"version"`
		Scale      int             `json:"scale"`
		Kind       string          `json:"kind"`
		Grid       [5]int          `json:"grid"`
		Background int             `json:"background_index"`
		Heads      []characterItem `json:"heads"`
		Bodies     []characterItem `json:"bodies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version != 1 || manifest.Scale != 3 || manifest.Kind != "combat-characters" ||
		manifest.Grid != [5]int{8, 8, 7, 7, 24} || manifest.Background != 8 ||
		len(manifest.Heads) < 1 || len(manifest.Heads) > 25 || len(manifest.Bodies) < 1 || len(manifest.Bodies) > 128 {
		return nil, fmt.Errorf("characters.json：版本、倍率、種類、格線、背景或件數不符規格 010 §6.2")
	}
	keys := map[string]bool{}
	load := func(items []characterItem, head bool) ([]*characterComponent, error) {
		var result []*characterComponent
		signatures := map[[32]byte]bool{}
		for _, item := range items {
			w, h := item.Size[0], item.Size[1]
			if item.Key == "" || keys[item.Key] || w != 24 || (head && h != 8 && h != 9 && h != 10) || (!head && h != 24) {
				return nil, fmt.Errorf("characters.json：%q key 或尺寸不合法", item.Key)
			}
			keys[item.Key] = true
			ref, err := readChecked(filepath.Join(dir, item.Indexed), item.IndexSHA)
			if err != nil || len(ref) != w*h {
				return nil, fmt.Errorf("%s：色號長度或 SHA 不符：%v", item.Key, err)
			}
			foreground := 0
			for _, c := range ref {
				if c > 15 {
					return nil, fmt.Errorf("%s：色號超過 15", item.Key)
				}
				if c != 8 {
					foreground++
				}
			}
			if (head && (foreground < 21 || foreground > 37)) || (!head && (foreground < 101 || foreground > 190)) {
				return nil, fmt.Errorf("%s：前景數不符來源範圍", item.Key)
			}
			sum := sha256.Sum256(ref)
			if signatures[sum] {
				return nil, fmt.Errorf("%s：組件來源重複", item.Key)
			}
			signatures[sum] = true
			encoded, err := readChecked(filepath.Join(dir, item.PNG), item.PNGSHA)
			if err != nil {
				return nil, err
			}
			if len(encoded) < 26 || encoded[24] != 8 || encoded[25] != 6 {
				return nil, fmt.Errorf("%s：需要八位元 RGBA PNG", item.Key)
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
			result = append(result, &characterComponent{key: item.Key, w: w, h: h, ref: ref, pixels: rgba.Pix})
		}
		return result, nil
	}
	heads, err := load(manifest.Heads, true)
	if err != nil {
		return nil, err
	}
	bodies, err := load(manifest.Bodies, false)
	if err != nil {
		return nil, err
	}
	result := &actorHD{sources: len(heads) + len(bodies), full: map[[32]byte]int{}}
	for _, body := range bodies {
		for _, head := range heads {
			combined := append([]byte(nil), body.ref...)
			for n, c := range head.ref {
				if c != 8 {
					combined[n] = c
				}
			}
			for _, flip := range []bool{false, true} {
				pattern := actorPattern{key: body.key + "/" + head.key, w: 24, h: 24,
					full: make([]byte, 24*24), parts: &characterLayers{head: head, body: body, flip: flip}}
				var colors [16]bool
				count := 0
				for y := 0; y < 24; y++ {
					for x := 0; x < 24; x++ {
						sx := x
						if flip {
							sx = 23 - x
						}
						c := combined[y*24+sx]
						pattern.full[y*24+x] = c
						if !colors[c] {
							colors[c] = true
							count++
						}
						if c != 8 {
							pattern.ink = append(pattern.ink, actorInk{x, y, y*320 + x, c})
						}
					}
				}
				if count < 3 || len(pattern.ink) < 122 || len(pattern.ink) > 226 {
					return nil, fmt.Errorf("%s：組合前景資訊不符來源", pattern.key)
				}
				sum := sha256.Sum256(pattern.full)
				if _, duplicate := result.full[sum]; duplicate {
					return nil, fmt.Errorf("%s：組合或鏡像來源重複", pattern.key)
				}
				result.full[sum] = len(result.patterns)
				result.patterns = append(result.patterns, pattern)
			}
		}
	}
	return result, nil
}

func mergeCharacterHD(actors, characters *actorHD) *actorHD {
	if actors == nil {
		return characters
	}
	if characters == nil {
		return actors
	}
	actors.full = make(map[[32]byte]int, len(characters.full))
	for sum, n := range characters.full {
		actors.full[sum] = len(actors.patterns) + n
	}
	actors.patterns = append(actors.patterns, characters.patterns...)
	actors.sources += characters.sources
	return actors
}

func (p *actorPattern) drawCharacterPoint(out []byte, point actorInk, x, y int) {
	parts := p.parts
	sx := point.x
	if parts.flip {
		sx = 23 - sx
	}
	component := parts.body
	if point.y < parts.head.h && parts.head.ref[point.y*24+sx] != 8 {
		component = parts.head
	}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			pixelX := point.x*3 + col
			if parts.flip {
				pixelX = 71 - pixelX
			}
			src := ((point.y*3+row)*72 + pixelX) * 4
			dst := (((y+point.y)*3+row)*960 + (x+point.x)*3 + col) * 4
			copy(out[dst:dst+4], component.pixels[src:src+4])
		}
	}
}
