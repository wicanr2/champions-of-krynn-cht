// cok-hd-frame-check 比對獨立建立的整幀 HD／原版預期，不執行或修改 DOS 遊戲。
// 主專案規格 010 §6；來源 fixture 與原版保存畫面由 cases.group 分開。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/xlate"
)

type frameCase struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Indexed     string `json:"indexed"`
	IndexedSHA  string `json:"indexed_sha256"`
	Expected    string `json:"expected"`
	ExpectedSHA string `json:"expected_sha256"`
	Original    string `json:"original"`
	OriginalSHA string `json:"original_sha256"`
}

func checked(path, sha string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(b)
	if hex.EncodeToString(hash[:]) != sha {
		return nil, fmt.Errorf("%s SHA 不符", path)
	}
	return b, nil
}

func pixels(path, sha string) ([]byte, error) {
	b, err := checked(path, sha)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != 960 || img.Bounds().Dy() != 600 {
		return nil, fmt.Errorf("%s 預期圖不是 960×600", path)
	}
	rgba := image.NewNRGBA(image.Rect(0, 0, 960, 600))
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba.Pix, nil
}

func main() {
	dir := flag.String("hd", "", "含 theme.json 與可選 combat.json 的主題目錄")
	casesPath := flag.String("cases", "", "獨立預期整幀索引 JSON")
	group := flag.String("group", "", "只執行指定證據類型")
	receipt := flag.String("receipt", "", "輸出 JSONL 收據；空值使用 stdout")
	flag.Parse()
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	data, err := os.ReadFile(*casesPath)
	if err != nil {
		fail(err)
	}
	var cases struct {
		Cases []frameCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		fail(err)
	}
	live := cok.NewHDLive(&xlate.Font{W: 16, H: 16, Glyphs: map[rune][]byte{}}, nil, nil)
	if err := live.AddHD(*dir); err != nil {
		fail(err)
	}
	ega := [16][3]uint8{{0, 0, 0}, {0, 0, 170}, {0, 170, 0}, {0, 170, 170}, {170, 0, 0}, {170, 0, 170}, {170, 85, 0}, {170, 170, 170},
		{85, 85, 85}, {85, 85, 255}, {85, 255, 85}, {85, 255, 255}, {255, 85, 85}, {255, 85, 255}, {255, 255, 85}, {255, 255, 255}}
	var palette [256][3]uint8
	for i := range palette {
		palette[i] = ega[i&15]
	}
	out := os.Stdout
	if *receipt != "" {
		out, err = os.Create(*receipt)
		if err != nil {
			fail(err)
		}
		defer out.Close()
	}
	encoder := json.NewEncoder(out)
	allOK, count := true, 0
	for _, test := range cases.Cases {
		if *group != "" && test.Group != *group {
			continue
		}
		indexed, err := checked(test.Indexed, test.IndexedSHA)
		if err != nil {
			fail(err)
		}
		if len(indexed) != 320*200 {
			fail(fmt.Errorf("%s 原版色號長度不符", test.Name))
		}
		expected, err := pixels(test.Expected, test.ExpectedSHA)
		if err != nil {
			fail(err)
		}
		original, err := pixels(test.Original, test.OriginalSHA)
		if err != nil {
			fail(err)
		}
		before := sha256.Sum256(indexed)
		live.Frame(indexed, palette)
		hd := live.Compose(3, true, nil)
		difference := 0
		for n := 0; n < len(expected); n += 4 {
			if !bytes.Equal(hd[n:n+4], expected[n:n+4]) {
				difference++
			}
		}
		sameInput := before == sha256.Sum256(indexed) && bytes.Equal(live.Indexed(), indexed)
		sameOriginal := bytes.Equal(live.Compose(3, false, nil), original)
		sameTwo := bytes.Equal(live.Compose(2, true, nil), live.Compose(2, false, nil))
		sameToggle := bytes.Equal(hd, live.Compose(3, true, nil))
		ok := difference == 0 && sameInput && sameOriginal && sameTwo && sameToggle
		row := map[string]any{"screen": test.Indexed, "name": test.Name, "group": test.Group,
			"diff_pixels": difference, "same_input": sameInput, "same_original": sameOriginal,
			"same_2x": sameTwo, "same_toggle": sameToggle, "ok": ok}
		if err := encoder.Encode(row); err != nil {
			fail(err)
		}
		allOK = allOK && ok
		count++
	}
	if count == 0 {
		fail(fmt.Errorf("沒有指定類型的案例"))
	}
	if !allOK {
		os.Exit(1)
	}
}
