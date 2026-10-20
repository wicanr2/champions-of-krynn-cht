// cok-hd-probe 從既有 dosgolem 狀態取得 HD 原版畫格證據及 A/B 最終狀態。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/xlate"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type view struct{ m *machine.Machine }

func (v view) Read8(a uint32) uint8   { return v.m.Peek8(a) }
func (v view) Read16(a uint32) uint16 { return v.m.Peek16(a) }
func (v view) Steps() uint64          { return v.m.Steps }
func (v view) CS() uint16             { return v.m.CPU.Seg[cpu.CS] }
func (v view) IP() uint16             { return v.m.CPU.IP }
func (v view) SS() uint16             { return v.m.CPU.Seg[cpu.SS] }
func (v view) SP() uint16             { return v.m.CPU.R[cpu.SP] }
func (v view) DS() uint16             { return v.m.CPU.Seg[cpu.DS] }

func main() {
	st := flag.String("state", "", "原版保存狀態")
	root := flag.String("root", "", "唯讀原版資料")
	scratch := flag.String("scratch", "", "可寫存檔目錄")
	output := flag.String("out", "", "輸出目錄")
	until := flag.Uint64("until", 0, "絕對步數上限")
	sample := flag.Uint64("sample", 1_000_000, "畫面採樣間隔指令數")
	hd := flag.String("hd", "", "可選 HD 主題")
	fontpath := flag.String("font", "", "可選中文字型")
	text := flag.String("text-dir", "", "可選譯文")
	scale := flag.Int("scale", 3, "輸出倍率")
	keys := flag.String("keys", "", "依序送字元")
	keyNames := flag.String("key-names", "", "接續送具名鍵，逗號分隔")
	poke := flag.String("initial-poke", "", "診斷起點注入：線性十六進位位址=hex bytes，以分號分隔")
	from := flag.Uint64("key-from", 0, "第一個鍵步數")
	every := flag.Uint64("key-every", 30_000_000, "鍵間隔")
	limit := flag.Int("limit", 200, "最多保存獨立畫格")
	saveState := flag.String("save-state", "", "可選：保存續跑終點，僅供本機收據")
	flag.Parse()
	must := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var inputKeys []dos.Key
	for _, r := range *keys {
		k, ok := dos.KeyForRune(r)
		if !ok {
			k = dos.Key{ASCII: uint8(r)}
		}
		inputKeys = append(inputKeys, k)
	}
	if strings.TrimSpace(*keyNames) != "" {
		for _, name := range strings.Split(*keyNames, ",") {
			k, ok := dos.KeyNamed(strings.TrimSpace(name))
			if r := []rune(strings.TrimSpace(name)); !ok && len(r) == 1 {
				k, ok = dos.KeyForRune(r[0])
			}
			if !ok {
				must(fmt.Errorf("未知具名鍵 %q", name))
			}
			inputKeys = append(inputKeys, k)
		}
	}
	if *st == "" || *root == "" || *scratch == "" || *output == "" || *until == 0 || *sample == 0 || (*scale != 2 && *scale != 3) {
		must(fmt.Errorf("必要參數或倍率不合法"))
	}
	must(os.MkdirAll(*output, 0755))
	must(os.MkdirAll(*scratch, 0755))
	input, err := os.ReadFile(*st)
	must(err)
	m := machine.New()
	d := dos.New(m, *root)
	d.Install()
	d.Scratch = *scratch
	must(state.Load(*st, m, d))
	if *poke != "" {
		for _, entry := range strings.Split(*poke, ";") {
			address, value, ok := strings.Cut(entry, "=")
			if !ok {
				must(fmt.Errorf("起點注入格式不合法"))
			}
			a, err := strconv.ParseUint(address, 16, 20)
			must(err)
			data, err := hex.DecodeString(value)
			must(err)
			if a+uint64(len(data)) > 1<<20 {
				must(fmt.Errorf("起點注入越界"))
			}
			for n, v := range data {
				m.Write8(uint32(a)+uint32(n), v)
			}
		}
	}
	initial := m.Steps
	font := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	if *fontpath != "" {
		font, err = xlate.LoadFont(*fontpath)
		must(err)
	}
	var ui cok.Lookuper
	var ecl *cok.Catalog
	if *text != "" {
		ui, ecl, err = cok.LoadTextDir(*text)
		must(err)
	}
	live := cok.NewLive(font, ui, ecl)
	if *hd != "" {
		must(live.AddHD(*hd))
	}
	if *text != "" {
		live.Arm()
		m.ObserveVideoWrites(func(w machine.VideoWrite) { live.VideoWrite(w.Offset) })
		m.ObserveModeChanges(func(machine.ModeChange) { live.ModeChange() })
	}
	seen := map[[32]byte]bool{}
	var prev [32]byte
	next := m.Steps
	var shots []map[string]any
	capture := func(name string) {
		idx := m.Indexed()
		if len(idx) != 64000 {
			return
		}
		live.Frame(idx, m.FramePalette())
		must(os.WriteFile(filepath.Join(*output, name+".bin"), idx, 0644))
		pix := live.Compose(*scale, true, nil)
		f, e := os.Create(filepath.Join(*output, name+".png"))
		must(e)
		must(png.Encode(f, &image.NRGBA{Pix: pix, Stride: 320 * (*scale) * 4, Rect: image.Rect(0, 0, 320*(*scale), 200*(*scale))}))
		must(f.Close())
		shots = append(shots, map[string]any{"name": name, "steps": m.Steps, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(idx))})
	}
	m.SetOnFrame(func() {
		idx := m.Indexed()
		live.Frame(idx, m.FramePalette())
		if m.Steps < next || len(idx) != 64000 {
			return
		}
		next = m.Steps + *sample
		// 完整左側畫面穩定兩次才存，避免半畫格被列為已觀測來源。
		region := make([]byte, 0, 176*168)
		for y := 8; y < 176; y++ {
			region = append(region, idx[y*320+8:y*320+176]...)
		}
		sig := sha256.Sum256(region)
		if sig == prev && !seen[sig] && len(shots) < *limit {
			seen[sig] = true
			capture(fmt.Sprintf("frame-%012d", m.Steps))
		}
		prev = sig
	})
	ki := 0
	for m.Steps < *until && !d.Exited {
		if ki < len(inputKeys) && m.Steps >= *from+uint64(ki)*(*every) {
			if _, pending := m.PeekKey(); !pending {
				d.PushKey(inputKeys[ki])
				ki++
			}
		}
		if *text != "" {
			must(live.BeforeStep(view{m}))
		}
		must(m.Step())
	}
	capture("final")
	if *saveState != "" {
		must(state.Save(*saveState, m, d))
	}
	ram := make([]byte, 1<<20)
	for n := range ram {
		ram[n] = m.Peek8(uint32(n))
	}
	registers := struct {
		R         [8]uint16
		Seg       [4]uint16
		IP, Flags uint16
	}{m.CPU.R, m.CPU.Seg, m.CPU.IP, m.CPU.Flags}
	raw, err := json.MarshalIndent(map[string]any{"initial_state_sha256": fmt.Sprintf("%x", sha256.Sum256(input)), "initial_steps": initial, "initial_poke": *poke, "steps": m.Steps, "memory_sha256": fmt.Sprintf("%x", sha256.Sum256(ram)), "registers": registers, "shots": shots, "keys_sent": ki, "keys": *keys, "key_names": *keyNames, "key_from": *from, "key_every": *every, "hd_items": live.HDItems, "scope": "既有狀態續跑；相同state固定初始亂數與虛擬時鐘。initial_poke明列診斷注入，空字串代表本輪無注入。不是新冷開機玩家路徑。"}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(*output, "receipt.json"), append(raw, '\n'), 0644))
	fmt.Printf("steps=%d shots=%d HD=%d\n", m.Steps, len(shots), live.HDItems)
}
