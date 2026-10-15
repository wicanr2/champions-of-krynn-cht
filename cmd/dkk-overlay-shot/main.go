// Command dkk-overlay-shot 是《Death Knights of Krynn》路徑 A 覆繪的 headless A/B 收據
// （death-knights-of-krynn-cht 規格 001 §3）。
//
// 從 probe 狀態檔接著跑，照指令數送 BIOS 鍵；每道指令前交給 apps/dkk 的攔截點，
// 每個畫格送進 xlate 定色。在 -shots 指定的步數各輸出原版與覆繪的 2× PNG，
// 最後印 JSON：每張圖的 SHA-256、差異是否落在 Stamp 矩形內、每次印字呼叫的命中紀錄
// （只有長度與 key，不含原文）。
//
//	tools/go.sh run ./cmd/dkk-overlay-shot -state /st/start-200m.state -root /orig \
//	  -scratch /out/scratch -font /font/dkk-eten.golemfnt \
//	  -events /text/ui-text-events.tsv -translations /text/ui-text.zh-TW.tsv \
//	  -bios-keys ' p' -bios-key-from 210000000 -bios-key-every 50000000 \
//	  -shots 399000000:menu -until 400000000 -out /out
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/dkk"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/xlate"
)

const scale = 2

type shot struct {
	step uint64
	name string
}

type shotResult struct {
	Name          string `json:"name"`
	Step          uint64 `json:"step"`
	IndexedSHA256 string `json:"indexed_sha256"` // 原版色號陣列，與 probe -dump-at 的 .bin 同一份資料
	BaseSHA256    string `json:"base_sha256"`
	OverlaySHA256 string `json:"overlay_sha256"`
	DiffPixels    int    `json:"diff_pixels"`
	DiffOutside   int    `json:"diff_outside_stamps"`
	Stamps        int    `json:"stamps"`
	StampsShown   int    `json:"stamps_shown"`
	Missing       string `json:"missing_glyphs,omitempty"`
}

