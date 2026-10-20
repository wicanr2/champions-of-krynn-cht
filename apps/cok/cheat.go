package cok

import "fmt"

// 作弊：我方無敵（主專案規格 017，欄位見主專案 docs/re/010）。開啟時，遊戲寫入我方紀錄的目前 HP 變小、
// 健康狀態由 0 變成非 0，都立即還原；回復與治療照常。關閉時不監看任何位址，與原版相同。

const (
	partyHead    = 0x0CD6*16 + 0x609A // DS:609A 隊伍串列頭（遠指標）
	recNext      = 0x17F              // 下一位紀錄（遠指標）；戰鬥中會接到怪物
	recStatus    = 0x188              // 健康狀態，0 正常
	recTeam      = 0x18A              // 陣營，我方 0
	recHP        = 0x197              // 目前 HP
	maxPartyRecs = 12                 // 隊員 6 名加 NPC，留餘裕；也擋住壞掉的串列
)

// Invincible 是無敵開關與這一輪要守的我方紀錄。
type Invincible struct {
	On       bool
	recs     []uint32 // 我方紀錄的線性位址（GuardRange 時更新）
	Report   string   // 最後一次 GuardRange 看到的隊員「名字 HP/狀態」，收據用
	Restored int      // 還原過幾次寫入
}

// party 走訪隊伍串列，回陣營為我方的紀錄；遇到怪物、空指標、位址超界或重複就停。
func party(peek func(uint32) uint8) []uint32 {
	far := func(a uint32) uint32 {
		off := uint32(peek(a)) | uint32(peek(a+1))<<8
		seg := uint32(peek(a+2)) | uint32(peek(a+3))<<8
		return seg*16 + off
	}
	var out []uint32
	seen := map[uint32]bool{}
	for a := far(partyHead); a != 0 && a+recHP < 0xA0000 && !seen[a] && len(out) < maxPartyRecs; a = far(a + recNext) {
		if peek(a+recTeam) != 0 {
			break
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// GuardRange 實作 session.WriteGuard：關閉或沒有隊員時回空範圍。
func (c *Invincible) GuardRange(peek func(uint32) uint8) (lo, hi uint32) {
	if c == nil || !c.On {
		return 1, 0
	}
	c.recs = party(peek)
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

// GuardWrite 實作 session.WriteGuard：HP 變小、狀態由正常變異常時回 true（還原）。
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
