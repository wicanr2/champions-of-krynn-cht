package cok

import (
	"bytes"
	"fmt"

	"github.com/wicanr2/dosgolem/xlate"
)

// Signature 是一段位元組簽章；Mask 為 false 的位置不比對（重定位改寫的段值）。
type Signature struct {
	Bytes []byte
	Mask  []bool
}

// sig 由「-1 表示不比對」的整數表建 Signature。
func sig(v ...int) Signature {
	s := Signature{Bytes: make([]byte, len(v)), Mask: make([]bool, len(v))}
	for i, x := range v {
		if x >= 0 {
			s.Bytes[i], s.Mask[i] = byte(x), true
		}
	}
	return s
}

// PathASignature 是 resident Pascal 字串印字程序的進入點（幽靈騎士規格 001 §2.1 的同一函式）。
// 本作 START.EXE 檔案 0x552A，執行期 0631:0353（本專案 docs/re/004，已證實：主選單 5 行經過此處）。
// 前 23 bytes 另有一處相同（0x69DB，第 28 byte 為 0x8A）；第 25–26、30–31 byte 是遠呼叫的段值，不比對。
// 第 28–29 byte 是遠呼叫的位移，本作為 51 00（幽靈騎士為 58 00）。
var PathASignature = sig(
	0x55, 0x89, 0xE5, 0x81, 0xEC, 0x04, 0x01, 0xC4, 0x7E, 0x06, 0x06, 0x57, 0x8D, 0xBE, 0x00, 0xFF,
	0x16, 0x57, 0xB8, 0xFF, 0x00, 0x50, 0x9A, 0xDE, 0x0A, -1, -1, 0x9A, 0x51, 0x00, -1, -1,
)

// Find 在 [lo, hi) 的線性位址找簽章，回所有命中位址。
func (s Signature) Find(read func(addr uint32) uint8, lo, hi uint32) []uint32 {
	var hits []uint32
	n := uint32(len(s.Bytes))
	if n == 0 || hi < lo+n {
		return nil
	}
	for a := lo; a+n <= hi; a++ {
		ok := true
		for i := uint32(0); i < n; i++ {
			if s.Mask[i] && read(a+i) != s.Bytes[i] {
				ok = false
				break
			}
		}
		if ok {
			hits = append(hits, a)
		}
	}
	return hits
}

