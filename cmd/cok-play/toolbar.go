package main

// 視窗頂端切換列（death-knights-of-krynn-cht 規格 011）：平常隱藏，游標移到頂端時疊在畫面上方出現。
// 每個項目送出的前端動作與對應的功能鍵相同（F2、F12、F5…），由 game.apply 處理。這一層不認識 ebiten。

import (
	"fmt"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/xlate"
)

const (
	barH        = 12 // 切換列高度（邏輯像素）
	barReveal   = 3  // 游標 y 小於此值時出現
	barTextY    = 2  // 文字列起點
	barFirstCol = 1
)

type itemState int

const (
	itemEnabled  itemState = iota
	itemActive             // 目前狀態（例如目前倍率），已在這個狀態時按下不動作
	itemDisabled           // 灰色，不能按
)

// toolbarItem 是切換列的一段文字；col、cols 以 8 邏輯像素為一格。
type toolbarItem struct {
	key   string
	label []rune
	act   hostAction
	state itemState
	col   int
	cols  int
}

// toolbarState 是決定項目文字與狀態的前端狀態。
type toolbarState struct {
	scale                             int
	hasHD, chinese, muted, fullscreen bool
	hasMusic, music                   bool // 有配樂素材、配樂開啟中（規格 014）
	answers                           bool // 查驗題答案顯示中
	cheat                             bool // 我方無敵開啟中（主專案規格 017）
	journal, guide, mapOpen           bool
	langs                             int  // 已載入的語言數（規格 015）；兩種以上才顯示語言項
	packASCII                         bool // 英文版：切換列以預設字型畫，ASCII 也兩字併一格
	noJournal, noGuide                bool // 目前語言沒有手札、攻略（規格 015），項目停用
}

// toolbarItems 依規格 011 §2.2 排出項目，項目間空一格。text 為 host-text 的 toolbar.* 鍵。
func toolbarItems(text map[string]string, st toolbarState) []toolbarItem {
	on := func(b bool) itemState {
		if b {
			return itemActive
		}
		return itemEnabled
	}
	panel := func(open bool) itemState {
		if !st.chinese {
			return itemDisabled // 原版顯示時 F5／F6／F7 不動作
		}
		return on(open)
	}
	scale3 := "toolbar.scale3"
	if st.hasHD {
		scale3 = "toolbar.scale3hd"
	}
	lang, langAct := "toolbar.original", actToggle
	if st.chinese {
		lang = "toolbar.chinese"
	}
	if st.langs > 1 { // 規格 015：語言項顯示目前語言名稱，按下依序換下一種語言，最後一種之後回原版
		langAct = actLangCycle
		if st.chinese {
			lang = "lang.name"
		}
	}
	items := []toolbarItem{
		{key: "toolbar.scale2", act: actScale, state: on(st.scale == 2)},
		{key: scale3, act: actScale, state: on(st.scale == 3)},
	}
	if !st.hasHD {
		items = append(items, toolbarItem{key: "toolbar.nohd", state: itemDisabled})
	}
	items = append(items,
		toolbarItem{key: lang, act: langAct, state: on(st.chinese)},
		toolbarItem{key: "toolbar.journal", act: actJournal, state: absent(st.noJournal, panel(st.journal))},
		toolbarItem{key: "toolbar.guide", act: actGuide, state: absent(st.noGuide, panel(st.guide))},
		toolbarItem{key: "toolbar.map", act: actMap, state: panel(st.mapOpen)},
		toolbarItem{key: "toolbar.answers", act: actAnswers, state: on(st.answers)},
		toolbarItem{key: "toolbar.cheat", act: actCheat, state: on(st.cheat)},
		toolbarItem{key: "toolbar.music", act: actMusic, state: musicState(st)},
		toolbarItem{key: "toolbar.mute", act: actMute, state: on(st.muted)},
		toolbarItem{key: "toolbar.shot", act: actShot},
		toolbarItem{key: "toolbar.fullscreen", act: actFullscreen, state: on(st.fullscreen)},
	)
	col := barFirstCol
	for i := range items {
		items[i].label = cok.PackHalfWidth([]rune(text[items[i].key])) // 半形兩字併一格（規格 005 §2；#18）
		if st.packASCII {
			items[i].label = cok.PackHalfWidthAll([]rune(text[items[i].key]))
		}
		items[i].col, items[i].cols = col, len(items[i].label)
		col += items[i].cols + 1
	}
	return items
}

// toolbarHit 回邏輯像素 (x, y) 所在的項目索引，不在任何項目上回 -1。
func toolbarHit(items []toolbarItem, x, y int) int {
	if y < 0 || y >= barH {
		return -1
	}
	for i, it := range items {
		if x >= it.col*8 && x < (it.col+it.cols)*8 {
			return i
		}
	}
	return -1
}

