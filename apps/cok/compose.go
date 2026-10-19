package cok

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Lookuper 以原字串 bytes 查譯文。Catalog 與 Translator 都滿足。
type Lookuper interface {
	Lookup(original []byte) (Entry, bool)
}

// Translator 在 catalog 未命中之後依規格 004 組出譯文：名稱、數字模板、標籤、物品名。
type Translator struct {
	Exact     []Lookuper   // 依序查的 catalog
	Names     Lookuper     // 怪物、人物、地名（可為 nil）
	Words     Lookuper     // 物品名詞組來源（介面 catalog；可為 nil）
	Menus     *MenuMasters // 底列選單母串（規格 003 §2.2.1；可為 nil）
	Players   *PlayerNames // 玩家名音譯（規格 013；可為 nil）
	templates map[string]string
	named     []namedTemplate // 含 {name} 的模板（規格 004 §2.1.1），依表中順序
}

// namedTemplate 是含 {name} 的模板：re 的子群組依 kinds 為名字（"name"）或數字（"n"）。
type namedTemplate struct {
	re    *regexp.Regexp
	kinds []string
	zh    string
}

var templateToken = regexp.MustCompile(`\{name\}|\{n\}`)

// compileNamed 把模板轉成整串比對的正規式：{name} 取最短非空子字串，{n} 取連續數字，其餘逐字。
func compileNamed(tpl string) (*regexp.Regexp, []string) {
	var b strings.Builder
	var kinds []string
	b.WriteString("^")
	last := 0
	for _, m := range templateToken.FindAllStringIndex(tpl, -1) {
		b.WriteString(regexp.QuoteMeta(tpl[last:m[0]]))
		if tpl[m[0]:m[1]] == "{name}" {
			b.WriteString("(.+?)")
			kinds = append(kinds, "name")
		} else {
			b.WriteString("([0-9]+)")
			kinds = append(kinds, "n")
		}
		last = m[1]
	}
	b.WriteString(regexp.QuoteMeta(tpl[last:]) + "$")
	return regexp.MustCompile(b.String()), kinds
}

var digits = regexp.MustCompile(`[0-9]+`)
var modifier = regexp.MustCompile(`^([+-][0-9]+|vs\.)$`)
var placeholder = regexp.MustCompile(`\{([0-9]*)n?\}`)

// LoadTemplates 讀 text/composed-templates.tsv（template、translation、note）。
func (t *Translator) LoadTemplates(b []byte) error {
	rows, err := readTSV(b, []string{"template", "translation"})
	if err != nil {
		return fmt.Errorf("cok: 模板 TSV：%w", err)
	}
	t.templates = make(map[string]string, len(rows))
	for _, r := range rows {
		if _, dup := t.templates[r[0]]; dup {
			return fmt.Errorf("cok: 模板重複：%q", r[0])
		}
		t.templates[r[0]] = r[1]
		if strings.Contains(r[0], "{name}") {
			re, kinds := compileNamed(r[0])
			t.named = append(t.named, namedTemplate{re: re, kinds: kinds, zh: r[1]})
		}
	}
	return nil
}

// MenuWords 回選單詞列在母串表中的逐詞譯文（規格 003 §2.2.1）。
func (t *Translator) MenuWords(words []string) []string { return t.Menus.Words(words) }

// Lookup 實作 Lookuper（規格 004 §2）。
func (t *Translator) Lookup(original []byte) (Entry, bool) {
	if e, ok := t.exact(original); ok {
		return e, true
	}
	if e, ok := t.template(string(original)); ok {
		return e, true
	}
	if e, ok := t.label(string(original)); ok {
		return e, true
	}
	if e, ok := t.readyRow(string(original)); ok {
		return e, true
	}
	if e, ok := t.item(string(original)); ok {
		return e, true
	}
	if e, ok := t.pricedRow(string(original)); ok {
		return e, true
	}
	if e, ok := t.buys(string(original)); ok {
		return e, true
	}
	// 數量＋物品整串（戰鬥訊息的投擲武器 `3 Darts`，docs/re/008 跨區盤點）：數量置後
	if s := strings.TrimSpace(string(original)); quantityRe.MatchString(s) {
		if zh, ok := t.goodsName(s); ok {
			return Entry{Key: "goods", Translation: zh}, true
		}
	}
	// 詞組：不以空白結尾、兩詞以上、每段都是介面詞組（例 `Paladine Cleric` 神祇＋職業；規格 004 §2.2）。
	if s := string(original); !strings.HasSuffix(s, " ") && len(strings.Fields(s)) >= 2 {
		if e, ok := t.item(s + " "); ok {
			return e, true
		}
	}
	// 反白列不帶清單列的前導空白，catalog 收的是帶空白的寫法（例 ` <No Spell Effects>`）。
	if len(original) > 0 && original[0] != ' ' {
		return t.exact(append([]byte{' '}, original...))
	}
	return Entry{}, false
}

