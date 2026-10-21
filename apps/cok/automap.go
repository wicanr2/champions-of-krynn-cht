package cok

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/xlate"
)

// 幽靈騎士規格 009：遊戲內地圖疊層。牆面從記憶體裡已載入的 GEO 區塊讀，位置讀隊伍 x、y、面向，
// 地圖身分以 0x400 本體雜湊對 text/map-geo.tsv；攻略點取自 text/map-guide.zh-TW.tsv。

const (
	// 本作 DS:6D7C 是指向目前 GEO 區塊本體的遠指標（off, seg），DS:6D85 起是 x、y、面向（0 北、2 東、4 南、6 西）。
	// 開場 7,13 面北時 DS:6D85 為 07 0D 00，前 9 bytes 的遠指標 0000:45AC；與幽靈騎士 C06A／C073 相差 9 bytes 的配置相同
	// （本專案 docs/re/006，強推論，地圖雜湊比對後升級）。
	geoPtrOff   = 0x6D7C
	partyXOff   = 0x6D85
	geoBodySize = 0x400
	// 遊戲狀態區塊指標（幽靈騎士 DS:8484，用來以 ECL 區塊區分重用的 GEO）本作尚未找到，0 表示停用。
	statePtrOff = 0
	stateEclOff = 0xF2
)

// MapPoint 是一個攻略點。
type MapPoint struct {
	N              int
	X, Y           int
	Label, Summary string
}

// MapGuide 是地圖資料：雜湊→地圖鍵、名稱、攻略點。
type MapGuide struct {
	ByHash map[string]string
	Titles map[string]string
	Points map[string][]MapPoint
}

// LoadAutoMapDir 讀地圖資料與面板格式；沒有 map-geo.tsv 時回 nil（不啟用）。
func LoadAutoMapDir(dir string) (*AutoMap, error) {
	return LoadAutoMapDirLang(dir, DefaultLang)
}

// LoadAutoMapDirLang 同 LoadAutoMapDir，地圖名稱、攻略點與面板格式取 lang 的表。
func LoadAutoMapDirLang(dir, lang string) (*AutoMap, error) {
	g, err := LoadMapGuideDirLang(dir, lang)
	if err != nil || g == nil {
		return nil, err
	}
	host, err := loadHostDir(dir, lang)
	if err != nil {
		return nil, err
	}
	f, ok := MapFormatOf(host)
	if !ok {
		return nil, fmt.Errorf("cok: %s 缺 map.* 或 map.dirs 不是四個字", langName("host-text", lang))
	}
	return &AutoMap{Guide: g, Format: f}, nil
}

// LoadMapGuideDir 讀 map-geo.tsv 與 map-guide.zh-TW.tsv；兩者都沒有時回 nil。
func LoadMapGuideDir(dir string) (*MapGuide, error) {
	return LoadMapGuideDirLang(dir, DefaultLang)
}

