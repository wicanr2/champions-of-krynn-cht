package cok

// 主專案規格 010 §6.2：COMSPR 只辨識完整矩形。
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

func loadEffectHD(dir string) (*actorHD, error) {
	data, err := os.ReadFile(filepath.Join(dir, "effects.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m struct {
		Version    int             `json:"version"`
		Scale      int             `json:"scale"`
		Kind       string          `json:"kind"`
		Grid       [5]int          `json:"grid"`
		Background int             `json:"background_index"`
		Items      []characterItem `json:"items"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || m.Scale != 3 || m.Kind != "combat-effects" || m.Grid != [5]int{8, 8, 7, 7, 24} || m.Background != 8 || len(m.Items) < 1 || len(m.Items) > 26 {
		return nil, fmt.Errorf("effects.json：欄位不符規格 010 §6.2")
	}
	keys, sources := map[string]bool{}, map[[32]byte]bool{}
	var components []*characterComponent
	for _, item := range m.Items {
		if item.Key == "" || keys[item.Key] || item.Size != [2]int{24, 24} {
			return nil, fmt.Errorf("%s：key 或尺寸錯誤", item.Key)
		}
		keys[item.Key] = true
		ref, err := readChecked(filepath.Join(dir, item.Indexed), item.IndexSHA)
		if err != nil || len(ref) != 576 {
			return nil, fmt.Errorf("%s：色號長度或 SHA 不符：%v", item.Key, err)
		}
		colors, ink := map[byte]bool{}, 0
		for _, c := range ref {
			if c > 15 {
				return nil, fmt.Errorf("%s：色號超過15", item.Key)
			}
			colors[c] = true
			if c != 8 {
				ink++
			}
		}
		if len(colors) < 3 || ink < 12 {
			return nil, fmt.Errorf("%s：來源資訊不足", item.Key)
		}
		sum := sha256.Sum256(ref)
		if sources[sum] {
			return nil, fmt.Errorf("%s：原方向來源重複", item.Key)
		}
		sources[sum] = true
		encoded, err := readChecked(filepath.Join(dir, item.PNG), item.PNGSHA)
		if err != nil {
			return nil, err
		}
		if len(encoded) < 26 || encoded[24] != 8 || encoded[25] != 6 {
			return nil, fmt.Errorf("%s：需要八位元RGBA PNG", item.Key)
		}
		decoded, err := png.Decode(bytes.NewReader(encoded))
		if err != nil || decoded.Bounds().Dx() != 72 || decoded.Bounds().Dy() != 72 {
			return nil, fmt.Errorf("%s：PNG尺寸不符", item.Key)
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, 72, 72))
		draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
		for n := 3; n < len(rgba.Pix); n += 4 {
			if rgba.Pix[n] != 255 {
				return nil, fmt.Errorf("%s：PNG含透明點", item.Key)
			}
		}
		components = append(components, &characterComponent{key: item.Key, w: 24, h: 24, ref: ref, pixels: rgba.Pix})
	}
	result := &actorHD{sources: len(components), full: map[[32]byte]int{}}
	// 所有原方向先登錄；鏡像遇到既有圖樣便沿用，避免同圖兩個候選。
	for _, flip := range []bool{false, true} {
		for _, component := range components {
			p := actorPattern{key: component.key, w: 24, h: 24, full: make([]byte, 576), pixels: make([]byte, 72*72*4)}
			for y := 0; y < 24; y++ {
				for x := 0; x < 24; x++ {
					sx := x
					if flip {
						sx = 23 - x
					}
					c := component.ref[y*24+sx]
					p.full[y*24+x] = c
					if c != 8 {
						p.ink = append(p.ink, actorInk{x, y, y*320 + x, c})
					}
				}
			}
			sum := sha256.Sum256(p.full)
			if _, exists := result.full[sum]; exists {
				continue
			}
			for y := 0; y < 72; y++ {
				for x := 0; x < 72; x++ {
					sx := x
					if flip {
						sx = 71 - x
					}
					copy(p.pixels[(y*72+x)*4:], component.pixels[(y*72+sx)*4:(y*72+sx+1)*4])
				}
			}
			result.full[sum] = len(result.patterns)
			result.patterns = append(result.patterns, p)
		}
	}
	return result, nil
}

func mergeEffectHD(actors, effects *actorHD) (*actorHD, error) {
	if actors == nil {
		return effects, nil
	}
	if effects == nil {
		return actors, nil
	}
	for sum := range effects.full {
		if _, exists := actors.full[sum]; exists {
			return nil, fmt.Errorf("effects.json：與其他完整來源重複")
		}
	}
	if actors.full == nil {
		actors.full = map[[32]byte]int{}
	}
	for sum, n := range effects.full {
		actors.full[sum] = len(actors.patterns) + n
	}
	actors.patterns = append(actors.patterns, effects.patterns...)
	actors.sources += effects.sources
	return actors, nil
}
