package main

// 發行版啟動（主專案規格 012 §3.1）：路徑解析、第一次匯入原版、之後的雜湊核對與致命錯誤的呈現。
// 沒給 -save 時走發行模式；給了就是開發模式（tools/play.sh 與收據）。
// 做法沿用 cmd/buckrogers-play/launcher.go（拯救地球規格 035 §3.1–3.3）。

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wicanr2/dosgolem/bootroot"
)

// 由打包以 -ldflags -X 注入（規格 012 §3.1）。
var version, dosgolemCommit string

// 原版必要檔（主專案 docs/re/001）。覆繪依賴這兩個檔的位址。
var requiredOriginals = []struct{ name, sha string }{
	{"START.EXE", startEXESHA256},
	{"GAME.OVR", "fdbe7265ac9b04c67f8648331d9845bf33c6a1438e676389eae3f4526796c817"},
}

const errorLogName = "dkk-error.log"

// logDir 是致命錯誤紀錄的位置；發行模式確定使用者資料目錄後設定。
var logDir string

func versionString() string {
	v, c := version, dosgolemCommit
	if v == "" {
		v = "dev"
	}
	if c == "" {
		c = "unknown"
	}
	return fmt.Sprintf("版本 %s（dosgolem %s）\n", v, c)
}

// userDataDir 回各平台的使用者資料目錄。
func userDataDir() (string, error) {
	home, err := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "DKKCHT"), nil
		}
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming", "DKKCHT"), nil
	case "darwin":
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "DKKCHT"), nil
	default:
		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
			return filepath.Join(d, "dkk-cht"), nil
		}
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "dkk-cht"), nil
	}
}

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Dir(p)
}

// packageDir 是玩家看得到的發行包所在目錄：AppImage 檔、.app 或 exe 的目錄。
func packageDir() string {
	if a := os.Getenv("APPIMAGE"); a != "" {
		return filepath.Dir(a)
	}
	d := exeDir()
	if i := strings.Index(d, ".app"+string(filepath.Separator)+"Contents"); i >= 0 {
		return filepath.Dir(d[:i+len(".app")])
	}
	return d
}

// resourceCandidates 是發行包內資料的搜尋順序：執行檔目錄、macOS 的 Resources、工作目錄。
func resourceCandidates(rel string) []string {
	d := exeDir()
	cands := []string{filepath.Join(d, rel), filepath.Join(d, "..", "Resources", rel)}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, filepath.Join(wd, rel))
	}
	return cands
}

// resourcePath 回第一個存在的發行包內資料。
func resourcePath(rel string) (string, error) {
	cands := resourceCandidates(rel)
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("找不到 %s（發行包可能不完整，請重新下載）；找過：%s", rel, strings.Join(cands, "、"))
}

// releaseFont 回發行模式的字型：本機完整版附的倚天優先，其次 Unifont。
func releaseFont() (string, error) {
	var err error
	for _, name := range []string{"dkk-eten.golemfnt", "dkk-unifont.golemfnt"} {
		var p string
		if p, err = resourcePath(filepath.Join("font", name)); err == nil {
			return p, nil
		}
	}
	return "", err
}

// releaseHD 回發行包內的 HD 主題目錄；原版包沒有，回空字串。
func releaseHD() string {
	if p, err := resourcePath(filepath.Join("hd", "theme.json")); err == nil {
		return filepath.Dir(p)
	}
	return ""
}

// findFold 在 dir 的直接項目中找與 name 大小寫不拘相同的一般檔。
func findFold(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var hits []string
	for _, e := range entries {
		if strings.EqualFold(e.Name(), name) {
			hits = append(hits, e.Name())
		}
	}
	switch {
	case len(hits) == 0:
		return "", fmt.Errorf("%s 裡沒有 %s", dir, name)
	case len(hits) > 1:
		return "", fmt.Errorf("%s 裡有多個大小寫不同的 %s：%s", dir, name, strings.Join(hits, "、"))
	}
	info, err := os.Lstat(filepath.Join(dir, hits[0]))
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s 不是一般檔案", filepath.Join(dir, hits[0]))
	}
	return hits[0], nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// locateOriginals 找出 dir 裡必要檔的實際檔名並核對雜湊。
