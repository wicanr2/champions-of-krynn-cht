package cok

import "testing"

type mapLookup map[string]string

func (m mapLookup) Lookup(b []byte) (Entry, bool) {
	zh, ok := m[string(b)]
	return Entry{Key: string(b), Translation: zh}, ok
}

func TestPricedRowKeepsPriceRightAligned(t *testing.T) {
	words := mapLookup{"Long Sword": "長劍", "Arrow": "箭矢", "Hoopak": "呼帕克杖"}
	tr := &Translator{Words: words}
	for _, in := range []string{"Long Sword                   7", "20 Arrows                    1", "Hoopak                       2"} {
		e, ok := tr.Lookup([]byte(in))
		if !ok {
			t.Fatalf("%q 沒有命中", in)
		}
		if n := len(PackHalfWidth([]rune(e.Translation))); n != len(in) {
			t.Errorf("%q → %q：%d 格，要 %d 格", in, e.Translation, n, len(in))
		}
	}
	if e, _ := tr.Lookup([]byte("20 Arrows                    1")); e.Translation[:len("箭矢 20")] != "箭矢 20" {
		t.Errorf("數量要置後：%q", e.Translation)
	}
	if e, ok := tr.Lookup([]byte(" Yes  60 Arrows ")); !ok || e.Translation != " 是 箭矢 60" {
		t.Errorf("備妥列帶數量：%q %v", e.Translation, ok)
	}
	if e, ok := tr.Lookup([]byte("3 Arrows")); !ok || e.Translation != "箭矢 3" {
		t.Errorf("戰鬥訊息帶數量：%q %v", e.Translation, ok)
	}
	if e, ok := tr.Lookup([]byte("SIR STRONGSWORD buys a Hoopak ")); !ok || e.Translation != "SIR STRONGSWORD買下呼帕克杖" {
		t.Errorf("購買訊息：%q %v", e.Translation, ok)
	}
}
