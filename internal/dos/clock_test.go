package dos

import (
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
)

// `AH=2Ch` 的基準時刻要聽 Now 的。
//
// **播種靠時鐘的程式把這個欄位變成輸入的一部分**：Borland 的 randomize()
// 是 srand((unsigned)time(NULL))，而 time() 走 getdate／gettime。基準不能設
// 的話，種子雖然是決定性的，卻綁在 tick 數上——時序模型一改、秒數跨過整數
// 邊界，種子就跳一格，而症狀是「同一份輸入跑出另一條亂數流」。
func TestClockBaseIsSettable(t *testing.T) {
	m, d := newTest(t)
	d.Now = Time{Hour: 13, Min: 45, Sec: 30, Hundredth: 0}
	call(m, d, 0x21, 0x2C00)
	hour, minute := uint8(m.CPU.R[cpu.CX]>>8), uint8(m.CPU.R[cpu.CX])
	sec := uint8(m.CPU.R[cpu.DX] >> 8)
	if hour != 13 || minute != 45 {
		t.Fatalf("AH=2Ch 回 %02d:%02d，預期 13:45", hour, minute)
	}
	if sec < 30 {
		t.Fatalf("秒數是 %d，比基準的 30 還小——基準沒被吃進去", sec)
	}
}

// 時刻仍要隨 tick 往前走。固定回同一個值的話，「等 N 個百分之一秒」
// 那種迴圈永遠轉不出來，而從外面看程式還活著。
func TestClockAdvancesWithTicks(t *testing.T) {
	m, d := newTest(t)
	d.Now = Time{}
	before := d.Clock()
	m.Ticks += 200 // 約 11 秒
	after := d.Clock()
	if before == after {
		t.Fatal("推進 200 個 tick 之後時刻沒變")
	}
	if after.Sec == 0 && after.Min == 0 {
		t.Fatalf("推進 200 個 tick 之後是 %02d:%02d:%02d，預期至少走到秒",
			after.Hour, after.Min, after.Sec)
	}
}