func locateOriginals(dir string) (map[string]string, error) {
	names := map[string]string{}
	for _, r := range requiredOriginals {
		n, err := findFold(dir, r.name)
		if err != nil {
			return nil, err
		}
		got, err := fileSHA256(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		if got != r.sha {
			return nil, fmt.Errorf("%s 的 SHA-256 不符：預期 %s，實際 %s（本中文化只支援 DOS 英文版 1.10）", filepath.Join(dir, n), r.sha, got)
		}
		names[r.name] = n
	}
	return names, nil
}

// originalCandidates 是原版目錄的搜尋順序：旗標 → 發行包旁 → 發行包內（本機完整版）→ 使用者資料目錄。
func originalCandidates(flagDir, data string) []string {
	if flagDir != "" {
		return []string{flagDir}
	}
	d := exeDir()
	return []string{filepath.Join(packageDir(), "original"), filepath.Join(d, "original"),
		filepath.Join(d, "..", "Resources", "original"), filepath.Join(data, "original")}
}

// prepareGame 回可開機的遊戲樹（存檔也在裡面）與 START.EXE 的實際檔名。
func prepareGame(flagOriginal, data string) (string, string, error) {
	game := filepath.Join(data, "game")
	if info, err := os.Stat(game); err == nil && info.IsDir() {
		names, err := locateOriginals(game)
		if err != nil {
			return "", "", fmt.Errorf("遊戲資料 %s 核對失敗：%v\n請刪除這個資料夾後重新啟動，程式會重新匯入原版（存檔也在裡面，請先備份）。", game, err)
		}
		return game, names["START.EXE"], nil
	}
	if err := os.MkdirAll(data, 0o755); err != nil {
		return "", "", err
	}
	if stale, _ := filepath.Glob(filepath.Join(data, "game.importing-*")); len(stale) > 0 {
		for _, s := range stale {
			if err := os.RemoveAll(s); err != nil {
				return "", "", err
			}
		}
	}
	var orig string
	var names map[string]string
	var tried []string
	for _, c := range originalCandidates(flagOriginal, data) {
		n, err := locateOriginals(c)
		if err == nil {
			orig, names = c, n
			break
		}
		tried = append(tried, fmt.Sprintf("  %s：%v", c, err))
	}
	if orig == "" {
		return "", "", fmt.Errorf("%s\n找過：\n%s", missingOriginalHelp(data), strings.Join(tried, "\n"))
	}
	var req []bootroot.RequiredFile
	for _, r := range requiredOriginals {
		sum, _ := hex.DecodeString(r.sha)
		var want [32]byte
		copy(want[:], sum)
		req = append(req, bootroot.RequiredFile{Name: names[r.name], SHA256: want})
	}
	tmp := filepath.Join(data, fmt.Sprintf("game.importing-%d", os.Getpid()))
	if _, err := bootroot.Prepare(bootroot.BootRootInput{OriginalRoot: orig, SaveRoot: tmp, Required: req}); err != nil {
		os.RemoveAll(tmp)
		return "", "", fmt.Errorf("匯入原版失敗：%w", err)
	}
	if err := os.Rename(tmp, game); err != nil {
		os.RemoveAll(tmp)
		return "", "", fmt.Errorf("匯入原版失敗：%w", err)
	}
	return game, names["START.EXE"], nil
}

func missingOriginalHelp(data string) string {
	example := "把原版遊戲資料夾（含 START.EXE、GAME.OVR）複製成本程式旁邊的 original 資料夾"
	switch runtime.GOOS {
	case "windows":
		example += `，或執行 DKKCHT.exe -original D:\games\DKKRYNN`
	case "darwin":
		example += "（與 DKKCHT.app 同一層），或在終端機執行 DKKCHT.app/Contents/MacOS/dkk-play -original ~/games/DKKRYNN"
	default:
		example += "（與 AppImage 同一層），或執行 ./DKKCHT-x86_64.AppImage -original ~/games/DKKRYNN"
	}
	return fmt.Sprintf("找不到原版《Death Knights of Krynn》（DOS 英文版 1.10）。\n本程式不附原版遊戲，請自備合法取得的原版：\n%s；\n也可以放在 %s。",
		example, filepath.Join(data, "original"))
}

// fatal 同時輸出 stderr、錯誤紀錄與（Windows）對話框，然後結束。
func fatal(err error) {
	msg := err.Error()
	fmt.Fprintln(os.Stderr, "dkk-play：", msg)
	path := ""
	if logDir != "" {
		if os.MkdirAll(logDir, 0o755) == nil {
			path = filepath.Join(logDir, errorLogName)
			wd, _ := os.Getwd()
			body := fmt.Sprintf("時間：%s\n%s命令列：%q\n工作目錄：%s\n\n%s\n", time.Now().Format(time.RFC3339), versionString(), os.Args, wd, msg)
			if os.WriteFile(path, []byte(body), 0o644) != nil {
				path = ""
			}
		}
	}
	if os.Getenv("DKK_NO_DIALOG") != "1" {
		text := msg
		if path != "" {
			text += "\n\n錯誤紀錄：" + path
		}
		messageBox(windowTitle, text)
	}
	os.Exit(1)
}

// clearErrorLog 在啟動成功後刪掉上一次的錯誤紀錄。
func clearErrorLog() {
	if logDir != "" {
		if err := os.Remove(filepath.Join(logDir, errorLogName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "dkk-play: 刪除舊錯誤紀錄失敗：", err)
		}
	}
}
