package xlate

import "testing"

// 呼叫端自己定的顏色在色盤不變時要留著，色盤真的變了才跟著原版走。
//
// 兩條分支各自只測了一半：psychic-war 測「色盤變了要跟」
// （TestShownPaletteRecoveryAndSnapshot），phantasie 測「已顯示的不重新定色」
// （apps/phantasie TestRecolorOnlyGroupsNewlyShown，它依原版字模遮罩定色，
// 結果可能與多數色規則前後景相反）。合併後兩件事同時成立才對。
func TestShownColorsKeepCallerOverrideUntilPaletteChanges(t *testing.T) {
	idx := make([]uint8, DefaultScreenW*DefaultScreenH)
	idx[8*DefaultScreenW+9] = 15
	idx[8*DefaultScreenW+17] = 15
	rgb := func(fg [3]uint8) []uint8 {
		out := make([]uint8, len(idx)*3)
		for i, c := range idx {
			if c == 15 {
				copy(out[i*3:i*3+3], fg[:])
			}
		}
		return out
	}
	grey, white := [3]uint8{128, 128, 128}, [3]uint8{255, 255, 255}

	s := &Stamp{Key: "override", X: 8, Y: 8, Cells: 2, CellW: 8, CellH: 8, State: Pending}
	l := &Layer{Stamps: []*Stamp{s}}
	l.Frame(idx, rgb(grey))
	if s.State != Shown || s.FG != grey {
		t.Fatalf("定色後 State=%v FG=%v，要 Shown、%v", s.State, s.FG, grey)
	}

	// 呼叫端把前景背景對調（phantasie 依字模遮罩定色時會這樣做）。
	s.BG, s.FG = s.FG, s.BG
	override := [2][3]uint8{s.BG, s.FG}
	for i := 0; i < 3; i++ {
		l.Frame(idx, rgb(grey))
		if [2][3]uint8{s.BG, s.FG} != override {
			t.Fatalf("色盤沒變，第 %d 幀把呼叫端的顏色蓋成 BG=%v FG=%v", i+1, s.BG, s.FG)
		}
	}

	// 色盤真的變了（淡入淡出）：前景色號的 RGB 改了，就跟著原版走。
	l.Frame(idx, rgb(white))
	if s.FG != white {
		t.Fatalf("色盤改變後 FG=%v，要跟到 %v", s.FG, white)
	}
	if s.BG != override[0] {
		t.Errorf("背景色號的 RGB 沒變，BG 不該動：%v，原本 %v", s.BG, override[0])
	}
}
