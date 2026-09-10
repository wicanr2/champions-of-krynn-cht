package eob1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wicanr2/dosgolem/oracle"
)

// 走到第二層要打過整層第一層。對拍不需要那段戰鬥的真實性，需要的是
// **能重跑**——所以照 `docs/spec/014` 的原則：固定狀態，不要動亂數。
//
// 鎖 HP 與食物是往記憶體寫值，一次抽值都不消費，所以原版的亂數流仍是自然的；
// 強制攻擊是送真的點擊，走的也是原版自己的命中與傷害計算。
// 這與先前那套「把亂數回傳值換掉」完全不同，那個會讓量到的流不是原版的流。
//
// ⚠ **收據要記完整**：鎖了哪些欄位、從第幾步開始鎖、攻擊點了幾次，
// 沒記下來的注入等於沒有發生過。

// 隊伍陣列在 DGROUP 裡的位移與版面，由角色名字當錨點量出來，
// 再拿存檔解析出來的 HP、職業逐人核對過（四人全中）。
//
// 位移是相對**名字起點**的：
//
//	+00 名字 11 bytes  +0B 力量  +0D 力量ext  +0F 智力  +11 感知
//	+13 敏捷  +15 體質  +17 魅力  +19 HP 目前  +1A HP 最大
//	+1B 護甲  +1E 職業  +1F 陣營  +20 肖像  +21 食物  +22 等級×3
const (
	partyDGroupOffset = 0xF582
	partyRecordStride = 243
	partySlots        = 6

	offHitPoints    = 0x19
	offHitPointsMax = 0x1A
	offFood         = 0x21
)

// randSeedOffset 是 Borland RANDSEED 在 DGROUP 裡的位移（低字）。
// 演算法與位址見 eob_123_remake 的 docs/re/eob1-dos-rng-algorithm.md。
const randSeedOffset = 0x8784

// RandomState 讀原版目前的 RANDSEED。
//
// **收據一定要帶它。** 鎖 HP 本身不消費亂數，但它會改變後面發生哪些事
// （沒人倒下就不會有倒下的流程），所以「這一段是在哪個亂數狀態下跑的」
// 是輸入的一部分，不是背景資訊。
func RandomState(o *oracle.Oracle) (uint32, error) {
	base, err := partyBase(o)
	if err != nil {
		return 0, err
	}
	// partyBase 已經把 DGROUP 算好了，退回去取 DGROUP 起點。
	dgroup := uint32(base.Seg)*16 + uint32(base.Off) - partyDGroupOffset
	at := dgroup + randSeedOffset
	lo := uint32(o.Word(oracle.Addr{Seg: uint16(at >> 4), Off: uint16(at & 0xF)}))
	hi := uint32(o.Word(oracle.Addr{Seg: uint16((at + 2) >> 4), Off: uint16((at + 2) & 0xF)}))
	return hi<<16 | lo, nil
}

// PartyMember 是一筆隊伍記錄的位置與目前的關鍵欄位。
type PartyMember struct {
	At      oracle.Addr
	Name    string
	HP, Max uint8
	Food    uint8
}

// dgroupInModule 是 DGROUP 在 EOB.EXE 載入模組裡的位移（IDA 312E0 減載入基底）。
const dgroupInModule = 0x312E0 - 0x10000

// partyBase 算出隊伍陣列的位址。
//
// ⚠ **不要用 `o.DS()`。** 那一支的基底是 Load 時決定的，而 EOB.EXE 是被
// chain 載到 0110h 的另一支程式——`o.DS()` 會回一個看起來很正常的位址，
// 讀到的是別的東西**而且不會報錯**。位址要從 overlay 的載入紀錄自己算。
func partyBase(o *oracle.Oracle) (oracle.Addr, error) {
	var base uint32
	for _, ov := range o.Overlays() {
		if strings.Contains(strings.ToUpper(ov.Name), "EOB.EXE") {
			base = uint32(ov.Seg) * 16
		}
	}
	if base == 0 {
		return oracle.Addr{}, fmt.Errorf("還沒看到 EOB.EXE 的 overlay 載入紀錄，算不出 DGROUP")
	}
	at := base + dgroupInModule + partyDGroupOffset
	return oracle.Addr{Seg: uint16(at >> 4), Off: uint16(at & 0xF)}, nil
}

// readName 讀一筆記錄的名字。**至少兩個可見字元才算數**——
// 放寬到一個字元的話，記憶體裡任何一個孤立的可見位元組加一個 0
// 都會被當成角色，於是位址錯了也照樣「找到隊伍」。
func readName(o *oracle.Oracle, at oracle.Addr) string {
	raw := o.Bytes(at, 11)
	name := ""
	for _, b := range raw {
		if b == 0 {
			break
		}
		if b < 0x20 || b > 0x7E {
			return ""
		}
		name += string(rune(b))
	}
	if len(name) < 2 {
		return ""
	}
	return name
}

