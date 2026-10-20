// Command cok-play 是《Death Knights of Krynn》繁中覆繪的可遊玩視窗（death-knights-of-krynn-cht #6）。
//
// 以 session 從原版目錄開機（bootroot 複製到可寫存檔目錄，原版唯讀），每個主機畫格推進
// 原版節奏的指令數，以 apps/cok.Live 疊上路徑 A、B、C 的中文，2× 或 3× 輸出（F2 切換）。
// F12 即時切換中文／原版（原版看不到這個鍵），F2 切換 2×／3×，F11 全螢幕，F3 靜音，F10 截圖，F9 查驗題答案顯示，
// F5 翻閱已讀手札（PgUp／PgDn 翻頁，←／→ 換則，Esc 關閉）。
// 滑鼠移到視窗頂端時出現切換列（主專案規格 011），點項目等同按對應的功能鍵。
//
// 沒給 -save 時是發行模式（主專案規格 012）：資料從發行包找，原版第一次匯入到使用者資料目錄。
// 自動模式（-frames）供收據使用：依 -script 送鍵，結束時印記憶體與畫面雜湊，
// 並可輸出 -shot（中文合成 RGBA）與 -shot-original（原版合成 RGBA）。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/apps/cok/bgm"
	"github.com/wicanr2/dosgolem/audio/mixer"
	"github.com/wicanr2/dosgolem/bootroot"
	"github.com/wicanr2/dosgolem/host"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/session"
	"github.com/wicanr2/dosgolem/xlate"
)

const (
	windowTitle = "克萊恩英豪"
	// startEXESHA256 是 G1 盤點的 START.EXE v1.2（champions-of-krynn-cht docs/re/001）。
	startEXESHA256 = "e087c20258cc87a06fb9db62a3898896b47c0b94796f748f72eb8986919ee384"
)

func stepsPerHostFrame(clock int) uint64 {
	return machine.ScaleSteps(machine.DefaultVGAFrameEvery, clock) * 70 / 60
}

// observer 把 session 的觀測事件轉給 cok.Live 與音訊混音器，寫入守衛轉給無敵開關（主專案規格 017）。
type observer struct {
	live  *cok.Live
	mix   *mixer.Mixer
	cheat *cok.Invincible
}

func (o observer) GuardRange(peek func(uint32) uint8) (uint32, uint32) {
	return o.cheat.GuardRange(peek)
}
func (o observer) GuardWrite(addr uint32, old, new uint8) bool {
	return o.cheat.GuardWrite(addr, old, new)
}

func (o observer) BeforeStep(v session.StepView) error { return o.live.BeforeStep(v) }
func (o observer) VideoWrite(w machine.VideoWrite)     { o.live.VideoWrite(w.Offset) }
func (o observer) VideoModeChange(machine.ModeChange)  { o.live.ModeChange() }
func (o observer) Frame(indexed []byte, palette [256][3]uint8) {
	o.live.Frame(indexed, palette)
}
func (o observer) OPLWrite(w machine.OPLWrite)           { o.mix.OPLWrite(w) }
func (o observer) SpeakerSample(s machine.SpeakerSample) { o.mix.SpeakerSample(s) }
func (o observer) PITChannel2(c machine.PIT2Change)      { o.mix.PITChannel2(c) }

var namedKeys = map[ebiten.Key]string{
	ebiten.KeyEnter: "Enter", ebiten.KeyNumpadEnter: "Enter", ebiten.KeyEscape: "Escape",
	ebiten.KeyBackspace: "Backspace", ebiten.KeyTab: "Tab",
	ebiten.KeyArrowUp: "Up", ebiten.KeyArrowDown: "Down", ebiten.KeyArrowLeft: "Left", ebiten.KeyArrowRight: "Right",
	ebiten.KeyHome: "Home", ebiten.KeyEnd: "End", ebiten.KeyPageUp: "PgUp", ebiten.KeyPageDown: "PgDn",
	ebiten.KeyInsert: "Insert", ebiten.KeyDelete: "Delete",
	ebiten.KeyNumpad0: "KP0", ebiten.KeyNumpad1: "KP1", ebiten.KeyNumpad2: "KP2", ebiten.KeyNumpad3: "KP3",
	ebiten.KeyNumpad4: "KP4", ebiten.KeyNumpad5: "KP5", ebiten.KeyNumpad6: "KP6", ebiten.KeyNumpad7: "KP7",
	ebiten.KeyNumpad8: "KP8", ebiten.KeyNumpad9: "KP9", ebiten.KeyNumpadDecimal: "KPDot",
	ebiten.KeyF1: "F1", ebiten.KeyF2: "F2", ebiten.KeyF3: "F3", ebiten.KeyF4: "F4", ebiten.KeyF5: "F5",
	ebiten.KeyF6: "F6", ebiten.KeyF7: "F7", ebiten.KeyF8: "F8", ebiten.KeyF9: "F9", ebiten.KeyF10: "F10",
	ebiten.KeyF11: "F11", ebiten.KeyF12: "F12",
}

