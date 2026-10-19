package cok

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// plantGeo 在合成記憶體放一塊 GEO 本體與 DS:C06A 指標、DS:C073 位置，回本體雜湊。
func plantGeo(m fakeMem, ds uint16, seg uint16, x, y, dir byte) string {
	body := make([]byte, geoBodySize)
	body[3*16+2] = 0x10       // (2,3) 北牆
	body[0x100+3*16+2] = 0x01 // (2,3) 西牆
	copy(m[linear(seg, 0):], body)
	base := linear(ds, 0)
	m.put16(base+geoPtrOff, 0)
	m.put16(base+geoPtrOff+2, seg)
	m[base+partyXOff], m[base+partyXOff+1], m[base+partyXOff+2] = x, y, dir
	s := sha256.Sum256(body)
	return hex.EncodeToString(s[:])
}

func TestAutoMapSample(t *testing.T) {
	m := make(fakeMem, 0x100000)
	sum := plantGeo(m, 0x0CDE, 0x4C28, 2, 3, 4)
	var visits []string
	a := &AutoMap{Guide: &MapGuide{ByHash: map[string]string{sum: "1/24"}, Titles: map[string]string{"1/24": "蓋加斯屯墾區"},
		Points: map[string][]MapPoint{"1/24": {{N: 9, X: 2, Y: 3, Label: "銀行", Summary: "唯一的銀行"}, {N: 11, X: 6, Y: 9, Label: "神殿"}}}},
		Format:  MapFormat{"未知", "無資料", "{x},{y} 面向{dir}", "北東南西", "V 全圖", "V 只看走過", "{page}/{pages}"},
		OnVisit: func(k string, x, y int) { visits = append(visits, k) }}
	a.Sample(m, 0x0CDE)
	if a.Key != "1/24" || a.X != 2 || a.Y != 3 || a.Dir != 2 {
		t.Fatalf("取樣：key=%q (%d,%d) dir=%d", a.Key, a.X, a.Y, a.Dir)
	}
	if w := a.walls[3][2]; w[0] != 1 || w[3] != 1 || w[1] != 0 || w[2] != 0 {
		t.Errorf("牆面：%v", w)
	}
	if len(visits) != 1 {
		t.Errorf("第一次走到要通知一次：%v", visits)
	}
	a.Sample(m, 0x0CDE)
	if len(visits) != 1 {
		t.Errorf("同一格不重複通知：%v", visits)
	}
	// 只看走過：銀行那格走過、神殿沒有
	if p := a.VisiblePoints(); len(p) != 1 || p[0].N != 9 {
		t.Errorf("可見點：%+v", p)
	}
	a.open = true
	a.ToggleReveal()
	if p := a.VisiblePoints(); len(p) != 2 {
		t.Errorf("全圖：%+v", p)
	}
	// 同一緩衝區小改動（開門）沿用地圖鍵；換緩衝區又對不到則未知
	m[linear(0x4C28, 0)+5] = 0xFF
	a.Sample(m, 0x0CDE)
	if a.Key != "1/24" {
		t.Errorf("同緩衝區小改動應沿用：%q", a.Key)
	}
	base := linear(0x0CDE, 0)
	m.put16(base+geoPtrOff+2, 0x5000)
	a.Sample(m, 0x0CDE)
	if a.Key != "" {
		t.Errorf("換緩衝區對不到應為未知：%q", a.Key)
	}
}

func TestVisitedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "v.txt")
	if err := AppendVisited(p, "1/24", 6, 9); err != nil {
		t.Fatal(err)
	}
	if err := AppendVisited(p, "1/24", 7, 5); err != nil {
		t.Fatal(err)
	}
	v, err := ReadVisited(p)
	if err != nil || len(v["1/24"]) != 2 || !v["1/24"][cellPos{7, 5}] {
		t.Fatalf("讀回：%v %v", v, err)
	}
	if err := os.WriteFile(p, []byte("bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVisited(p); err == nil {
		t.Error("壞行要回錯")
	}
}

func TestLoadMapGuideRejectsUnknownMap(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "map-geo.tsv"), []byte("map\tpayload_sha256\ttitle\n1/24\tab\t蓋加斯\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "map-guide.zh-TW.tsv"), []byte("map\tn\tx\ty\tlabel\tsummary\tsource\n9/99\t1\t0\t0\tx\ty\tz\n"), 0o644)
	if _, err := LoadMapGuideDir(dir); err == nil {
		t.Error("地圖不在 map-geo.tsv 要回錯")
	}
}