// Hit 是一次路徑 A 呼叫的觀測紀錄。原文只存長度與命中 key，不存字串本身。
type Hit struct {
	Step     uint64 `json:"step"`
	Caller   string `json:"caller"`
	Length   int    `json:"length"`
	Key      string `json:"key,omitempty"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
	BG       int    `json:"bg"`
	FG       int    `json:"fg"`
	Stamped  bool   `json:"stamped"`
	Rejected string `json:"rejected,omitempty"`
	Str      uint32 `json:"str,omitempty"` // 路徑 A 字串（Pascal 長度位元組）的線性位址
}

// CPUView 是攔截點需要的最小機器介面。
type CPUView interface {
	Read8(addr uint32) uint8
	Read16(addr uint32) uint16
}

func linear(seg, off uint16) uint32 { return uint32(seg)<<4 + uint32(off) }

// PathA 在印字程序進入點讀參數、查譯文、加 xlate Stamp（規格 001）。
type PathA struct {
	Entry   uint32 // 進入點線性位址，由 PathASignature 找到
	Catalog Lookuper
	Font    *xlate.Font
	Layer   *xlate.Layer
	Hits    []Hit
	// OnText 在每次合法進入時以原字串呼叫（規格 014 配樂觸發）；可為 nil。
	OnText func(text []byte)

	// pending 是已加入、等原版印完的 Stamp：返回位址命中且 SP 回到呼叫前加參數時轉 Pending。
	pending []pendingStamp
	// bgOf 記每筆 Stamp 呼叫時給的背景 attribute，PrepareFrame 用它校正定色方向。
	bgOf map[*xlate.Stamp]uint8
	// src 記每筆 Stamp 查表用的原文，換語言時重譯（主專案規格 015 §2）。
	src map[*xlate.Stamp][]byte

	// ASCIIWide 為真時純 ASCII 譯文一字一格（英文版，主專案規格 015 §3）；否則兩字併一格，
	// 中文等語言的製作群英文名才不會每個字母佔一個全形格（使用者 2026-10-21 指示）。
	ASCIIWide bool

	// 規格 006 查驗題：答案表、是否隱藏答案、最近一次改寫。
	Answers     map[QuizKey]string
	QuizFormat  QuizFormat // 空白時不改寫查驗題
	HideAnswers bool
	QuizShown   *QuizShown
	quiz        quizTrack
}

// pack 把譯文轉成格子：半形兩字併一格；純 ASCII 只有 ASCIIWide（英文版）時維持一字一格。
func (p *PathA) pack(s string) []rune {
	if p.ASCIIWide {
		return PackHalfWidth([]rune(s))
	}
	return PackHalfWidthAll([]rune(s))
}

type pendingStamp struct {
	ret   uint32
	ss    uint16
	sp    uint16
	stamp *xlate.Stamp
}

// argBytes 是印字程序以 RETF 彈掉的參數長度：六個 word。
const argBytes = 12

// NewPathA 搜尋進入點；命中數不是 1 就回錯（規格 001 §2.1：不猜位址）。
func NewPathA(mem CPUView, cat Lookuper, font *xlate.Font, layer *xlate.Layer) (*PathA, error) {
	hits := PathASignature.Find(mem.Read8, 0, 0xA0000)
	if len(hits) != 1 {
		return nil, fmt.Errorf("cok: 路徑 A 簽章命中 %d 處，要恰好 1 處", len(hits))
	}
	if font == nil || font.W != 16 || font.H != 16 {
		return nil, fmt.Errorf("cok: 路徑 A 需要 16×16 字型")
	}
	return &PathA{Entry: hits[0], Catalog: cat, Font: font, Layer: layer}, nil
}

// Before 在每道指令執行前呼叫；CS:IP 等於進入點時處理一次呼叫。
func (p *PathA) Before(step uint64, cs, ip, ss, sp uint16, mem CPUView) {
	at := linear(cs, ip)
	if len(p.pending) > 0 {
		keep := p.pending[:0]
		for _, w := range p.pending {
			if at == w.ret && ss == w.ss && sp == w.sp {
				if w.stamp.State == xlate.Printing {
					w.stamp.State = xlate.Pending
				}
				continue
			}
			keep = append(keep, w)
		}
		p.pending = keep
	}
	if at != p.Entry {
		return
	}
	arg := func(i int) uint16 { return mem.Read16(linear(ss, sp+4+uint16(2*i))) }
	retOff, retSeg := mem.Read16(linear(ss, sp)), mem.Read16(linear(ss, sp+2))
	strOff, strSeg := arg(0), arg(1)
	bg, fg, row, col := int(arg(2)&0xFF), int(arg(3)&0xFF), int(arg(4)&0xFF), int(arg(5)&0xFF)
	base := linear(strSeg, strOff)
	n := int(mem.Read8(base))
	text := make([]byte, n)
	for i := range text {
		text[i] = mem.Read8(base + 1 + uint32(i))
	}
	if p.OnText != nil {
		p.OnText(text)
	}
	h := Hit{Step: step, Caller: fmt.Sprintf("%04X:%04X", retSeg, retOff), Length: n, Row: row, Col: col, BG: bg, FG: fg, Str: base}
	var added *xlate.Stamp
	defer func() {
		p.observeQuiz(text, row, col, bg, added)
		// 原版照原文印字的範圍，舊覆繪在那裡失效（規格 001 §2.2）：
		// 換畫面後舊 Stamp 的補白格會蓋住新印的字（查驗題的字序數字落在舊「Name」欄上）。
		if !h.Stamped && n > 0 && col*8+n*8 <= 320 && row*8+8 <= 200 {
			p.Layer.Clear(col*8, row*8, (col+n)*8, row*8+8)
		}
		p.Hits = append(p.Hits, h)
	}()
	if n == 0 {
		h.Rejected = "空字串"
		return
	}
	// lead 是前導空白數：清單列以「兩格縮排＋名字＋空白補到固定寬度」呼叫，反白列不帶縮排與補白。
	// 完整字串查不到時去頭尾空白再查；Stamp 從 col+lead 起，涵蓋到原字串結尾（規格 001 §2.2）。
	if l, ok := p.Catalog.(playerLearner); ok && inPartyList(row, col) {
		l.LearnPlayer(text, sexOf(mem.Read8(base+recordSexOff))) // 規格 013：隊伍欄只印隊員名字，指標是角色記錄
	}
	lead, key := 0, text
	e, ok := p.Catalog.Lookup(text)
	if !ok {
		trimmed := bytes.TrimRight(bytes.TrimLeft(text, " "), " ")
		if len(trimmed) > 0 && len(trimmed) < len(text) {
			if e, ok = p.Catalog.Lookup(trimmed); ok {
				lead, key = len(text)-len(bytes.TrimLeft(text, " ")), trimmed
			}
		}
	}
	if !ok {
		h.Rejected = "未命中"
		return
	}
	h.Key = e.Key
	runes := p.pack(e.Translation)
	cells := n - lead
	if len(runes) > cells {
		h.Rejected = "譯文超過原文格數"
		return
	}
	if col*8+n*8 > 320 || row*8+8 > 200 {
		h.Rejected = "超出畫面"
		return
	}
	for len(runes) < cells {
		runes = append(runes, ' ')
	}
	st := &xlate.Stamp{
		Key:   e.Key,
		X:     (col + lead) * 8,
		Y:     row * 8,
		Cells: cells,
		CellW: 8,
		CellH: 8,
		Font:  p.Font,
		Text:  runes,
		State: xlate.Printing,
	}
	p.Layer.Add(st)
	if p.bgOf == nil {
		p.bgOf = map[*xlate.Stamp]uint8{}
	}
	p.bgOf[st] = uint8(bg)
	if p.src == nil {
		p.src = map[*xlate.Stamp][]byte{}
	}
	p.src[st] = key
	if len(p.src) > 2*len(p.Layer.Stamps)+512 {
		p.prune()
	}
	p.pending = append(p.pending, pendingStamp{ret: linear(retSeg, retOff), ss: ss, sp: sp + 4 + argBytes, stamp: st})
	h.Stamped = true
	added = st
}

// PrepareFrame 在 Layer.Frame 之前呼叫。xlate 定色取區域內最多的顏色當背景；短字串
// 墨跡多於底色時會判反（2026-10-15「Human」列實測）。planar 畫面的 indexed 值就是
// attribute，與呼叫參數的 bg 同一個空間：最多的顏色不是 bg 時設 SwapColors。
func (p *PathA) PrepareFrame(indexed []uint8, w, h int) {
	for st, bg := range p.bgOf {
		if st.State != xlate.Pending {
			if st.State == xlate.Shown {
				delete(p.bgOf, st)
			}
			continue
		}
		x0, y0, x1, y1 := st.Rect()
		var region []uint8
		for y := y0; y < y1 && y < h; y++ {
			for x := x0; x < x1 && x < w; x++ {
				region = append(region, indexed[y*w+x])
			}
		}
		most, second := xlate.Colors(region)
		st.SwapColors = most != bg && second == bg
	}
}

// relang 換語言後重譯仍在 Layer 上的路徑 A 疊字（主專案規格 015 §2）：新譯文放不下或查不到的移除，露出原版。
func (p *PathA) relang(cat Lookuper, font *xlate.Font, f QuizFormat) {
	p.Catalog, p.Font, p.QuizFormat = cat, font, f
	shown := map[*xlate.Stamp]bool{}
	for _, s := range p.Layer.Stamps {
		shown[s] = true
	}
	drop := map[*xlate.Stamp]bool{}
	for st, key := range p.src {
		if !shown[st] {
			delete(p.src, st)
			delete(p.bgOf, st)
			continue
		}
		e, ok := cat.Lookup(key)
		runes := p.pack(e.Translation)
		if !ok || len(runes) > st.Cells {
			drop[st] = true
			delete(p.src, st)
			continue
		}
		for len(runes) < st.Cells {
			runes = append(runes, ' ')
		}
		st.Text, st.Font = runes, font
	}
	if len(drop) > 0 {
		keep := p.Layer.Stamps[:0]
		for _, s := range p.Layer.Stamps {
			if !drop[s] {
				keep = append(keep, s)
			}
		}
		p.Layer.Stamps = keep
	}
	p.relangQuiz()
}

// prune 丟掉已不在 Layer 上的 Stamp 紀錄（長時間遊玩時 src、bgOf 不無限成長）。
func (p *PathA) prune() {
	shown := map[*xlate.Stamp]bool{}
	for _, s := range p.Layer.Stamps {
		shown[s] = true
	}
	for st := range p.src {
		if !shown[st] {
			delete(p.src, st)
			delete(p.bgOf, st)
		}
	}
}