var readyFlag = regexp.MustCompile(`^( *)(Yes|No) +(\S.*?) *$`)

// readyRow：物品清單列「Yes／No（備妥旗標）＋物品名」（規格 004 §2.6）。旗標譯為「是」「否」，物品名走 item。
func (t *Translator) readyRow(s string) (Entry, bool) {
	m := readyFlag.FindStringSubmatch(s)
	if m == nil {
		return Entry{}, false
	}
	// 品名可帶數量（本作 ` Yes  60 Arrows `，docs/re/008）：走 goodsName，數量置後
	it, ok := t.goodsName(m[3])
	if !ok {
		return Entry{}, false
	}
	flag := map[string]string{"Yes": "是", "No": "否"}[m[2]]
	return Entry{Key: "ready", Translation: m[1] + flag + " " + it}, true // 前導空白保留，與反白列對齊
}

func (t *Translator) exact(b []byte) (Entry, bool) {
	for _, c := range t.Exact {
		if e, ok := c.Lookup(b); ok {
			return e, true
		}
	}
	if t.Names != nil {
		if e, ok := t.Names.Lookup(b); ok {
			return e, true
		}
		if s := strings.TrimSpace(string(b)); s != string(b) && s != "" {
			if e, ok := t.Names.Lookup([]byte(s)); ok {
				return e, true
			}
		}
	}
	if t.Players != nil {
		if e, ok := t.Players.Lookup(b); ok {
			return e, true
		}
	}
	return Entry{}, false
}

// template：連續數字換成 {n} 後整串比對；譯文的 {n} 依序代回，{k} 取第 k 個數字。
func (t *Translator) template(s string) (Entry, bool) {
	if t.templates == nil {
		return Entry{}, false
	}
	nums := digits.FindAllString(s, -1)
	if len(nums) == 0 {
		// 沒有數字的模板是串接片段的整句譯法（規格 004 §2.1，例 `Spells in Grimoire`）。
		if zh, ok := t.templates[s]; ok {
			return Entry{Key: "template", Translation: zh}, true
		}
		return t.namedMatch(s)
	}
	zh, ok := t.templates[digits.ReplaceAllString(s, "{n}")]
	if !ok {
		return t.namedMatch(s)
	}
	return Entry{Key: "template", Translation: fillNumbers(zh, nums)}, true
}

// namedMatch 試含 {name} 的模板（規格 004 §2.1.1）：去掉前導空白後整串比對，前導空白保留在譯文前。
func (t *Translator) namedMatch(s string) (Entry, bool) {
	body := strings.TrimLeft(s, " ")
	lead := s[:len(s)-len(body)]
	for _, nt := range t.named {
		m := nt.re.FindStringSubmatch(body)
		if m == nil {
			continue
		}
		var nums []string
		name := ""
		for i, k := range nt.kinds {
			if k == "n" {
				nums = append(nums, m[i+1])
			} else if name == "" {
				name = m[i+1]
				if ne, ok := t.exact([]byte(name)); ok {
					name = ne.Translation // 介面詞表（職業名等）或名稱表命中才譯，玩家角色名照原樣
				}
			}
		}
		zh := strings.ReplaceAll(nt.zh, "{name}", name)
		return Entry{Key: "template", Translation: lead + fillNumbers(zh, nums)}, true
	}
	return Entry{}, false
}

// fillNumbers 把譯文的 {n} 依序代回數字，{k} 取第 k 個數字（規格 004 §2.1）。
func fillNumbers(zh string, nums []string) string {
	next := 0
	out := placeholder.ReplaceAllStringFunc(zh, func(m string) string {
		k := strings.TrimSuffix(strings.TrimPrefix(m, "{"), "}")
		if k == "n" {
			if next >= len(nums) {
				return m
			}
			next++
			return nums[next-1]
		}
		if i, err := strconv.Atoi(k); err == nil && i >= 1 && i <= len(nums) {
			return nums[i-1]
		}
		return m
	})
	return out
}

