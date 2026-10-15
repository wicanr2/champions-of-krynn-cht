package dkk

import (
	"os"
	"path/filepath"
)

// LoadTextDir 從譯文目錄載入介面查表（含規格 004 的名稱、模板、物品名）與 ECL catalog。
// 檔名：ui-text-events.tsv、ui-text.zh-TW.tsv、ecl-text-events.tsv、ecl-text.zh-TW.tsv、
// name-events.tsv、name.zh-TW.tsv、composed-templates.tsv。
func LoadTextDir(dir string) (ui *Translator, ecl *Catalog, err error) {
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
	uiCat, err := load("ui-text-events.tsv", "ui-text.zh-TW.tsv")
	if err != nil {
		return nil, nil, err
	}
	if ecl, err = load("ecl-text-events.tsv", "ecl-text.zh-TW.tsv"); err != nil {
		return nil, nil, err
	}
	names, err := load("name-events.tsv", "name.zh-TW.tsv")
	if err != nil {
		return nil, nil, err
	}
	ui = &Translator{Exact: []Lookuper{uiCat}, Names: names, Words: uiCat}
	tpl, err := read("composed-templates.tsv")
	if err != nil {
		return nil, nil, err
	}
	if err := ui.LoadTemplates(tpl); err != nil {
		return nil, nil, err
	}
	return ui, ecl, nil
}
