package dos

import (
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
)

// BIOS 的 int 09h 預設處理（`docs/spec/202-bios-int09-default-handler`）。
//
// 情境：程式的 INT 9 ISR 只是包裝，做完記帳就 chain 回舊向量——舊向量是我們的
// `CD 09 / CF` stub，所以這一支要做真機 BIOS 做的事：讀埠 0x60、翻成
// (掃描碼, ASCII)、放進 BDA 的環形緩衝。**不做的話鍵送到了 ISR，卻到不了程式讀的
// 緩衝，而且不會有任何錯誤。**

// irq1 模擬「硬體送來一個掃描碼，ISR chain 進 BIOS」：設埠 0x60 再走 int 09h 分派。
func irq1(m interface{ SetKeyPort(uint8) }, d *DOS, code uint8) {
	m.SetKeyPort(code)
	d.handle(d.M.CPU, 0x09)
}

// readKey 用 `int 16h AH=00` 取一個鍵，回 AX。
func readKey(t *testing.T, d *DOS) uint16 {
	t.Helper()
	d.M.CPU.R[cpu.AX] = 0x0000
	d.handle(d.M.CPU, 0x16)
	return d.M.CPU.R[cpu.AX]
}

func TestBiosInt9TranslatesMakeCodes(t *testing.T) {
	m, d := newTest(t)
	cases := []struct {
		name string
		scan uint8
		want uint16
	}{
		{"Enter", 0x1C, 0x1C0D},
		{"Esc", 0x01, 0x011B},
		{"Space", 0x39, 0x3920},
		{"a", 0x1E, 0x1E61},
		{"1", 0x02, 0x0231},
		{"Backspace", 0x0E, 0x0E08},
		{"Tab", 0x0F, 0x0F09},
		{"[", 0x1A, 0x1A5B},
		{"F1（功能鍵：ASCII 0）", 0x3B, 0x3B00},
		{"F10", 0x44, 0x4400},
		{"Up", 0x48, 0x4800},
		{"Down", 0x50, 0x5000},
		{"Left", 0x4B, 0x4B00},
		{"Right", 0x4D, 0x4D00},
		{"Home", 0x47, 0x4700},
		{"Del", 0x53, 0x5300},
	}
	for _, tc := range cases {
		irq1(m, d, tc.scan)
		irq1(m, d, tc.scan|0x80) // 放開鍵：不進緩衝
		if got := m.BIOSKeyCount(); got != 1 {
			t.Fatalf("%s：緩衝裡有 %d 個鍵，預期 1（放開鍵不進）", tc.name, got)
		}
		if got := readKey(t, d); got != tc.want {
			t.Errorf("%s：int 16h 取到 %04X，預期 %04X", tc.name, got, tc.want)
		}
	}
}

func TestBiosInt9ModifierKeys(t *testing.T) {
	m, d := newTest(t)
	const flagsAt = 0x40*16 + 0x17

	irq1(m, d, 0x2A) // 左 Shift 按下
	if m.Read8(flagsAt)&0x02 == 0 {
		t.Error("左 Shift 按下後 BDA 0040:0017 的 bit1 要立起來")
	}
	if m.BIOSKeyCount() != 0 {
		t.Error("修飾鍵本身不進緩衝")
	}
	irq1(m, d, 0x1E)
	if got := readKey(t, d); got != 0x1E41 {
		t.Errorf("Shift+a ＝ %04X，預期 1E41（'A'）", got)
	}
	irq1(m, d, 0x02)
	if got := readKey(t, d); got != 0x0221 {
		t.Errorf("Shift+1 ＝ %04X，預期 0221（'!'）", got)
	}
	irq1(m, d, 0xAA) // 左 Shift 放開
	if m.Read8(flagsAt)&0x03 != 0 {
		t.Error("Shift 放開後旗標要清掉")
	}
	irq1(m, d, 0x1E)
	if got := readKey(t, d); got != 0x1E61 {
		t.Errorf("放開 Shift 後 a ＝ %04X，預期 1E61", got)
	}

	irq1(m, d, 0x1D) // Ctrl
	irq1(m, d, 0x2E) // c
	if got := readKey(t, d); got != 0x2E03 {
		t.Errorf("Ctrl+C ＝ %04X，預期 2E03", got)
	}
	irq1(m, d, 0x9D)
	irq1(m, d, 0x3A) // Caps Lock：沒有字元也不是功能鍵，BIOS 不放進緩衝
	if m.BIOSKeyCount() != 0 {
		t.Error("Caps Lock 不該進緩衝")
	}
}

// 向量被程式接管時，分派不能再插手（否則一個鍵會被處理兩次）。
func TestBiosInt9OnlyForOwnStub(t *testing.T) {
	m, d := newTest(t)
	m.Write16(0x09*4, 0x0000)
	m.Write16(0x09*4+2, 0x2000) // 程式自己的 ISR
	m.SetKeyPort(0x1C)
	if d.handle(m.CPU, 0x09) {
		t.Error("向量不在 stub 時 handle 應該回 false，讓 CPU 真的跳過去")
	}
	if m.BIOSKeyCount() != 0 {
		t.Error("程式自己的 ISR 在跑時不該有人替它填 BDA 緩衝")
	}
}
