package main

// 多語系（主專案規格 015 §2）：譯文目錄裡每種有字型的語言載成一個語言包，F8 與切換列循環切換，
// 選擇記在前端資料目錄的 cok-lang.txt。只有一種語言時不保留 F8，切換列與多語系前相同。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/xlate"
)

const langChoiceFile = "cok-lang.txt"

// parseLangFonts 解析 -lang-fonts（代碼=字型路徑，逗號分隔）。
func parseLangFonts(s string) (map[string]string, error) {
	m := map[string]string{}
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f == "" {
			continue
		}
		code, path, ok := strings.Cut(f, "=")
		if !ok || code == "" || path == "" {
			return nil, fmt.Errorf("-lang-fonts：%q 要寫成 代碼=路徑", f)
		}
		m[code] = path
	}
	return m, nil
}

// langFontPath 回 code 的字型：-lang-fonts 指定的優先，發行模式再找 font/cok-<代碼>.golemfnt；找不到回空字串。
func langFontPath(code string, fonts map[string]string, release bool) string {
	if p := fonts[code]; p != "" {
		return p
	}
	if release {
		if p, err := resourcePath(filepath.Join("font", "cok-"+code+".golemfnt")); err == nil {
			return p
		}
	}
	return ""
}

// loadLangs 依 cok.AvailableLangs 的順序載入語言包；繁中用 main（-font 或發行包預設字型）。
// 其他語言沒有字型就略過並在 stderr 說明；有字型但表不完整則回錯。
func loadLangs(textDir string, main *xlate.Font, fonts map[string]string, release bool) ([]*cok.LangPack, error) {
	codes, err := cok.AvailableLangs(textDir)
	if err != nil {
		return nil, err
	}
	var packs []*cok.LangPack
	for _, code := range codes {
		font := main
		if code != cok.DefaultLang {
			path := langFontPath(code, fonts, release)
			if path == "" {
				fmt.Fprintf(os.Stderr, "cok-play: 語言 %s 沒有字型，略過\n", code)
				continue
			}
			// 同目錄的 <名稱>-3x.golemfnt（24×24）是 3 倍畫面專用字模（英文版，規格 015 §3.1）
			if font, err = cok.LoadFontScaled(path); err != nil {
				return nil, fmt.Errorf("語言 %s 的字型：%w", code, err)
			}
		}
		p, err := cok.LoadLangPack(textDir, code, font)
		if err != nil {
			return nil, fmt.Errorf("語言 %s：%w", code, err)
		}
		if err := checkToolbarText(p.Host); err != nil {
			return nil, fmt.Errorf("語言 %s：%w", code, err)
		}
		packs = append(packs, p)
	}
	for _, p := range packs {
		if len(packs) > 1 && p.Host["lang.name"] == "" {
			return nil, fmt.Errorf("語言 %s：host-text.%s.tsv 缺 lang.name（切換列的語言名稱）", p.Code, p.Code)
		}
		// 項目最多的狀態（沒有 HD 素材時多一個說明項）也要放得進 320 像素寬
		items := toolbarItems(p.Host, toolbarState{scale: 2, langs: len(packs)})
		if last := items[len(items)-1]; (last.col+last.cols)*8 > 320 {
			return nil, fmt.Errorf("語言 %s：切換列寬 %d 像素，超過 320", p.Code, (last.col+last.cols)*8)
		}
	}
	if len(packs) == 0 {
		return nil, fmt.Errorf("%s 沒有可用的語言", textDir)
	}
	return packs, nil
}

// pickLang 回起始語言的索引：-lang 優先，其次是上次的選擇，再來是系統語言（第一次執行），都沒有就是第一種（繁中）。
func pickLang(packs []*cok.LangPack, flagLang, stateDir string) (int, error) {
	find := func(code string) int {
		for i, p := range packs {
			if code != "" && p.Code == code {
				return i
			}
		}
		return -1
	}
	if flagLang != "" {
		if i := find(flagLang); i >= 0 {
			return i, nil
		}
		return 0, fmt.Errorf("-lang %s：沒有這個語言或缺字型", flagLang)
	}
	if stateDir != "" {
		if b, err := os.ReadFile(filepath.Join(stateDir, langChoiceFile)); err == nil {
			if i := find(strings.TrimSpace(string(b))); i >= 0 {
				return i, nil
			}
		}
	}
	if i := find(localeLang(systemLocale())); i >= 0 {
		return i, nil
	}
	return 0, nil
}

// systemLocale 回作業系統的語言設定（lang_sys_*.go）；測試可以換掉。
var systemLocale = osLocale

// localeLang 把系統語言標記（zh_TW.UTF-8、zh-Hant-TW、ja-JP、ko_KR 等）對到語言代碼。
// 中文以字體（Hant、Hans）或地區分繁簡，只寫 zh 時用繁中；C、POSIX 與空字串回空；其他語言回英文。
func localeLang(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexAny(tag, ".@"); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" || tag == "c" || tag == "posix" {
		return ""
	}
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '_' || r == '-' })
	switch parts[0] {
	case "zh":
		for _, p := range parts[1:] {
			switch p {
			case "hans", "cn", "sg", "my":
				return "zh-CN"
			case "hant", "tw", "hk", "mo":
				return cok.DefaultLang
			}
		}
		return cok.DefaultLang
	case "ja":
		return "ja"
	case "ko":
		return "ko"
	}
	return cok.EnglishLang
}

// envLocale 依 POSIX 優先序讀語言環境變數：LANGUAGE（冒號分隔的第一項）、LC_ALL、LC_MESSAGES、LANG。
func envLocale() string {
	if v, _, _ := strings.Cut(os.Getenv("LANGUAGE"), ":"); v != "" {
		return v
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// nextLang 換到下一種語言（F8，最後一種之後回第一種）並記下選擇。
func (g *game) nextLang() {
	if len(g.langs) < 2 {
		return
	}
	g.setLang((g.lang + 1) % len(g.langs))
}

// cycleLang 是切換列語言項：原版 → 第一種語言 → … → 最後一種 → 原版。
func (g *game) cycleLang() {
	switch {
	case !g.chinese:
		g.chinese = true
		g.setLang(0)
	case g.lang == len(g.langs)-1:
		g.chinese = false
	default:
		g.setLang(g.lang + 1)
	}
}

func (g *game) setLang(i int) {
	if i == g.lang && g.live.Lang() == g.langs[i] {
		return
	}
	g.lang = i
	p := g.langs[g.lang]
	g.live.SetLang(p)
	g.barText = p.Host
	if g.langDir != "" {
		if err := os.WriteFile(filepath.Join(g.langDir, langChoiceFile), []byte(p.Code+"\n"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "cok-play: 語言選擇寫入失敗：", err)
		}
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// english 回目前是否為英文版。
func (g *game) english() bool {
	return len(g.langs) > 0 && g.langs[g.lang].Code == cok.EnglishLang
}

// barFont 回切換列的字型：英文字型沒有半形組合字與中文，英文版的切換列改用預設字型（-font）。
func (g *game) barFont() *xlate.Font {
	if g.english() && g.mainFont != nil {
		return g.mainFont
	}
	return g.live.Font
}
