package dos

import (
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/machine"
)

// PSP 區塊調整後的剩餘空間（`docs/spec/199-psp-block-resize-boundary`）。
//
// Borland C++ 的 `exec`（AESOP.EXE 的 `launch`）先 `AH=4Ah BX=FFFFh` 探出上限 X，
// 再把自己撐到 `X − (參數段數+1)`，接著 `AH=48h` 要「參數段數」那麼多段。
// 自由區塊的 MCB 緊接在 `PSP + 新大小`，所以剩下的恰好夠——**少算一段就
// ENOMEM**，程式印 `Secondary process launch failure 8` 離開。

func TestPSPBlockResizeLeavesExactTail(t *testing.T) {
	m, d := newTest(t)
	d.arena = nil
	d.freeSeg = machine.PSPSeg + 0x100 // 還沒配置過：可配置區從 PSP 區塊之後起算

	// 探測：要 FFFFh 段一定失敗，BX 回上限。
	m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, 0xFFFF
	call(m, d, 0x21, 0x4A00)
	if m.CPU.Flags&cpu.CF == 0 {
		t.Fatal("探測竟然成功")
	}
	max := m.CPU.R[cpu.BX]
	if want := uint16(machine.MemTop) - machine.PSPSeg; max != want {
		t.Fatalf("上限 %04X，預期 MemTop−PSP ＝ %04X", max, want)
	}

	// 照 Borland 的算法：撐到 max−5，再要 4 段。
	const args = 4
	m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, max-(args+1)
	call(m, d, 0x21, 0x4A00)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Fatalf("撐到 max−5 失敗：AX＝%04X", m.CPU.R[cpu.AX])
	}
	// 自由區塊的 MCB 在 PSP＋大小：剩 MemTop − (PSP+大小) 段，扣掉 MCB 本身一段。
	if d.freeSeg != machine.PSPSeg+max-(args+1) {
		t.Errorf("freeSeg＝%04X，預期 PSP＋新大小 ＝ %04X", d.freeSeg, machine.PSPSeg+max-(args+1))
	}
	m.CPU.R[cpu.BX] = args + 1
	call(m, d, 0x21, 0x4800)
	if m.CPU.Flags&cpu.CF == 0 {
		t.Errorf("要 5 段竟然成功了（只該剩 4 段資料）：AX＝%04X", m.CPU.R[cpu.AX])
	}
	m.CPU.R[cpu.BX] = args
	call(m, d, 0x21, 0x4800)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Fatalf("要 %d 段失敗：BX＝%04X（Borland exec 要的剛好是這麼多）", args, m.CPU.R[cpu.BX])
	}
	// 配到的段落在 PSP 區塊之後、MemTop 之內。
	seg := m.CPU.R[cpu.AX]
	if seg <= machine.PSPSeg+max-(args+1) || int(seg)+args > machine.MemTop {
		t.Errorf("配到的段 %04X 不在剩餘區間內", seg)
	}
}

// MCB 鏈要涵蓋到傳統記憶體上緣，而且程式自己的那一塊大小等於它要的大小。
func TestPSPBlockResizeMCBMatches(t *testing.T) {
	m, d := newTest(t)
	d.arena = nil
	d.freeSeg = machine.PSPSeg + 0x100
	const want = 0x1000
	m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, want
	call(m, d, 0x21, 0x4A00)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Fatal("調整失敗")
	}
	d.syncMCB()
	mcb := uint32(machine.PSPSeg-1) * 16
	if got := m.Read16(mcb + 3); got != want {
		t.Errorf("程式自己的 MCB 大小 ＝ %04X，預期 %04X", got, want)
	}
	next := uint32(machine.PSPSeg+want) * 16
	if sig := m.Read8(next); sig != 'M' && sig != 'Z' {
		t.Errorf("下一個 MCB 應該在 PSP＋大小（%04X），簽章是 %02X", machine.PSPSeg+want, sig)
	}
}