// LoadMapGuideDirLang 同 LoadMapGuideDir；繁中以外的語言以 map-titles.<lang>.tsv（map、title）
// 覆寫 map-geo.tsv 的繁中地圖名，攻略點取 map-guide.<lang>.tsv（沒有就不顯示攻略點）。
func LoadMapGuideDirLang(dir, lang string) (*MapGuide, error) {
	b, err := os.ReadFile(filepath.Join(dir, "map-geo.tsv"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := readTSV(b, []string{"map", "payload_sha256", "title"})
	if err != nil {
		return nil, fmt.Errorf("cok: 地圖雜湊表：%w", err)
	}
	g := &MapGuide{ByHash: map[string]string{}, Titles: map[string]string{}, Points: map[string][]MapPoint{}}
	for _, r := range rows {
		if !strings.Contains(r[0], "@") { // `鍵@ECL` 列與基本鍵同雜湊，只提供名稱與攻略點
			g.ByHash[r[1]] = r[0]
		}
		g.Titles[r[0]] = r[2]
	}
	if lang != DefaultLang {
		g.Titles = map[string]string{} // 沒有該語言名稱的地圖顯示 map.unknown，不露出繁中
		if b, err = os.ReadFile(filepath.Join(dir, langName("map-titles", lang))); err != nil {
			return nil, err
		}
		if rows, err = readTSV(b, []string{"map", "title"}); err != nil {
			return nil, fmt.Errorf("cok: 地圖名稱表：%w", err)
		}
		for _, r := range rows {
			g.Titles[r[0]] = r[1]
		}
	}
	b, err = os.ReadFile(filepath.Join(dir, langName("map-guide", lang)))
	if os.IsNotExist(err) {
		return g, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err = readTSV(b, []string{"map", "n", "x", "y", "label", "summary"})
	if err != nil {
		return nil, fmt.Errorf("cok: 地圖攻略表：%w", err)
	}
	for i, r := range rows {
		var p MapPoint
		var e1, e2, e3 error
		p.N, e1 = strconv.Atoi(r[1])
		p.X, e2 = strconv.Atoi(r[2])
		p.Y, e3 = strconv.Atoi(r[3])
		if e1 != nil || e2 != nil || e3 != nil || p.X < 0 || p.X > 15 || p.Y < 0 || p.Y > 15 {
			return nil, fmt.Errorf("cok: 地圖攻略表第 %d 列座標不合法", i+2)
		}
		if _, ok := g.Titles[r[0]]; !ok {
			return nil, fmt.Errorf("cok: 地圖攻略表第 %d 列地圖 %q 不在 map-geo.tsv", i+2, r[0])
		}
		p.Label, p.Summary = r[4], r[5]
		g.Points[r[0]] = append(g.Points[r[0]], p)
	}
	return g, nil
}

type cellPos struct{ x, y int }

// AutoMap 是地圖疊層狀態。
type AutoMap struct {
	Guide   *MapGuide
	Format  MapFormat
	OnVisit func(key string, x, y int) // 第一次走到某格時呼叫（前端存檔）

	open, reveal bool
	ptr          uint32
	sum          string
	ecl          int    // 目前的 ECL 區塊（-1 未知）
	Key          string // 目前地圖鍵（查不到雜湊時沿用同指標的上一個鍵，或空）
	walls        [16][16][4]uint8
	loaded       bool
	X, Y, Dir    int
	visited      map[string]map[cellPos]bool
	listPage     int // 右側清單頁（PgUp／PgDn）
}

// TurnList 翻右側清單頁（delta ±1）。
func (m *AutoMap) TurnList(delta int) {
	if m.Open() {
		m.listPage = max(0, m.listPage+delta)
	}
}

// MapFormat 是面板固定文字（host-text 的 map.*）。
type MapFormat struct{ Unknown, NoData, Pos, Dirs, Hint, All, More string }

// MapFormatOf 從前端文字表取面板格式。
func MapFormatOf(host map[string]string) (MapFormat, bool) {
	f := MapFormat{host["map.unknown"], host["map.nodata"], host["map.pos"], host["map.dirs"], host["map.hint"], host["map.all"], host["map.more"]}
	return f, f.Unknown != "" && f.NoData != "" && f.Pos != "" && len([]rune(f.Dirs)) == 4 && f.Hint != "" && f.All != ""
}

// Sample 從記憶體讀目前地圖與位置（每 1M 步呼叫一次）。ds 是遊戲資料段。
func (m *AutoMap) Sample(mem CPUView, ds uint16) {
	base := linear(ds, 0)
	off, seg := mem.Read16(base+geoPtrOff), mem.Read16(base+geoPtrOff+2)
	ptr := linear(seg, off)
	if seg == 0 {
		return
	}
	body := make([]byte, geoBodySize)
	for i := range body {
		body[i] = mem.Read8(ptr + uint32(i))
	}
	s := sha256.Sum256(body)
	sum := hex.EncodeToString(s[:])
	ecl := -1
	if so, ss := mem.Read16(base+statePtrOff), mem.Read16(base+statePtrOff+2); statePtrOff != 0 && ss != 0 {
		ecl = int(mem.Read8(linear(ss, so) + stateEclOff))
	}
	if sum != m.sum || ptr != m.ptr || ecl != m.ecl {
		m.ecl = ecl
		if k, ok := m.Guide.ByHash[sum]; ok {
			m.Key = k
			// 同一塊 GEO 給兩個區用時（3/50 塞普路斯與達西默），以 `鍵@ECL 區塊` 區分（規格 009 §2.1）
			if q := fmt.Sprintf("%s@%02X", k, ecl); ecl >= 0 && m.Guide.Titles[q] != "" {
				m.Key = q
			}
		} else if ptr != m.ptr {
			m.Key = "" // 換了緩衝區又對不到：未知地圖；同一緩衝區的小改動（開門等）沿用上一個鍵
		}
		m.ptr, m.sum = ptr, sum
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				i := y*16 + x
				m.walls[y][x] = [4]uint8{body[i] >> 4, body[i] & 15, body[0x100+i] >> 4, body[0x100+i] & 15}
			}
		}
		m.loaded = true
	}
	x, y, d := int(mem.Read8(base+partyXOff)), int(mem.Read8(base+partyXOff+1)), int(mem.Read8(base+partyXOff+2))
	if x > 15 || y > 15 {
		return
	}
	m.X, m.Y, m.Dir = x, y, d/2&3
	m.Visit(m.mapID(), x, y)
}

// mapID 是走過格子的分組鍵：已知地圖用鍵，未知地圖用本體雜湊前 12 碼。
func (m *AutoMap) mapID() string {
	if m.Key != "" {
		return m.Key
	}
	if len(m.sum) >= 12 {
		return "?" + m.sum[:12]
	}
	return "?"
}

// Visit 記錄走過一格。
func (m *AutoMap) Visit(id string, x, y int) {
	if m.visited == nil {
		m.visited = map[string]map[cellPos]bool{}
	}
	if m.visited[id] == nil {
		m.visited[id] = map[cellPos]bool{}
	}
	if !m.visited[id][cellPos{x, y}] {
		m.visited[id][cellPos{x, y}] = true
		if m.OnVisit != nil {
			m.OnVisit(id, x, y)
		}
	}
}

// Open 回面板是否開啟。
func (m *AutoMap) Open() bool { return m != nil && m.open }

// Toggle 開關面板；Close 關閉；ToggleReveal 切換全圖。
func (m *AutoMap) Toggle() {
	if m != nil {
		m.open = !m.open
	}
}
func (m *AutoMap) Close() {
	if m != nil {
		m.open = false
	}
}
func (m *AutoMap) ToggleReveal() {
	if m.Open() {
		m.reveal = !m.reveal
	}
}

func (m *AutoMap) seen(x, y int) bool { return m.reveal || m.visited[m.mapID()][cellPos{x, y}] }

// quadrant 回複合圖中隊伍所在的象限（0 西北、1 東北、2 西南、3 東南）與各象限名稱。
// 複合圖是一塊 GEO 分成四個 8×8 小地圖（例如 3/53：沉船、樹之父、遊樂迷宮、被詛咒的村莊），
// map-geo.tsv 的名稱以 `|` 依序寫四個名稱；不是複合圖回 -1。
func (m *AutoMap) quadrant() (int, []string) {
	if m.Key == "" || m.Guide == nil {
		return -1, nil
	}
	ts := strings.Split(m.Guide.Titles[m.Key], "|")
	if len(ts) != 4 {
		return -1, nil
	}
	return quadOf(m.X, m.Y), ts
}

func quadOf(x, y int) int { return y/8*2 + x/8 }

// VisiblePoints 回目前地圖可見的攻略點（依編號）。
func (m *AutoMap) VisiblePoints() []MapPoint {
	var out []MapPoint
	if m.Key == "" || m.Guide == nil {
		return nil
	}
	q, _ := m.quadrant()
	for _, p := range m.Guide.Points[m.Key] {
		if q >= 0 && quadOf(p.X, p.Y) != q {
			continue // 複合圖只列隊伍所在的小地圖
		}
		if m.seen(p.X, p.Y) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N < out[j].N })
	return out
}

