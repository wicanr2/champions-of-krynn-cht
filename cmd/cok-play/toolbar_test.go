package main

import (
	"testing"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/xlate"
)

// 主專案 text/host-text.zh-TW.tsv 的 toolbar.* 文字（規格 011 §2.2）。
var testBarText = map[string]string{
	"toolbar.scale2": "2倍", "toolbar.scale3": "3倍", "toolbar.scale3hd": "3倍HD", "toolbar.nohd": "無HD素材",
	"toolbar.chinese": "中文", "toolbar.original": "原版", "toolbar.journal": "手札", "toolbar.guide": "攻略",
	"toolbar.map": "地圖", "toolbar.answers": "答案", "toolbar.mute": "靜音", "toolbar.music": "配樂", "toolbar.shot": "截圖",
	"toolbar.fullscreen": "全螢幕",
}

func TestToolbarTextComplete(t *testing.T) {
	if err := checkToolbarText(testBarText); err != nil {
		t.Fatal(err)
	}
	if err := checkToolbarText(map[string]string{}); err == nil {
		t.Fatal("缺文字時應回錯誤")
	}
}

func TestToolbarLayoutFits(t *testing.T) {
	for _, hd := range []bool{false, true} {
		items := toolbarItems(testBarText, toolbarState{scale: 2, hasHD: hd, chinese: true})
		end := 0
		for _, it := range items {
			if it.col < end {
				t.Fatalf("hd=%v：%s 與前一項重疊", hd, it.key)
			}
			if it.cols != len(cok.PackHalfWidth([]rune(testBarText[it.key]))) || it.cols == 0 {
				t.Fatalf("hd=%v：%s 寬 %d 格", hd, it.key, it.cols)
			}
			end = it.col + it.cols + 1
		}
		if last := items[len(items)-1]; (last.col+last.cols)*8 > 320 {
			t.Fatalf("hd=%v：切換列寬 %d 超過 320", hd, (last.col+last.cols)*8)
		}
	}
}

func find(items []toolbarItem, key string) (toolbarItem, bool) {
	for _, it := range items {
		if it.key == key {
			return it, true
		}
	}
	return toolbarItem{}, false
}

func TestToolbarHDGate(t *testing.T) {
	no := toolbarItems(testBarText, toolbarState{scale: 2, chinese: true})
	if _, ok := find(no, "toolbar.scale3hd"); ok {
		t.Fatal("沒有 HD 素材時不得出現 3倍HD")
	}
	nohd, ok := find(no, "toolbar.nohd")
	if !ok || nohd.state != itemDisabled || nohd.act != 0 {
		t.Fatalf("沒有 HD 素材時應有停用的說明項：%+v", nohd)
	}
	if a := toolbarClick(no, nohd.col*8, barTextY); a != 0 {
		t.Fatalf("停用項按下送出 %v", a)
	}
	yes := toolbarItems(testBarText, toolbarState{scale: 2, hasHD: true, chinese: true})
	if _, ok := find(yes, "toolbar.nohd"); ok {
		t.Fatal("有 HD 素材時不顯示停用說明")
	}
	s3, ok := find(yes, "toolbar.scale3hd")
	if !ok || toolbarClick(yes, s3.col*8, barTextY) != actScale {
		t.Fatal("有 HD 素材時 3倍HD 應送出 F2 同一動作")
	}
}

func TestToolbarClickSameAsKeys(t *testing.T) {
	st := toolbarState{scale: 3, hasHD: true, chinese: true}
	items := toolbarItems(testBarText, st)
	want := map[string]hostAction{"toolbar.scale2": actScale, "toolbar.chinese": actToggle, "toolbar.journal": actJournal,
		"toolbar.guide": actGuide, "toolbar.map": actMap, "toolbar.answers": actAnswers, "toolbar.mute": actMute,
		"toolbar.shot": actShot, "toolbar.fullscreen": actFullscreen}
	for k, a := range want {
		it, ok := find(items, k)
		if !ok {
			t.Fatalf("缺 %s", k)
		}
		// 對應鍵與 hostKeysReserved 相同
		found := false
		for _, ka := range hostKeysReserved {
			found = found || ka == a
		}
		if !found {
			t.Fatalf("%s 的動作沒有對應鍵", k)
		}
		if got := toolbarClick(items, it.col*8+3, barH-2); got != a {
			t.Fatalf("%s 送出 %v，要 %v", k, got, a)
		}
	}
	// 目前倍率按下不動作
	s3, _ := find(items, "toolbar.scale3hd")
	if a := toolbarClick(items, s3.col*8, barTextY); a != 0 {
		t.Fatalf("已在 3× 按 3倍HD 送出 %v", a)
	}
	// 切換列以外或項目間空格不命中
	if toolbarClick(items, 0, barTextY) != 0 || toolbarClick(items, items[1].col*8+1, barH) != 0 {
		t.Fatal("範圍外不得命中")
	}
}

