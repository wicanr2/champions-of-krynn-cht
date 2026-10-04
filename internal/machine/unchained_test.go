package machine

import (
	"bytes"
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
)

// 非鏈結 256 色（Mode X，`docs/spec/201-vga-unchained-256`）的釘死測試。
//
// 這些行為錯了不會有錯誤訊息：畫面照樣有東西，只是 4 個平面的資料被當成線性
// 位元組讀，圖會變成左右縮成 1/4、並排四份（EOB3 的主選單就是這樣）。

// modeX 設 mode 13h，再照標準 Mode X 的做法解鏈結：序列器 04h←04h、
// CRTC 14h←00h、17h←E3h，然後 Map Mask 全開清平面。**走真正的埠。**
func modeX(t *testing.T) *Machine {
	t.Helper()
	m := New()
	m.SetVideoMode(0x13)
	if m.planarOn {
		t.Fatal("mode 13h 剛設好就不該走平面路徑（鏈結位元是立的）")
	}
	seqReg(m, 4, 0x04)
	m.Out8(0x3D4, 0x14)
	m.Out8(0x3D5, 0x00)
	m.Out8(0x3D4, 0x17)
	m.Out8(0x3D5, 0xE3)
	if !m.planarOn || !m.Unchained() {
		t.Fatal("清掉鏈結位元之後 A0000 要走平面路徑")
	}
	return m
}

func TestMode13hStaysLinearUntilUnchained(t *testing.T) {
	m := New()
	m.SetVideoMode(0x13)
	m.Write8(0xA0000+5, 0x7B)
	if m.Mem[0xA0000+5] != 0x7B {
		t.Error("鏈結（一般 mode 13h）的寫入要落在 Mem 的線性位址")
	}
	if got := m.Indexed()[5]; got != 0x7B {
		t.Errorf("Indexed()[5]＝%02X，預期 7B", got)
	}
	// 寫 Map Mask 不會讓 mode 13h 走平面（舊行為）。
	seqReg(m, 2, 0x01)
	if m.planarOn {
		t.Error("mode 13h 在鏈結狀態下，Map Mask 不該切到平面路徑")
	}
	// 清掉鏈結位元才算 Mode X；再立起來就回到線性。
	seqReg(m, 4, 0x04)
	if !m.Unchained() {
		t.Error("清掉鏈結位元後應是 Mode X")
	}
	seqReg(m, 4, 0x0E)
	if m.Unchained() || m.planarOn {
		t.Error("鏈結位元立起來後要回到線性")
	}
}

func TestUnchainedPixelLayout(t *testing.T) {
	m := modeX(t)
	seqReg(m, 2, 0x0F)
	// 全部平面寫 0，再各平面寫不同的值。
	for off := uint32(0); off < 0x10000; off++ {
		m.Write8(0xA0000+off, 0)
	}
	// 像素 (x,y) 在平面 x&3 的第 y*80 + x>>2 格。
	for p := 0; p < 4; p++ {
		seqReg(m, 2, 1<<p)
		m.Write8(0xA0000+uint32(3*80+10), uint8(0x40+p)) // y=3，x>>2＝10 → x ＝ 40+p
	}
	px := m.Indexed()
	if len(px) != 320*200 {
		t.Fatalf("畫面大小 %d，預期 64000", len(px))
	}
	for p := 0; p < 4; p++ {
		x := 40 + p
		if got := px[3*320+x]; got != uint8(0x40+p) {
			t.Errorf("(x=%d,y=3)＝%02X，預期 %02X（平面 %d）", x, got, 0x40+p, p)
		}
	}
	// 其他像素是 0。
	nz := 0
	for _, v := range px {
		if v != 0 {
			nz++
		}
	}
	if nz != 4 {
		t.Errorf("非零像素 %d 個，預期 4", nz)
	}
	// 回傳的是複本。
	px[3*320+40] = 0
	if m.Indexed()[3*320+40] != 0x40 {
		t.Error("Indexed() 應該回複本")
	}
	if w, h := m.PlanarSize(); w != 0 || h != 0 {
		t.Errorf("Mode X 不是 16 色平面模式，PlanarSize 要回 0,0，得到 %d,%d", w, h)
	}
	if w, h := m.VideoSize(); w != 320 || h != 200 {
		t.Errorf("VideoSize＝%d×%d，預期 320×200", w, h)
	}
}