// TestMapGuideTable 檢查正式地圖資料（規格 009 §3 靜態）：可載入、每點在 0..15、每塊有點的地圖都有名稱。
func TestMapGuideTable(t *testing.T) {
	dir := os.Getenv("DOSGOLEM_DKK_TEXT_DIR")
	if dir == "" {
		t.Skip("DOSGOLEM_DKK_TEXT_DIR 未設定")
	}
	g, err := LoadMapGuideDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g == nil || len(g.Points) == 0 {
		t.Fatal("沒有攻略點")
	}
	for k, ps := range g.Points {
		if g.Titles[k] == "" {
			t.Errorf("地圖 %s 有 %d 點但沒有名稱", k, len(ps))
		}
		for _, p := range ps {
			if p.Label == "" || len([]rune(p.Label)) > 10 {
				t.Errorf("%s %d 名稱長度不合：%q", k, p.N, p.Label)
			}
		}
	}
}

// 複合圖（規格 009 §2.1）：名稱以 `|` 分四個 8×8 小地圖，名稱與攻略點只取隊伍所在象限。
func TestAutoMapQuadrants(t *testing.T) {
	a := &AutoMap{Guide: &MapGuide{Titles: map[string]string{"3/53": "沉船|樹之父|遊樂迷宮|被詛咒的村莊"},
		Points: map[string][]MapPoint{"3/53": {{N: 1, X: 7, Y: 5, Label: "登船跳板"}, {N: 1, X: 12, Y: 7, Label: "大樹入口"},
			{N: 1, X: 6, Y: 15, Label: "迷宮主人"}, {N: 1, X: 15, Y: 15, Label: "出入口"}}}},
		Format: MapFormat{"未知", "無資料", "{x},{y} 面向{dir}", "北東南西", "V 全圖", "V 只看走過", "{page}/{pages}"},
		Key:    "3/53", reveal: true}
	for _, c := range []struct {
		x, y         int
		title, label string
	}{{1, 3, "沉船", "登船跳板"}, {13, 6, "樹之父", "大樹入口"}, {5, 9, "遊樂迷宮", "迷宮主人"}, {9, 9, "被詛咒的村莊", "出入口"}} {
		a.X, a.Y = c.x, c.y
		if p := a.VisiblePoints(); len(p) != 1 || p[0].Label != c.label {
			t.Errorf("(%d,%d) 可見點：%+v", c.x, c.y, p)
		}
		if l := a.Lines(); string(l[0]) != c.title || string(l[3]) != string(PackHalfWidth([]rune("1 "+c.label))) {
			t.Errorf("(%d,%d) 清單：%q %q", c.x, c.y, string(l[0]), string(l[3]))
		}
	}
	a.Guide.Titles["3/53"] = "單一名稱"
	if p := a.VisiblePoints(); len(p) != 4 {
		t.Errorf("非複合圖應列全部：%+v", p)
	}
}

// 同一塊 GEO 依目前 ECL 區塊分鍵（3/50 塞普路斯與達西默；規格 009 §2.1）。
func TestAutoMapEclQualifiedKey(t *testing.T) {
	m := make(fakeMem, 0x100000)
	sum := plantGeo(m, 0x0CDE, 0x4C28, 2, 3, 4)
	base := linear(0x0CDE, 0)
	m.put16(base+statePtrOff, 0)
	m.put16(base+statePtrOff+2, 0x48BD)
	state := linear(0x48BD, 0)
	g := &MapGuide{ByHash: map[string]string{sum: "3/50"}, Titles: map[string]string{"3/50": "塞普路斯", "3/50@51": "達西默"},
		Points: map[string][]MapPoint{}}
	a := &AutoMap{Guide: g}
	for _, c := range []struct {
		ecl  byte
		want string
	}{{0x50, "3/50"}, {0x51, "3/50@51"}, {0x50, "3/50"}} {
		m[state+stateEclOff] = c.ecl
		a.Sample(m, 0x0CDE)
		if a.Key != c.want {
			t.Errorf("ECL %02X：鍵 %q，要 %q", c.ecl, a.Key, c.want)
		}
	}
	// 雜湊表不收 `@` 列
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "map-geo.tsv"), []byte("map\tpayload_sha256\ttitle\n3/50\t"+sum+"\t塞普路斯\n3/50@51\t"+sum+"\t達西默\n"), 0o644)
	lg, err := LoadMapGuideDir(dir)
	if err != nil || lg.ByHash[sum] != "3/50" || lg.Titles["3/50@51"] != "達西默" {
		t.Fatalf("載入：%+v %v", lg, err)
	}
}
