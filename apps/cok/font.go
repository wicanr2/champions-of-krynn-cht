package cok

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/dosgolem/xlate"
)

// LoadFontScaled 載入 16×16 GOLEMFNT；同目錄有 <名稱>-3x.golemfnt（24×24）時登記為 3 倍字模
// （英文版 Courier、繁中圓體；主專案規格 015 §3.1、使用者 2026-10-21 指示），否則 3 倍由 16×16 放大。
func LoadFontScaled(path string) (*xlate.Font, error) {
	f, err := xlate.LoadFont(path)
	if err != nil {
		return nil, err
	}
	f.Name = filepath.Base(path)
	p3 := strings.TrimSuffix(path, ".golemfnt") + "-3x.golemfnt"
	if st, err := os.Stat(p3); err != nil || !st.Mode().IsRegular() {
		return f, nil
	}
	f3, err := xlate.LoadFont(p3)
	if err != nil {
		return nil, err
	}
	if f3.W != 24 || f3.H != 24 {
		return nil, fmt.Errorf("%s：3 倍字模要是 24×24", p3)
	}
	f3.Name = filepath.Base(p3)
	f.Scaled = map[int]xlate.ScaledFont{3: {Font: f3}}
	return f, nil
}
