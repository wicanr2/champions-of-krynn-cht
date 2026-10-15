package dos

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// 滑鼠驅動狀態在三條分支各存在不同地方：stubseg 分支放 MouseExtra，
// logh3 與 fix/dm-boot 放 Mouse 本身。合併後同一份寫兩處，讀的時候
// 三種舊檔都要回得來（`docs/spec/196-mouse-handler-survives-state-file`、
// `docs/spec/197-mouse-handler-in-saved-state`）。
func TestLoadStateAcceptsEveryMouseStateLayout(t *testing.T) {
	_, d := newTest(t)
	d.Mouse.Handler.Seg, d.Mouse.Handler.Off = 0x36AD, 0x021D
	d.Mouse.Handler.Mask, d.Mouse.Handler.Set = 0x1F, true
	d.Mouse.MinX, d.Mouse.MaxX, d.Mouse.MinY, d.Mouse.MaxY = 8, 0x277, 4, 0x18B
	d.Mouse.PressAt[0] = [2]uint16{300, 170}
	d.Mouse.ReleaseAt[1] = [2]uint16{12, 34}
	d.Mouse.MickeyX, d.Mouse.MickeyY = -3, 7
	var buf bytes.Buffer
	if err := d.SaveState(&buf); err != nil {
		t.Fatal(err)
	}
	var saved dosState
	if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&saved); err != nil {
		t.Fatal(err)
	}

	layouts := map[string]func(s *dosState){
		"兩處都有": func(s *dosState) {},
		"只有 MouseExtra（stubseg）": func(s *dosState) {
			s.Mouse = Mouse{X: s.Mouse.X, Y: s.Mouse.Y, Buttons: s.Mouse.Buttons,
				Press: s.Mouse.Press, Release: s.Mouse.Release, XScale: s.Mouse.XScale}
		},
		"只有 Mouse（logh3、fix/dm-boot）": func(s *dosState) { s.MouseExtra = mouseExtra{} },
	}
	for name, strip := range layouts {
		t.Run(name, func(t *testing.T) {
			s := saved
			strip(&s)
			var b bytes.Buffer
			if err := gob.NewEncoder(&b).Encode(&s); err != nil {
				t.Fatal(err)
			}
			_, e := newTest(t)
			if err := e.LoadState(&b); err != nil {
				t.Fatal(err)
			}
			if e.Mouse.Handler != d.Mouse.Handler {
				t.Errorf("事件常式 %+v，要 %+v", e.Mouse.Handler, d.Mouse.Handler)
			}
			if e.Mouse.MinX != 8 || e.Mouse.MaxX != 0x277 || e.Mouse.MinY != 4 || e.Mouse.MaxY != 0x18B {
				t.Errorf("範圍 %d–%d × %d–%d 沒還原", e.Mouse.MinX, e.Mouse.MaxX, e.Mouse.MinY, e.Mouse.MaxY)
			}
			if e.Mouse.PressAt != d.Mouse.PressAt || e.Mouse.ReleaseAt != d.Mouse.ReleaseAt {
				t.Errorf("按放位置 %v／%v 沒還原", e.Mouse.PressAt, e.Mouse.ReleaseAt)
			}
		})
	}

	// mickey 只存在 Mouse 本身；完整往返要留住（197 §2）。
	_, e := newTest(t)
	if err := e.LoadState(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if e.Mouse.MickeyX != -3 || e.Mouse.MickeyY != 7 {
		t.Errorf("相對位移 (%d,%d)，要 (-3,7)", e.Mouse.MickeyX, e.Mouse.MickeyY)
	}
}