// 面板幾何（規格 009 §2.3）
const (
	mapX0, mapY0 = 4, 4
	mapCell      = 11      // 16 格 × 11 = 176，地圖止於 y=180，不蓋第 25 列原版選單
	listX, listW = 200, 15 // 右側清單起點（邏輯像素）與欄數
	listRows     = 24
)

var (
	mapWall  = [3]uint8{85, 255, 85}
	mapFloor = [3]uint8{40, 40, 40}
	mapParty = [3]uint8{255, 255, 85}
	mapPoint = [3]uint8{255, 85, 255}
)

// listLayout 回右側清單的欄數、列數與每格裝置像素：3 倍時改用 16 像素小字（使用者 2026-10-21 指示），
// 22 欄、36 列，攻略點一頁列得完；其他倍率維持 8 邏輯像素一格（15 欄、24 列）。
func listLayout(scale int) (cols, rows, cell int) {
	if scale == 3 {
		return (320 - listX) * 3 / 16, 192 * 3 / 16, 16
	}
	return listW, listRows, 8 * scale
}

// Lines 回 2 倍版面的右側清單（測試用）。
func (m *AutoMap) Lines() [][]rune { return m.LinesFor(listW, listRows) }

// LinesFor 回 cols 欄、rows 列的右側清單：標題、座標、攻略點清單（編號固定兩格，名稱對齊），
// 所在格有攻略點時底部留給說明（依說明長度，最多三分之一），最後一列是提示。
func (m *AutoMap) LinesFor(cols, rows int) [][]rune {
	out := make([][]rune, rows)
	put := func(row int, s string) {
		if row < rows {
			out[row] = PackHalfWidth([]rune(s))
		}
	}
	title := m.Format.Unknown
	if m.Key != "" && m.Guide != nil && m.Guide.Titles[m.Key] != "" {
		title = m.Guide.Titles[m.Key]
	}
	if q, ts := m.quadrant(); q >= 0 {
		title = ts[q]
	}
	put(0, title)
	dir := string([]rune(m.Format.Dirs)[m.Dir&3])
	put(1, strings.NewReplacer("{x}", strconv.Itoa(m.X), "{y}", strconv.Itoa(m.Y), "{dir}", dir).Replace(m.Format.Pos))
	row := 3
	pts := m.VisiblePoints()
	if m.Key == "" || m.Guide == nil || len(m.Guide.Points[m.Key]) == 0 {
		for _, l := range wrapCols(m.Format.NoData, cols) {
			if row < rows-4 {
				out[row] = l
				row++
			}
		}
	}
	listed := map[int]bool{} // 同一編號標在多格（塔樓樓梯、前門）只列一次
	var items []string
	var summary [][]rune
	for _, p := range pts {
		if p.X == m.X && p.Y == m.Y && summary == nil {
			summary = wrapCols(fmt.Sprintf("%2d %s", p.N, p.Summary), cols)
		}
		if !listed[p.N] {
			listed[p.N] = true
			items = append(items, fmt.Sprintf("%2d %s", p.N, p.Label))
		}
	}
	end := rows - 1 // 清單可用到這列之前
	if summary != nil {
		n := min(len(summary), max(4, rows/3))
		summary = summary[:n]
		end = rows - 2 - n // 清單與說明之間空一列
	}
	per := max(1, end-row)
	pages := (len(items) + per - 1) / per
	m.listPage = max(0, min(m.listPage, pages-1))
	for i := m.listPage * per; i < len(items) && row < end; i++ {
		put(row, items[i])
		row++
	}
	for i, l := range summary {
		out[rows-1-len(summary)+i] = l
	}
	hint := m.Format.Hint
	if m.reveal {
		hint = m.Format.All
	}
	if pages > 1 && m.Format.More != "" {
		hint = strings.NewReplacer("{page}", strconv.Itoa(m.listPage+1), "{pages}", strconv.Itoa(pages)).Replace(m.Format.More)
	}
	put(rows-1, hint)
	return out
}