// 讀回：Read Map Select 選平面，而且讀會載入 latch（write mode 1 的區塊複製要靠它）。
func TestUnchainedReadAndLatchCopy(t *testing.T) {
	m := modeX(t)
	seqReg(m, 2, 0x0F)
	for p := 0; p < 4; p++ {
		seqReg(m, 2, 1<<p)
		m.Write8(0xA0000+0x100, uint8(0x10*(p+1)))
	}
	for p := 0; p < 4; p++ {
		gcReg(m, 4, uint8(p)) // Read Map Select
		if got := m.Read8(0xA0000 + 0x100); got != uint8(0x10*(p+1)) {
			t.Errorf("讀平面 %d＝%02X，預期 %02X", p, got, 0x10*(p+1))
		}
	}
	// write mode 1：latch 整組寫回，四個平面一次搬完。
	gcReg(m, 5, 0x01)
	seqReg(m, 2, 0x0F)
	_ = m.Read8(0xA0000 + 0x100) // 載入 latch
	m.Write8(0xA0000+0x200, 0)   // 資料位元組被忽略
	for p := 0; p < 4; p++ {
		if got := m.VGA.Planes[p][0x200]; got != uint8(0x10*(p+1)) {
			t.Errorf("latch 複製後平面 %d＝%02X", p, got)
		}
	}
}

// CRTC 的列距、顯示起點與分割線都要生效。
func TestUnchainedCRTCPitchStartAndSplit(t *testing.T) {
	m := modeX(t)
	seqReg(m, 2, 0x0F)
	// 第一個像素 (0,0) 與第二頁起點。
	m.Out8(0x3D4, 0x13)
	m.Out8(0x3D5, 0x28) // 列距 80 位元組
	m.VGA.Planes[0][0] = 0x11
	m.VGA.Planes[0][0x4000] = 0x22
	if m.Indexed()[0] != 0x11 {
		t.Fatal("顯示起點 0 應該是 0x11")
	}
	m.Out8(0x3D4, 0x0C)
	m.Out8(0x3D5, 0x40) // 起點高位元組：0x4000
	m.Out8(0x3D4, 0x0D)
	m.Out8(0x3D5, 0x00)
	if got := m.Indexed()[0]; got != 0x22 {
		t.Errorf("顯示起點 4000h：(0,0)＝%02X，預期 22", got)
	}
	// 虛擬寬度 160 位元組一列（offset 80）：第 1 列從 160 起。
	m.Out8(0x3D4, 0x0C)
	m.Out8(0x3D5, 0x00)
	m.Out8(0x3D4, 0x13)
	m.Out8(0x3D5, 80)
	m.VGA.Planes[0][160] = 0x33
	if got := m.Indexed()[320]; got != 0x33 {
		t.Errorf("列距 160：(0,1)＝%02X，預期 33", got)
	}
	// 分割線：line compare ＝ 100 掃描線 ＝ 第 50 列（每列掃兩次）之後的列從 0 起算。
	m.Out8(0x3D4, 0x13)
	m.Out8(0x3D5, 0x28)
	m.Out8(0x3D4, 0x0C)
	m.Out8(0x3D5, 0x40) // 起點 4000h
	m.Out8(0x3D4, 0x09)
	m.Out8(0x3D5, 0x41) // max scan line ＝ 1（每列兩次）、bit6＝LC 第 9 位
	m.Out8(0x3D4, 0x18)
	m.Out8(0x3D5, 100) // line compare 低 8 位
	m.Out8(0x3D4, 0x07)
	m.Out8(0x3D5, 0x00) // LC 第 8 位 ＝ 0
	m.VGA.crtc[0x09] &^= 0x40
	m.VGA.Planes[0][0x4000+80*40] = 0x44 // 上半部：第 40 列
	m.VGA.Planes[0][0] = 0x55            // 分割線之下從 0 起算的第一列
	px := m.Indexed()
	if px[40*320] != 0x44 {
		t.Errorf("分割線之上要跟著顯示起點：(0,40)＝%02X，預期 44", px[40*320])
	}
	if got := px[51*320]; got != 0x55 {
		// 分割在 line compare 的下一列生效：第 split＝50 列還是上半部，第 51 列起歸零。
		t.Errorf("分割線之下從位址 0 起算：(0,51)＝%02X，預期 55", got)
	}
}