func TestToolbarPanelsDisabledInOriginal(t *testing.T) {
	items := toolbarItems(testBarText, toolbarState{scale: 2})
	for _, k := range []string{"toolbar.journal", "toolbar.guide", "toolbar.map"} {
		if it, _ := find(items, k); it.state != itemDisabled {
			t.Fatalf("原版顯示時 %s 應停用", k)
		}
	}
	if _, ok := find(items, "toolbar.original"); !ok {
		t.Fatal("原版顯示時語言項應寫原版")
	}
}

func TestToolbarReveal(t *testing.T) {
	var b toolbar
	steps := []struct {
		x, y  int
		shown bool
	}{
		{100, 100, false}, {100, 5, false}, // 隱藏時游標在切換列高度內但不在頂端：不出現
		{100, 2, true}, {200, 10, true}, // 頂端出現，切換列內維持
		{200, 12, false}, // 離開收起
		{-1, 1, false},   // 視窗左外
		{50, 0, true}, {320, 0, false},
	}
	for i, s := range steps {
		b.track(s.x, s.y)
		if b.shown != s.shown {
			t.Fatalf("第 %d 步 (%d,%d)：shown=%v", i, s.x, s.y, b.shown)
		}
	}
	b.forced = true
	b.track(100, 150)
	if !b.shown {
		t.Fatal("bar 動作後應保持顯示")
	}
}

func TestToolbarTakesPointer(t *testing.T) {
	b := toolbar{shown: true}
	if !b.takesPointer(5, 0) || !b.takesPointer(319, barH-1) || b.takesPointer(5, barH) {
		t.Fatal("切換列顯示時只吃範圍內的點擊")
	}
	b.shown = false
	if b.takesPointer(5, 0) {
		t.Fatal("隱藏時不吃點擊")
	}
}

func TestDrawToolbarOnlyTopRows(t *testing.T) {
	font := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	full := make([]byte, 32)
	for i := range full {
		full[i] = 0xFF
	}
	for _, s := range testBarText {
		for _, r := range cok.PackHalfWidth([]rune(s)) {
			font.Glyphs[r] = full
		}
	}
	for _, scale := range []int{2, 3} {
		dst := make([]byte, 4*320*scale*200*scale)
		for i := range dst {
			dst[i] = 7
		}
		missing := 0
		drawToolbar(dst, scale, font, toolbarItems(testBarText, toolbarState{scale: scale, chinese: true}), -1,
			func(rune) { missing++ })
		if missing != 0 {
			t.Fatalf("%d×：缺字 %d", scale, missing)
		}
		w := 320 * scale
		for y := 0; y < 200*scale; y++ {
			for x := 0; x < w; x++ {
				untouched := dst[4*(y*w+x)] == 7
				if y >= barH*scale && !untouched {
					t.Fatalf("%d×：切換列外 (%d,%d) 被改", scale, x, y)
				}
				if y < barH*scale && untouched {
					t.Fatalf("%d×：切換列內 (%d,%d) 沒畫", scale, x, y)
				}
			}
		}
	}
}

func TestToolbarHalfWidthPacked(t *testing.T) {
	items := toolbarItems(testBarText, toolbarState{scale: 2})
	it, _ := find(items, "toolbar.nohd")
	if it.cols != 4 { // 無、HD（併一格）、素、材
		t.Fatalf("無HD素材佔 %d 格，要 4", it.cols)
	}
}

