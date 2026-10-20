// cok-promo 從既有狀態續跑，逐格輸出覆繪後的畫面（PNG）與遊戲音訊加配樂（PCM），供推廣影片剪輯
// （主專案規格 016）。畫面、HD、語言包、面板與配樂的接法同 cok-play。時間基準也同 cok-play：VGA 回掃每秒 70 次，
// 每秒 VGAFrameEvery×70 步（cok-play 的 stepsPerHostFrame×60），從載入後的機器讀，所以任何時脈比例的狀態檔都對；
// 影片每格與音訊樣本都從這個量換算。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/apps/cok/bgm"
	"github.com/wicanr2/dosgolem/audio/mixer"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/xlate"
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

type timedKey struct {
	at  uint64
	key dos.Key
}

type action struct {
	at   uint64
	name string
	arg  string
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cok-promo:", err)
	os.Exit(1)
}

func parseKey(name string) (dos.Key, error) {
	name = strings.TrimSpace(name)
	if k, ok := dos.KeyNamed(name); ok {
		return k, nil
	}
	if r := []rune(name); len(r) == 1 {
		if k, ok := dos.KeyForRune(r[0]); ok {
			return k, nil
		}
		return dos.Key{ASCII: uint8(r[0])}, nil
	}
	return dos.Key{}, fmt.Errorf("未知具名鍵 %q", name)
}

// parseTimed 解析 `步數:內容`，逗號分隔。
func parseTimed(s string) ([][2]string, []uint64, error) {
	var items [][2]string
	var at []uint64
	for _, it := range strings.Split(s, ",") {
		if it = strings.TrimSpace(it); it == "" {
			continue
		}
		n, rest, ok := strings.Cut(it, ":")
		v, err := strconv.ParseUint(n, 10, 64)
		if !ok || err != nil {
			return nil, nil, fmt.Errorf("%q 不是 步數:內容", it)
		}
		name, arg, _ := strings.Cut(rest, "=")
		items = append(items, [2]string{name, arg})
		at = append(at, v)
	}
	return items, at, nil
}

