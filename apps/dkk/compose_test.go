package dkk

import "testing"

func newTranslator(t *testing.T) *Translator {
	t.Helper()
	ui := catalogOf(t, "Long Bow", "長弓", "Long", "長", "Mace", "釘頭錘", "Spell:", "法術：", "Hold Person", "人類定身術",
		"Dart", "飛鏢", "of", "之", "Hornet's Nest", "黃蜂巢")
	names := catalogOf(t, "DREAD WOLF", "恐狼")
	tr := &Translator{Exact: []Lookuper{ui}, Names: names, Words: ui}
	tpl := "template\ttranslation\tnote\n" +
		"Hitting for {n} points  of damage\t造成 {n} 點傷害\t\n" +
		"{n},{n} N {n}:{n}\t{1},{2} 北 {3}:{4}\t\n"
	if err := tr.LoadTemplates([]byte(tpl)); err != nil {
		t.Fatal(err)
	}
	return tr
}

func TestTranslatorTemplates(t *testing.T) {
	tr := newTranslator(t)
	for in, want := range map[string]string{
		"Hitting for 15 points  of damage": "造成 15 點傷害",
		"6,6 N 06:08":                      "6,6 北 06:08",
	} {
		if e, ok := tr.Lookup([]byte(in)); !ok || e.Translation != want {
			t.Errorf("%q → %q，要 %q", in, e.Translation, want)
		}
	}
	if _, ok := tr.Lookup([]byte("Hitting for many points  of damage")); ok {
		t.Error("沒有數字的字串不走模板")
	}
}

func TestTranslatorItemNames(t *testing.T) {
	tr := newTranslator(t)
	for in, want := range map[string]string{
		"Long Bow +3 ":           "長弓 +3",
		"Mace +4 ":               "釘頭錘 +4",
		"Dart of Hornet's Nest ": "飛鏢之黃蜂巢",
	} {
		if e, ok := tr.Lookup([]byte(in)); !ok || e.Translation != want {
			t.Errorf("%q → %q，要 %q", in, e.Translation, want)
		}
	}
	if _, ok := tr.Lookup([]byte("Long Unknown +1 ")); ok {
		t.Error("任一詞組查不到就整串不處理")
	}
	if _, ok := tr.Lookup([]byte("Long Bow +3")); ok {
		t.Error("不以空白結尾的字串不走物品名規則")
	}
}

func TestTranslatorNamesAndLabels(t *testing.T) {
	tr := newTranslator(t)
	if e, ok := tr.Lookup([]byte("DREAD WOLF")); !ok || e.Translation != "恐狼" {
		t.Errorf("怪物名：%+v", e)
	}
	if e, ok := tr.Lookup([]byte("Spell:Hold Person")); !ok || e.Translation != "法術：人類定身術" {
		t.Errorf("標籤：%+v", e)
	}
	if _, ok := tr.Lookup([]byte("Spell:Unknown")); ok {
		t.Error("標籤後段查不到就不處理")
	}
	if _, ok := tr.Lookup([]byte("ANNA")); ok {
		t.Error("玩家名不該命中")
	}
}