// Draw 把面板畫進 scale 倍的 RGBA。
func (m *AutoMap) Draw(dst []uint8, scale int, font *xlate.Font, missing func(rune)) {
	if !m.Open() || font == nil || scale <= 0 {
		return
	}
	w := 320 * scale
	for y := 0; y < 192*scale; y++ {
		for x := 0; x < 320*scale; x++ {
			putRGB(dst, w, x, y, journalBG)
		}
	}
	px := func(lx, ly int) (int, int) { return (mapX0 + lx) * scale, (mapY0 + ly) * scale }
	if m.loaded {
		for cy := 0; cy < 16; cy++ {
			for cx := 0; cx < 16; cx++ {
				if !m.seen(cx, cy) {
					continue
				}
				x0, y0 := px(cx*mapCell, cy*mapCell)
				for y := y0 + scale; y < y0+mapCell*scale; y++ {
					for x := x0 + scale; x < x0+mapCell*scale; x++ {
						putRGB(dst, w, x, y, mapFloor)
					}
				}
				wl := m.walls[cy][cx]
				for k := 0; k <= mapCell*scale; k++ {
					for t := 0; t < scale; t++ {
						if wl[0] != 0 {
							putRGB(dst, w, x0+k, y0+t, mapWall)
						}
						if wl[2] != 0 {
							putRGB(dst, w, x0+k, y0+mapCell*scale-t, mapWall)
						}
						if wl[3] != 0 {
							putRGB(dst, w, x0+t, y0+k, mapWall)
						}
						if wl[1] != 0 {
							putRGB(dst, w, x0+mapCell*scale-t, y0+k, mapWall)
						}
					}
				}
			}
		}
	}
	for _, p := range m.VisiblePoints() {
		x0, y0 := px(p.X*mapCell+2, p.Y*mapCell+2)
		drawLine(dst, scale, font, x0/scale, y0/scale, 1, []rune{pointGlyph(p.N)}, mapPoint, missing)
	}
	// 隊伍：格中央的三角箭頭
	cx, cy := px(m.X*mapCell+mapCell/2, m.Y*mapCell+mapCell/2)
	r := 4 * scale
	for i := -r; i <= r; i++ {
		for j := 0; j <= r-abs(i); j++ {
			var x, y int
			switch m.Dir & 3 {
			case 0: // 北：底邊在下、尖端朝上
				x, y = cx+i, cy+r/2-j
			case 1:
				x, y = cx-r/2+j, cy+i
			case 2:
				x, y = cx+i, cy-r/2+j
			default:
				x, y = cx+r/2-j, cy+i
			}
			putRGB(dst, w, x, y, mapParty)
		}
	}
	cols, rows, cell := listLayout(scale)
	for row, line := range m.LinesFor(cols, rows) {
		c := journalText
		if row == 0 {
			c = journalTitle
		} else if row == rows-1 {
			c = journalPage
		}
		if cell == 8*scale {
			drawLine(dst, scale, font, listX, row*8, cols, line, c, missing)
		} else {
			drawLinePx(dst, w, font, listX*scale, row*cell, cols, cell, line, c, missing)
		}
	}
}

