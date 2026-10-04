// Package eob3 保存《Eye of the Beholder III》原版（`AESOP.EXE`）的導航知識。
// 原版執行檔與資料由使用者提供，套件本身不含任何遊戲內容。
//
// 規格：`docs/spec/200-eob3-oracle-adapter`。
package eob3

import (
	"fmt"

	"github.com/wicanr2/dosgolem/oracle"
)

// ProgramPath 是環境區塊裡的程式全路徑（argv[0]）。
//
// **這個值有後果**：`dos.New` 由它推出 `RootDir`（`EOB3`），相對路徑
// （`SAVEGAME\ITEMS_00.BIN`、`CHARGEN\CHGEN.EXE`）才會落在資料夾底下
// （`docs/spec/198-dos-relative-directories`）。給錯目錄名，第二層以下的
// 檔案全部找不到，而程式把「找不到」當成要從 `.TXT` 翻譯，開機以離開碼 1 結束。
const ProgramPath = `C:\EOB3\AESOP.EXE`

// MarkerOffset 是 DOS 低記憶體裡程式間通訊的階段標記位址（0000:04F0，BIOS 資料區之後
// 的 Intra-Application Communication Area）。`AESOP.EXE` 的 `start` 腳本一開頭
// `peekmem(0x4F0)`：0 走「第一次開機」（建初始檔、初始化音效、寫 `INTR`、`launch cine.exe`
// 播片頭），`INTR` 走「片頭播完」那一支（進主選單）。**32 位元，小端**。
const MarkerOffset = 0x4F0

// CommandTail 是命令列尾（PSP+80h 的內容，**含前導空白**，DOS 的行為）。
// 原版 `EYE.BAT` 是 `aesop eye start`。
const CommandTail = " eye start"

// Load 載入 `AESOP.EXE`，設好原版開機需要的環境：
//
//   - 命令列 `aesop eye start`、程式路徑 `C:\EOB3\AESOP.EXE`；
//   - 可寫暫存層 scratch（存檔、翻譯檔落在這裡，root 永遠唯讀；
//     **每次執行要給一個空的目錄**，暫存層會蓋過 root，殘留檔案會改變結果）；
//   - AdLib（OPL2）偵測存在、輸出丟棄（`hardware-spec approximation`）：
//     音效驅動 `ADLIB.ADV` 偵測失敗時程式印「Music initialization failed」並
//     等一個鍵，**停在文字模式**，看起來像開機卡住。
func Load(exe, root, scratch string) (*oracle.Oracle, error) {
	o, err := oracle.LoadWith(exe, root, oracle.Options{ProgramPath: ProgramPath})
	if err != nil {
		return nil, err
	}
	o.SetCmdLine(CommandTail)
	if scratch != "" {
		o.SetScratch(scratch)
	}
	o.AdLib(true)
	// Borland exec 的載入器 stub 與環境區塊要與真 DOS 一樣不重疊
	// （`docs/spec/199-psp-block-resize-boundary` §3）。
	o.StrictPSPGrowth(true)
	// `launch` 會 chdir 到一個不存在的目錄名；真 DOS 回失敗、目前目錄不變，
	// 之後 exec 的路徑才是 `C:\EOB3\CINE.EXE` 而不是 `C:\EOB3\CINE.EXE\CINE.EXE`
	// （`docs/spec/198-dos-relative-directories` §5）。
	o.StrictChdir(true)
	return o, nil
}

// Keys 是 IBM PC/AT 鍵盤 Set 1 的 make 掃描碼，只放腳本會用到的鍵。
// 名字全小寫。
var Keys = map[string]oracle.Key{
	"esc": 0x01, "enter": 0x1C, "space": 0x39, "tab": 0x0F, "backspace": 0x0E,
	"up": 0x48, "down": 0x50, "left": 0x4B, "right": 0x4D,
	"f1": 0x3B, "f2": 0x3C, "f3": 0x3D, "f4": 0x3E, "f5": 0x3F,
	"f6": 0x40, "f7": 0x41, "f8": 0x42, "f9": 0x43, "f10": 0x44,
	"1": 0x02, "2": 0x03, "3": 0x04, "4": 0x05, "5": 0x06,
	"6": 0x07, "7": 0x08, "8": 0x09, "9": 0x0A, "0": 0x0B,
	"q": 0x10, "w": 0x11, "e": 0x12, "r": 0x13, "t": 0x14,
	"y": 0x15, "u": 0x16, "i": 0x17, "o": 0x18, "p": 0x19,
	"a": 0x1E, "s": 0x1F, "d": 0x20, "f": 0x21, "g": 0x22,
	"h": 0x23, "j": 0x24, "k": 0x25, "l": 0x26,
	"z": 0x2C, "x": 0x2D, "c": 0x2E, "v": 0x2F, "b": 0x30,
	"n": 0x31, "m": 0x32,
}

// KeyByName 依名字找掃描碼。
func KeyByName(name string) (oracle.Key, error) {
	if k, ok := Keys[name]; ok {
		return k, nil
	}
	return 0, fmt.Errorf("eob3：不認得按鍵 %q", name)
}