// 行程的 PSP 區塊不能長過已配出的區塊（`StrictPSPGrowth`，
// `docs/spec/199-psp-block-resize-boundary` §3）。
//
// 情境（EOB3 的 Borland exec 連續兩次）：第一次 exec 把自己縮到「最大值減 5」，
// 再 `AH=48h` 要 4 段放環境，環境區塊於是落在記憶體最上緣。第二次 exec 又先探測
// 最大值再撐到那個值，**真 DOS 的最大值不含那個環境區塊**；舊行為回 `MemTop−PSP`，
// 載入器 stub（區塊尾端往下 0x17 段）因此蓋到環境區塊。
func TestStrictPSPGrowthStopsAtAllocatedBlock(t *testing.T) {
	probe := func(m *machine.Machine, d *DOS) (uint16, bool) {
		m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, 0xFFFF
		call(m, d, 0x21, 0x4A00)
		return m.CPU.R[cpu.BX], m.CPU.Flags&cpu.CF != 0
	}
	build := func(strict bool) (*machine.Machine, *DOS, uint16) {
		m, d := newTest(t)
		d.arena = nil
		d.freeSeg = machine.PSPSeg + 0x100
		d.StrictPSPGrowth = strict
		max, _ := probe(m, d)
		// 第一次 exec：縮到 max−5，再要 4 段放環境。
		m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, max-5
		call(m, d, 0x21, 0x4A00)
		m.CPU.R[cpu.BX] = 4
		call(m, d, 0x21, 0x4800)
		if m.CPU.Flags&cpu.CF != 0 {
			t.Fatalf("環境區塊配置失敗：AX＝%04X", m.CPU.R[cpu.AX])
		}
		return m, d, m.CPU.R[cpu.AX] // 環境區塊的資料段
	}

	// 預設（舊行為）：探測不看已配出的區塊。
	m, d, _ := build(false)
	want := uint16(machine.MemTop) - machine.PSPSeg
	if got, ok := probe(m, d); !ok || got != want {
		t.Errorf("預設探測 BX＝%04X ok＝%v，預期舊行為 %04X", got, ok, want)
	}

	// 嚴格：上限是環境區塊的 MCB（資料段 −1）減 PSP。
	m, d, env := build(true)
	limit := (env - 1) - machine.PSPSeg
	got, failed := probe(m, d)
	if !failed || got != limit {
		t.Fatalf("嚴格探測 BX＝%04X failed＝%v，預期上限 %04X", got, failed, limit)
	}
	// 撐到上限成功；多一段失敗，BX 回上限。
	m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, limit
	call(m, d, 0x21, 0x4A00)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Errorf("撐到上限 %04X 失敗：AX＝%04X", limit, m.CPU.R[cpu.AX])
	}
	m.CPU.Seg[cpu.ES], m.CPU.R[cpu.BX] = machine.PSPSeg, limit+1
	call(m, d, 0x21, 0x4A00)
	if m.CPU.Flags&cpu.CF == 0 || m.CPU.R[cpu.BX] != limit {
		t.Errorf("多要一段應失敗並回上限 %04X：CF＝%v BX＝%04X", limit, m.CPU.Flags&cpu.CF != 0, m.CPU.R[cpu.BX])
	}
	// Borland 的載入器 stub（0x160 bytes ＝ 0x16 段）放在區塊尾端往下 0x17 段，
	// 環境區塊的資料在 env：stub 從開頭到環境資料的距離要放得下整個 stub，
	// 否則環境字串會蓋掉 stub 的尾端（EOB3 的症狀）。
	end := machine.PSPSeg + limit
	if end > env-1 {
		t.Errorf("區塊尾端 %04X 超過環境區塊的 MCB %04X", end, env-1)
	}
	stub := end - 0x17
	if env-stub < 0x16 {
		t.Errorf("stub 段 %04X 到環境資料段 %04X 只有 %d 段，放不下 0x16 段的 stub", stub, env, env-stub)
	}
}