// drawLinePx 以字型原尺寸（不放大）畫一列，每字佔 cell 裝置像素；給 3 倍的小字清單用。
func drawLinePx(dst []uint8, w int, font *xlate.Font, x0, y0, cols, cell int, line []rune, c [3]uint8, missing func(rune)) {
	rb := (font.W + 7) / 8
	for col, r := range line {
		if col >= cols || r == ' ' {
			continue
		}
		g, ok := font.Glyphs[r]
		if !ok {
			if missing != nil {
				missing(r)
			}
			continue
		}
		for gy := 0; gy < font.H && gy < cell; gy++ {
			for gx := 0; gx < font.W && gx < cell; gx++ {
				if g[gy*rb+gx/8]&(0x80>>uint(gx%8)) != 0 {
					putRGB(dst, w, x0+col*cell+gx, y0+gy, c)
				}
			}
		}
	}
}

// pointGlyph 把 1–99 的編號畫成一個 8×8 字格：一位數用 ASCII 本身，兩位數用半形組合字（規格 005）。
func pointGlyph(n int) rune {
	t := []rune(strconv.Itoa(n))
	if len(t) == 1 {
		return t[0]
	}
	return pairRune(t[0], t[1])
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// InMapPanel 回邏輯像素是否在面板範圍內（收據用）。
func InMapPanel(x, y int) bool { return x >= 0 && x < 320 && y >= 0 && y < 192 }

// ReadVisited 讀走過的格子（每行 `地圖 x y`）。
func ReadVisited(path string) (map[string]map[cellPos]bool, error) {
	out := map[string]map[cellPos]bool{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var k string
		var x, y int
		if _, err := fmt.Sscan(sc.Text(), &k, &x, &y); err != nil {
			return nil, fmt.Errorf("cok: %s：%q：%w", path, sc.Text(), err)
		}
		if out[k] == nil {
			out[k] = map[cellPos]bool{}
		}
		out[k][cellPos{x, y}] = true
	}
	return out, sc.Err()
}

// SetVisited 換入讀回的走過格子。
func (m *AutoMap) SetVisited(v map[string]map[cellPos]bool) { m.visited = v }

// AppendVisited 追加一格。
func AppendVisited(path, key string, x, y int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "%s %d %d\n", key, x, y); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
