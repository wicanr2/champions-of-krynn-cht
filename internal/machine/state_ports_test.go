package machine

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// 同一台機器存兩次要得到同一串 bytes。埠統計若用 map 編 gob，順序每次不同，
// 狀態檔就沒辦法拿雜湊驗決定性（master 分支 `de5e2d2`）。
func TestSaveStateIsByteIdenticalWithManyPorts(t *testing.T) {
	m := New()
	for p := uint16(0x300); p < 0x340; p++ { // 埠夠多，map 走訪順序幾乎必然打亂
		m.Out8(p, uint8(p))
		m.In8(p)
	}
	var first []byte
	for i := 0; i < 8; i++ {
		var buf bytes.Buffer
		if err := m.SaveState(&buf); err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = buf.Bytes()
			continue
		}
		if !bytes.Equal(first, buf.Bytes()) {
			t.Fatalf("第 %d 次存檔與第一次不同", i+1)
		}
	}
}

// 舊狀態檔把埠統計存在 Ports／PortsIn 兩個 map 裡。換成 slice 之後
// 舊檔仍要讀得回來：同名欄位換型別的話 gob 解不開，既有收據全部作廢。
func TestLoadStateReadsLegacyPortMaps(t *testing.T) {
	src := New()
	src.Out8(0x3C8, 0x12)
	src.In8(0x3DA)
	var buf bytes.Buffer
	if err := src.SaveState(&buf); err != nil {
		t.Fatal(err)
	}
	// 把新檔改寫成舊格式：slice 清掉，內容搬回 map。
	var s machineState
	if err := gob.NewDecoder(&buf).Decode(&s); err != nil {
		t.Fatal(err)
	}
	s.Ports, s.PortsIn = map[uint16]uint8{}, map[uint16]uint64{}
	for _, pv := range s.PortList {
		s.Ports[pv.Port] = pv.Val
	}
	for _, pc := range s.PortInList {
		s.PortsIn[pc.Port] = pc.Count
	}
	s.PortList, s.PortInList = nil, nil
	var legacy bytes.Buffer
	if err := gob.NewEncoder(&legacy).Encode(&s); err != nil {
		t.Fatal(err)
	}

	m := New()
	if err := m.LoadState(&legacy); err != nil {
		t.Fatalf("舊格式讀不回來：%v", err)
	}
	if m.Ports[0x3C8] != 0x12 || m.PortsIn[0x3DA] == 0 {
		t.Fatalf("埠統計沒還原：Ports[3C8]=%02X PortsIn[3DA]=%d", m.Ports[0x3C8], m.PortsIn[0x3DA])
	}
}
