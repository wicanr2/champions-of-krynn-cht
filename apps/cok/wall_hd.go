package cok

// 規格010 §7：來源推得的完整8×8組合格。全視窗逐點吻合後才畫。
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type wallAssetSpec struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"`
	Size     [2]int `json:"size"`
	Indexed  string `json:"indexed"`
	IndexSHA string `json:"indexed_sha256"`
	PNG      string `json:"png"`
	PNGSHA   string `json:"png_sha256"`
}
type wallLayerSpec struct {
	Key string `json:"key"`
	At  [2]int `json:"at"`
}
type wallCellSpec struct {
	Key                string          `json:"key"`
	Positions          []int           `json:"positions"`
	Base               int             `json:"base_color"`
	Layers             []wallLayerSpec `json:"layers"`
	IndexSHA           string          `json:"indexed_sha256"`
	Paint              bool            `json:"paint"`
	CompleteBackground bool            `json:"complete_background,omitempty"`
}
type wallManifest struct {
	Version  int             `json:"version"`
	Scale    int             `json:"scale"`
	Kind     string          `json:"kind"`
	Viewport [5]int          `json:"viewport"`
	Assets   []wallAssetSpec `json:"assets"`
	Cells    []wallCellSpec  `json:"cells"`
}
type wallAsset struct {
	w, h     int
	wall     bool
	ref, pix []byte
}
type wallCell struct {
	ref   [64]byte
	pix   []byte
	guard bool
}
type wallHD struct {
	sources int
	global  map[[64]byte]*wallCell
	cells   [121]map[[64]byte]*wallCell
	matches [121]*wallCell
	panel   [88 * 88]byte
	active  bool
}

func loadWallHD(dir string) (*wallHD, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "walls.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m wallManifest
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || m.Scale != 3 || m.Kind != "wall-grid" || m.Viewport != [5]int{24, 24, 11, 11, 8} || len(m.Assets) < 1 || len(m.Assets) > 2048 || len(m.Cells) < 1 || len(m.Cells) > 32768 {
		return nil, fmt.Errorf("walls.json不符規格010 §7")
	}
	assets := map[string]wallAsset{}
	checked := func(name, hash string) ([]byte, error) {
		if name == "" || filepath.IsAbs(name) || filepath.Clean(name) != name || name == ".." || len(name) >= 3 && name[:3] == "../" {
			return nil, fmt.Errorf("牆面資產路徑不合法")
		}
		return readChecked(filepath.Join(dir, name), hash)
	}
	for _, s := range m.Assets {
		w, h := s.Size[0], s.Size[1]
		if _, ok := assets[s.Key]; ok || s.Key == "" {
			return nil, fmt.Errorf("牆面key重複或空白")
		}
		if s.Kind != "wall" && s.Kind != "sky" || s.Kind == "wall" && (w != 8 || h != 8) || s.Kind == "sky" && s.Size != [2]int{88, 16} && s.Size != [2]int{24, 24} && s.Size != [2]int{88, 48} {
			return nil, fmt.Errorf("牆面來源種類或尺寸不合法")
		}
		ref, err := checked(s.Indexed, s.IndexSHA)
		if err != nil {
			return nil, err
		}
		if len(ref) != w*h {
			return nil, fmt.Errorf("牆面來源長度不符")
		}
		for _, v := range ref {
			if v > 15 {
				return nil, fmt.Errorf("牆面非EGA色號")
			}
		}
		encoded, err := checked(s.PNG, s.PNGSHA)
		if err != nil {
			return nil, err
		}
		im, err := png.Decode(bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		if im.Bounds().Size() != (image.Point{X: w * 3, Y: h * 3}) {
			return nil, fmt.Errorf("牆面三倍PNG尺寸不符")
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, w*3, h*3))
		draw.Draw(rgba, rgba.Bounds(), im, im.Bounds().Min, draw.Src)
		for y := 0; y < h*3; y++ {
			for x := 0; x < w*3; x++ {
				n := (y*w*3 + x) * 4
				v := ref[y/3*w+x/3]
				if rgba.Pix[n+3] != 255 {
					return nil, fmt.Errorf("牆面PNG非不透明")
				}
				if v == 13 && !bytes.Equal(rgba.Pix[n:n+3], []byte{85, 85, 85}) {
					return nil, fmt.Errorf("牆面留底遮罩被改動")
				}
				if s.Kind == "wall" && (x == 0 || y == 0 || x == w*3-1 || y == h*3-1) {
					c := v
					if c == 13 {
						c = 8
					}
					if !bytes.Equal(rgba.Pix[n:n+3], combatEGA[c][:]) {
						return nil, fmt.Errorf("牆面邊界接縫不符")
					}
				}
			}
		}
		assets[s.Key] = wallAsset{w, h, s.Kind == "wall", ref, rgba.Pix}
	}
	h := &wallHD{sources: len(assets), global: map[[64]byte]*wallCell{}}
	for i := range h.cells {
		h.cells[i] = map[[64]byte]*wallCell{}
	}
	keys := map[string]bool{}
	for _, s := range m.Cells {
		if s.Key == "" || keys[s.Key] || s.Base < 0 || s.Base > 15 || len(s.Layers) > 6 || len(s.Positions) > 121 {
			return nil, fmt.Errorf("牆面組合格契約不符")
		}
		keys[s.Key] = true
		cell := &wallCell{pix: make([]byte, 24*24*4)}
		var owners [64]bool
		for n := range cell.ref {
			cell.ref[n] = byte(s.Base)
		}
		for n := 0; n < len(cell.pix); n += 4 {
			copy(cell.pix[n:n+3], combatEGA[s.Base][:])
			cell.pix[n+3] = 255
		}
		for _, layer := range s.Layers {
			a, ok := assets[layer.Key]
			if !ok {
				return nil, fmt.Errorf("牆面組合來源不存在")
			}
			dx, dy := layer.At[0], layer.At[1]
			if dx <= -a.w || dy <= -a.h || dx >= 8 || dy >= 8 {
				return nil, fmt.Errorf("牆面組合來源不與格子相交")
			}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					sx, sy := x-dx, y-dy
					if sx < 0 || sy < 0 || sx >= a.w || sy >= a.h {
						continue
					}
					v := a.ref[sy*a.w+sx]
					if v == 13 {
						continue
					}
					cell.ref[y*8+x] = v
					owners[y*8+x] = a.wall
					for row := 0; row < 3; row++ {
						src := ((sy*3+row)*a.w*3 + sx*3) * 4
						dst := ((y*3+row)*24 + x*3) * 4
						copy(cell.pix[dst:dst+12], a.pix[src:src+12])
					}
				}
			}
		}
		digest := sha256.Sum256(cell.ref[:])
		if hex.EncodeToString(digest[:]) != s.IndexSHA {
			return nil, fmt.Errorf("牆面組合基準SHA不符")
		}
		ink, colors := 0, map[byte]bool{}
		for n, wall := range owners {
			if wall {
				ink++
				colors[cell.ref[n]] = true
			}
		}
		cell.guard = ink >= 16 && len(colors) >= 2
		if !s.Paint {
			for y := 0; y < 24; y++ {
				for x := 0; x < 24; x++ {
					n := (y*24 + x) * 4
					copy(cell.pix[n:n+3], combatEGA[cell.ref[y/3*8+x/3]][:])
				}
			}
		}
		if len(s.Positions) == 0 {
			// 通用位置只允許一個完整不留底8×8牆面來源。
			if len(s.Layers) != 1 || s.Layers[0].At != [2]int{} {
				return nil, fmt.Errorf("牆面通用格須為單一完整來源")
			}
			a := assets[s.Layers[0].Key]
			masked := bytes.IndexByte(a.ref, 13) >= 0
			if !a.wall || masked != s.CompleteBackground {
				return nil, fmt.Errorf("牆面通用格留底須有完整背景契約")
			}
			if s.CompleteBackground {
				nonempty := false
				for _, v := range a.ref {
					if v != 13 {
						nonempty = true
						break
					}
				}
				if !nonempty {
					return nil, fmt.Errorf("牆面通用格沒有來源像素")
				}
			}
			if h.global[cell.ref] != nil {
				return nil, fmt.Errorf("牆面重複通用基準")
			}
			h.global[cell.ref] = cell
		} else {
			for _, pos := range s.Positions {
				if pos < 0 || pos >= 121 || h.cells[pos][cell.ref] != nil {
					return nil, fmt.Errorf("牆面位置越界或重複基準")
				}
				h.cells[pos][cell.ref] = cell
			}
		}
	}
	for _, lookup := range h.cells {
		for ref := range lookup {
			if h.global[ref] != nil {
				return nil, fmt.Errorf("牆面局部與通用基準歧義")
			}
		}
	}
	return h, nil
}

