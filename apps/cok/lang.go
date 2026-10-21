package cok

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/dosgolem/xlate"
)

// 多語系（主專案規格 015 §2）：一種語言是一組譯文表、前端文字與字型，Live 執行中可以整組替換。
// 檔名 <表>.<語言>.tsv；繁中另沿用無後綴的 composed-templates.tsv、menu-masters.tsv，輸出與多語系前相同。

// DefaultLang 是沒有指定語言時的譯文語言。
const DefaultLang = "zh-TW"

// EnglishLang 是英文版（規格 015 §3）：不查譯文表，攔截到的原文照原樣以英文字型重繪。
const EnglishLang = "en"

// identityLookup 把原文當譯文（英文版）。
type identityLookup struct{}

func (identityLookup) Lookup(original []byte) (Entry, bool) {
	return Entry{Key: EnglishLang, Translation: string(original)}, true
}

// langName 回 stem 在 lang 的檔名。
func langName(stem, lang string) string { return stem + "." + lang + ".tsv" }

// langShared 回語言相依、但繁中沿用無後綴檔名的表：有 <stem>.<lang>.tsv 就用它，繁中退回 <stem>.tsv。
func langShared(dir, stem, lang string) string {
	n := langName(stem, lang)
	if lang == DefaultLang {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			return stem + ".tsv"
		}
	}
	return n
}

func loadHostDir(dir, lang string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(dir, langName("host-text", lang)))
	if err != nil {
		return nil, err
	}
	return LoadHostText(b)
}

// AvailableLangs 列出譯文目錄裡有前端文字表（host-text.<語言>.tsv）的語言，繁中排第一，其餘依代碼排序。
func AvailableLangs(dir string) ([]string, error) {
	m, err := filepath.Glob(filepath.Join(dir, "host-text.*.tsv"))
	if err != nil {
		return nil, err
	}
	var langs []string
	for _, p := range m {
		code := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "host-text."), ".tsv")
		if code != "" && !strings.Contains(code, ".") {
			langs = append(langs, code)
		}
	}
	sort.Slice(langs, func(i, j int) bool {
		if (langs[i] == DefaultLang) != (langs[j] == DefaultLang) {
			return langs[i] == DefaultLang
		}
		return langs[i] < langs[j]
	})
	return langs, nil
}

// LangPack 是一種語言的全部可替換內容。
type LangPack struct {
	Code       string
	Font       *xlate.Font
	UI         *Translator
	ECL        *Catalog
	QuizFormat QuizFormat
	Ref        JournalRef
	Host       map[string]string // 前端文字（切換列、面板格式）
	Journal    *JournalPanel     // 只取 Entries、Format；可為 nil
	Guide      *GuidePanel       // 只取 Chapters、Format；可為 nil
	Map        *AutoMap          // 只取 Guide、Format；可為 nil
}

// Name 回語言在切換列上的名稱（host-text 的 lang.name），沒有時回代碼。
func (p *LangPack) Name() string {
	if n := p.Host["lang.name"]; n != "" {
		return n
	}
	return p.Code
}

