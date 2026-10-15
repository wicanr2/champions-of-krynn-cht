// Command oracle 在 dosgolem 上執行原版 EOB3 的 `AESOP.EXE eye start`，
// 依腳本在固定的指令數注入滑鼠與鍵盤，擷取 320×200 的畫面。
//
// 規格與操作說明：`docs/spec/200-eob3-oracle-adapter`。
//
//	oracle -exe EOB3/AESOP.EXE -root EOB3 -out out/ \
//	       -script 'until=620000000;shot=menu'
//
// ⚠ **本程式不含任何原版檔案。** `-exe` 與 `-root` 由使用者提供，唯讀；
// 輸出（PNG、色號、調色盤）是原版畫面的衍生物，只放在 `-out`，不得提交或散布。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/eob3"
	"github.com/wicanr2/dosgolem/oracle"
)

func main() {
	exe := flag.String("exe", "", "AESOP.EXE 的路徑（必填）")
	root := flag.String("root", "", "EOB3 資料夾（必填，唯讀）")
	out := flag.String("out", "", "輸出目錄（必填；PNG、色號、調色盤、receipt.json）")
	scratch := flag.String("scratch", "", "可寫暫存層，必須是空的（預設 <out>/scratch）")
	script := flag.String("script", "", "腳本，動作用 `;` 或換行分隔（見 docs/spec/200）")
	budget := flag.Uint64("budget", 2_000_000_000, "單一動作的指令數上限")
	flag.Parse()
	if *exe == "" || *root == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*exe, *root, *out, *scratch, *script, *budget); err != nil {
		fmt.Fprintln(os.Stderr, "oracle:", err)
		os.Exit(1)
	}
}

// shotRecord 是一張擷取畫面的收據。**不含任何路徑**（只有輸出檔的 basename）。
type shotRecord struct {
	Name       string `json:"name"`
	Step       uint64 `json:"step"`
	VideoMode  uint8  `json:"video_mode"`
	IndexedSHA string `json:"indexed_sha256"`
	PaletteSHA string `json:"palette_sha256"`
	PNG        string `json:"png"`
}

type receipt struct {
	Exe        string       `json:"exe"`
	ExeSHA     string       `json:"exe_sha256"`
	ResSHA     string       `json:"eye_res_sha256"`
	Script     string       `json:"script"`
	Steps      uint64       `json:"steps"`
	Shots      []shotRecord `json:"shots"`
	Opened     []string     `json:"opened"`
	Exec       []string     `json:"exec"`
	Unimpl     []string     `json:"unimplemented"`
	Console    string       `json:"console"`
	FinalState string       `json:"final_state"`
}

func run(exe, root, out, scratch, script string, budget uint64) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if scratch == "" {
		scratch = filepath.Join(out, "scratch")
	}
	if ents, err := os.ReadDir(scratch); err == nil && len(ents) > 0 {
		return fmt.Errorf("暫存層 %s 不是空的；殘留檔案會蓋過原版目錄並改變結果，請換一個空目錄", scratch)
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		return err
	}
	o, err := eob3.Load(exe, root, scratch)
	if err != nil {
		return err
	}
	defer o.Close()

	rc := receipt{Exe: filepath.Base(exe), Script: script}
	rc.ExeSHA = fileSHA(exe)
	rc.ResSHA = fileSHA(filepath.Join(root, "EYE.RES"))

	var runErr error
	for _, act := range splitScript(script) {
		if runErr = doAction(o, act, out, budget, &rc); runErr != nil {
			runErr = fmt.Errorf("動作 %q（第 %d 道指令）：%w", act, o.Steps(), runErr)
			break
		}
	}
	rc.Steps = o.Steps()
	rc.Opened = o.Opened()
	for _, e := range o.ExecLog() {
		rc.Exec = append(rc.Exec, fmt.Sprintf("%s exit=%d", e.Name, e.Exit))
	}
	rc.Unimpl = o.Unimplemented()
	rc.Console = o.Console()
	rc.FinalState = fmt.Sprintf("CS:IP=%s mode=%02Xh", o.IP(), o.VideoMode())
	b, _ := json.MarshalIndent(rc, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), b, 0o644); err != nil {
		return err
	}
	return runErr
}

