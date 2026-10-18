package dkk

import (
	"os"
	"strings"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// fakeTranslit 把每個字母換成固定的字，讓測試不依賴音譯資料；含 Q 的名字當作無法音譯。
func fakeTranslit(s string) (string, bool) {
	if strings.Contains(s, "Q") {
		return "", false
	}
	return map[string]string{"ANNA": "安娜", "BRAM": "布拉姆", "SKELETON": "斯凱", "MARY JANE": "瑪麗•珍"}[s], true
}

func playerTranslator(t *testing.T) *Translator {
	t.Helper()
	tr := &Translator{Exact: []Lookuper{catalogFor(t, "Exit", "ui.exit", "離開")},
		Names: catalogFor(t, "SKELETON", "name.skeleton", "骷髏"), Players: &PlayerNames{Translit: fakeTranslit}}
	if err := tr.LoadTemplates([]byte("template\ttranslation\tnote\n{name} is hit FOR {n} points of Damage.\t{name}受到 {n} 點傷害。\t\n")); err != nil {
		t.Fatal(err)
	}
	return tr
}

// 路徑 A 在隊伍欄（列 4–9、欄 17）印名字才學；學到後其他位置與組句都換成音譯（規格 013 §3.2、§3.3）。
func TestPlayerNameLearnedFromPartyList(t *testing.T) {
	tr := playerTranslator(t)
	m := make(fakeMem, 0x100000)
	plantSignature(m, 0x62AA, 0)
	font := &xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}
	p, err := NewPathA(m, tr, font, &xlate.Layer{})
	if err != nil {
		t.Fatal(err)
	}
	print := func(text string, row, col uint8) Hit {
		base := linear(0x5777, 8)
		m[base] = byte(len(text))
		copy(m[base+1:], text)
		callFrame(m, 0x1CBD, 0x3F00, 0x34CF, 0x1605, 0x5777, 8, 0, 10, row, col)
		p.Before(uint64(len(p.Hits)+1), 0x05F4, 0x036A, 0x1CBD, 0x3F00, m)
		return p.Hits[len(p.Hits)-1]
	}
	// 戰鬥右窗（欄 23）先印：還沒學到，留英文
	if h := print("ANNA", 1, 23); h.Stamped {
		t.Fatalf("未學到前不應覆繪：%+v", h)
	}
	if h := print("ANNA", 4, 17); !h.Stamped || h.Key != "name:ANNA" {
		t.Fatalf("隊伍欄應學到並覆繪：%+v", h)
	}
	if h := print("ANNA", 1, 23); !h.Stamped || h.Key != "name:ANNA" {
		t.Fatalf("學到後戰鬥右窗應覆繪：%+v", h)
	}
	// 其他位置的未知字串不學
	print("CORA", 1, 23)
	if _, ok := tr.Lookup([]byte("CORA")); ok {
		t.Fatal("非隊伍欄不應學名字")
	}
	// 組句的 {name}
	if e, ok := tr.Lookup([]byte("ANNA is hit FOR 4 points of Damage.")); !ok || e.Translation != "安娜受到 4 點傷害。" {
		t.Fatalf("組句：%+v %v", e, ok)
	}
}

func TestPlayerNameRules(t *testing.T) {
	tr := playerTranslator(t)
	for _, s := range []string{"  BRAM ", "SKELETON", "Exit", "R2D2", "QUINN", "A", "MARY JANE"} {
		tr.LearnPlayer([]byte(s))
	}
	cases := map[string]string{
		"BRAM":      "布拉姆",  // 去頭尾空白後學
		"SKELETON":  "骷髏",   // 與名稱表同名：用名稱表
		"Exit":      "離開",   // UI 目錄優先
		"MARY JANE": "瑪麗•珍", // 多字名
	}
	for s, want := range cases {
		if e, ok := tr.Lookup([]byte(s)); !ok || e.Translation != want {
			t.Errorf("%s：%+v %v，要 %s", s, e, ok, want)
		}
	}
	for _, s := range []string{"R2D2", "QUINN", "A"} { // 含數字、無法音譯、單一字母
		if e, ok := tr.Players.Lookup([]byte(s)); ok {
			t.Errorf("%s 不應學到：%+v", s, e)
		}
	}
	if _, ok := tr.Players.Lookup([]byte("SKELETON")); ok {
		t.Error("與名稱表同名不應記成玩家名")
	}
}

// 實際譯文目錄：有音譯資料檔時啟用（規格 013 §3.1）。
func TestLoadTextDirTranslit(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_DKK_TEXT_DIR")
	if dir == "" {
		t.Skip("DOSGOLEM_DKK_TEXT_DIR 未設定")
	}
	ui, _, err := LoadTextDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ui.Players == nil {
		t.Fatal("有音譯資料時應啟用")
	}
	// 建角收據的六人：音譯成功，且格數不超過原名（路徑 A 放得下）
	for _, s := range []string{"ANNA", "BRAM", "CORA", "DORN", "ELIA", "FINN"} {
		ui.LearnPlayer([]byte(s))
		e, ok := ui.Lookup([]byte(s))
		if !ok || e.Key != "name:"+s || len(PackHalfWidth([]rune(e.Translation))) > len(s) {
			t.Errorf("%s 音譯：%+v %v", s, e, ok)
		}
		t.Logf("%s → %s", s, e.Translation)
	}
}