// Party 讀出目前的隊伍。
//
// ⚠ **讀不到就要說出來，不要回空的隊伍。** 位址錯了而安靜回空，
// 之後的「鎖 HP」就變成什麼都沒鎖，而症狀是隊伍在某一場戰鬥全滅——
// 離成因很遠，看起來像運氣不好。
func Party(o *oracle.Oracle) ([]PartyMember, error) {
	base, err := partyBase(o)
	if err != nil {
		return nil, err
	}
	var out []PartyMember
	for slot := 0; slot < partySlots; slot++ {
		at := oracle.Addr{Seg: base.Seg, Off: base.Off + uint16(slot*partyRecordStride)}
		name := readName(o, at)
		if name == "" {
			continue // 空槽
		}
		hp := o.Byte(oracle.Addr{Seg: at.Seg, Off: at.Off + offHitPoints})
		max := o.Byte(oracle.Addr{Seg: at.Seg, Off: at.Off + offHitPointsMax})
		if max == 0 || max > 200 || hp > max {
			return nil, fmt.Errorf("隊伍第 %d 格 %q 的 HP=%d/%d 說不通——位址或版面不對，不要往這裡寫",
				slot, name, hp, max)
		}
		out = append(out, PartyMember{At: at, Name: name, HP: hp, Max: max,
			Food: o.Byte(oracle.Addr{Seg: at.Seg, Off: at.Off + offFood})})
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("隊伍陣列 %s 只讀到 %d 個角色——正常隊伍至少兩人，位址八成不對",
			base, len(out))
	}
	return out, nil
}

// VitalsLock 是鎖 HP 與食物的紀錄，收工時要印進收據。
type VitalsLock struct {
	Writes  int
	Members int
	From    uint64
}

// LockPartyVitals 每一幀把全隊的 HP 補滿、食物補到 100。
//
// 用幀而不是用 tick：幀是顯示硬體的事實，`cli` 期間照樣發生；
// 掛在 tick 上的話，戰鬥中那段關中斷的處理正好是最需要補血的時候。
func LockPartyVitals(o *oracle.Oracle) (*VitalsLock, error) {
	members, err := Party(o)
	if err != nil {
		return nil, err
	}
	lock := &VitalsLock{Members: len(members), From: o.Steps()}
	o.OnFrame(func(o *oracle.Oracle) {
		for _, m := range members {
			max := o.Byte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + offHitPointsMax})
			if max == 0 {
				continue
			}
			if o.Byte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + offHitPoints}) != max {
				o.SetByte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + offHitPoints}, max)
				lock.Writes++
			}
			if o.Byte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + offFood}) < 100 {
				o.SetByte(oracle.Addr{Seg: m.At.Seg, Off: m.At.Off + offFood}, 100)
				lock.Writes++
			}
		}
	})
	return lock, nil
}

// 武器格的點擊區，來源 `buttonDefs[13..16]`（`clickedWeaponSlot`）：
// 四個角色各 31×33，涵蓋上下兩隻手各 16 高。
var weaponSlotRect = [4][2]int{{216, 10}, {288, 10}, {216, 62}, {288, 62}}

// AttackSlot 點某個角色的某隻手（hand 0 ＝ 上、1 ＝ 下）。
func AttackSlot(o *oracle.Oracle, character, hand int) error {
	if character < 0 || character >= len(weaponSlotRect) || hand < 0 || hand > 1 {
		return fmt.Errorf("攻擊格 character=%d hand=%d 超出範圍", character, hand)
	}
	x := weaponSlotRect[character][0] + 15
	y := weaponSlotRect[character][1] + 8 + hand*17
	return o.Click(x, y, oracle.Hover(0), oracle.Hold(200_000), oracle.Settle(400_000))
}

// AttackAll 讓全隊各揮一次上手武器。
//
// ⚠ **點不到不算錯。** 角色沒武器、在冷卻中、或前排擋著時原版本來就不回應；
// 這裡把「畫面完全沒變」當成正常，只把真正的錯誤往上傳。
func AttackAll(o *oracle.Oracle, count int) (int, error) {
	swings := 0
	for round := 0; round < count; round++ {
		for character := range weaponSlotRect {
			err := AttackSlot(o, character, 0)
			if err == nil {
				swings++
				continue
			}
			var noResponse *oracle.NoResponseError
			if !errors.As(err, &noResponse) {
				return swings, err
			}
		}
	}
	return swings, nil
}
