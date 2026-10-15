package basic_test

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
	"github.com/wicanr2/dosgolem/runtime/basic"
)

// 測試程式（載在 LoadSeg 0110h，CS:IP ＝ 0110:0000），模仿編譯後 BASIC 的呼叫慣例：
// 參數以近指標由左到右 push，再 far call。
//
//	0000  call far 0110:0021     RANDOMIZE（一次）
//	0005  mov ax,0310 / push ax  第 1 個參數的指標
//	0009  mov ax,0312 / push ax  第 2 個參數的指標
//	000D  call far 0110:0020     RND
//	0012  add sp,4
//	0015  nop                    ← 測試在這裡推進 LCG（代替真的 RND 本體）
//	0016  jmp 0005
//	0020  retf                   RND
//	0021  retf                   RANDOMIZE
var program = map[int][]byte{
	0x00: {0x9A, 0x21, 0x00, 0x10, 0x01},
	0x05: {0xB8, 0x10, 0x03, 0x50},
	0x09: {0xB8, 0x12, 0x03, 0x50},
	0x0D: {0x9A, 0x20, 0x00, 0x10, 0x01},
	0x12: {0x83, 0xC4, 0x04},
	0x15: {0x90},
	0x16: {0xEB, 0xED},
	0x20: {0xCB},
	0x21: {0xCB},
}

const (
	codeSeg     = 0x0110
	advanceOff  = 0x0015
	returnOff   = 0x0012
	stateLo     = 0x0300 // DGROUP 偏移
	stateHi     = 0x0302
	arg1, arg2  = 0x0310, 0x0312
	descriptor  = 0x0340
	arrayDataAt = 0x0380
)

