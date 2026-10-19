package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// installCoK 讓遊戲樹可以在 dosgolem 開機，可重複呼叫：
//   - KRYNN.CFG 第 2 行是音效卡。原版附的設定選 R（Roland LAPC），dosgolem 沒有 MPU-401，程式會關中斷輪詢埠 331h
//     不結束（主專案 docs/re/003）。改成 A（AdLib），等同玩家在安裝選單選 [3] AdLib。
//   - 原版的預建隊伍放在 SAVE/ 子目錄，安裝程式 COPYSAVE.BAT 會複製到存檔目錄；dosgolem 只在根目錄依檔名找，
//     所以比照複製到根目錄，已存在的檔不覆寫（主專案 docs/re/004）。
func installCoK(root string) error {
	if name, err := findFold(root, "KRYNN.CFG"); err == nil {
		p := filepath.Join(root, name)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines := bytes.Split(b, []byte("\r\n"))
		if len(lines) > 1 && string(lines[1]) == "R" {
			lines[1] = []byte("A")
			if err := os.WriteFile(p, bytes.Join(lines, []byte("\r\n")), 0o644); err != nil {
				return fmt.Errorf("改寫 KRYNN.CFG：%w", err)
			}
		}
	}
	saveDir, err := findFold(root, "SAVE")
	if err != nil {
		return nil
	}
	ents, err := os.ReadDir(filepath.Join(root, saveDir))
	if err != nil {
		return nil
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if _, err := findFold(root, e.Name()); err == nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, saveDir, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, strings.ToUpper(e.Name())), b, 0o644); err != nil {
			return fmt.Errorf("複製預建隊伍 %s：%w", e.Name(), err)
		}
	}
	return nil
}
