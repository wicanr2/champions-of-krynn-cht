package main

import "syscall"

// osLocale 回 Windows 介面語言：GetUserDefaultUILanguage 的 LANGID 對到語言標記。
// 中文以子語言分繁簡（0x0404 台灣、0x0C04 香港、0x1404 澳門為繁中，其餘為簡中）。
func osLocale() string {
	if v := envLocale(); v != "" {
		return v
	}
	r, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage").Call()
	return langIDTag(uint16(r))
}

func langIDTag(id uint16) string {
	switch id & 0x3ff {
	case 0x04:
		switch id {
		case 0x0404, 0x0c04, 0x1404:
			return "zh-TW"
		}
		return "zh-CN"
	case 0x11:
		return "ja"
	case 0x12:
		return "ko"
	case 0x00:
		return ""
	}
	return "en"
}
