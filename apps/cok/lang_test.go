package cok

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

func TestPathARelangRetranslatesAndDropsOverlong(t *testing.T) {
	m := make(fakeMem, 0x100000)
	plantSignature(m, 0x62AA, 0)
	layer := &xlate.Layer{}
	font := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	p, err := NewPathA(m, catalogOf(t, "Human", "人類", "Exit", "離開"), font, layer)
	if err != nil {
		t.Fatal(err)
	}
	printA(p, m, 1, "  Human ", 8, 6)
	printA(p, m, 2, "Exit", 10, 6)
	if len(layer.Stamps) != 2 {
		t.Fatalf("前置：%d 筆", len(layer.Stamps))
	}
	font2 := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	p.relang(catalogOf(t, "Human", "人类", "Exit", "离开这个画面"), font2, QuizFormat{})
	r := quizRows(layer)
	if r[8] != "人类" || r[10] != "" {
		t.Fatalf("重譯結果 %q；放不下的要移除：%v", r[8], r)
	}
	if len(layer.Stamps) != 1 || layer.Stamps[0].Font != font2 {
		t.Fatal("留下的 Stamp 要換成新字型")
	}
}

func TestPathBRelangRelaysOutPagesAndKeepsRemovedRows(t *testing.T) {
	b, m, l := newB(t, catalogOf(t, "EVEN AFTER ", "即使已擊敗麥坦尼，終結了", "THE THREAT.", "龍人的威脅。"))
	bCallSetup(m, "EVEN AFTER ", 1, story, 0, 0)
	call(b, m, 1)
	ret(b, m)
	bCallSetup(m, "THE THREAT.", 0, story, 17, 12)
	call(b, m, 2)
	ret(b, m)
	b.VideoWrite(uint32(22*8*planarStride + 3)) // 原版蓋掉底列
	n := len(l.Stamps)
	font2 := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	b.relang(font2, JournalRef{Format: " %d%s", Dot: "."}, catalogOf(t, "EVEN AFTER ", "Even after ", "THE THREAT.", "the threat."))
	if got := rowText(l, 17); got != string(PackHalfWidth([]rune("Even after the threat."))) {
		t.Fatalf("重排第 17 列 %q", got)
	}
	if len(l.Stamps) != n || rowText(l, 22) != "" {
		t.Fatalf("被原版蓋掉的列要維持移除：%d → %d", n, len(l.Stamps))
	}
	for _, s := range l.Stamps {
		if s.Font != font2 {
			t.Fatal("Stamp 要換成新字型")
		}
	}
	b.relang(font2, DefaultJournalRef, catalogOf(t, "EVEN AFTER ", strings.Repeat("長", 300)))
	if len(l.Stamps) != 0 || len(b.pages) != 0 {
		t.Fatal("新語言排不下的頁要整頁移除")
	}
}

func TestPathBJournalRefUsesLanguageFormat(t *testing.T) {
	b, m, l := newB(t, catalogOf(t, "READ JOURNAL ENTRY ", "請讀手札第"))
	bCallSetup(m, "READ JOURNAL ENTRY ", 1, story, 0, 0)
	call(b, m, 1)
	ret(b, m)
	bCallSetup(m, "51.", 0, story, 17, 20)
	call(b, m, 2)
	if got := rowText(l, 17); got != string(PackHalfWidth([]rune("請讀手札第 51 則。"))) {
		t.Fatalf("繁中 %q", got)
	}
	b.relang(b.Font, JournalRef{Format: " %d%s", Dot: "."}, catalogOf(t, "READ JOURNAL ENTRY ", "Read journal entry"))
	if got := rowText(l, 17); got != string(PackHalfWidth([]rune("Read journal entry 51."))) {
		t.Fatalf("換語言 %q", got)
	}
}

func TestPathCRelangDoesNotFireOnLine(t *testing.T) {
	c, m, l := newC(t, catalogOf(t, "Select", "選擇", "Exit", "離開"))
	lines := 0
	c.OnLine = func(int, string) { lines++ }
	drawWord(c, m, 24, 0, "Select", 0, false)
	c.SetPalette(c.pal)
	before := lines
	c.relang(catalogOf(t, "Select", "选择"), c.Font)
	if lines != before {
		t.Fatal("換語言不觸發 OnLine（配樂不換曲）")
	}
	if got := rowText(l, 24); !strings.HasPrefix(got, "选择") {
		t.Fatalf("重組 %q", got)
	}
	if c.OnLine == nil {
		t.Fatal("OnLine 要還原")
	}
}

func TestQuizRelangRewritesAllRows(t *testing.T) {
	p, m, l := newQuizA(t)
	askQuiz(p, m, "4    ")
	host, err := LoadHostText([]byte("key\ttranslation\tnote\n" +
		"quiz.entry\t旅行手札第 {entry} 则\t\nquiz.word\t（英文手册第 {page} 页）的第 {word} 个字？\t\nquiz.answer\t答案：{answer}\t\n"))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := QuizFormatOf(host)
	p.relang(p.Catalog, p.Font, f)
	r := quizRows(l)
	if r[2] != string(PackHalfWidth([]rune("旅行手札第 60 则"))) || r[4] != string(PackHalfWidth([]rune("（英文手册第 49 页）的第 4 个字？"))) {
		t.Fatalf("查驗題要以新格式重寫：%v", r)
	}
}

func TestTranslatorRelearnPlayers(t *testing.T) {
	old := &Translator{Players: &PlayerNames{Translit: func(s string, _ Sex) (string, bool) { return "舊" + s, true }}}
	old.LearnPlayer([]byte(" ANNA "), SexFemale)
	nw := &Translator{Players: &PlayerNames{Translit: func(s string, _ Sex) (string, bool) { return "新" + s, true }}}
	nw.Relearn(old)
	if e, ok := nw.Players.Lookup([]byte("ANNA")); !ok || e.Translation != "新ANNA" {
		t.Fatalf("新語言要重學隊員名：%+v %v", e, ok)
	}
}

