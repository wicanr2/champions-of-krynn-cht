package cok

// 戰鬥地形：主專案規格 010 §6.2。只比對已知地形的完整色號，不讀遊戲狀態。
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

var combatEGA = [16][3]uint8{
	{0, 0, 0}, {0, 0, 170}, {0, 170, 0}, {0, 170, 170},
	{170, 0, 0}, {170, 0, 170}, {170, 85, 0}, {170, 170, 170},
	{85, 85, 85}, {85, 85, 255}, {85, 255, 85}, {85, 255, 255},
	{255, 85, 85}, {255, 85, 255}, {255, 255, 85}, {255, 255, 255},
}

type combatTile struct {
	key       string
	reference []byte
	pixels    []byte
	guard     bool
}

type combatMatch struct{ tile, x, y int }

type combatHD struct {
	tiles   []combatTile
	lookup  map[[32]byte]int
	matches []combatMatch
}

// loadCombatHD 先完整驗證，呼叫端成功後才登記。缺少可選檔案時維持舊主題行為。
func loadCombatHD(dir string) (*combatHD, error) {
	b, err := os.ReadFile(filepath.Join(dir, "combat.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Version int    `json:"version"`
		Scale   int    `json:"scale"`
		Kind    string `json:"kind"`
		Grid    [5]int `json:"grid"`
		Items   []struct {
			Key      string `json:"key"`
			Size     [2]int `json:"size"`
			PNG      string `json:"png"`
			PNGSHA   string `json:"png_sha256"`
			Indexed  string `json:"indexed"`
			IndexSHA string `json:"indexed_sha256"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, fmt.Errorf("combat.json：%w", err)
	}
	if manifest.Version != 1 || manifest.Scale != 3 || manifest.Kind != "combat-terrain" ||
		manifest.Grid != [5]int{8, 8, 7, 7, 24} || len(manifest.Items) == 0 || len(manifest.Items) > 256 {
		return nil, fmt.Errorf("combat.json：版本、倍率、類型、格線或件數不符規格 010 §6.2")
	}
	theme := &combatHD{lookup: make(map[[32]byte]int)}
	keys := make(map[string]bool)
	for _, item := range manifest.Items {
		if item.Key == "" || keys[item.Key] || item.Size != [2]int{24, 24} {
			return nil, fmt.Errorf("combat.json：%q key 或尺寸不合法", item.Key)
		}
		keys[item.Key] = true
		ref, err := readChecked(filepath.Join(dir, item.Indexed), item.IndexSHA)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", item.Key, err)
		}
		if len(ref) != 24*24 {
			return nil, fmt.Errorf("%s：地形基準長度不符", item.Key)
		}
		var colors [16]bool
		colorCount, nonGray := 0, 0
		for _, c := range ref {
			if c > 15 {
				return nil, fmt.Errorf("%s：色號 %d 超過 15", item.Key, c)
			}
			if !colors[c] {
				colorCount++
				colors[c] = true
			}
			if c != 8 {
				nonGray++
			}
		}
		sum := sha256.Sum256(ref)
		if _, found := theme.lookup[sum]; found {
			return nil, fmt.Errorf("%s：地形基準重複", item.Key)
		}
		raw, err := readChecked(filepath.Join(dir, item.PNG), item.PNGSHA)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", item.Key, err)
		}
		decoded, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%s：%w", item.Key, err)
		}
		if decoded.Bounds().Dx() != 72 || decoded.Bounds().Dy() != 72 {
			return nil, fmt.Errorf("%s：HD 地形不是 72×72", item.Key)
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, 72, 72))
		draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
		for n := 3; n < len(rgba.Pix); n += 4 {
			if rgba.Pix[n] != 255 {
				return nil, fmt.Errorf("%s：不透明地形含 alpha", item.Key)
			}
		}
		theme.lookup[sum] = len(theme.tiles)
		theme.tiles = append(theme.tiles, combatTile{item.Key, ref, rgba.Pix, colorCount > 1 && nonGray >= 16})
	}
	return theme, nil
}

func (c *combatHD) frame(indexed []byte, palette [256][3]uint8) {
	if c == nil {
		return
	}
	c.matches = c.matches[:0]
	if len(indexed) != 320*200 {
		return
	}
	for n, color := range combatEGA {
		if palette[n] != color {
			return // 未證實色盤及閃光變色保留原版。
		}
	}
	guards := 0
	var tile [24 * 24]byte
	for y := 8; y <= 152; y += 24 {
		for x := 8; x <= 152; x += 24 {
			for row := 0; row < 24; row++ {
				copy(tile[row*24:], indexed[(y+row)*320+x:(y+row)*320+x+24])
			}
			if n, found := c.lookup[sha256.Sum256(tile[:])]; found {
				c.matches = append(c.matches, combatMatch{n, x, y})
				if c.tiles[n].guard {
					guards++
				}
			}
		}
	}
	if guards < 3 {
		c.matches = c.matches[:0]
	}
}

func (c *combatHD) draw(out, indexed []byte, scale int) {
	if c == nil || scale != 3 || len(out) != 960*600*4 || len(indexed) != 320*200 {
		return
	}
	for _, match := range c.matches {
		tile := &c.tiles[match.tile]
		valid := true
		for row := 0; row < 24 && valid; row++ {
			valid = bytes.Equal(indexed[(match.y+row)*320+match.x:(match.y+row)*320+match.x+24], tile.reference[row*24:(row+1)*24])
		}
		if !valid {
			continue
		}
		for row := 0; row < 72; row++ {
			dst := ((match.y*3+row)*960 + match.x*3) * 4
			copy(out[dst:dst+72*4], tile.pixels[row*72*4:(row+1)*72*4])
		}
	}
}