func init() {
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		namedKeys[k] = string(rune('A' + (k - ebiten.KeyA)))
	}
}

type game struct {
	owner    *session.Owner
	scale    int // 輸出倍率 2 或 3（F2 切換，比照拯救地球）
	live     *cok.Live
	chinese  bool
	frame    int
	frames   int
	script   map[int][]scriptAction
	shot     string
	shotOrig string
	shotDir  string
	budget   uint64
	mix      *mixer.Mixer
	ring     *mixer.Ring
	player   *audio.Player
	muted    bool
	pressing bool
	focused  bool
	samples  []float32
	missing  map[rune]bool
	bar      toolbar           // 視窗頂端切換列（主專案規格 011）
	barText  map[string]string // host-text 的 toolbar.* 文字
	music    *bgm.Player       // 遊戲中配樂（規格 014）；nil 表示沒有素材
	cheat    *cok.Invincible   // 我方無敵（主專案規格 017），預設關
	langs    []*cok.LangPack   // 已載入的語言（規格 015），索引 lang 為目前語言
	lang     int
	langDir  string      // 語言選擇記錄的目錄
	mainFont *xlate.Font // -font（切換列在英文版用它）
	cursorX  int         // 游標邏輯位置（切換列懸停用）
	cursorY  int
}

func (g *game) hostInput(u *session.CapturedUpdate) []hostAction {
	in := frameInput{Chars: ebiten.AppendInputChars(nil), Held: map[string]int{},
		Ctrl: ebiten.IsKeyPressed(ebiten.KeyControl), Alt: ebiten.IsKeyPressed(ebiten.KeyAlt)}
	for k, name := range namedKeys {
		if d := inpututil.KeyPressDuration(k); d > 0 && (in.Held[name] == 0 || d < in.Held[name]) {
			in.Held[name] = d
		}
	}
	keys, acts := mapKeys(in, g.panelMode())
	u.BIOSKeys = append(u.BIOSKeys, keys...)
	cx, cy := ebiten.CursorPosition()
	dx, dy := dosPoint(cx, cy, g.scale)
	g.cursorX, g.cursorY = dx, dy
	g.bar.track(dx, dy)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.pressing && g.bar.takesPointer(dx, dy) {
		// 切換列範圍內的點擊由前端處理，不送進遊戲（規格 011 §2.3）
		if a := toolbarClick(g.toolbarItems(), dx, dy); a != 0 {
			acts = append(acts, a)
		}
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.pressing {
		if ev := mouseEvent(host.MouseEventDown, dx, dy); ev.Target == host.MouseTargetCanvas {
			u.PointerDown, g.pressing = &ev, true
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.pressing {
		ev := mouseEvent(host.MouseEventUp, dx, dy)
		u.PointerUp, g.pressing = &ev, false
	}
	focused := ebiten.IsFocused()
	if g.focused && !focused && g.pressing {
		u.FocusLost, g.pressing = true, false
	}
	g.focused = focused
	return acts
}

func (g *game) apply(acts []hostAction) {
	for _, a := range acts {
		switch a {
		case actToggle:
			g.chinese = !g.chinese
		case actMute:
			g.muted = !g.muted
			if g.player != nil {
				g.player.SetVolume(map[bool]float64{true: 0, false: 1}[g.muted])
			}
		case actMusic:
			g.music.Toggle()
		case actCheat:
			g.cheat.On = !g.cheat.On
		case actLang:
			g.nextLang()
		case actLangCycle:
			g.cycleLang()
		case actScale:
			g.scale = 5 - g.scale // 2 ↔ 3
			if !ebiten.IsFullscreen() {
				ebiten.SetWindowSize(320*g.scale, 200*g.scale)
			}
		case actFullscreen:
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
		case actShot:
			g.screenshot()
		case actAnswers:
			g.live.SetHideAnswers(!g.live.HideAnswers())
		case actJournal:
			if g.chinese && g.live.Journal != nil {
				g.live.Guide.Close() // 與攻略面板、地圖互斥（規格 008 §2.1、009 §2.3）
				g.live.Map.Close()
				g.live.Journal.ToggleBrowse()
			}
		case actGuide:
			if g.chinese && g.live.Guide != nil {
				if g.live.Journal.Browsing() {
					g.live.Journal.ToggleBrowse()
				}
				g.live.Map.Close()
				g.live.Guide.Toggle()
			}
		case actMap:
			if g.chinese && g.live.Map != nil {
				if g.live.Journal.Browsing() {
					g.live.Journal.ToggleBrowse()
				}
				g.live.Guide.Close()
				g.live.Map.Toggle()
			}
		case actReveal:
			g.live.Map.ToggleReveal()
		case actCloseJournal:
			if g.live.Map.Open() {
				g.live.Map.Close()
			} else if g.live.Guide.Open() {
				g.live.Guide.Close()
			} else if g.live.Journal.Browsing() {
				g.live.Journal.ToggleBrowse()
			}
		case actPageUp, actPageDown, actPrevEntry, actNextEntry:
			d := map[hostAction]int{actPageUp: -1, actPageDown: 1, actPrevEntry: -1, actNextEntry: 1}[a]
			switch {
			case g.live.Map.Open() && (a == actPageUp || a == actPageDown):
				g.live.Map.TurnList(d)
			case g.live.Map.Open():
			case g.live.Guide.Open() && (a == actPageUp || a == actPageDown):
				g.live.Guide.TurnPage(d)
			case g.live.Guide.Open():
				g.live.Guide.NextChapter(d)
			case a == actPageUp || a == actPageDown:
				g.live.Journal.TurnPage(d)
			default:
				g.live.Journal.NextEntry(d)
			}
		}
	}
}

func (g *game) panelMode() panelMode {
	if g.chinese && (g.live.Guide.Open() || g.live.Map.Open()) {
		return panelBrowse // 攻略面板與手札翻閱同一組鍵（規格 008 §2.1）
	}
	switch j := g.live.Journal; {
	case !g.chinese || !j.Open():
		return panelNone
	case j.Browsing():
		return panelBrowse
	}
	return panelAuto
}

func (g *game) compose() []byte {
	return g.live.Compose(g.scale, g.chinese, func(r rune) {
		if !g.missing[r] {
			g.missing[r] = true
			fmt.Fprintf(os.Stderr, "cok-play: 字型缺字 %q（U+%04X）\n", r, r)
		}
	})
}

func (g *game) screenshot() {
	rgba := g.compose()
	if rgba == nil {
		return
	}
	img := &image.NRGBA{Pix: rgba, Stride: 4 * 320 * g.scale, Rect: image.Rect(0, 0, 320*g.scale, 200*g.scale)}
	name := fmt.Sprintf("cok-%s.png", time.Now().Format("20060102-150405.000"))
	err := os.MkdirAll(g.shotDir, 0o755)
	if err == nil {
		var f *os.File
		if f, err = os.Create(filepath.Join(g.shotDir, name)); err == nil {
			err = png.Encode(f, img)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cok-play: 截圖失敗：", err)
	}
}

func (g *game) Update() error {
	g.frame++
	var u session.CapturedUpdate
	var acts []hostAction
	for _, a := range g.script[g.frame] {
		if a.bar {
			g.bar.forced, g.bar.shown = true, true
		} else if a.toggle {
			acts = append(acts, actToggle)
		} else if a.lang {
			acts = append(acts, actLang)
		} else if a.cheat {
			acts = append(acts, actCheat)
		} else if a.key != nil {
			u.BIOSKeys = append(u.BIOSKeys, *a.key)
		}
	}
	if g.frames == 0 {
		acts = append(acts, g.hostInput(&u)...)
	}
	g.apply(acts)
	view, err := g.owner.View()
	if err != nil {
		return err
	}
	u.StartedPanel, u.Layout, u.SourceGeneration = view.Panel, view.Layout, view.SourceGeneration
	if _, err := g.owner.Deliver(u); err != nil {
		return err
	}
	tick, err := g.owner.Advance(session.InstructionBudget(g.budget))
	if err != nil {
		return err
	}
	g.samples = g.mix.Render(g.samples[:0], tick.MachineStepsBefore, tick.MachineStepsAfter)
	g.music.Mix(g.samples)
	if g.ring != nil {
		g.ring.Push(g.samples)
	}
	if tick.Phase == session.PhaseStopped {
		return ebiten.Termination
	}
	if g.frames > 0 && g.frame >= g.frames {
		return g.finish()
	}
	return nil
}

func (g *game) finish() error {
	write := func(path string, overlay bool) error {
		if path == "" {
			return nil
		}
		rgba := g.live.Compose(g.scale, overlay, nil)
		if rgba == nil {
			return errors.New("尚無畫格可截圖")
		}
		g.drawBar(rgba) // 腳本動作 bar 時含切換列（規格 011 §3 第 3 項）
		return os.WriteFile(path, rgba, 0o600)
	}
	if err := write(g.shot, true); err != nil {
		return err
	}
	if err := write(g.shotOrig, false); err != nil {
		return err
	}
	if g.live.BindErr != nil {
		fmt.Fprintln(os.Stderr, "cok-play:", g.live.BindErr)
	} else {
		fmt.Fprintf(os.Stderr, "cok-play: 攔截點綁定於 #%d：A %05X（%d 次）C %05X\n", g.live.BoundAt, g.live.A.Entry, len(g.live.A.Hits), g.live.C.Entry)
		if g.live.B != nil {
			fmt.Fprintf(os.Stderr, "cok-play: B %05X（%d 次）\n", g.live.B.Entry, len(g.live.B.Hits))
		}
		tally := func(name string, hs []cok.Hit) {
			c := map[string]int{}
			for _, h := range hs {
				r := h.Rejected
				if r == "" {
					r = "覆繪"
				}
				c[r]++
			}
			fmt.Fprintf(os.Stderr, "cok-play: %s %v\n", name, c)
		}
		tally("A", g.live.A.Hits)
		if g.live.B != nil {
			tally("B", g.live.B.Hits)
		}
		fmt.Fprintf(os.Stderr, "cok-play: Layer stamps=%d\n", len(g.live.Layer.Stamps))
	}
	if g.cheat.On {
		fmt.Fprintf(os.Stderr, "cok-play: 無敵 還原 %d 次；隊伍 %s\n", g.cheat.Restored, g.cheat.Report)
	}
	if d, err := g.owner.Digest(); err == nil {
		fmt.Fprintf(os.Stderr, "cok-play: frames=%d steps=%d memory_sha256=%x indexed_sha256=%x\n", g.frame, d.Steps, d.MemorySHA256, d.IndexedSHA256)
	}
	return ebiten.Termination
}

func (g *game) toolbarItems() []toolbarItem {
	return toolbarItems(g.barText, toolbarState{scale: g.scale, hasHD: g.live.HDItems > 0, chinese: g.chinese, muted: g.muted,
		hasMusic: g.music != nil, music: g.music.Enabled(),
		fullscreen: ebiten.IsFullscreen(), answers: !g.live.HideAnswers(),
		journal: g.live.Journal.Browsing(), guide: g.live.Guide.Open(), mapOpen: g.live.Map.Open(), langs: len(g.langs),
		packASCII: g.english(), cheat: g.cheat.On, noJournal: g.live.Journal == nil, noGuide: g.live.Guide == nil})
}

// drawBar 在切換列顯示時把它疊到 rgba 上（截圖 F10 不含切換列）。
func (g *game) drawBar(rgba []byte) {
	if rgba == nil || !g.bar.shown {
		return
	}
	items := g.toolbarItems()
	drawToolbar(rgba, g.scale, g.barFont(), items, toolbarHit(items, g.cursorX, g.cursorY), func(r rune) {
		if !g.missing[r] {
			g.missing[r] = true
			fmt.Fprintf(os.Stderr, "cok-play: 字型缺字 %q（U+%04X）\n", r, r)
		}
	})
}

func (g *game) Draw(screen *ebiten.Image) {
	rgba := g.compose()
	g.drawBar(rgba)
	if b := screen.Bounds(); rgba != nil && len(rgba) == 4*b.Dx()*b.Dy() {
		screen.WritePixels(rgba)
	}
}

func (g *game) Layout(_, _ int) (int, int) { return 320 * g.scale, 200 * g.scale }

// fail 是致命錯誤：stderr、錯誤紀錄（發行模式）與 Windows 對話框（launcher.go）。
func fail(err error) { fatal(err) }

func main() {
	original := flag.String("original", "", "唯讀原版目錄（含 START.EXE）")
	save := flag.String("save", "", "可寫存檔目錄：不存在或為空時由原版複製；已有 START.EXE（雜湊相符）則沿用，存檔保留")
	textDir := flag.String("text-dir", "", "譯文目錄（ui-text-*、ecl-text-* TSV）")
	fontPath := flag.String("font", "", "16×16 GOLEMFNT")
	frames := flag.Int("frames", 0, "自動模式：跑這麼多畫格後結束（0＝互動）")
	script := flag.String("script", "", "自動模式腳本：畫格:動作[,…]（鍵名、單一字元、toggle）")
	shot := flag.String("shot", "", "自動模式結束時輸出中文合成 RGBA")
	shotOrig := flag.String("shot-original", "", "自動模式結束時輸出原版合成 RGBA")
	shotDir := flag.String("shot-dir", "", "F10 截圖目錄（預設存檔目錄旁的 screenshots）")
	clock := flag.Int("clock", 50, "機器時脈比例 10–100")
	adlib := flag.Bool("adlib", true, "模擬 AdLib")
	audioOn := flag.Bool("audio", true, "互動模式播放聲音（容器內沒有音訊裝置時關閉）")
	hideAnswers := flag.Bool("hide-answers", false, "開機時隱藏查驗題答案（F9 切換）")
	original0 := flag.Bool("original-view", false, "開機時顯示原版（F12 切換）")
	invPath := flag.String("inventory", "", "結束時把印字路徑盤點（issue #11）寫成這個 TSV；含原文，只放工作區")
	scaleFlag := flag.Int("scale", 2, "起始倍率（2 或 3；F2 切換）")
	musicDir := flag.String("music-dir", "", "配樂目錄（music.tsv 與 <曲目>.opl；主專案規格 014）；選曲表讀 text-dir 的 music-cues.tsv")
	noMusic := flag.Bool("no-music", false, "開機時配樂關閉（切換列或 F4 可再打開）")
	langFlag := flag.String("lang", "", "起始語言代碼（主專案規格 015；預設為上次的選擇或 zh-TW）")
	langFonts := flag.String("lang-fonts", "", "繁中以外語言的字型：代碼=路徑，逗號分隔（發行模式另找 font/cok-<代碼>.golemfnt）")
	cheatOn := flag.Bool("cheat", false, "開機時開啟我方無敵（主專案規格 017；F1 或切換列切換）")
	hdDir := flag.String("hd", "", "HD 主題目錄（theme.json；主專案規格 010，只在 3× 顯示）")
	showVersion := flag.Bool("version", false, "印版號與 dosgolem commit 後結束")
	flag.Parse()
	if *showVersion {
		fmt.Print(versionString())
		return
	}
	// 沒給 -save 是發行模式（主專案規格 012 §3.1）：資料從發行包找，原版第一次匯入到使用者資料目錄的 game/。
	release := *save == ""
	stateDir := *save // 手札已讀、地圖走過、攻略位置等前端檔的目錄
	var gameDir, startName string
	if release {
		data, err := userDataDir()
		if err != nil {
			fail(err)
		}
		logDir, stateDir = data, data
		if *textDir == "" {
			if *textDir, err = resourcePath("text"); err != nil {
				fail(err)
			}
		}
		if *fontPath == "" {
			if *fontPath, err = releaseFont(); err != nil {
				fail(err)
			}
		}
		if *hdDir == "" {
			*hdDir = releaseHD()
		}
		if *musicDir == "" {
			*musicDir = releaseMusic()
		}
		if gameDir, startName, err = prepareGame(*original, data); err != nil {
			fail(err)
		}
	} else if *original == "" || *textDir == "" || *fontPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *clock < 10 || *clock > 100 {
		fail(fmt.Errorf("-clock 須在 10–100"))
	}
	if *scaleFlag != 2 && *scaleFlag != 3 {
		fail(fmt.Errorf("-scale 須為 2 或 3"))
	}
	font, err := xlate.LoadFont(*fontPath)
	if err != nil {
		fail(err)
	}
	font.Name = filepath.Base(*fontPath)
	ui, ecl, err := cok.LoadTextDir(*textDir)
	if err != nil {
		fail(err)
	}
	keys, err := parseScript(*script)
	if err != nil {
		fail(err)
	}
	sum, _ := hex.DecodeString(startEXESHA256)
	var want [32]byte
	copy(want[:], sum)
	saveRoot := *save
	if release {
		saveRoot = gameDir
	} else if _, err := os.Stat(filepath.Join(saveRoot, "START.EXE")); err != nil {
		startName = "START.EXE"
		out, err := bootroot.Prepare(bootroot.BootRootInput{OriginalRoot: *original, SaveRoot: *save,
			Required: []bootroot.RequiredFile{{Name: "START.EXE", SHA256: want}}})
		if err != nil {
			fail(err)
		}
		saveRoot = out.SaveRoot
	}
	if startName == "" {
		startName = "START.EXE"
	}
	if err := installCoK(saveRoot); err != nil {
		fail(err)
	}
	exe, err := os.ReadFile(filepath.Join(saveRoot, startName))
	if err != nil {
		fail(err)
	}
	if got := sha256.Sum256(exe); got != want {
		fail(errors.New("START.EXE 雜湊不符"))
	}
	live := cok.NewLive(font, ui, ecl)
	if *hdDir != "" {
		if err := live.AddHD(*hdDir); err != nil {
			fail(err)
		}
	}
	live.Inventory = *invPath != ""
	if live.Answers, live.QuizFormat, err = cok.LoadQuizDir(*textDir); err != nil {
		fail(err)
	}
	live.SetHideAnswers(*hideAnswers)
	hostText, err := os.ReadFile(filepath.Join(*textDir, "host-text.zh-TW.tsv"))
	if err != nil {
		fail(err)
	}
	barText, err := cok.LoadHostText(hostText)
	if err != nil {
		fail(err)
	}
	if err := checkToolbarText(barText); err != nil {
		fail(err)
	}
	if live.Journal, err = cok.LoadJournalDir(*textDir); err != nil {
		fail(err)
	}
	if live.Guide, err = cok.LoadGuideDir(*textDir); err != nil {
		fail(err)
	}
	if live.Guide != nil {
		posPath := filepath.Join(stateDir, "cok-walkthrough-pos.txt")
		ch, page, err := cok.ReadGuidePos(posPath)
		if err != nil {
			fail(err)
		}
		if ch != 0 || page != 0 {
			live.Guide.SetPos(ch, page)
		}
		live.Guide.OnMove = func(ch, page int) {
			if err := cok.WriteGuidePos(posPath, ch, page); err != nil {
				fmt.Fprintln(os.Stderr, "cok-play: 攻略位置寫入失敗：", err)
			}
		}
	}
	if live.Map, err = cok.LoadAutoMapDir(*textDir); err != nil {
		fail(err)
	}
	if cues, err := os.ReadFile(filepath.Join(*textDir, "music-cues.tsv")); err == nil {
		if live.Music, err = cok.LoadMusicCues(cues); err != nil {
			fail(err)
		}
	} else if !os.IsNotExist(err) {
		fail(err)
	}
	if live.Map != nil {
		visitedPath := filepath.Join(stateDir, "cok-map-visited.txt")
		v, err := cok.ReadVisited(visitedPath)
		if err != nil {
			fail(err)
		}
		live.Map.SetVisited(v)
		live.Map.OnVisit = func(key string, x, y int) {
			if err := cok.AppendVisited(visitedPath, key, x, y); err != nil {
				fmt.Fprintln(os.Stderr, "cok-play: 地圖走過格子寫入失敗：", err)
			}
		}
	}
	seenPath := filepath.Join(stateDir, "cok-journal-seen.txt")
	if live.Journal.Seen, err = cok.ReadJournalSeen(seenPath); err != nil {
		fail(err)
	}
	live.Journal.OnSeen = func(n int) {
		if err := cok.AppendJournalSeen(seenPath, n); err != nil {
			fmt.Fprintln(os.Stderr, "cok-play: 已讀手札寫入失敗：", err)
		}
	}
	s := host.OutputScale2
	layout := host.MouseLayout{Epoch: 1, Scale: s, ChromeHeight: 18 * int(s), Canvas: host.Canvas{Width: 320, Height: 200},
		FrameWidth: 320 * int(s), FrameHeight: 18*int(s) + 200*int(s)}
	budget := stepsPerHostFrame(*clock)
	fontsByLang, err := parseLangFonts(*langFonts)
	if err != nil {
		fail(err)
	}
	langs, err := loadLangs(*textDir, font, fontsByLang, release)
	if err != nil {
		fail(err)
	}
	langIdx, err := pickLang(langs, *langFlag, stateDir)
	if err != nil {
		fail(err)
	}
	live.SetLang(langs[langIdx])
	barText = langs[langIdx].Host
	if len(langs) > 1 {
		hostKeysReserved["F8"] = actLang
	}
	mix := mixer.New(budget * 60)
	cheat := &cok.Invincible{On: *cheatOn}
	owner, err := session.New(session.Config{InitialScale: s, InitialLayout: layout, AdLib: *adlib, ClockPercent: *clock,
		Observer: observer{live: live, mix: mix, cheat: cheat}})
	if err != nil {
		fail(err)
	}
	defer owner.Close()
	if _, err := owner.BootOriginal(session.BootInput{EXE: exe, ExpectedEXESHA256: want, SaveRoot: saveRoot}); err != nil {
		fail(err)
	}
	dir := *shotDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(filepath.Clean(saveRoot)), "screenshots")
	}
	g := &game{owner: owner, live: live, scale: *scaleFlag, chinese: !*original0, frames: *frames, script: keys, shot: *shot, shotOrig: *shotOrig,
		shotDir: dir, budget: budget, mix: mix, focused: true, missing: map[rune]bool{}, barText: barText,
		langs: langs, lang: langIdx, langDir: stateDir, mainFont: font, cheat: cheat}
	if *musicDir != "" {
		if g.music, err = bgm.Load(*musicDir, live.Music); err != nil {
			fail(err)
		}
		if *noMusic {
			g.music.Toggle()
		}
	}
	if *frames == 0 && *audioOn {
		g.ring = mixer.NewRing(200)
		player, err := audio.NewContext(mixer.SampleRate).NewPlayerF32(g.ring)
		if err != nil {
			fail(err)
		}
		player.SetBufferSize(50 * time.Millisecond)
		player.Play()
		g.player = player
	}
	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowSize(320*g.scale, 200*g.scale)
	ebiten.SetTPS(60)
	clearErrorLog()
	err = ebiten.RunGame(g)
	if live.Inv != nil {
		ovrName, _ := findFold(saveRoot, "GAME.OVR")
		ovr, _ := os.ReadFile(filepath.Join(saveRoot, ovrName))
		if werr := live.Inv.WriteTSV(*invPath, ovr); werr != nil {
			fmt.Fprintln(os.Stderr, "cok-play: 盤點寫入失敗：", werr)
		} else {
			fmt.Fprintf(os.Stderr, "cok-play: 盤點 %d 筆寫到 %s\n", len(live.Inv.Events), *invPath)
		}
	}
	if err != nil {
		fail(err)
	}
}
