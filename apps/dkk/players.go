package dkk

import (
	"bytes"
	"regexp"
)

// 規格 013：玩家角色名自動音譯。名字從冒險畫面隊伍欄（路徑 A 列 4–9、欄 17）學到，
// 之後任何路徑整串等於該名字時（含組句的 {name}）都換成音譯。

// 隊伍欄的位置：全部盤點裡只印過隊員名字（規格 013 §2）。
const (
	partyListCol                 = 17
	partyListRow0, partyListRow1 = 4, 9
)

// playerNameShape 是可學的名字：2–15 字元，只含大寫字母、空白、撇號與連字號。
var playerNameShape = regexp.MustCompile(`^[A-Z][A-Z' -]{1,14}$`)

// PlayerNames 記學到的玩家名與音譯。Translit 回音譯結果；ok 為 false 時不學。
type PlayerNames struct {
	Translit func(name string) (zh string, ok bool)
	names    map[string]string
}

// learn 記一個名字；回是否新學到。
func (p *PlayerNames) learn(s string) bool {
	if _, ok := p.names[s]; ok || !playerNameShape.MatchString(s) {
		return false
	}
	zh, ok := p.Translit(s)
	if !ok || zh == "" {
		return false
	}
	if p.names == nil {
		p.names = map[string]string{}
	}
	p.names[s] = zh
	return true
}

// Lookup 實作 Lookuper：整串等於學到的名字才命中（不去空白；路徑 A 會自己去頭尾空白再查）。
func (p *PlayerNames) Lookup(original []byte) (Entry, bool) {
	zh, ok := p.names[string(original)]
	if !ok {
		return Entry{}, false
	}
	return Entry{Key: "name:" + string(original), Translation: zh}, true
}

// LearnPlayer 由路徑 A 在隊伍欄印字時呼叫：去頭尾空白後，UI 目錄與名稱表都查不到才學成玩家名，
// 所以和怪物、人物同名時沿用名稱表的譯名（規格 013 §3.3）。
func (t *Translator) LearnPlayer(text []byte) {
	if t.Players == nil {
		return
	}
	s := bytes.TrimSpace(text)
	if len(s) == 0 {
		return
	}
	for _, c := range t.Exact {
		if _, ok := c.Lookup(s); ok {
			return
		}
	}
	if t.Names != nil {
		if _, ok := t.Names.Lookup(s); ok {
			return
		}
	}
	t.Players.learn(string(s))
}

// playerLearner 是路徑 A 的查表若能學名字時實作的介面。
type playerLearner interface{ LearnPlayer(text []byte) }

func inPartyList(row, col int) bool {
	return col == partyListCol && row >= partyListRow0 && row <= partyListRow1
}
