package machine

import "testing"

// civ1 的 mode 13h `3DA` 相位模型是 opt-in（`Machine.Mode13Phase`）。
//
// 預設要維持行為模型：既有 mode 13h 收據（大富翁2 等）的 `in` 序列建立在
// 「讀取次數翻轉」上，相位模型會讓同一份收據跑出另一個時間軸，而畫面照樣
// 畫得出來，不會有任何地方報錯。
func TestMode13PhaseIsOptIn(t *testing.T) {
	read := func(phase bool) map[uint8]bool {
		m := New()
		m.Mode13Phase = phase
		m.SetVideoMode(0x13)
		m.Steps = 12345 // 讀取期間步數不動
		seen := map[uint8]bool{}
		for i := 0; i < 64; i++ {
			seen[m.In8(0x3DA)] = true
		}
		return seen
	}
	if seen := read(false); len(seen) < 2 {
		t.Errorf("預設（行為模型）同一步數下讀 3DA 應該會翻轉，只看到 %v", seen)
	}
	if seen := read(true); len(seen) != 1 {
		t.Errorf("Mode13Phase 開啟時讀取不推進時間，同一步數應該只有一個值，看到 %v", seen)
	}
}