func splitScript(s string) []string {
	s = strings.ReplaceAll(s, "\n", ";")
	var out []string
	for _, a := range strings.Split(s, ";") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func doAction(o *oracle.Oracle, act, out string, budget uint64, rc *receipt) error {
	name, arg, _ := strings.Cut(act, "=")
	switch name {
	case "run": // run=N：再跑 N 道
		n, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return err
		}
		return o.RunUntil(oracle.Steps(n), oracle.Budget(n+1))
	case "until": // until=N：跑到絕對步數 N
		n, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return err
		}
		if o.Steps() > n {
			return fmt.Errorf("已經跑過第 %d 道指令", n)
		}
		return o.RunUntil(oracle.NewCond(fmt.Sprintf("到第 %d 道", n),
			func(o *oracle.Oracle) bool { return o.Steps() >= n }), oracle.Budget(n-o.Steps()+1))
	case "idle": // idle=N：畫面連續 N 道指令不變
		n, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return err
		}
		return o.RunUntil(oracle.ScreenIdle(n), oracle.Budget(budget))
	case "opened": // opened=NAME：程式開了這個檔
		return o.RunUntil(oracle.Opened(arg), oracle.Budget(budget))
	case "key": // key=NAME：送一次按下與放開（硬體 IRQ1）
		k, err := eob3.KeyByName(strings.ToLower(arg))
		if err != nil {
			return err
		}
		o.PressKey(k)
		return nil
	case "move": // move=X,Y：游標移到像素座標（320×200）
		x, y, err := xy(arg)
		if err != nil {
			return err
		}
		return o.MoveMouse(x, y)
	case "click": // click=X,Y[,r]
		parts := strings.Split(arg, ",")
		if len(parts) < 2 {
			return fmt.Errorf("click 要 X,Y")
		}
		x, y, err := xy(parts[0] + "," + parts[1])
		if err != nil {
			return err
		}
		opts := []oracle.ClickOpt{oracle.NoCursorWait()}
		if len(parts) > 2 && parts[2] == "r" {
			opts = append(opts, oracle.RightButton())
		}
		// 點擊後畫面沒變不是錯誤（有些點擊的回饋晚於預設的等待）；
		// 收據只記步數與之後的畫面。
		if err := o.Click(x, y, opts...); err != nil {
			if _, ok := err.(*oracle.NoResponseError); ok {
				return nil
			}
			return err
		}
		return nil
	case "marker": // marker=INTR：在 DOS 低記憶體 0:04F0 寫階段標記（`docs/spec/200` §4）
		if len(arg) != 4 {
			return fmt.Errorf("標記是四個字元（INTR、CINE、VICT、CHGN）：%q", arg)
		}
		// 腳本把 32 位元值 0x494E5452 寫成小端位元組，所以記憶體裡是反過來的
		// 字元順序（"RTNI"）。這裡收的是「讀作」的字串，與 docs/re 的寫法一致。
		b := []byte{arg[3], arg[2], arg[1], arg[0]}
		o.SetBytes(oracle.Far(0, eob3.MarkerOffset), b)
		return nil
	case "shot": // shot=NAME：存 NAME.png、NAME.idx、NAME.pal
		return shot(o, arg, out, rc)
	case "text": // text=NAME：存 NAME.txt（文字模式畫面）
		return os.WriteFile(filepath.Join(out, arg+".txt"),
			[]byte(strings.Join(o.TextScreen(), "\n")+"\n"), 0o644)
	case "mark": // mark=LABEL：印一行目前狀態
		fmt.Printf("[%s] step=%d ip=%s mode=%02Xh\n", arg, o.Steps(), o.IP(), o.VideoMode())
		return nil
	}
	return fmt.Errorf("不認得的動作 %q", name)
}

func xy(s string) (int, int, error) {
	a, b, ok := strings.Cut(s, ",")
	if !ok {
		return 0, 0, fmt.Errorf("座標要 X,Y：%q", s)
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(a))
	y, err2 := strconv.Atoi(strings.TrimSpace(b))
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("座標要整數：%q", s)
	}
	return x, y, nil
}

func shot(o *oracle.Oracle, name, out string, rc *receipt) error {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("shot 的名字不能含路徑：%q", name)
	}
	idx := o.Indexed()
	pal := o.Palette()
	flat := make([]byte, 0, 768)
	for _, c := range pal {
		flat = append(flat, c[0], c[1], c[2])
	}
	png := filepath.Join(out, name+".png")
	if err := o.WritePNG(png); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, name+".idx"), idx, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, name+".pal"), flat, 0o644); err != nil {
		return err
	}
	rec := shotRecord{Name: name, Step: o.Steps(), VideoMode: o.VideoMode(),
		IndexedSHA: sum(idx), PaletteSHA: sum(flat), PNG: name + ".png"}
	rc.Shots = append(rc.Shots, rec)
	fmt.Printf("shot %s step=%d mode=%02Xh indexed=%s… palette=%s…\n",
		name, rec.Step, rec.VideoMode, rec.IndexedSHA[:12], rec.PaletteSHA[:12])
	return nil
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func fileSHA(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "（讀不到）"
	}
	return sum(b)
}