func TestLangFilesAndAvailableLangs(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"host-text.zh-TW.tsv", "host-text.ja.tsv", "host-text.zh-CN.tsv", "ui-text-events.tsv", "composed-templates.tsv"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := AvailableLangs(dir)
	if err != nil || !reflect.DeepEqual(got, []string{"zh-TW", "ja", "zh-CN"}) {
		t.Fatalf("語言清單 %v %v", got, err)
	}
	if n := langShared(dir, "composed-templates", "zh-TW"); n != "composed-templates.tsv" {
		t.Fatalf("繁中沿用無後綴檔：%s", n)
	}
	if n := langShared(dir, "composed-templates", "ja"); n != "composed-templates.ja.tsv" {
		t.Fatalf("其他語言要用自己的檔：%s", n)
	}
}

// TestLoadLangPackRealText 以主專案的正式譯文目錄載入繁中語言包（DOSGOLEM_COK_TEXT_DIR 未設定時略過）。
func TestLoadLangPackRealText(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_COK_TEXT_DIR")
	if dir == "" {
		t.Skip("DOSGOLEM_COK_TEXT_DIR 未設定")
	}
	p, err := LoadLangPack(dir, DefaultLang, &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Ref != DefaultJournalRef || p.Journal == nil || p.Guide == nil || p.Map == nil || p.Name() == "" {
		t.Fatalf("繁中語言包不完整：%+v", p.Ref)
	}
}

const enHost = "key\ttranslation\tnote\n" +
	"quiz.entry\tJournal entry {entry}\t\nquiz.word\t(page {page}) word {word}?\t\nquiz.answer\tAnswer: {answer}\t\n" +
	"journal.ref\t%d%s\t\njournal.refdot\t.\t\nlang.name\tEN\t\n"

func TestEnglishPackIsIdentityAndHidesMissingPanels(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "host-text.en.tsv"), []byte(enHost), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "quiz-answers.tsv"), []byte("page\tentry\tword\tanswer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	en, err := LoadLangPack(dir, EnglishLang, &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}})
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := en.UI.Lookup([]byte("View Take Pool Exit")); !ok || e.Translation != "View Take Pool Exit" {
		t.Fatalf("英文原文即譯文：%+v", e)
	}
	if en.ECL == nil || en.Journal != nil || en.Guide != nil || en.Ref.Format != "%d%s" {
		t.Fatal("英文語言包：空 ECL catalog、沒有手札與攻略、手札編號不補空白")
	}
	l := NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	l.Journal = &JournalPanel{Entries: map[int]JournalEntry{1: {}}, Seen: map[int]bool{}}
	l.Guide = &GuidePanel{Chapters: []GuideChapter{{Title: "x"}}}
	keepJ, keepG := l.Journal, l.Guide
	l.SetLang(en)
	if l.Journal != nil || l.Guide != nil {
		t.Fatal("語言沒有手札、攻略時面板要停用")
	}
	zh := &LangPack{Code: DefaultLang, UI: &Translator{}, Journal: &JournalPanel{Entries: map[int]JournalEntry{2: {}}},
		Guide: &GuidePanel{Chapters: []GuideChapter{{Title: "y"}}}}
	l.SetLang(zh)
	if l.Journal != keepJ || l.Guide != keepG || l.Guide.Chapters[0].Title != "y" || len(l.Journal.Entries) != 1 {
		t.Fatal("換回有面板的語言要還原同一個面板物件並換內容")
	}
}

func TestPathCEnglishKeepsHotkeyColor(t *testing.T) {
	c, m, l := newC(t, nil)
	c.Catalog = &Translator{Exact: []Lookuper{identityLookup{}}}
	drawWord(c, m, 24, 7, "Exit", 0, false)
	c.SetPalette(c.pal)
	var word, hot *xlate.Stamp
	for _, s := range l.Stamps {
		switch s.Key {
		case pathCKey + "24.7":
			word = s
		case pathCKey + "24.7.hot":
			hot = s
		}
	}
	if word == nil || string(word.Text) != "Exit" || hot == nil || hot.X != 7*8 || hot.Text[0] != 'E' || hot.FG != c.pal[15] {
		t.Fatalf("英文詞整詞重畫，熱鍵字母在原位置用熱鍵色：%+v %+v", word, hot)
	}
}

func TestWrapColsEnglishDropsLeadingSpace(t *testing.T) {
	lines := wrapCols("I want you to find the adventurers and put them out of our misery", 12)
	lines = append(lines, wrapCols("abcdefghijk lmnop", 11)...) // 換行點正好是空白
	for i, ln := range lines {
		if i > 0 && len(ln) > 0 && ln[0] == ' ' {
			t.Fatalf("第 %d 列行首空白：%q", i, string(ln))
		}
	}
}

func TestKoreanHalfwidthPunctNotAtLineStart(t *testing.T) {
	lines := wrapCols("가나다라마바사아자차. 카타파", 10)
	for i, ln := range lines {
		if i > 0 && len(ln) > 0 && (ln[0] == '.' || ln[0] == pairRune('.', ' ')) {
			t.Fatalf("第 %d 列以半形句號開頭：%q", i, string(ln))
		}
	}
	if !closeAfter('다', '.') || !closeAfter('다', pairRune('.', ' ')) || closeAfter('a', '.') || !closeAfter('a', '。') {
		t.Fatal("closeAfter：韓文後的半形標點與全形標點不放行首，拉丁字後的半形標點照舊")
	}
}