func TestToolbarLangItem(t *testing.T) {
	one := toolbarItems(testBarText, toolbarState{scale: 2, chinese: true, langs: 1})
	if it, _ := find(one, "toolbar.chinese"); it.act != actToggle {
		t.Fatal("只有一種語言時語言項維持中文／原版切換")
	}
	for _, name := range []string{"繁中", "简中", "日本", "한국", "EN"} {
		text := map[string]string{"lang.name": name}
		for k, v := range testBarText {
			text[k] = v
		}
		items := toolbarItems(text, toolbarState{scale: 2, chinese: true, langs: 5})
		it, ok := find(items, "lang.name")
		if !ok || toolbarClick(items, it.col*8, barTextY) != actLangCycle {
			t.Fatalf("%s：語言項要顯示語言名稱並送出 actLangCycle", name)
		}
		if last := items[len(items)-1]; (last.col+last.cols)*8 > 320 {
			t.Fatalf("%s：切換列寬 %d 超過 320", name, (last.col+last.cols)*8)
		}
		orig := toolbarItems(text, toolbarState{scale: 2, langs: 5})
		if it, ok := find(orig, "toolbar.original"); !ok || it.act != actLangCycle {
			t.Fatalf("%s：原版時語言項寫原版", name)
		}
	}
}

func TestParseLangFontsAndPick(t *testing.T) {
	m, err := parseLangFonts("zh-CN=/f/cn.golemfnt, ja=/f/ja.golemfnt")
	if err != nil || m["zh-CN"] != "/f/cn.golemfnt" || m["ja"] != "/f/ja.golemfnt" {
		t.Fatalf("%v %v", m, err)
	}
	if _, err := parseLangFonts("ja"); err == nil {
		t.Fatal("缺路徑要報錯")
	}
	packs := []*cok.LangPack{{Code: "zh-TW"}, {Code: "ja"}}
	dir := t.TempDir()
	defer func(f func() string) { systemLocale = f }(systemLocale)
	systemLocale = func() string { return "" }
	if i, _ := pickLang(packs, "", dir); i != 0 {
		t.Fatal("沒有選擇時用第一種")
	}
	systemLocale = func() string { return "ja_JP.UTF-8" }
	if i, _ := pickLang(packs, "", dir); i != 1 {
		t.Fatal("第一次執行依系統語言")
	}
	systemLocale = func() string { return "ko_KR" }
	if i, _ := pickLang(packs, "", dir); i != 0 {
		t.Fatal("系統語言沒有對應的語言包時用第一種")
	}
	g := &game{langs: packs, langDir: dir, live: cok.NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)}
	packs[1].Host = map[string]string{"lang.name": "日本語"}
	packs[1].UI = &cok.Translator{}
	g.chinese = true
	g.nextLang()
	if g.lang != 1 || g.barText["lang.name"] != "日本語" || g.live.Lang() != packs[1] {
		t.Fatal("換語言要更新語言包與切換列文字")
	}
	systemLocale = func() string { return "zh_TW" }
	if i, _ := pickLang(packs, "", dir); i != 1 {
		t.Fatal("記住的選擇優先於系統語言")
	}
	g.cycleLang()
	if g.chinese || g.lang != 1 {
		t.Fatal("最後一種語言之後回原版")
	}
	packs[0].UI = &cok.Translator{}
	g.cycleLang()
	if !g.chinese || g.lang != 0 || g.live.Lang() != packs[0] {
		t.Fatal("原版之後回第一種語言")
	}
	if _, err := pickLang(packs, "ko", dir); err == nil {
		t.Fatal("-lang 指定不存在的語言要報錯")
	}
}

func TestToolbarPanelAbsentDisabled(t *testing.T) {
	items := toolbarItems(testBarText, toolbarState{scale: 2, chinese: true, noGuide: true})
	if it, _ := find(items, "toolbar.guide"); it.state != itemDisabled || toolbarClick(items, it.col*8, barTextY) != 0 {
		t.Fatal("語言沒有攻略時攻略項停用")
	}
	if it, _ := find(items, "toolbar.journal"); it.state == itemDisabled {
		t.Fatal("手札仍可用")
	}
}

func TestLocaleLang(t *testing.T) {
	for tag, want := range map[string]string{
		"zh_TW.UTF-8": "zh-TW", "zh-Hant-TW": "zh-TW", "zh_HK": "zh-TW", "zh": "zh-TW",
		"zh_CN.UTF-8": "zh-CN", "zh-Hans-CN": "zh-CN", "zh-Hans": "zh-CN", "zh_SG": "zh-CN",
		"ja_JP.UTF-8": "ja", "ja": "ja", "ko-KR": "ko", "en_US.UTF-8": "en", "fr_FR": "en",
		"C": "", "POSIX": "", "C.UTF-8": "", "": "",
	} {
		if got := localeLang(tag); got != want {
			t.Errorf("localeLang(%q) = %q，要 %q", tag, got, want)
		}
	}
}