func main() {
	statePath := flag.String("state", "", "probe 狀態檔")
	root := flag.String("root", "", "原版素材目錄（唯讀）")
	scratch := flag.String("scratch", "", "可寫暫存層（存檔落這裡）")
	fontPath := flag.String("font", "", "16×16 GOLEMFNT")
	events := flag.String("events", "", "ui-text-events.tsv")
	translations := flag.String("translations", "", "ui-text.zh-TW.tsv")
	eclEvents := flag.String("ecl-events", "", "ecl-text-events.tsv（給了才啟用路徑 B）")
	eclTranslations := flag.String("ecl-translations", "", "ecl-text.zh-TW.tsv")
	biosKeys := flag.String("bios-keys", "", "逐個送進 BIOS 鍵盤緩衝區的字元")
	biosKeyNames := flag.String("bios-key-names", "", "接在 -bios-keys 後送的具名鍵，逗號分隔")
	keyFrom := flag.Uint64("bios-key-from", 0, "第一個鍵的步數")
	keyEvery := flag.Uint64("bios-key-every", 50_000_000, "兩鍵間隔步數")
	shotsSpec := flag.String("shots", "", "步數:名稱，逗號分隔")
	until := flag.Uint64("until", 0, "絕對步數上限")
	outDir := flag.String("out", "", "輸出目錄")
	flag.Parse()
	if *statePath == "" || *root == "" || *fontPath == "" || *events == "" || *translations == "" || *until == 0 || *outDir == "" {
		fail(fmt.Errorf("state、root、font、events、translations、until、out 均為必填"))
	}
	shots, err := parseShots(*shotsSpec)
	if err != nil {
		fail(err)
	}
	keys, err := parseKeys(*biosKeys, *biosKeyNames)
	if err != nil {
		fail(err)
	}
	font, err := xlate.LoadFont(*fontPath)
	if err != nil {
		fail(err)
	}
	font.Name = filepath.Base(*fontPath)
	cat, err := dkk.LoadCatalog(mustRead(*events), mustRead(*translations))
	if err != nil {
		fail(err)
	}

	m := machine.New()
	d := dos.New(m, *root)
	d.Install()
	d.Scratch = *scratch
	if err := state.Load(*statePath, m, d); err != nil {
		fail(err)
	}
	layer := &xlate.Layer{W: 320, H: 200}
	pa, err := dkk.NewPathA(m, cat, font, layer)
	if err != nil {
		fail(err)
	}
	pc, err := dkk.NewPathC(m, cat, font, layer)
	if err != nil {
		fail(err)
	}
	pc.SetPalette(m.FramePalette())
	var pb *dkk.PathB
	if *eclEvents != "" {
		ecl, err := dkk.LoadCatalog(mustRead(*eclEvents), mustRead(*eclTranslations))
		if err != nil {
			fail(err)
		}
		if pb, err = dkk.NewPathB(m, font, layer, ecl, cat); err != nil {
			fail(err)
		}
		pb.SetPalette(m.FramePalette())
	}
	m.ObserveVideoWrites(func(w machine.VideoWrite) {
		pc.VideoWrite(w.Offset)
		if pb != nil {
			pb.VideoWrite(w.Offset)
		}
	})
	m.ObserveModeChanges(func(machine.ModeChange) {
		pc.Reset()
		if pb != nil {
			pb.Reset()
		}
	})
	m.SetOnFrame(func() {
		idx := m.Indexed()
		if len(idx) == 320*200 {
			pal := m.FramePalette()
			pa.PrepareFrame(idx, 320, 200)
			pc.SetPalette(pal)
			if pb != nil {
				pb.SetPalette(pal)
			}
			layer.Frame(idx, rgbOf(idx, pal))
		}
	}) // planar 以 FramePalette 取色（#12）

	var results []shotResult
	ki, si := 0, 0
	for m.Steps < *until && !d.Exited {
		if ki < len(keys) && m.Steps >= *keyFrom+uint64(ki)*(*keyEvery) {
			if _, pending := m.PeekKey(); !pending {
				d.PushKey(keys[ki])
				ki++
			}
		}
		if si < len(shots) && m.Steps >= shots[si].step {
			results = append(results, capture(m, layer, shots[si], *outDir))
			si++
		}
		pa.Before(m.Steps, m.CPU.Seg[cpu.CS], m.CPU.IP, m.CPU.Seg[cpu.SS], m.CPU.R[cpu.SP], m)
		pc.Before(m.CPU.Seg[cpu.CS], m.CPU.IP, m.CPU.Seg[cpu.SS], m.CPU.R[cpu.SP], m)
		if pb != nil {
			pb.Before(m.Steps, m.CPU.Seg[cpu.CS], m.CPU.IP, m.CPU.Seg[cpu.SS], m.CPU.R[cpu.SP], m.CPU.Seg[cpu.DS], m)
		}
		if err := m.Step(); err != nil {
			fail(err)
		}
	}
	out := struct {
		Entry   string         `json:"path_a_entry"`
		Catalog int            `json:"catalog_entries"`
		KeysOut int            `json:"keys_sent"`
		Stopped uint64         `json:"stopped_at"`
		Shots   []shotResult   `json:"shots"`
		Hits    []dkk.Hit      `json:"hits"`
		HitsB   []dkk.Hit      `json:"hits_b,omitempty"`
		MenuC   []dkk.MenuWord `json:"menu_words"`
	}{Entry: fmt.Sprintf("%05X", pa.Entry), Catalog: cat.Len(), KeysOut: ki, Stopped: m.Steps, Shots: results, Hits: pa.Hits}
	if pb != nil {
		out.HitsB = pb.Hits
	}
	out.MenuC = pc.Words
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func capture(m *machine.Machine, layer *xlate.Layer, s shot, dir string) shotResult {
	idx := m.Indexed()
	if len(idx) != 320*200 {
		fail(fmt.Errorf("#%d 畫面不是 320×200（%d 點）", m.Steps, len(idx)))
	}
	pal := m.FramePalette()
	base := upscale(idx, pal)
	over := append([]uint8(nil), base...)
	var missing []rune
	layer.Draw(over, scale, func(r rune) { missing = append(missing, r) })
	isum := sha256.Sum256(idx)
	r := shotResult{Name: s.name, Step: m.Steps, IndexedSHA256: hex.EncodeToString(isum[:]), Stamps: len(layer.Stamps)}
	for _, st := range layer.Stamps {
		if st.State == xlate.Shown {
			r.StampsShown++
		}
	}
	for i := 0; i < len(base); i += 4 {
		if base[i] == over[i] && base[i+1] == over[i+1] && base[i+2] == over[i+2] {
			continue
		}
		r.DiffPixels++
		x, y := (i/4)%(320*scale)/scale, (i/4)/(320*scale)/scale
		inside := false
		for _, st := range layer.Stamps {
			x0, y0, x1, y1 := st.Rect()
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				inside = true
				break
			}
		}
		if !inside {
			r.DiffOutside++
		}
	}
	r.BaseSHA256 = writePNG(filepath.Join(dir, s.name+"-base.png"), base)
	r.OverlaySHA256 = writePNG(filepath.Join(dir, s.name+"-overlay.png"), over)
	r.Missing = string(missing)
	return r
}

