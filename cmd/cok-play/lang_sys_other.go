//go:build !windows

package main

import (
	"os/exec"
	"regexp"
	"runtime"
	"time"
)

var appleLang = regexp.MustCompile(`"?([A-Za-z]{2,3}(?:[-_][A-Za-z0-9]+)*)"?`)

// osLocale 回語言環境變數；macOS 從 Finder 開啟時沒有 LANG，改讀 defaults 的 AppleLanguages 第一項。
func osLocale() string {
	if v := envLocale(); v != "" || runtime.GOOS != "darwin" {
		return v
	}
	cmd := exec.Command("/usr/bin/defaults", "read", "-g", "AppleLanguages")
	done := make(chan []byte, 1)
	go func() { out, _ := cmd.Output(); done <- out }()
	select {
	case out := <-done:
		// 輸出是 ( "zh-Hant-TW", "en-US" ) 這樣的清單
		for _, m := range appleLang.FindAllStringSubmatch(string(out), -1) {
			return m[1]
		}
	case <-time.After(2 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	return ""
}