// toolbarClick 回點擊 (x, y) 要送出的動作；停用、已是目前倍率或不在項目上回 0。
func toolbarClick(items []toolbarItem, x, y int) hostAction {
	i := toolbarHit(items, x, y)
	if i < 0 || items[i].state == itemDisabled || items[i].state == itemActive && items[i].act == actScale {
		return 0
	}
	return items[i].act
}

// toolbar 是切換列的顯示狀態。forced 由自動模式腳本的 bar 動作設定。
type toolbar struct {
	shown, forced bool
}

// track 依游標邏輯位置更新顯示：頂端 barReveal 列內出現，離開切換列範圍收起。
func (t *toolbar) track(x, y int) {
	inside := x >= 0 && x < 320 && y >= 0
	switch {
	case t.forced:
		t.shown = true
	case !t.shown:
		t.shown = inside && y < barReveal
	default:
		t.shown = inside && y < barH
	}
}

// takesPointer 回切換列是否吃掉 (x, y) 的滑鼠按下（顯示中且在切換列範圍內）。
func (t *toolbar) takesPointer(x, y int) bool {
	return t.shown && x >= 0 && x < 320 && y >= 0 && y < barH
}

var (
	barBG       = [3]uint8{0, 0, 0}
	barLine     = [3]uint8{170, 170, 170}
	barText     = [3]uint8{255, 255, 255}
	barHover    = [3]uint8{255, 255, 85}
	barActive   = [3]uint8{85, 255, 255}
	barDisabled = [3]uint8{110, 110, 110}
)

// drawToolbar 把切換列畫進 scale 倍的 RGBA（320×200 畫面）；hover 為游標所在項目（-1 為無）。
func drawToolbar(dst []uint8, scale int, font *xlate.Font, items []toolbarItem, hover int, missing func(rune)) {
	if font == nil || scale <= 0 || len(dst) != 4*320*scale*200*scale {
		return
	}
	w := 320 * scale
	for y := 0; y < barH*scale; y++ {
		c := barBG
		if y >= (barH-1)*scale {
			c = barLine
		}
		for x := 0; x < w; x++ {
			setRGB(dst, w, x, y, c)
		}
	}
	f, k, ox, oy := font.ForScale(scale) // 3× 用 22×22 變體（apps/cok AddScaledVariants）
	rb := (f.W + 7) / 8
	for i, it := range items {
		c := barText
		switch {
		case it.state == itemDisabled:
			c = barDisabled
		case i == hover && toolbarClick(items, it.col*8, barTextY) != 0:
			c = barHover
		case it.state == itemActive:
			c = barActive
		}
		for n, r := range it.label {
			if r == ' ' {
				continue
			}
			g, ok := f.Glyphs[r]
			if !ok {
				if missing != nil {
					missing(r)
				}
				continue
			}
			cx, cy := (it.col+n)*8*scale, barTextY*scale
			for gy := 0; gy < f.H; gy++ {
				for gx := 0; gx < f.W; gx++ {
					if g[gy*rb+gx/8]&(0x80>>uint(gx%8)) == 0 {
						continue
					}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							if lx, ly := ox+gx*k+dx, oy+gy*k+dy; lx < 8*scale && ly < 8*scale {
								setRGB(dst, w, cx+lx, cy+ly, c)
							}
						}
					}
				}
			}
		}
	}
}

func setRGB(dst []uint8, w, x, y int, c [3]uint8) {
	i := 4 * (y*w + x)
	if x < 0 || x >= w || i < 0 || i+3 >= len(dst) {
		return
	}
	dst[i], dst[i+1], dst[i+2], dst[i+3] = c[0], c[1], c[2], 0xFF
}

// toolbarKeys 是切換列用到的全部 host-text 鍵。
var toolbarKeys = []string{"toolbar.scale2", "toolbar.scale3", "toolbar.scale3hd", "toolbar.nohd", "toolbar.chinese",
	"toolbar.original", "toolbar.journal", "toolbar.guide", "toolbar.map", "toolbar.answers", "toolbar.cheat", "toolbar.mute", "toolbar.music",
	"toolbar.shot", "toolbar.fullscreen"}

// checkToolbarText 確認 host-text 有全部切換列文字。
func checkToolbarText(text map[string]string) error {
	for _, k := range toolbarKeys {
		if text[k] == "" {
			return fmt.Errorf("host-text.zh-TW.tsv 缺 %s", k)
		}
	}
	return nil
}

// musicState：沒有配樂素材時停用；開啟中為目前狀態，但與倍率不同，按下仍會切換（toolbarClick 只擋倍率）。
func musicState(st toolbarState) itemState {
	switch {
	case !st.hasMusic:
		return itemDisabled
	case st.music:
		return itemActive
	}
	return itemEnabled
}

// absent：目前語言沒有這個面板時停用。
func absent(none bool, st itemState) itemState {
	if none {
		return itemDisabled
	}
	return st
}
