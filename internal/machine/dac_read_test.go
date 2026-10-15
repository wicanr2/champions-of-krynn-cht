package machine

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestDACReadPortSelectsRGBAndAdvances(t *testing.T) {
	m := New()
	m.Out8(0x3C8, 5)
	for _, v := range []uint8{10, 20, 30, 40, 50, 60} {
		m.Out8(0x3C9, v)
	}
	writeIndex, writePhase := m.dacIndex, m.dacPhase
	m.Out8(0x3C7, 5)
	for i, want := range []uint8{10, 20, 30, 40, 50, 60} {
		if got := m.In8(0x3C9); got != want {
			t.Fatalf("DAC 第 %d 個分量 = %d，應為 %d", i, got, want)
		}
	}
	if m.dacIndex != writeIndex || m.dacPhase != writePhase {
		t.Fatalf("讀 DAC 改動寫入游標：%d/%d → %d/%d", writeIndex, writePhase, m.dacIndex, m.dacPhase)
	}
	m.Out8(0x3C7, 6)
	if got := m.In8(0x3C9); got != 40 {
		t.Fatalf("重新選色 6 後首項 = %d，應為 40", got)
	}
	m.DAC[255*3], m.DAC[0] = 23, 31
	m.Out8(0x3C7, 255)
	for i := 0; i < 3; i++ {
		got := m.In8(0x3C9)
		if i == 0 && got != 23 {
			t.Fatalf("末色 R = %d，應為 23", got)
		}
	}
	if got := m.In8(0x3C9); got != 31 {
		t.Fatalf("索引應由 255 回到 0，讀得 %d", got)
	}
}

func TestDACReadPositionSurvivesSnapshotsAndStateFiles(t *testing.T) {
	m := New()
	m.DAC[5*3], m.DAC[5*3+1], m.DAC[5*3+2] = 10, 20, 30
	m.Out8(0x3C7, 5)
	if got := m.In8(0x3C9); got != 10 {
		t.Fatalf("快照前 R = %d", got)
	}
	snap := m.Snapshot()
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatal(err)
	}
	saved := append([]byte(nil), buf.Bytes()...)
	m.Out8(0x3C7, 0)
	m.Restore(snap)
	if got := m.In8(0x3C9); got != 20 {
		t.Fatalf("快照後應接續 G，讀得 %d", got)
	}
	m2 := New()
	if err := m2.LoadState(bytes.NewReader(saved)); err != nil {
		t.Fatal(err)
	}
	if got := m2.In8(0x3C9); got != 20 {
		t.Fatalf("狀態檔後應接續 G，讀得 %d", got)
	}

	// 舊版狀態檔只知道已讀過 DAC，沒有記下相位；須在改動目標前拒絕。
	var old machineState
	if err := gob.NewDecoder(bytes.NewReader(saved)).Decode(&old); err != nil {
		t.Fatal(err)
	}
	old.DACReadKnown = false
	var oldBytes bytes.Buffer
	if err := gob.NewEncoder(&oldBytes).Encode(&old); err != nil {
		t.Fatal(err)
	}
	m3 := New()
	m3.DAC[0] = 42
	if err := m3.LoadState(&oldBytes); err == nil {
		t.Fatal("舊版 DAC 讀取狀態不應默默載入")
	}
	if m3.DAC[0] != 42 {
		t.Fatal("拒絕舊檔後機器已被部分改動")
	}
}
