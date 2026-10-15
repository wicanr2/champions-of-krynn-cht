package dos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/machine"
)

// 相對子目錄解析（`docs/spec/198-dos-relative-directories`）。
//
// 背景：`resolve` 原本只認 basename。AESOP 的 `SAVEGAME\ITEMS_00.BIN`、
// `CHARGEN\CHGEN.EXE` 是真實子目錄，只看 basename 會找不到，而程式把「找不到」
// 當成要從 `.TXT` 翻譯、翻譯又失敗，開機以離開碼 1 結束。

// newTree 造一棵 EOB3 形狀的資料夾，回 (機器, DOS, root, scratch)。
func newTree(t *testing.T, programPath string) (*machine.Machine, *DOS, string, string) {
	t.Helper()
	root, scratch := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{
		"EYE.RES":                "res",
		"SAVEGAME/ITEMS_00.BIN":  "items",
		"SAVEGAME/LVL01_00.BIN":  "lvl1",
		"CHARGEN/ITEM.DAT":       "item",
		"CHARGEN/CHGEN.EXE":      "exe",
		"CHARGEN/SUB/DEEP.TXT":   "deep",
		"DATA/readme.txt":        "readme",
		"SAVEGAME/lowercase.bin": "low",
		"SAVEGAME/LONGFILE.BIN":  "long",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := machine.New()
	m.ProgramPath = programPath
	d := New(m, root)
	d.Scratch = scratch
	d.Install()
	return m, d, root, scratch
}

func TestProgramDir(t *testing.T) {
	for in, want := range map[string]string{
		`C:\EOB3\AESOP.EXE`:   "EOB3",
		`c:\eob3\aesop.exe`:   "EOB3",
		`C:\GAMES\EOB3\A.EXE`: `GAMES\EOB3`,
		`C:\START.EXE`:        "",
		``:                    "",
		`AESOP.EXE`:           "",
		`C:/EOB3/AESOP.EXE`:   "EOB3",
	} {
		if got := programDir(in); got != want {
			t.Errorf("programDir(%q)＝%q，預期 %q", in, got, want)
		}
	}
	m, d, _, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	_ = m
	if d.Dir != "EOB3" || d.RootDir != "EOB3" {
		t.Errorf("Dir＝%q RootDir＝%q，預期都是 EOB3", d.Dir, d.RootDir)
	}
	// 沒給程式路徑：維持舊的預設（rich2 的 RICH2）。
	_, d2, _, _ := newTree(t, "")
	if d2.Dir != "RICH2" || d2.RootDir != "RICH2" {
		t.Errorf("預設 Dir＝%q RootDir＝%q，預期 RICH2", d2.Dir, d2.RootDir)
	}
}

func TestResolveRealSubdirectories(t *testing.T) {
	_, d, root, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	want := func(name, rel string) {
		t.Helper()
		got := d.resolve(name)
		if got != filepath.Join(root, filepath.FromSlash(rel)) {
			t.Errorf("resolve(%q)＝%q，預期 root 底下的 %s", name, got, rel)
		}
	}
	want(`SAVEGAME\ITEMS_00.BIN`, "SAVEGAME/ITEMS_00.BIN")
	want(`savegame\items_00.bin`, "SAVEGAME/ITEMS_00.BIN") // 大小寫不分
	want(`SAVEGAME/ITEMS_00.BIN`, "SAVEGAME/ITEMS_00.BIN") // 正斜線
	want(`.\SAVEGAME\ITEMS_00.BIN`, "SAVEGAME/ITEMS_00.BIN")
	want(`C:\EOB3\SAVEGAME\ITEMS_00.BIN`, "SAVEGAME/ITEMS_00.BIN") // 絕對路徑，RootDir 前綴去掉
	want(`\EOB3\CHARGEN\ITEM.DAT`, "CHARGEN/ITEM.DAT")
	want(`CHARGEN\SUB\DEEP.TXT`, "CHARGEN/SUB/DEEP.TXT")
	want(`SAVEGAME\LONGFILENAME.BIN`, "SAVEGAME/LONGFILE.BIN") // 程式給長名字、磁碟上是 8.3
	want(`DATA\README.TXT`, "DATA/readme.txt")
	// 根目錄的檔：舊規則（basename）。
	want(`EYE.RES`, "EYE.RES")
	want(`C:\EOB3\EYE.RES`, "EYE.RES")
	// junk 路徑（多磁片殘留）：目錄對不上，退回 basename。
	want(`A:\<垃圾>\EYE.RES`, "EYE.RES")
	want(`C:\OTHER\EYE.RES`, "EYE.RES")
	// 目錄存在但檔案不在裡面：退回 basename，仍找不到就是找不到。
	if got := d.resolve(`CHARGEN\EYE.RES`); got != filepath.Join(root, "EYE.RES") {
		t.Errorf("目錄存在但檔不在：預期退回 basename 找到根目錄的 EYE.RES，得到 %q", got)
	}
	if got := d.resolve(`SAVEGAME\NOPE.BIN`); got != "" {
		t.Errorf("不存在的檔回了 %q", got)
	}
}

// 目前目錄：chdir 之後相對路徑跟著換；AH=47 回得出來；.. 退得回去。
func TestChdirMovesRelativeLookups(t *testing.T) {
	m, d, root, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	chdir := func(path string) {
		t.Helper()
		putFileName(m, path)
		call(m, d, 0x21, 0x3B00)
		if m.CPU.Flags&cpu.CF != 0 {
			t.Fatalf("chdir %s 失敗", path)
		}
	}
	getcwd := func() string {
		m.CPU.Seg[cpu.DS], m.CPU.R[cpu.SI] = fileScratchSeg, 0x200
		call(m, d, 0x21, 0x4700)
		return d.readCString(fileScratchSeg, 0x200, 64)
	}
	if got := getcwd(); got != "EOB3" {
		t.Fatalf("起始目錄 %q", got)
	}
	chdir("CHARGEN")
	if got := getcwd(); got != `EOB3\CHARGEN` {
		t.Errorf("chdir CHARGEN 後 getcwd＝%q，預期 EOB3\\CHARGEN", got)
	}
	// 在 CHARGEN 底下，沒有目錄的名字落在 CHARGEN。
	if got := d.resolve(`ITEM.DAT`); got != filepath.Join(root, "CHARGEN", "ITEM.DAT") {
		t.Errorf("cwd＝CHARGEN 時 resolve(ITEM.DAT)＝%q", got)
	}
	chdir("SUB")
	if got := d.resolve(`DEEP.TXT`); got != filepath.Join(root, "CHARGEN", "SUB", "DEEP.TXT") {
		t.Errorf("cwd＝CHARGEN\\SUB 時 resolve(DEEP.TXT)＝%q", got)
	}
	chdir(`..\..`)
	if got := getcwd(); got != "EOB3" {
		t.Errorf("chdir ..\\.. 之後 getcwd＝%q，預期回到 EOB3", got)
	}
	if got := d.resolve(`SAVEGAME\ITEMS_00.BIN`); got != filepath.Join(root, "SAVEGAME", "ITEMS_00.BIN") {
		t.Errorf("回到根目錄後 resolve＝%q", got)
	}
	// 退過磁碟機根目錄就停在根目錄。
	chdir(`..\..\..`)
	if got := getcwd(); got != "" {
		t.Errorf("退過根目錄：getcwd＝%q，預期空字串（磁碟機根目錄）", got)
	}
	chdir(`\EOB3\CHARGEN`) // 絕對路徑
	if got := getcwd(); got != `EOB3\CHARGEN` {
		t.Errorf("絕對 chdir：%q", got)
	}
}

// 寫入：帶真實目錄的路徑寫進暫存層的同名子目錄，原版目錄不動；讀得回來。
func TestScratchWritesFollowDirectories(t *testing.T) {
	m, d, root, scratch := newTree(t, `C:\EOB3\AESOP.EXE`)
	// 建新檔。
	putFileName(m, `SAVEGAME\NEWFILE.BIN`)
	call(m, d, 0x21, 0x3C00)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Fatalf("create 失敗：AX＝%04X", m.CPU.R[cpu.AX])
	}
	h := m.CPU.R[cpu.AX]
	m.CPU.R[cpu.BX], m.CPU.R[cpu.CX] = h, 4
	m.CPU.Seg[cpu.DS], m.CPU.R[cpu.DX] = fileScratchSeg, 0x100
	m.WriteBytes(cpu.Addr(fileScratchSeg, 0x100), []byte("abcd"))
	call(m, d, 0x21, 0x4000)
	m.CPU.R[cpu.BX] = h
	call(m, d, 0x21, 0x3E00)

	if _, err := os.Stat(filepath.Join(scratch, "SAVEGAME", "NEWFILE.BIN")); err != nil {
		t.Errorf("新檔應該落在 scratch/SAVEGAME/：%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "SAVEGAME", "NEWFILE.BIN")); err == nil {
		t.Error("原版目錄不該被寫入")
	}
	if got := d.resolve(`SAVEGAME\NEWFILE.BIN`); got != filepath.Join(scratch, "SAVEGAME", "NEWFILE.BIN") {
		t.Errorf("寫進去的檔讀不回來：resolve＝%q", got)
	}

	// 以寫入模式開原版目錄裡的檔：寫時複製到暫存層同名位置。
	putFileName(m, `SAVEGAME\ITEMS_00.BIN`)
	call(m, d, 0x21, 0x3D02)
	if m.CPU.Flags&cpu.CF != 0 {
		t.Fatalf("開寫入失敗：AX＝%04X", m.CPU.R[cpu.AX])
	}
	if _, err := os.Stat(filepath.Join(scratch, "SAVEGAME", "ITEMS_00.BIN")); err != nil {
		t.Errorf("寫時複製應該落在 scratch/SAVEGAME/：%v", err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "SAVEGAME", "ITEMS_00.BIN"))
	if string(b) != "items" {
		t.Errorf("原版檔被改了：%q", b)
	}

	// 沒有真實目錄的路徑：維持舊規則，攤平成 scratch/<basename>。
	putFileName(m, `C:\NOWHERE\FLAT.DAT`)
	call(m, d, 0x21, 0x3C00)
	if _, err := os.Stat(filepath.Join(scratch, "FLAT.DAT")); err != nil {
		t.Errorf("junk 路徑應該攤平：%v", err)
	}

	// 刪除與改名也走同一套位置。
	putFileName(m, `SAVEGAME\NEWFILE.BIN`)
	call(m, d, 0x21, 0x4100)
	if _, err := os.Stat(filepath.Join(scratch, "SAVEGAME", "NEWFILE.BIN")); err == nil {
		t.Error("unlink 沒刪到 scratch/SAVEGAME/NEWFILE.BIN")
	}
}

func TestFindFirstInSubdirectory(t *testing.T) {
	m, d, _, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	m.CPU.Seg[cpu.DS], m.CPU.R[cpu.DX] = fileScratchSeg, 0
	m.WriteBytes(cpu.Addr(fileScratchSeg, 0), append([]byte(`SAVEGAME\*.BIN`), 0))
	st := d.searchFor(`SAVEGAME\*.BIN`, 0)
	got := strings.Join(st.names, ",")
	if got != "ITEMS_00.BIN,LONGFILE.BIN,LOWERCASE.BIN,LVL01_00.BIN" {
		t.Errorf(`SAVEGAME\*.BIN ＝ %q`, got)
	}
	// 根目錄的樣式：不看子目錄（舊行為）。
	if got := strings.Join(d.searchFor(`*.RES`, 0).names, ","); got != "EYE.RES" {
		t.Errorf("*.RES ＝ %q", got)
	}
}

func TestDirectoryAttributes(t *testing.T) {
	m, d, _, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	attr := func(name string) (uint16, bool) {
		putFileName(m, name)
		m.CPU.R[cpu.CX] = 0xFFFF
		call(m, d, 0x21, 0x4300)
		return m.CPU.R[cpu.CX], m.CPU.Flags&cpu.CF == 0
	}
	if cx, ok := attr(`SAVEGAME`); !ok || cx != 0x10 {
		t.Errorf("目錄屬性：CX＝%04X ok＝%v，預期 0010", cx, ok)
	}
	if cx, ok := attr(`SAVEGAME\ITEMS_00.BIN`); !ok || cx != 0x20 {
		t.Errorf("檔案屬性：CX＝%04X ok＝%v，預期 0020", cx, ok)
	}
	if _, ok := attr(`NODIR`); ok {
		t.Error("不存在的名字不該成功")
	}
	if _, ok := attr(`SAVEGAME\NOPE.BIN`); ok {
		t.Error("不存在的檔不該成功")
	}
}

// 預設（舊行為）：Dir 是 RICH2，Root 底下的相對路徑照舊走 basename 規則。
func TestLegacyDefaultKeepsBasenameRule(t *testing.T) {
	_, d, root, _ := newTree(t, "")
	if got := d.resolve(`EYE.RES`); got != filepath.Join(root, "EYE.RES") {
		t.Errorf("resolve(EYE.RES)＝%q", got)
	}
	if got := d.resolve(`C:\RICH2\EYE.RES`); got != filepath.Join(root, "EYE.RES") {
		t.Errorf("絕對路徑 RICH2：%q", got)
	}
	// 子目錄在預設情況下也成立（路徑存在就走目錄）。
	if got := d.resolve(`SAVEGAME\ITEMS_00.BIN`); got != filepath.Join(root, "SAVEGAME", "ITEMS_00.BIN") {
		t.Errorf("預設 RootDir 下的子目錄：%q", got)
	}
}

// 狀態檔要留住 RootDir 與 Dir。
func TestStateKeepsRootDir(t *testing.T) {
	m, d, _, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
	d.Dir = `EOB3\CHARGEN`
	var buf strings.Builder
	_ = buf
	tmp := filepath.Join(t.TempDir(), "d.state")
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SaveState(f); err != nil {
		t.Fatal(err)
	}
	f.Close()
	m2 := machine.New()
	d2 := New(m2, d.Root) // 沒給 ProgramPath：RootDir 先是預設 RICH2
	d2.Install()
	r, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := d2.LoadState(r); err != nil {
		t.Fatal(err)
	}
	if d2.RootDir != "EOB3" || d2.Dir != `EOB3\CHARGEN` {
		t.Errorf("還原後 RootDir＝%q Dir＝%q", d2.RootDir, d2.Dir)
	}
	_ = m
}

// `StrictChdir`（`docs/spec/198` §5）：目標目錄不存在就失敗，目前目錄不變。
// 預設（舊行為）一律成功。
func TestStrictChdir(t *testing.T) {
	try := func(strict bool, path string) (ok bool, dir string, ax uint16) {
		m, d, _, _ := newTree(t, `C:\EOB3\AESOP.EXE`)
		d.StrictChdir = strict
		putFileName(m, path)
		call(m, d, 0x21, 0x3B00)
		return m.CPU.Flags&cpu.CF == 0, d.Dir, m.CPU.R[cpu.AX]
	}
	// 預設：不存在的目錄也成功，Dir 跟著變（舊行為）。
	if ok, dir, _ := try(false, "CINE.EXE"); !ok || dir != `EOB3\CINE.EXE` {
		t.Errorf("預設 chdir 不存在的目錄：ok＝%v Dir＝%q，預期成功且 Dir ＝ EOB3\\CINE.EXE", ok, dir)
	}
	// 嚴格：不存在 → CF、AX＝3、Dir 不變。
	if ok, dir, ax := try(true, "CINE.EXE"); ok || dir != "EOB3" || ax != 3 {
		t.Errorf("嚴格 chdir 不存在的目錄：ok＝%v Dir＝%q AX＝%d，預期失敗、Dir ＝ EOB3、AX ＝ 3", ok, dir, ax)
	}
	// 嚴格：存在的目錄、絕對路徑、根目錄、`..` 都成功。
	for path, want := range map[string]string{
		"CHARGEN":        `EOB3\CHARGEN`,
		`CHARGEN\SUB`:    `EOB3\CHARGEN\SUB`,
		`\EOB3\SAVEGAME`: `EOB3\SAVEGAME`,
		`C:\EOB3`:        "EOB3",
		`\`:              "",
		"..":             "",
	} {
		if ok, dir, _ := try(true, path); !ok || dir != want {
			t.Errorf("嚴格 chdir %q：ok＝%v Dir＝%q，預期成功、Dir ＝ %q", path, ok, dir, want)
		}
	}
}