// 快照與狀態檔要留住 CRTC、Misc 與暫存器高半（`docs/spec/197`、`201`）。
func TestSnapshotAndStateKeepCRTCAndHighHalves(t *testing.T) {
	m := modeX(t)
	m.Out8(0x3D4, 0x13)
	m.Out8(0x3D5, 0x30)
	m.Out8(0x3D4, 0x0D)
	m.Out8(0x3D5, 0x12)
	m.Out8(0x3C2, 0x67)
	m.CPU.Hi = [8]uint16{0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666, 0x7777, 0x8888}
	m.CPU.FSeg, m.CPU.GSeg = 0xAAAA, 0xBBBB
	m.VGA.Planes[2][7] = 0x99

	snap := m.Snapshot()
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		t.Fatal(err)
	}
	wantCRTC := m.VGA.crtc

	// 弄壞再還原。
	m.CPU.Hi = [8]uint16{}
	m.CPU.FSeg, m.CPU.GSeg = 0, 0
	m.VGA.crtc = [32]uint8{}
	m.VGA.misc = 0
	m.Restore(snap)
	if m.CPU.Hi[cpu.AX] != 0x1111 || m.CPU.Hi[cpu.DI] != 0x8888 || m.CPU.FSeg != 0xAAAA || m.CPU.GSeg != 0xBBBB {
		t.Errorf("Snapshot 沒留住 Hi／FS／GS：%v %04X %04X", m.CPU.Hi, m.CPU.FSeg, m.CPU.GSeg)
	}
	if m.VGA.crtc != wantCRTC || m.VGA.misc != 0x67 {
		t.Errorf("Snapshot 沒留住 CRTC／Misc")
	}
	if !m.Unchained() || m.Indexed()[0] != 0 {
		t.Error("還原後應仍是 Mode X")
	}

	// 狀態檔往返，到一台全新的機器。
	m2 := New()
	if err := m2.LoadState(&buf); err != nil {
		t.Fatal(err)
	}
	if m2.CPU.Hi != [8]uint16{0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666, 0x7777, 0x8888} ||
		m2.CPU.FSeg != 0xAAAA || m2.CPU.GSeg != 0xBBBB {
		t.Errorf("狀態檔沒留住 Hi／FS／GS：%v %04X %04X", m2.CPU.Hi, m2.CPU.FSeg, m2.CPU.GSeg)
	}
	if m2.VGA.crtc != wantCRTC || m2.VGA.misc != 0x67 || m2.VGA.Planes[2][7] != 0x99 {
		t.Errorf("狀態檔沒留住 CRTC／Misc／平面")
	}
	if !m2.Unchained() {
		t.Error("讀檔後應仍是 Mode X")
	}
}

// 回呼（滑鼠事件那一類）打斷程式時，被打斷的程式的暫存器高半與 FS／GS 要原樣還回去。
func TestCallbackPreservesHighHalves(t *testing.T) {
	m := New()
	m.CPU.Hi[cpu.AX] = 0xDEAD
	m.CPU.Hi[cpu.SI] = 0xBEEF
	m.CPU.FSeg, m.CPU.GSeg = 0x1234, 0x5678
	m.QueueCallback(QueuedCall{Seg: 0x2000, Off: 0x10})
	if !m.startCallback() {
		t.Fatal("回呼沒送出去")
	}
	// 回呼本身用 32 位元暫存器弄髒高半。
	m.CPU.Hi[cpu.AX], m.CPU.Hi[cpu.SI] = 0, 0
	m.CPU.FSeg, m.CPU.GSeg = 0, 0
	if !m.FinishCallback() {
		t.Fatal("FinishCallback 失敗")
	}
	if m.CPU.Hi[cpu.AX] != 0xDEAD || m.CPU.Hi[cpu.SI] != 0xBEEF || m.CPU.FSeg != 0x1234 || m.CPU.GSeg != 0x5678 {
		t.Errorf("回呼結束後高半沒還原：Hi＝%v FS＝%04X GS＝%04X", m.CPU.Hi, m.CPU.FSeg, m.CPU.GSeg)
	}
}