func (h *wallHD) frame(idx []byte, pal [256][3]uint8) {
	if h == nil {
		return
	}
	h.active = false
	if len(idx) != 64000 {
		return
	}
	for n, c := range combatEGA {
		if pal[n] != c {
			return
		}
	}
	guards := 0
	for pos := 0; pos < 121; pos++ {
		x, y := 24+pos%11*8, 24+pos/11*8
		var ref [64]byte
		for row := 0; row < 8; row++ {
			copy(ref[row*8:row*8+8], idx[(y+row)*320+x:(y+row)*320+x+8])
		}
		cell := h.global[ref]
		if cell == nil {
			cell = h.cells[pos][ref]
		}
		if cell == nil {
			return
		}
		h.matches[pos] = cell
		if cell.guard {
			guards++
		}
	}
	if guards < 3 {
		return
	}
	for row := 0; row < 88; row++ {
		copy(h.panel[row*88:(row+1)*88], idx[(24+row)*320+24:(24+row)*320+112])
	}
	h.active = true
}
func (h *wallHD) draw(out, idx []byte, scale int) {
	if h == nil || !h.active || scale != 3 || len(out) != 960*600*4 || len(idx) != 64000 {
		return
	}
	for row := 0; row < 88; row++ {
		if !bytes.Equal(h.panel[row*88:(row+1)*88], idx[(24+row)*320+24:(24+row)*320+112]) {
			return
		}
	}
	for pos, cell := range h.matches {
		x, y := (24+pos%11*8)*3, (24+pos/11*8)*3
		for row := 0; row < 24; row++ {
			dst := ((y+row)*960 + x) * 4
			copy(out[dst:dst+96], cell.pix[row*96:(row+1)*96])
		}
	}
}
