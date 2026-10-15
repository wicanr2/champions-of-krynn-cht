package dos

import (
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/machine"
)

// 會寫進 StubSeg 的服務都叫過一次之後，`int 08h` 的 BIOS 計時器 stub 還要能
// 完整跑完（`docs/spec/194-stubseg-font-collision-and-program-path` §1.2）。
//
// 這是兩條分支各自正確、合起來才出錯的位置：civ1 把計時器 stub 加長到 35 byte
// （`0x420`–`0x442`），而系統設定表原本放在 `0x440`。程式一叫 `int 15h AH=C0h`，
// stub 結尾的 `pop ax／pop ds／iret` 就被換成設定表，下一次計時器中斷走進資料。
// 只比對位元組的話新加的寫入者容易漏列，所以這裡直接把 stub 跑一次。
func TestStubSegWritersLeaveTimerStubRunnable(t *testing.T) {
	m, d := newTest(t)
	call(m, d, 0x15, 0xC000) // 系統設定表
	call(m, d, 0x21, 0x3400) // InDOS 旗標
	d.installFont()          // DOS/V 字型 stub

	const scratch = 0x2000
	// 只測計時器 stub 本身：int 1Ch 接到一個 IRET，跑完停在 HLT。
	m.WriteBytes(cpu.Addr(scratch, 0x300), []byte{0xCF})
	m.Write16(0x1C*4, 0x300)
	m.Write16(0x1C*4+2, scratch)
	m.WriteBytes(cpu.Addr(scratch, 0x400), []byte{0x90, 0xF4})
	m.CPU.Seg[cpu.CS], m.CPU.IP = scratch, 0x400
	m.CPU.Seg[cpu.SS], m.CPU.R[cpu.SP] = scratch, 0x200
	m.CPU.Seg[cpu.DS], m.CPU.R[cpu.AX], m.CPU.R[cpu.DX] = 0x1234, 0x5678, 0x9ABC
	m.CPU.SetFlags(m.CPU.Flags | cpu.IF)
	flags := m.CPU.Flags

	if seg := m.Read16(0x08*4 + 2); seg != machine.StubSeg {
		t.Fatalf("int 08h 向量的段是 %04X，預期 StubSeg %04X", seg, machine.StubSeg)
	}
	ticks := m.Read16(0x046C)
	m.CPU.Interrupt(8)
	for i := 0; i < 100 && !m.CPU.Halted; i++ {
		if err := m.Step(); err != nil {
			t.Fatalf("第 %d 步：%v", i, err)
		}
	}
	if !m.CPU.Halted {
		t.Fatal("計時器 stub 沒有回到被中斷的地方")
	}
	if got := m.Read16(0x046C); got != ticks+1 {
		t.Errorf("0040:006C 是 %04X，預期 %04X", got, ticks+1)
	}
	if m.CPU.Seg[cpu.DS] != 0x1234 || m.CPU.R[cpu.AX] != 0x5678 || m.CPU.R[cpu.DX] != 0x9ABC || m.CPU.Flags != flags {
		t.Errorf("暫存器沒還原：DS=%04X AX=%04X DX=%04X flags=%04X",
			m.CPU.Seg[cpu.DS], m.CPU.R[cpu.AX], m.CPU.R[cpu.DX], m.CPU.Flags)
	}
}