// label：「前段:後段」兩段各自查表（後段可走名稱或物品名），都命中才組成。
func (t *Translator) label(s string) (Entry, bool) {
	i := strings.IndexByte(s, ':')
	if i <= 0 || i == len(s)-1 {
		return Entry{}, false
	}
	pre, ok := t.exact([]byte(s[:i+1]))
	if !ok {
		return Entry{}, false
	}
	rest, ok := t.exact([]byte(s[i+1:]))
	if !ok {
		if rest, ok = t.item(s[i+1:] + " "); !ok {
			return Entry{}, false
		}
	}
	p := strings.TrimRight(pre.Translation, ":：")
	return Entry{Key: "label", Translation: p + "：" + rest.Translation}, true
}

// item：只處理以空白結尾的字串（物品名的印法）；從頭貪婪取最長命中詞組，加值照原樣置後。
func (t *Translator) item(s string) (Entry, bool) {
	if t.Words == nil || !strings.HasSuffix(s, " ") {
		return Entry{}, false
	}
	tok := strings.Fields(s)
	if len(tok) == 0 {
		return Entry{}, false
	}
	var b strings.Builder
	phrases := 0
	for i := 0; i < len(tok); {
		if modifier.MatchString(tok[i]) {
			b.WriteString(" " + tok[i])
			i++
			continue
		}
		found := false
		for j := len(tok); j > i; j-- {
			if e, ok := t.Words.Lookup([]byte(strings.Join(tok[i:j], " "))); ok {
				b.WriteString(e.Translation)
				phrases++
				i, found = j, true
				break
			}
		}
		if !found {
			return Entry{}, false
		}
	}
	if phrases == 0 {
		return Entry{}, false
	}
	return Entry{Key: "item", Translation: strings.TrimSpace(b.String())}, true
}

// 本作商店清單（本專案 docs/re/004 商店盤點）：一列是「品名＋空白補位＋價格」整串印出，例 `Long Sword                   7`；
// 品名可帶數量前綴（`20 Arrows`）。譯成「長劍＋補位＋7」，價格仍靠右對齊在原欄位。
var (
	pricedRowRe = regexp.MustCompile(`^(\S.*?\S) {2,}([0-9]+)$`)
	quantityRe  = regexp.MustCompile(`^([0-9]+) (.+)$`)
	buysRe      = regexp.MustCompile(`^(.+?) buys (?:a |an )?(.+?) ?$`)
)

// goodsName 譯品名：數量前綴置後（「箭矢 20」，依物品名加值置後的原則）。
func (t *Translator) goodsName(name string) (string, bool) {
	q := ""
	if m := quantityRe.FindStringSubmatch(name); m != nil {
		q, name = m[1], m[2]
	}
	var zh string
	found := false
	// 數量大於 1 時引擎在詞尾加 s（`20 Arrows`），查不到就試單數
	for _, n := range []string{name, strings.TrimSuffix(name, "s")} {
		if e, ok := t.item(n + " "); ok {
			zh, found = e.Translation, true
		} else if e, ok := t.exact([]byte(n)); ok {
			zh, found = strings.TrimSpace(e.Translation), true
		}
		if found {
			break
		}
	}
	if !found {
		return "", false
	}
	if q != "" {
		zh += " " + q
	}
	return zh, true
}

func (t *Translator) pricedRow(s string) (Entry, bool) {
	m := pricedRowRe.FindStringSubmatch(s)
	if m == nil {
		return Entry{}, false
	}
	zh, ok := t.goodsName(m[1])
	if !ok {
		return Entry{}, false
	}
	cells := len([]rune(s))
	best := ""
	for pad := 1; pad <= 2*cells; pad++ {
		cand := zh + strings.Repeat(" ", pad) + m[2]
		if n := len(PackHalfWidth([]rune(cand))); n == cells {
			best = cand
		} else if n > cells {
			break
		}
	}
	if best == "" {
		return Entry{}, false
	}
	return Entry{Key: "priced", Translation: best}, true
}

// buys：`<名字> buys a <物品> `（商店購買訊息）→「<名字>買下<物品>」。名字走名稱表或玩家名音譯，查不到照原文。
func (t *Translator) buys(s string) (Entry, bool) {
	m := buysRe.FindStringSubmatch(s)
	if m == nil {
		return Entry{}, false
	}
	zh, ok := t.goodsName(m[2])
	if !ok {
		return Entry{}, false
	}
	name := m[1]
	if t.Players != nil {
		if e, ok := t.Players.Lookup([]byte(name)); ok {
			name = e.Translation
		}
	}
	return Entry{Key: "buys", Translation: name + "買下" + zh}, true
}
