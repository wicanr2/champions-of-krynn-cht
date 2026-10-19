package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withFakeOriginals 把必要檔換成測試內容（不需原版）。
func withFakeOriginals(t *testing.T) map[string][]byte {
	t.Helper()
	files := map[string][]byte{"START.EXE": []byte("fake start"), "GAME.OVR": []byte("fake ovr")}
	old := requiredOriginals
	t.Cleanup(func() { requiredOriginals = old })
	requiredOriginals = nil
	for _, n := range []string{"START.EXE", "GAME.OVR"} {
		s := sha256.Sum256(files[n])
		requiredOriginals = append(requiredOriginals, struct{ name, sha string }{n, hex.EncodeToString(s[:])})
	}
	return files
}

func TestPrepareGameImportsOnceCaseInsensitive(t *testing.T) {
	files := withFakeOriginals(t)
	orig, data := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(orig, "start.exe"), files["START.EXE"], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orig, "Game.Ovr"), files["GAME.OVR"], 0o644); err != nil {
		t.Fatal(err)
	}
	game, start, err := prepareGame(orig, data)
	if err != nil {
		t.Fatal(err)
	}
	if game != filepath.Join(data, "game") || start != "start.exe" {
		t.Fatalf("game=%s start=%s", game, start)
	}
	// 第二次不再需要原版
	if _, _, err := prepareGame(filepath.Join(orig, "沒有"), data); err != nil {
		t.Fatalf("已匯入後應只核對 game/：%v", err)
	}
	// game/ 被改壞時要求重新匯入
	if err := os.WriteFile(filepath.Join(game, "Game.Ovr"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareGame(orig, data); err == nil || !strings.Contains(err.Error(), "核對失敗") {
		t.Fatalf("game/ 雜湊不符時應失敗：%v", err)
	}
}

func TestPrepareGameMissingOriginal(t *testing.T) {
	withFakeOriginals(t)
	data := t.TempDir()
	_, _, err := prepareGame(filepath.Join(data, "nowhere"), data)
	if err == nil || !strings.Contains(err.Error(), "找不到原版") {
		t.Fatalf("缺原版時應給說明：%v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "game")); err == nil {
		t.Fatal("失敗時不得留下 game/")
	}
}

func TestPrepareGameHashMismatch(t *testing.T) {
	files := withFakeOriginals(t)
	orig, data := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(orig, "START.EXE"), files["START.EXE"], 0o644)
	os.WriteFile(filepath.Join(orig, "GAME.OVR"), []byte("other version"), 0o644)
	_, _, err := prepareGame(orig, data)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 不符") {
		t.Fatalf("版本不符時應說明雜湊：%v", err)
	}
}

func TestRequiredOriginalsMatchBaseline(t *testing.T) {
	// 主專案 docs/re/001
	if requiredOriginals[0].sha != startEXESHA256 || requiredOriginals[1].sha != "c3f448c391ea860f97dec99ce88b0b8b0a4a7e807cf3b1f5a8f1515e2a7a0529" {
		t.Fatal("必要檔雜湊與 docs/re/001 不符")
	}
}
