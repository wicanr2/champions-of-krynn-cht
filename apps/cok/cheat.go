package cok

import "fmt"

// 作弊：我方無敵（主專案規格 017，欄位見主專案 docs/re/010）。開啟時，遊戲寫入我方紀錄的目前 HP 變小、
// 健康狀態由 0 變成非 0、陣營由我方變成其他（魅惑），都立即還原；每格開頭把我方效果串列裡的有害效果
// （麻痺、睡眠等）解除連結。回復、治療與有益效果照常。關閉時不監看、不改任何位址，與原版相同。

const (
	partyHead    = 0x0CD6*16 + 0x609A // DS:609A 隊伍串列頭（遠指標）
	recNext      = 0x17F              // 下一位紀錄（遠指標）；戰鬥中會接到怪物
	recAffects   = 0xE3               // 效果串列頭（遠指標）；每筆 9 bytes，+0 類型、+5 下一筆
	affectNext   = 5
	maxAffects   = 64    // 防壞掉的串列
	recStatus    = 0x188 // 健康狀態，0 正常
	recTeam      = 0x18A // 陣營，我方 0
	recHP        = 0x197 // 目前 HP
	maxPartyRecs = 12    // 隊員 6 名加 NPC，留餘裕；也擋住壞掉的串列
)

// harmfulAffect 是要從我方移除的效果類型。0x34 麻痺為本作實機證實（docs/re/010）；其餘取自同引擎
// 《詛咒藍符》反組譯（失能 0x1F、失明 0x21、緩慢 0x2A、蛇咒 0x33、睡眠 0x35），本作等級為假說。
var harmfulAffect = map[uint8]bool{0x1F: true, 0x21: true, 0x2A: true, 0x33: true, 0x34: true, 0x35: true}

// Memory 是無敵在每格開頭直接讀寫記憶體的介面（session.GuardMemory 同一組方法）。
type Memory interface {
	Peek8(addr uint32) uint8
	Poke8(addr uint32, v uint8)
}

// Invincible 是無敵開關與這一輪要守的我方紀錄。
type Invincible struct {
	On       bool
	recs     []uint32 // 我方紀錄的線性位址（GuardRange 時更新）
	Report   string   // 最後一次 GuardRange 看到的隊員「名字 HP/狀態」，收據用
	Restored int      // 還原過幾次寫入
	Removed  int      // 解除連結過幾筆有害效果
}

func far(peek func(uint32) uint8, a uint32) uint32 {
	off := uint32(peek(a)) | uint32(peek(a+1))<<8
	seg := uint32(peek(a+2)) | uint32(peek(a+3))<<8
	return seg*16 + off
}

// stripAffects 把紀錄 rec 效果串列中的有害效果解除連結：前一個連結改指向該筆的下一筆。
// 被移除的 9 bytes 留在原處不釋放（Turbo Pascal 堆積的小洩漏，換取不碰配置器）。
func (c *Invincible) stripAffects(mem Memory, rec uint32) {
	link := rec + recAffects
	for n := 0; n < maxAffects; n++ {
		node := far(mem.Peek8, link)
		if node == 0 || node+affectNext+4 > 0xA0000 {
			return
		}
		if !harmfulAffect[mem.Peek8(node)] {
			link = node + affectNext
			continue
		}
		for i := uint32(0); i < 4; i++ {
			mem.Poke8(link+i, mem.Peek8(node+affectNext+i))
		}
		c.Removed++
	}
}

// party 走訪隊伍串列，回陣營為我方的紀錄；遇到怪物、空指標、位址超界或重複就停。
func party(peek func(uint32) uint8) []uint32 {
	var out []uint32
	seen := map[uint32]bool{}
	for a := far(peek, partyHead); a != 0 && a+recHP < 0xA0000 && !seen[a] && len(out) < maxPartyRecs; a = far(peek, a+recNext) {
		if peek(a+recTeam) != 0 {
			break
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// GuardRange 實作 session.WriteGuard：先移除我方有害效果，再回要監看的範圍；關閉或沒有隊員時回空範圍。
func (c *Invincible) GuardRange(mem Memory) (lo, hi uint32) {
	if c == nil || !c.On {
		return 1, 0
	}
	peek := mem.Peek8
	c.recs = party(peek)
	for _, a := range c.recs {
		c.stripAffects(mem, a)
	}
	c.Report = ""
	for _, a := range c.recs {
		name := make([]byte, min(int(peek(a)), 15))
		for i := range name {
			name[i] = peek(a + 1 + uint32(i))
		}
		c.Report += fmt.Sprintf("%s %d/%d；", name, peek(a+recHP), peek(a+recStatus))
	}
	if len(c.recs) == 0 {
		return 1, 0
	}
	lo, hi = ^uint32(0), 0
	for _, a := range c.recs {
		lo, hi = min(lo, a+recStatus), max(hi, a+recHP)
	}
	return lo, hi
}

// GuardWrite 實作 session.WriteGuard：HP 變小、狀態由正常變異常、陣營由我方變成其他時回 true（還原）。
func (c *Invincible) GuardWrite(addr uint32, old, new uint8) bool {
	if c == nil || !c.On {
		return false
	}
	for _, a := range c.recs {
		restore := false
		switch addr - a {
		case recHP:
			restore = new < old
		case recStatus:
			restore = old == 0 && new != 0
		case recTeam:
			restore = old == 0 && new != 0 // 魅惑：我方被改成敵方
		default:
			continue
		}
		if restore {
			c.Restored++
		}
		return restore
	}
	return false
}