func load(t *testing.T) *oracle.Oracle {
	t.Helper()
	code := make([]byte, 0x400)
	for off, b := range program {
		copy(code[off:], b)
	}
	hdr := make([]byte, 32)
	put := func(off int, v uint16) { binary.LittleEndian.PutUint16(hdr[off:], v) }
	total := len(hdr) + len(code)
	copy(hdr, "MZ")
	put(0x02, uint16(total%512))
	put(0x04, uint16((total+511)/512))
	put(0x08, 2)      // 表頭 2 段
	put(0x0A, 0x10)   // 最少多要 16 段
	put(0x0C, 0xFFFF) // 最多
	put(0x10, 0x0400) // SP：程式區尾端
	put(0x18, 0x001C)
	path := filepath.Join(t.TempDir(), "BASIC.EXE")
	if err := os.WriteFile(path, append(hdr, code...), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err := oracle.Load(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(o.Close)
	o.SyncDGroupFromDS() // DGROUP ＝ 載入時的 DS（PSP 段）
	return o
}

func config(o *oracle.Oracle) basic.Config {
	return basic.Config{
		RNDEntry:  o.ToIDA(oracle.Addr{Seg: codeSeg, Off: 0x20}),
		Randomize: o.ToIDA(oracle.Addr{Seg: codeSeg, Off: 0x21}),
		StateLo:   stateLo,
		StateHi:   stateHi,
	}
}

// 公式本身：大富翁2 的 DGROUP 上讀出來的三個常數，結果一律落在 24 位元內。
func TestLCGNextFormula(t *testing.T) {
	for _, x := range []uint32{0, 1, 0x123456, 0xFFFFFF} {
		want := uint32((uint64(x)*0x43FD43FD + 0xC39EC3) % (1 << 24))
		if got := basic.LCGNext(x); got != want || got >= 1<<24 {
			t.Errorf("LCGNext(%06X) ＝ %06X，預期 %06X", x, got, want)
		}
	}
	if got := basic.LCGNext(0); got != 0xC39EC3 {
		t.Errorf("LCGNext(0) ＝ %06X，應該就是加數 C39EC3", got)
	}
	c := basic.Call{State: 0x123456}
	if c.Next() != basic.LCGNext(0x123456) || c.Value() != float64(c.Next())/(1<<24) {
		t.Error("Call.Next／Value 與 LCGNext 不一致")
	}
	if v := c.Value(); v < 0 || v >= 1 {
		t.Errorf("RND 值 %v 不在 [0,1)", v)
	}
}

// 狀態是 16＋8 位元分兩處存：高位元組只用 8 位元，寫入不能波及鄰居。
func TestStateRoundTripUsesBothHalves(t *testing.T) {
	o := load(t)
	cfg := config(o)
	o.WriteU8(o.DS(stateHi+1), 0xEE) // 鄰居
	basic.SetState(o, cfg, 0xABCDEF)
	if got := basic.State(o, cfg); got != 0xABCDEF {
		t.Fatalf("State ＝ %06X，預期 ABCDEF", got)
	}
	if lo, hi := o.Word(o.DS(stateLo)), o.Byte(o.DS(stateHi)); lo != 0xCDEF || hi != 0xAB {
		t.Fatalf("低 16 位 %04X、高 8 位 %02X，預期 CDEF／AB", lo, hi)
	}
	if o.Byte(o.DS(stateHi+1)) != 0xEE {
		t.Fatal("SetState 寫到高位元組之外")
	}
	basic.SetState(o, cfg, 0x1FFFFFF) // 超過 24 位元的部分丟掉
	if got := basic.State(o, cfg); got != 0xFFFFFF {
		t.Fatalf("超過 24 位元時 State ＝ %06X，預期 FFFFFF", got)
	}
}

// 真的跑起來：每一次 RND 都攔得到、序列照 LCG 推進、呼叫端與參數讀得對。
func TestTraceRNDFollowsTheSequence(t *testing.T) {
	o := load(t)
	cfg := config(o)
	basic.SetState(o, cfg, 0x123456)
	o.WriteU16(o.DS(arg1), 111)
	o.WriteU16(o.DS(arg2), 222)
	tr := basic.TraceRND(o, cfg)
	var args [][]uint16
	o.OnCall(o.IDA(cfg.RNDEntry), func(o *oracle.Oracle) {
		args = append(args, basic.CallArgs(o, 2))
	})
	o.OnCall(oracle.Addr{Seg: codeSeg, Off: advanceOff}, func(o *oracle.Oracle) {
		basic.SetState(o, cfg, basic.LCGNext(basic.State(o, cfg)))
	})
	if err := o.Run(800); err != nil {
		t.Fatal(err)
	}
	if len(tr.Calls) < 50 {
		t.Fatalf("只攔到 %d 次 RND", len(tr.Calls))
	}
	if len(tr.RandomizeAt) != 1 {
		t.Errorf("RANDOMIZE 攔到 %d 次，預期 1", len(tr.RandomizeAt))
	}
	if tr.InitialState != 0x123456 {
		t.Errorf("InitialState ＝ %06X，預期 123456", tr.InitialState)
	}
	// 每一次 RND 都在推進之前攔到；最後一次之後若還沒推進，停在那一次的狀態上。
	if basic.State(o, cfg) != tr.Calls[len(tr.Calls)-1].Next() {
		basic.SetState(o, cfg, tr.Calls[len(tr.Calls)-1].Next())
	}
	if err := tr.Verify(o); err != nil {
		t.Fatalf("Verify：%v", err)
	}
	by := tr.ByCaller(o)
	ret := o.ToIDA(oracle.Addr{Seg: codeSeg, Off: returnOff})
	if len(by) != 1 || by[ret] != len(tr.Calls) {
		t.Errorf("ByCaller ＝ %v，預期只有 %05X 一個呼叫端、%d 次", by, ret, len(tr.Calls))
	}
	for i, a := range args {
		if len(a) != 2 || a[0] != 111 || a[1] != 222 {
			t.Fatalf("第 %d 次 CallArgs ＝ %v，預期 [111 222]（第 1 個參數在 [0]）", i, a)
		}
	}
}

// Verify 要抓得到「公式不對」與「位址不對」兩種錯——這是它存在的理由。
func TestVerifyRejectsWrongSequenceAndWrongAddress(t *testing.T) {
	o := load(t)
	cfg := config(o)
	tr := basic.TraceRND(o, cfg)
	o.OnCall(oracle.Addr{Seg: codeSeg, Off: advanceOff}, func(o *oracle.Oracle) {
		basic.SetState(o, cfg, basic.State(o, cfg)+1) // 不是 LCG
	})
	if err := o.Run(200); err != nil {
		t.Fatal(err)
	}
	if err := tr.Verify(o); err == nil || !strings.Contains(err.Error(), "預期") {
		t.Errorf("錯的推進方式沒被抓到：%v", err)
	}

	o2 := load(t)
	wrong := config(o2)
	wrong.RNDEntry += 0x100 // 沒有程式碼的位址
	tr2 := basic.TraceRND(o2, wrong)
	if err := o2.Run(200); err != nil {
		t.Fatal(err)
	}
	if err := tr2.Verify(o2); err == nil || !strings.Contains(err.Error(), "一次都沒呼叫") {
		t.Errorf("位址不對沒被抓到：%v", err)
	}
}

// 陣列：描述子是 (位移, 段)，索引是列主序（第一維變化最快）。
func TestArrayIsColumnMajorFromDescriptor(t *testing.T) {
	o := load(t)
	ds := o.DS(0).Seg
	o.WriteU16(o.DS(descriptor), arrayDataAt)
	o.WriteU16(o.DS(descriptor+2), ds)
	dims := []basic.Dim{{Lo: 1, N: 6}, {Lo: 0, N: 3}} // A(1..6, 0..2)
	a := basic.NewArray(o, descriptor, dims, 4)
	base := uint32(ds)*16 + arrayDataAt
	if a.Base != base {
		t.Fatalf("Base ＝ %05X，預期 %05X", a.Base, base)
	}
	if a.Size() != 6*3*4 {
		t.Fatalf("Size ＝ %d，預期 72", a.Size())
	}
	if b := basic.Open(o, descriptor, dims, 4); b.Base != a.Base || b.Size() != a.Size() {
		t.Fatal("Open 與 NewArray 讀出不同的陣列")
	}

	// A(1,1) 與 A(1,0) 差一整個第一維（6 格 × 4 bytes），不是相鄰。
	o.WriteU16(oracle.Phys(base+uint32((1*6+0)*4)), 0x1234)
	if got := a.Int16(1, 1); got != 0x1234 {
		t.Fatalf("A(1,1) ＝ %04X，列主序位址錯", got)
	}
	a.SetInt16(-5, 6, 2)
	if got := int16(o.Word(oracle.Phys(base + uint32((2*6+5)*4)))); got != -5 {
		t.Fatalf("SetInt16 寫到別處：%d", got)
	}
	if got := a.Int16(6, 2); got != -5 {
		t.Fatalf("A(6,2) 讀回 %d", got)
	}

	at := base + uint32((0*6+2)*4) // A(3,0)
	o.WriteU16(oracle.Phys(at), 0x5678)
	o.WriteU16(oracle.Phys(at+2), 0x9ABC)
	if got := a.Int32(3, 0); got != int32(-0x6543A988) {
		t.Fatalf("Int32 ＝ %08X", uint32(got))
	}
	bits := math.Float32bits(1.5)
	o.WriteU16(oracle.Phys(at), uint16(bits))
	o.WriteU16(oracle.Phys(at+2), uint16(bits>>16))
	if got := a.Float32(3, 0); got != 1.5 {
		t.Fatalf("Float32 ＝ %v，預期 1.5（IEEE，不是 MBF）", got)
	}
	o.WriteBytes(oracle.Phys(at), []byte("AB  "))
	if got := string(a.Bytes(3, 0)); got != "AB  " {
		t.Fatalf("Bytes ＝ %q", got)
	}

	// Index 是 addr 的反函數；格內任一個位元組都對回同一格。
	for _, idx := range [][]int{{1, 0}, {6, 0}, {1, 2}, {4, 1}, {6, 2}} {
		lin := base + uint32(((idx[1]-0)*6+(idx[0]-1))*4)
		for k := uint32(0); k < 4; k++ {
			got, ok := a.Index(lin + k)
			if !ok || got[0] != idx[0] || got[1] != idx[1] {
				t.Fatalf("Index(%05X) ＝ %v %v，預期 %v", lin+k, got, ok, idx)
			}
		}
	}
	if _, ok := a.Index(base - 1); ok {
		t.Error("Index 在陣列之前回了 ok")
	}
	if _, ok := a.Index(base + uint32(a.Size())); ok {
		t.Error("Index 在陣列之後回了 ok")
	}

	for _, c := range []struct {
		idx []int
		ok  bool
	}{
		{[]int{1, 0}, true}, {[]int{6, 2}, true},
		{[]int{0, 0}, false}, {[]int{7, 0}, false}, {[]int{1, 3}, false}, {[]int{1, -1}, false},
		{[]int{1}, false},
	} {
		if got := a.InRange(c.idx...); got != c.ok {
			t.Errorf("InRange(%v) ＝ %v，預期 %v", c.idx, got, c.ok)
		}
	}
}

// 索引個數不符要 panic，不能安靜地讀到別的格子。
func TestArrayPanicsOnWrongIndexCount(t *testing.T) {
	o := load(t)
	a := basic.Open(o, descriptor, []basic.Dim{{Lo: 0, N: 2}, {Lo: 0, N: 2}}, 2)
	defer func() {
		if recover() == nil {
			t.Fatal("索引個數不符沒有 panic")
		}
	}()
	a.Int16(1)
}
