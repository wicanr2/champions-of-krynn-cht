package cok

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/wicanr2/dosgolem/xlate"
)

// 每個相位是完整色號狀態。這是來源衍生控制，不宣稱本作實際動畫時序。
func TestHDAnimationStatesAndTransition(t *testing.T) {
	live := NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	var pal [256][3]uint8
	for i := range pal {
		pal[i] = [3]uint8{uint8(i), uint8(i), uint8(i)}
	}
	states := make([][]byte, 3)
	for k := range states {
		ref := make([]byte, 88*88)
		for i := range ref {
			ref[i] = byte(i % 16)
		}
		ref[44*88+44] = byte(8 + k)
		ref[44*88+45] = byte(8 + k)
		pix := make([]byte, 264*264*4)
		for i := 0; i < len(pix); i += 4 {
			pix[i], pix[i+3] = byte(100+k), 255
		}
		it := HDItem{Key: fmt.Sprintf("phase@%d", k), X: 24, Y: 24, W: 88, H: 88, Scale: 3, Pix: pix, Reference: ref}
		live.Layer.Watch(it.Watcher())
		states[k] = ref
	}
	screen := make([]byte, 64000)
	put := func(ref []byte) {
		for y := 0; y < 88; y++ {
			copy(screen[(24+y)*320+24:], ref[y*88:(y+1)*88])
		}
	}
	for _, k := range []int{0, 1, 2, 1, 0} {
		put(states[k])
		before := append([]byte(nil), screen...)
		live.Frame(screen, pal)
		out := live.Compose(3, true, nil)
		if got := out[(24*3*960+24*3)*4]; got != byte(100+k) {
			t.Fatalf("phase %d: %d", k, got)
		}
		if !bytes.Equal(screen, before) {
			t.Fatal("原版色號被修改")
		}
		if !bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil)) {
			t.Fatal("2 倍有 HD 差異")
		}
	}
	// 半張切到下一相位，任一完整基準都不吻合，整塊立即回退。
	screen[(24+44)*320+24+44] = 9
	live.Frame(screen, pal)
	if !bytes.Equal(live.Compose(3, true, nil), live.Compose(3, false, nil)) {
		t.Fatal("過渡相位留下 HD")
	}
	put(states[2])
	live.Frame(screen, pal)
	if live.Compose(3, true, nil)[(24*3*960+24*3)*4] != 102 {
		t.Fatal("完整相位未恢復")
	}
	// 撤除所有已登記素材的負對照。
	empty := NewLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	empty.Frame(screen, pal)
	if !bytes.Equal(empty.Compose(3, true, nil), empty.Compose(3, false, nil)) {
		t.Fatal("無素材仍繪圖")
	}
}
