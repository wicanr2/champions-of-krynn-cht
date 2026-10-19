package cok

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/xlate/translit"
)

// LoadTextDir 從譯文目錄載入介面查表（含規格 004 的名稱、模板、物品名）與 ECL catalog。
// 檔名：ui-text-events.tsv、ui-text.zh-TW.tsv、ecl-text-events.tsv、ecl-text.zh-TW.tsv、
// name-events.tsv、name.zh-TW.tsv、composed-templates.tsv、menu-masters.tsv。
func LoadTextDir(dir string) (ui *Translator, ecl *Catalog, err error) {
	return LoadTextDirLang(dir, DefaultLang)
}

// LoadTextDirLang 同 LoadTextDir，譯文表取 lang 的檔名（主專案規格 015 §2）。
func LoadTextDirLang(dir, lang string) (ui *Translator, ecl *Catalog, err error) {
	read := func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, name)) }
	load := func(events, trans string) (*Catalog, error) {
		e, err := read(events)
		if err != nil {
			return nil, err
		}
		t, err := read(trans)
		if err != nil {
			return nil, err
		}
		return LoadCatalog(e, t)
	}
	uiCat, err := load("ui-text-events.tsv", langName("ui-text", lang))
	if err != nil {
		return nil, nil, err
	}
	if ecl, err = load("ecl-text-events.tsv", langName("ecl-text", lang)); err != nil {
		return nil, nil, err
	}
	names, err := load("name-events.tsv", langName("name", lang))
	if err != nil {
		return nil, nil, err
	}
	// 本作 ECL 腳本的選單選項是逐詞的獨立字串（例 INN、HALL、ARMOURY、LEAVE），由路徑 A／C 印出，
	// 介面 catalog 查不到時再查 ECL catalog（本專案 docs/re/004）。選單把選項改成首字大寫（例 Armoury），
	// ECL catalog 收的是全大寫原文，查之前轉大寫。
	ui = &Translator{Exact: []Lookuper{uiCat, upperLookup{ecl}}, Names: names, Words: chainLookup{uiCat, names}}
	mm, err := read(langShared(dir, "menu-masters", lang))
	if err != nil {
		return nil, nil, err
	}
	if ui.Menus, err = LoadMenuMasters(mm); err != nil {
		return nil, nil, err
	}
	tpl, err := read(langShared(dir, "composed-templates", lang))
	if err != nil {
		return nil, nil, err
	}
	if err := ui.LoadTemplates(tpl); err != nil {
		return nil, nil, err
	}
	// 規格 013：有音譯資料檔才啟用玩家名音譯；繁中以外的語言放在 translit-<lang>/ 子目錄
	trDir := dir
	if lang != DefaultLang {
		trDir = filepath.Join(dir, "translit-"+lang)
	}
	if _, err := os.Stat(filepath.Join(trDir, "translit-table.tsv")); err == nil {
		tr, err := translit.Load(trDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cok: 音譯資料：%w", err)
		}
		ui.Players = &PlayerNames{Translit: func(s string, sex Sex) (string, bool) {
			g := map[Sex]translit.Gender{SexMale: translit.Male, SexFemale: translit.Female}[sex] // 未知為 GenderUnknown
			zh, _, ok := tr.Transliterate(s, g)
			return zh, ok
		}}
	}
	return ui, ecl, nil
}

// LoadQuizDir 從譯文目錄讀查驗題答案表與前端文字表的格式（規格 006）。
func LoadQuizDir(dir string) (map[QuizKey]string, QuizFormat, error) {
	return LoadQuizDirLang(dir, DefaultLang)
}

// LoadQuizDirLang 同 LoadQuizDir，格式取 lang 的前端文字表。
func LoadQuizDirLang(dir, lang string) (map[QuizKey]string, QuizFormat, error) {
	ans, err := LoadQuizAnswersFile(filepath.Join(dir, "quiz-answers.tsv"))
	if err != nil {
		return nil, QuizFormat{}, err
	}
	host, err := loadHostDir(dir, lang)
	if err != nil {
		return nil, QuizFormat{}, err
	}
	f, ok := QuizFormatOf(host)
	if !ok {
		return nil, QuizFormat{}, fmt.Errorf("cok: %s 缺 quiz.entry／quiz.word／quiz.answer", langName("host-text", lang))
	}
	return ans, f, nil
}

// LoadJournalDir 從譯文目錄讀手札表與面板格式（規格 007）。
func LoadJournalDir(dir string) (*JournalPanel, error) {
	return LoadJournalDirLang(dir, DefaultLang)
}

// LoadJournalDirLang 同 LoadJournalDir，取 lang 的手札表與前端文字表。
func LoadJournalDirLang(dir, lang string) (*JournalPanel, error) {
	b, err := os.ReadFile(filepath.Join(dir, langName("journal", lang)))
	if err != nil {
		return nil, err
	}
	entries, err := LoadJournal(b)
	if err != nil {
		return nil, err
	}
	host, err := loadHostDir(dir, lang)
	if err != nil {
		return nil, err
	}
	f, ok := JournalFormatOf(host)
	if !ok {
		return nil, fmt.Errorf("cok: %s 缺 journal.title／journal.page／journal.none", langName("host-text", lang))
	}
	return &JournalPanel{Entries: entries, Format: f, Seen: map[int]bool{}}, nil
}

// ReadJournalSeen 讀已讀手札清單（每行一個編號）；檔案不存在時回空集合。
func ReadJournalSeen(path string) (map[int]bool, error) {
	seen := map[int]bool{}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return seen, nil
	}
	if err != nil {
		return nil, err
	}
	for i, l := range strings.Fields(string(b)) {
		n, err := strconv.Atoi(l)
		if err != nil {
			return nil, fmt.Errorf("cok: %s 第 %d 筆：%w", path, i+1, err)
		}
		seen[n] = true
	}
	return seen, nil
}

// AppendJournalSeen 在已讀清單加一筆。
func AppendJournalSeen(path string, n int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, n)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// upperLookup 把查詢字串轉成大寫再查（ECL 原文全大寫，選單印出時改成首字大寫）。
type upperLookup struct{ c Lookuper }

func (u upperLookup) Lookup(original []byte) (Entry, bool) {
	return u.c.Lookup([]byte(strings.ToUpper(string(original))))
}

// chainLookup 依序查多個 catalog：物品名詞組（例 Ring Mail）在術語表的名稱 catalog，單詞在介面 catalog。
type chainLookup []Lookuper

func (c chainLookup) Lookup(original []byte) (Entry, bool) {
	for _, l := range c {
		if e, ok := l.Lookup(original); ok {
			return e, true
		}
	}
	return Entry{}, false
}