// LoadLangPack 讀 dir 裡 lang 的全部表；font 是該語言的 16×16 字型，會補上半形組合字與 3× 變體（同 NewLive）。
// 繁中缺 journal.ref 時用 DefaultJournalRef；其他語言要在 host-text 給 journal.ref（含 %d、%s）與 journal.refdot。
func LoadLangPack(dir, lang string, font *xlate.Font) (*LangPack, error) {
	p := &LangPack{Code: lang, Font: font}
	var err error
	if lang == EnglishLang {
		// 路徑 B 由 ECL catalog 是否存在決定要不要綁，英文給空 catalog，查表全落到原文即譯文
		if p.ECL, err = LoadCatalog([]byte("event_key\toriginal_length\toriginal_sha256\tsources\n"), []byte("key\ttranslation\tsource\n")); err != nil {
			return nil, err
		}
		p.UI = &Translator{Exact: []Lookuper{identityLookup{}}}
	} else if p.UI, p.ECL, err = LoadTextDirLang(dir, lang); err != nil {
		return nil, err
	}
	if _, p.QuizFormat, err = LoadQuizDirLang(dir, lang); err != nil {
		return nil, err
	}
	if p.Host, err = loadHostDir(dir, lang); err != nil {
		return nil, err
	}
	p.Ref = JournalRef{Format: p.Host["journal.ref"], Dot: p.Host["journal.refdot"]}
	switch {
	case p.Ref.Format == "" && lang == DefaultLang:
		p.Ref = DefaultJournalRef
	case p.Ref.Format == "" || p.Ref.Dot == "" || strings.Count(p.Ref.Format, "%d") != 1 || strings.Count(p.Ref.Format, "%s") != 1:
		return nil, fmt.Errorf("cok: %s 的 journal.ref 要有一個 %%d 與一個 %%s，journal.refdot 不可空白", langName("host-text", lang))
	}
	if _, err := os.Stat(filepath.Join(dir, langName("journal", lang))); err == nil {
		if p.Journal, err = LoadJournalDirLang(dir, lang); err != nil {
			return nil, err
		}
	}
	if p.Guide, err = LoadGuideDirLang(dir, lang); err != nil {
		return nil, err
	}
	if p.Map, err = LoadAutoMapDirLang(dir, lang); err != nil {
		return nil, err
	}
	AddHalfWidthPairs(font)
	if _, ok := font.Scaled[3]; !ok { // 英文字型另有 3 倍字模（cok-en-3x），不從 16×16 放大
		AddScaledVariants(font)
	}
	return p, nil
}

// Lang 回目前的語言包（還沒呼叫過 SetLang 時為 nil）。
func (l *Live) Lang() *LangPack { return l.lang }

// SetLang 換成 p 的語言：查表、字型與面板內容整組替換，畫面上已有的疊字以保留的原文重譯，
// 隊伍欄學過的名字交給新語言重學。面板開關、頁碼、已讀與走過的格子不變。原版狀態不受影響。
func (l *Live) SetLang(p *LangPack) {
	if p == nil || p == l.lang {
		return
	}
	if old, ok := l.ui.(*Translator); ok {
		p.UI.Relearn(old)
	}
	l.lang = p
	l.Font, l.ui, l.ecl, l.QuizFormat = p.Font, p.UI, p.ECL, p.QuizFormat
	// 語言沒有手札或攻略時面板停用（nil），不沿用上一種語言的內容；面板物件與狀態留在 keep 欄位
	if l.journalKeep == nil {
		l.journalKeep, l.guideKeep = l.Journal, l.Guide
	}
	l.Journal, l.Guide = nil, nil
	if l.journalKeep != nil && p.Journal != nil {
		l.Journal = l.journalKeep
		l.Journal.setContent(p.Journal.Entries, p.Journal.Format)
	} else {
		l.journalKeep.closeAll()
	}
	if l.guideKeep != nil && p.Guide != nil {
		l.Guide = l.guideKeep
		l.Guide.setContent(p.Guide.Chapters, p.Guide.Format)
	} else {
		l.guideKeep.Close()
	}
	if l.Map != nil && p.Map != nil {
		l.Map.Guide, l.Map.Format = p.Map.Guide, p.Map.Format
	}
	if !l.ready {
		return
	}
	l.A.ASCIIWide = p.Code == EnglishLang
	l.A.relang(p.UI, p.Font, p.QuizFormat)
	l.C.relang(p.UI, p.Font)
	if l.B != nil {
		l.B.relang(p.Font, p.Ref, p.ECL, p.UI)
	}
}

// setContent 換手札內容與格式；面板開著時以同一則、同一頁重排（頁數變少就停在最後一頁）。
func (j *JournalPanel) setContent(entries map[int]JournalEntry, f JournalFormat) {
	j.Entries, j.Format = entries, f
	if j.pages != nil {
		page := j.page
		j.show(j.n)
		j.page = max(0, min(len(j.pages)-1, page))
	}
}

// setContent 換攻略內容與格式；面板開過時以同章同頁重排，不觸發 OnMove（位置沒有變）。
func (g *GuidePanel) setContent(chs []GuideChapter, f GuideFormat) {
	g.Chapters, g.Format = chs, f
	if g.pages != nil {
		on := g.OnMove
		g.OnMove = nil
		g.show(g.ch, g.page)
		g.OnMove = on
	}
}

// closeAll 關掉面板（自動與翻閱模式）。
func (j *JournalPanel) closeAll() {
	if j != nil {
		j.open, j.browse = false, false
	}
}
