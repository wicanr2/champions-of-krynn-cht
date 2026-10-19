package cok

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 全部來源組合的 RGBA 預期由 tools/hd_battle_receipts.py 的 NumPy 合成產生。
func TestHDIssue13CharacterMatrix(t *testing.T) {
	root := os.Getenv("DOSGOLEM_COK_HD_MATRIX")
	if root == "" {
		t.Skip("未指定本機頭身矩陣")
	}
	raw, err := os.ReadFile(filepath.Join(root, "out/issue13-frames/character-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Count int `json:"count"`
		Items []struct {
			Key     string `json:"key"`
			Indexed string `json:"indexed_sha256"`
			RGBA    string `json:"rgba_sha256"`
		} `json:"items"`
	}
	if err = json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for _, r := range expected.Items {
		if _, dup := want[r.Indexed]; dup {
			t.Fatal("預期矩陣重複")
		}
		want[r.Indexed] = r.RGBA
	}
	chars, err := loadCharacterHD(filepath.Join(root, "theme"))
	if err != nil || chars == nil {
		t.Fatalf("頭身載入：%v", err)
	}
	if len(chars.patterns) != 6400 || expected.Count != 6400 || len(want) != 6400 {
		t.Fatal("矩陣不是6400個方向")
	}
	out := make([]byte, 960*600*4)
	pixels := make([]byte, 72*72*4)
	for _, p := range chars.patterns {
		for y := 0; y < 72; y++ {
			clear(out[y*960*4 : y*960*4+72*4])
		}
		for _, point := range p.ink {
			p.drawCharacterPoint(out, point, 0, 0)
		}
		for y := 0; y < 72; y++ {
			copy(pixels[y*72*4:(y+1)*72*4], out[y*960*4:y*960*4+72*4])
		}
		indexed := sha256.Sum256(p.full)
		rgba := sha256.Sum256(pixels)
		key := hex.EncodeToString(indexed[:])
		if want[key] != hex.EncodeToString(rgba[:]) {
			t.Fatalf("%s 來源或RGBA不同", p.key)
		}
	}
	t.Log("6400個來源與方向的完整RGBA全部相同")
}
