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
			if font, err = xlate.LoadFont(path); err != nil {
				return nil, fmt.Errorf("語言 %s 的字型：%w", code, err)
			}
			font.Name = filepath.Base(path)
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

// pickLang 回起始語言的索引：-lang 優先，其次是上次的選擇，都沒有就是第一種（繁中）。
func pickLang(packs []*cok.LangPack, flagLang, stateDir string) (int, error) {
	want := flagLang
	if want == "" && stateDir != "" {
		if b, err := os.ReadFile(filepath.Join(stateDir, langChoiceFile)); err == nil {
			want = strings.TrimSpace(string(b))
		}
	}
	for i, p := range packs {
		if p.Code == want {
			return i, nil
		}
	}
	if flagLang != "" {
		return 0, fmt.Errorf("-lang %s：沒有這個語言或缺字型", flagLang)
	}
	return 0, nil
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
