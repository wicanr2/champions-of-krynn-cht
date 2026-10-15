package dos

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
)

// 同一個 DOS 狀態存兩次要得到同一串 bytes。開著的檔、EMS handle、XMS EMB
// 在 Go 這一端都是 map，照走訪順序寫出去的話每次都不同，狀態檔就沒辦法拿
// 雜湊驗決定性。
func TestDOSSaveStateIsByteIdentical(t *testing.T) {
	_, d := newTest(t)
	d.emb = map[uint16][]byte{}
	for h := uint16(1); h <= 16; h++ {
		d.emb[h] = []byte{byte(h), byte(h >> 8)}
	}
	d.ems = &ems{handles: map[uint16]*emsHandle{}, next: 17}
	for h := uint16(1); h <= 16; h++ {
		d.ems.handles[h] = &emsHandle{pages: [][]byte{{byte(h)}}}
	}
	for i := 0; i < 8; i++ {
		name := filepath.Join(d.Root, string(rune('A'+i))+".DAT")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		d.handles[uint16(5+i)] = &handle{name: filepath.Base(name), path: name, f: f, size: 1, refs: 1}
	}
	var first []byte
	for i := 0; i < 8; i++ {
		var b bytes.Buffer
		if err := d.SaveState(&b); err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = b.Bytes()
		} else if !bytes.Equal(first, b.Bytes()) {
			t.Fatalf("第 %d 次存檔與第一次不同", i+1)
		}
	}
}

// 舊狀態檔把 EMB 存成 map；換成 EMBList 之後舊檔仍要讀得回來。
func TestDOSLoadStateReadsLegacyEMBMap(t *testing.T) {
	_, d := newTest(t)
	d.emb = map[uint16][]byte{3: {0xAA, 0xBB}, 9: {0xCC}}
	var buf bytes.Buffer
	if err := d.SaveState(&buf); err != nil {
		t.Fatal(err)
	}
	var s dosState
	if err := gob.NewDecoder(&buf).Decode(&s); err != nil {
		t.Fatal(err)
	}
	s.EMB = map[uint16][]byte{}
	for _, e := range s.EMBList {
		s.EMB[e.H] = e.Data
	}
	s.EMBList = nil
	var legacy bytes.Buffer
	if err := gob.NewEncoder(&legacy).Encode(&s); err != nil {
		t.Fatal(err)
	}
	_, e := newTest(t)
	if err := e.LoadState(&legacy); err != nil {
		t.Fatalf("舊格式讀不回來：%v", err)
	}
	if !bytes.Equal(e.emb[3], []byte{0xAA, 0xBB}) || !bytes.Equal(e.emb[9], []byte{0xCC}) {
		t.Fatalf("EMB 沒還原：%v", e.emb)
	}
}