func rgbOf(idx []uint8, pal [256][3]uint8) []uint8 {
	out := make([]uint8, len(idx)*3)
	for i, c := range idx {
		out[i*3], out[i*3+1], out[i*3+2] = pal[c][0], pal[c][1], pal[c][2]
	}
	return out
}

func upscale(idx []uint8, pal [256][3]uint8) []uint8 {
	w := 320 * scale
	out := make([]uint8, w*200*scale*4)
	for y := 0; y < 200*scale; y++ {
		for x := 0; x < w; x++ {
			c := pal[idx[(y/scale)*320+x/scale]]
			o := (y*w + x) * 4
			out[o], out[o+1], out[o+2], out[o+3] = c[0], c[1], c[2], 0xFF
		}
	}
	return out
}

func writePNG(path string, rgba []uint8) string {
	img := &image.RGBA{Pix: rgba, Stride: 320 * scale * 4, Rect: image.Rect(0, 0, 320*scale, 200*scale)}
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	if err := png.Encode(f, img); err != nil {
		fail(err)
	}
	if err := f.Close(); err != nil {
		fail(err)
	}
	sum := sha256.Sum256(rgba)
	return hex.EncodeToString(sum[:])
}

func parseShots(spec string) ([]shot, error) {
	var out []shot
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	for _, part := range strings.Split(spec, ",") {
		i := strings.Index(part, ":")
		if i < 0 {
			return nil, fmt.Errorf("-shots %q 不是 步數:名稱", part)
		}
		n, err := strconv.ParseUint(part[:i], 10, 64)
		if err != nil || part[i+1:] == "" || strings.ContainsAny(part[i+1:], "/\\") {
			return nil, fmt.Errorf("-shots %q 不合法", part)
		}
		if len(out) > 0 && n <= out[len(out)-1].step {
			return nil, fmt.Errorf("-shots 步數要遞增")
		}
		out = append(out, shot{n, part[i+1:]})
	}
	return out, nil
}

func parseKeys(text, names string) ([]dos.Key, error) {
	var q []dos.Key
	for _, r := range text {
		if k, ok := dos.KeyForRune(r); ok {
			q = append(q, k)
			continue
		}
		switch r {
		case '\r', '\n':
			q = append(q, dos.Key{Scan: 0x1C, ASCII: '\r'})
		default:
			q = append(q, dos.Key{ASCII: uint8(r)})
		}
	}
	if strings.TrimSpace(names) != "" {
		for _, n := range strings.Split(names, ",") {
			k, ok := dos.KeyNamed(strings.TrimSpace(n))
			if !ok {
				return nil, fmt.Errorf("不認得的鍵名 %q", n)
			}
			q = append(q, k)
		}
	}
	return q, nil
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	return b
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "dkk-overlay-shot：", err)
	os.Exit(1)
}