func main() {
	st := flag.String("state", "", "dosgolem 狀態檔")
	root := flag.String("root", "", "唯讀原版資料")
	scratch := flag.String("scratch", "", "可寫暫存層")
	output := flag.String("out", "", "輸出目錄：frames/NNNNNN.png、audio.s16le（48 kHz 立體聲）、receipt.json")
	fontPath := flag.String("font", "", "16×16 GOLEMFNT（繁中）")
	textDir := flag.String("text-dir", "", "譯文目錄")
	hdDir := flag.String("hd", "", "HD 主題目錄（只在 3× 顯示）")
	musicDir := flag.String("music-dir", "", "配樂目錄（music.tsv 與 <曲目>.opl）；選曲表讀 text-dir 的 music-cues.tsv")
	langFonts := flag.String("lang-fonts", "", "繁中以外語言的字型：代碼=路徑，逗號分隔；同目錄 <名稱>-3x.golemfnt 當 3 倍字模")
	langCode := flag.String("lang", "zh-TW", "起始語言")
	scale := flag.Int("scale", 3, "輸出倍率 2 或 3")
	fps := flag.Int("fps", 30, "影片畫格率")
	from := flag.Uint64("from", 0, "開始錄的絕對步數（之前照跑、不輸出）")
	until := flag.Uint64("until", 0, "絕對步數上限")
	keysAt := flag.String("keys-at", "", "依步數送鍵 `步數:鍵名`，逗號分隔；鍵盤緩衝區有鍵時順延")
	keys := flag.String("keys", "", "從 -key-from 起每 -key-every 步送一個字元")
	keyNames := flag.String("key-names", "", "接在 -keys 後送的具名鍵，逗號分隔")
	keyFrom := flag.Uint64("key-from", 0, "-keys 第一個鍵的步數")
	keyEvery := flag.Uint64("key-every", 30_000_000, "-keys 兩鍵間隔步數")
	actions := flag.String("actions", "", "依步數做前端動作 `步數:動作[=參數]`，逗號分隔："+
		"journal、journal-next、guide=章:頁、guide-page、guide-close、map、map-reveal、map-close、lang=代碼、original、chinese")
	dumpAt := flag.String("dump-mem-at", "", "診斷：到這些步數把線性 0–9FFFF 寫成 <out>/mem-<步數>.bin，逗號分隔")
	cheatOn := flag.Bool("cheat", false, "我方無敵（主專案規格 017）：同 cok-play 的判斷，以 machine 寫入監看實作，每格更新監看範圍")
	poke := flag.String("initial-poke", "", "起點注入：線性十六進位位址=hex bytes，分號分隔（同 cok-hd-probe）")
	flag.Parse()
	if *st == "" || *root == "" || *scratch == "" || *output == "" || *fontPath == "" || *textDir == "" || *until == 0 ||
		(*scale != 2 && *scale != 3) || *fps <= 0 {
		flag.Usage()
		os.Exit(2)
	}

	var timed []timedKey
	items, at, err := parseTimed(*keysAt)
	if err != nil {
		fail(fmt.Errorf("-keys-at：%w", err))
	}
	for i, it := range items {
		k, err := parseKey(it[0])
		if err != nil {
			fail(err)
		}
		timed = append(timed, timedKey{at[i], k})
	}
	n := 0
	for _, r := range *keys {
		k, ok := dos.KeyForRune(r)
		if !ok {
			k = dos.Key{ASCII: uint8(r)}
		}
		timed = append(timed, timedKey{*keyFrom + uint64(n)*(*keyEvery), k})
		n++
	}
	if strings.TrimSpace(*keyNames) != "" {
		for _, name := range strings.Split(*keyNames, ",") {
			k, err := parseKey(name)
			if err != nil {
				fail(err)
			}
			timed = append(timed, timedKey{*keyFrom + uint64(n)*(*keyEvery), k})
			n++
		}
	}
	sort.SliceStable(timed, func(a, b int) bool { return timed[a].at < timed[b].at })
	var acts []action
	items, at, err = parseTimed(*actions)
	if err != nil {
		fail(fmt.Errorf("-actions：%w", err))
	}
	for i, it := range items {
		acts = append(acts, action{at[i], it[0], it[1]})
	}
	sort.SliceStable(acts, func(a, b int) bool { return acts[a].at < acts[b].at })

	font, err := xlate.LoadFont(*fontPath)
	if err != nil {
		fail(err)
	}
	font.Name = filepath.Base(*fontPath)
	ui, ecl, err := cok.LoadTextDir(*textDir)
	if err != nil {
		fail(err)
	}
	live := cok.NewLive(font, ui, ecl)
	if *hdDir != "" {
		if err := live.AddHD(*hdDir); err != nil {
			fail(err)
		}
	}
	if live.Answers, live.QuizFormat, err = cok.LoadQuizDir(*textDir); err != nil {
		fail(err)
	}
	if live.Journal, err = cok.LoadJournalDir(*textDir); err != nil {
		fail(err)
	}
	if live.Guide, err = cok.LoadGuideDir(*textDir); err != nil {
		fail(err)
	}
	if live.Map, err = cok.LoadAutoMapDir(*textDir); err != nil {
		fail(err)
	}
	if cues, err := os.ReadFile(filepath.Join(*textDir, "music-cues.tsv")); err == nil {
		if live.Music, err = cok.LoadMusicCues(cues); err != nil {
			fail(err)
		}
	}
	packs := map[string]*cok.LangPack{}
	loadPack := func(code, path string) {
		f := font
		if code != cok.DefaultLang {
			if f, err = xlate.LoadFont(path); err != nil {
				fail(fmt.Errorf("語言 %s 的字型：%w", code, err))
			}
			f.Name = filepath.Base(path)
			if p3 := strings.TrimSuffix(path, ".golemfnt") + "-3x.golemfnt"; fileExists(p3) {
				f3, err := xlate.LoadFont(p3)
				if err != nil {
					fail(err)
				}
				f.Scaled = map[int]xlate.ScaledFont{3: {Font: f3}}
			}
		}
		p, err := cok.LoadLangPack(*textDir, code, f)
		if err != nil {
			fail(fmt.Errorf("語言 %s：%w", code, err))
		}
		packs[code] = p
	}
	loadPack(cok.DefaultLang, "")
	for _, f := range strings.Split(*langFonts, ",") {
		if code, path, ok := strings.Cut(strings.TrimSpace(f), "="); ok {
			loadPack(code, path)
		}
	}
	setLang := func(code string) {
		p := packs[code]
		if p == nil {
			fail(fmt.Errorf("語言 %s 沒有載入（-lang-fonts）", code))
		}
		live.SetLang(p)
	}
	setLang(*langCode)

	music := (*bgm.Player)(nil)
	if *musicDir != "" {
		if music, err = bgm.Load(*musicDir, live.Music); err != nil {
			fail(err)
		}
	}

	input, err := os.ReadFile(*st)
	if err != nil {
		fail(err)
	}
	m := machine.New()
	m.SetAdLib(true)
	d := dos.New(m, *root)
	d.Install()
	d.Scratch = *scratch
	if err := state.Load(*st, m, d); err != nil {
		fail(err)
	}
	m.SetAdLib(true)
	// 不用 StepsPerSecondNow()：它是 IRQ0 間隔×PIT 頻率，本作狀態下約 890k 步／秒，影片會慢 13 倍；
	// 這裡要的是與 cok-play 相同的時間（VGA 回掃 70 Hz）。
	stepsPerSecond := m.VGAFrameEvery * 70
	mix := mixer.New(stepsPerSecond)
	if *poke != "" {
		for _, entry := range strings.Split(*poke, ";") {
			address, value, ok := strings.Cut(entry, "=")
			a, err1 := strconv.ParseUint(address, 16, 20)
			data, err2 := hex.DecodeString(value)
			if !ok || err1 != nil || err2 != nil || a+uint64(len(data)) > 1<<20 {
				fail(fmt.Errorf("-initial-poke %q 不合法", entry))
			}
			for i, v := range data {
				m.Write8(uint32(a)+uint32(i), v)
			}
		}
	}
	initial := m.Steps
	live.Arm() // 狀態檔已在遊戲中
	m.ObserveVideoWrites(func(w machine.VideoWrite) { live.VideoWrite(w.Offset) })
	m.ObserveModeChanges(func(machine.ModeChange) { live.ModeChange() })
	m.SetOnFrame(func() { live.Frame(m.Indexed(), m.FramePalette()) })
	m.ObserveOPLWrites(mix.OPLWrite)
	m.ObserveSpeaker(mix.SpeakerSample)
	m.ObservePITChannel2(mix.PITChannel2)

	framesDir := filepath.Join(*output, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		fail(err)
	}
	audioFile, err := os.Create(filepath.Join(*output, "audio.s16le"))
	if err != nil {
		fail(err)
	}
	audioHash, frameHash := sha256.New(), sha256.New()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	chinese := true
	missing := map[rune]bool{}
	frameEvery := stepsPerSecond / uint64(*fps)
	var samples []float32
	var pcm []byte
	frames, totalSamples := 0, 0
	last := m.Steps
	next := m.Steps + frameEvery
	ki, ai := 0, 0
	cheat := &cok.Invincible{On: *cheatOn}
	watchID, restoring := 0, false
	rewatch := func() {
		if watchID != 0 {
			m.Unwatch(watchID)
			watchID = 0
		}
		if lo, hi := cheat.GuardRange(machineMem{m}); lo <= hi {
			watchID = m.WatchWrite(lo, hi, func(m *machine.Machine, a uint32, old, v uint8) {
				if restoring || old == v || !cheat.GuardWrite(a, old, v) {
					return
				}
				restoring = true
				m.Write8(a, old)
				restoring = false
			})
		}
	}
	rewatch()
	var dumps []uint64
	for _, f := range strings.Split(*dumpAt, ",") {
		if v, err := strconv.ParseUint(strings.TrimSpace(f), 10, 64); err == nil {
			dumps = append(dumps, v)
		}
	}
	sort.Slice(dumps, func(a, b int) bool { return dumps[a] < dumps[b] })

	doAction := func(a action) {
		switch a.name {
		case "journal":
			live.Journal.ToggleBrowse()
		case "journal-next":
			live.Journal.NextEntry(1)
		case "guide":
			var ch, page int
			if _, err := fmt.Sscanf(a.arg, "%d:%d", &ch, &page); err != nil {
				fail(fmt.Errorf("guide=%q 要寫 章:頁", a.arg))
			}
			live.Guide.SetPos(ch, page)
			if !live.Guide.Open() {
				live.Guide.Toggle()
			}
		case "guide-page":
			live.Guide.TurnPage(1)
		case "guide-close":
			live.Guide.Close()
		case "map":
			live.Map.Toggle()
		case "map-reveal":
			live.Map.ToggleReveal()
		case "map-close":
			live.Map.Close()
		case "lang":
			setLang(a.arg)
		case "original":
			chinese = false
		case "chinese":
			chinese = true
		default:
			fail(fmt.Errorf("未知動作 %q", a.name))
		}
	}
	emit := func() {
		samples = mix.Render(samples[:0], last, m.Steps)
		music.Mix(samples)
		last = m.Steps
		if m.Steps < *from {
			return
		}
		pcm = pcm[:0]
		for _, v := range samples {
			pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(math.Round(float64(v)*32767))))
		}
		if _, err := audioFile.Write(pcm); err != nil {
			fail(err)
		}
		audioHash.Write(pcm)
		totalSamples += len(samples) / 2
		pix := live.Compose(*scale, chinese, func(r rune) {
			if !missing[r] {
				missing[r] = true
				fmt.Fprintf(os.Stderr, "cok-promo: 字型缺字 %q（U+%04X）\n", r, r)
			}
		})
		if pix == nil {
			pix = make([]byte, 4*320*200*(*scale)*(*scale))
			for i := 3; i < len(pix); i += 4 {
				pix[i] = 0xFF
			}
		}
		writeFrame(&enc, filepath.Join(framesDir, fmt.Sprintf("%06d.png", frames)), pix, *scale, frameHash)
		frames++
	}
	for m.Steps < *until && !d.Exited {
		for ai < len(acts) && m.Steps >= acts[ai].at {
			doAction(acts[ai])
			ai++
		}
		if ki < len(timed) && m.Steps >= timed[ki].at {
			if _, pending := m.PeekKey(); !pending {
				d.PushKey(timed[ki].key)
				ki++
			}
		}
		if err := live.BeforeStep(view{m}); err != nil {
			fail(err)
		}
		if err := m.Step(); err != nil {
			fail(err)
		}
		for len(dumps) > 0 && m.Steps >= dumps[0] {
			mem := make([]byte, 0xA0000)
			for i := range mem {
				mem[i] = m.Peek8(uint32(i))
			}
			if err := os.WriteFile(filepath.Join(*output, fmt.Sprintf("mem-%d.bin", dumps[0])), mem, 0o644); err != nil {
				fail(err)
			}
			dumps = dumps[1:]
		}
		if m.Steps >= next {
			emit()
			next += frameEvery
			rewatch()
		}
	}
	if err := audioFile.Close(); err != nil {
		fail(err)
	}
	raw, err := json.MarshalIndent(map[string]any{
		"initial_state_sha256": fmt.Sprintf("%x", sha256.Sum256(input)), "initial_steps": initial, "initial_poke": *poke,
		"from": *from, "steps": m.Steps, "clock_percent": m.ClockPercent(), "steps_per_second": stepsPerSecond,
		"machine_steps_per_second_now": m.StepsPerSecondNow(), "fps": *fps, "frames": frames,
		"sample_rate": mixer.SampleRate, "samples": totalSamples,
		"audio_seconds": float64(totalSamples) / mixer.SampleRate, "video_seconds": float64(frames) / float64(*fps),
		"audio_sha256": hexSum(audioHash), "frames_sha256": hexSum(frameHash),
		"keys_sent": ki, "keys_total": len(timed), "actions_done": ai, "hd_items": live.HDItems, "lang": *langCode,
		"music": *musicDir != "", "missing_glyphs": len(missing),
		"cheat": *cheatOn, "cheat_restored": cheat.Restored, "cheat_removed_affects": cheat.Removed, "cheat_party": cheat.Report,
	}, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*output, "receipt.json"), append(raw, '\n'), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("steps=%d frames=%d audio=%.3fs video=%.3fs keys=%d/%d HD=%d\n", m.Steps, frames,
		float64(totalSamples)/mixer.SampleRate, float64(frames)/float64(*fps), ki, len(timed), live.HDItems)
}

// writeFrame 存一格 PNG；畫格雜湊取 RGBA 原始位元組，不受 PNG 壓縮影響。
func writeFrame(enc *png.Encoder, path string, pix []byte, scale int, h hash.Hash) {
	h.Write(pix)
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	img := &image.NRGBA{Pix: pix, Stride: 4 * 320 * scale, Rect: image.Rect(0, 0, 320*scale, 200*scale)}
	if err := enc.Encode(f, img); err != nil {
		fail(err)
	}
	if err := f.Close(); err != nil {
		fail(err)
	}
}

// machineMem 讓無敵在每格開頭直接讀寫記憶體（同 session.GuardMemory）。
type machineMem struct{ m *machine.Machine }

func (g machineMem) Peek8(a uint32) uint8    { return g.m.Peek8(a) }
func (g machineMem) Poke8(a uint32, v uint8) { g.m.Write8(a, v) }

func hexSum(h hash.Hash) string { return fmt.Sprintf("%x", h.Sum(nil)) }

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}
