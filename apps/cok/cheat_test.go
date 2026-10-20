package cok

import "testing"

// fakeParty 在 64 KB 以上擺三筆紀錄：兩名隊員、一隻怪物，串成隊伍串列。
func fakeParty() map[uint32]uint8 {
	mem := map[uint32]uint8{}
	put := func(a uint32, seg uint16) {
		mem[a], mem[a+1], mem[a+2], mem[a+3] = 0, 0, uint8(seg), uint8(seg>>8)
	}
	put(partyHead, 0x5219)       // 隊員 1：0x52190
	put(0x52190+recNext, 0x5233) // 隊員 2：0x52330
	put(0x52330+recNext, 0x57BA) // 怪物：0x57BA0
	mem[0x57BA0+recTeam] = 1
	put(0x57BA0+recNext, 0x5219) // 怪物再指回隊員 1，走訪要在怪物就停
	return mem
}

func TestInvinciblePartyWalk(t *testing.T) {
	mem := fakeParty()
	peek := func(a uint32) uint8 { return mem[a] }
	got := party(peek)
	if len(got) != 2 || got[0] != 0x52190 || got[1] != 0x52330 {
		t.Fatalf("隊伍串列走訪 %x，要 52190、52330（遇怪物停）", got)
	}
	// 迴圈：隊員 2 指回隊員 1，不可無限走
	mem[0x52330+recNext+2], mem[0x52330+recNext+3] = 0x19, 0x52
	if got := party(peek); len(got) != 2 {
		t.Fatalf("迴圈串列走訪 %x，要兩筆", got)
	}
}

func TestInvincibleGuard(t *testing.T) {
	mem := fakeParty()
	peek := func(a uint32) uint8 { return mem[a] }
	c := &Invincible{}
	if lo, hi := c.GuardRange(peek); lo <= hi {
		t.Fatal("關閉時不監看")
	}
	if c.GuardWrite(0x52190+recHP, 50, 40) {
		t.Fatal("關閉時不還原")
	}
	c.On = true
	lo, hi := c.GuardRange(peek)
	if lo != 0x52190+recStatus || hi != 0x52330+recHP {
		t.Fatalf("範圍 %x–%x", lo, hi)
	}
	cases := []struct {
		addr     uint32
		old, new uint8
		restore  bool
		why      string
	}{
		{0x52190 + recHP, 50, 40, true, "扣血"},
		{0x52330 + recHP, 50, 51, false, "回血"},
		{0x52190 + recStatus, 0, 4, true, "正常變異常"},
		{0x52190 + recStatus, 6, 0, false, "復活"},
		{0x52190 + recTeam, 0, 1, false, "其他欄位"},
		{0x57BA0 + recHP, 9, 0, false, "怪物"},
	}
	for _, k := range cases {
		if got := c.GuardWrite(k.addr, k.old, k.new); got != k.restore {
			t.Errorf("%s：還原 %v，要 %v", k.why, got, k.restore)
		}
	}
}
