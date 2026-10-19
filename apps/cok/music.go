package cok

import (
	"fmt"
	"strings"
)

// MusicDirector 依畫面上的訊號決定該播哪一首配樂（主專案規格 014 §4）。只讀覆繪已攔到的文字與地圖鍵，
// 不改遊戲狀態；選曲表來自 text/music-cues.tsv。
//
// 三層：區域曲（地圖鍵、城鎮與大地圖選單、營地與神殿選單）、戰鬥曲（戰鬥選單出現到勝負訊息）、
// 短曲（勝利、全滅，播完回到下面一層）。
type MusicDirector struct {
	menus []musicCue          // 底列選單：整列以 pattern 開頭
	texts []musicCue          // 路徑 A、B 文字：包含 pattern
	maps  map[string]string   // 地圖鍵 → 區域曲
	battleTrack, bossTrack string

	area    string // 目前區域曲
	mapKey  string
	combat  bool
	boss    bool
	oneShot string // 進行中的短曲；播放端播完呼叫 OneShotDone
	serial  int    // 短曲序號：同一首連續觸發兩次也要重播
}

type cueKind int

const (
	cueArea   cueKind = iota // 區域曲
	cueBattle                // 戰鬥開始
	cueBoss                  // 強敵名（戰鬥中）
	cueEnd                   // 戰鬥結束後的短曲
	cueLeave                 // 回到 3D 移動（營地、神殿結束），區域曲回到目前地圖
)

type musicCue struct {
	pattern string
	track   string
	kind    cueKind
}

// LoadMusicCues 讀 music-cues.tsv：kind、pattern、track。kind 為 map（地圖鍵）、menu（area）、
// battle、leave、boss、end。
func LoadMusicCues(b []byte) (*MusicDirector, error) {
	rows, err := readTSV(b, []string{"kind", "pattern", "track"})
	if err != nil {
		return nil, err
	}
	d := &MusicDirector{maps: map[string]string{}}
	for _, r := range rows {
		kind, pat, track := r[0], r[1], r[2]
		switch kind {
		case "map":
			d.maps[pat] = track
		case "menu":
			d.menus = append(d.menus, musicCue{pat, track, cueArea})
		case "battle":
			d.menus = append(d.menus, musicCue{pat, track, cueBattle})
			d.battleTrack = track
		case "leave":
			d.menus = append(d.menus, musicCue{pat, track, cueLeave})
		case "boss":
			d.texts = append(d.texts, musicCue{pat, track, cueBoss})
			d.bossTrack = track
		case "end":
			d.texts = append(d.texts, musicCue{pat, track, cueEnd})
		default:
			return nil, fmt.Errorf("music-cues：未知 kind %q", kind)
		}
	}
	return d, nil
}

// Menu 收一列底列選單（詞以單一空白相連）。
func (d *MusicDirector) Menu(line string) {
	if d == nil {
		return
	}
	for _, c := range d.menus {
		if !strings.HasPrefix(line, c.pattern) {
			continue
		}
		switch c.kind {
		case cueArea:
			d.combat, d.boss = false, false
			d.area = c.track
		case cueBattle:
			if !d.combat {
				d.combat, d.boss = true, false
			}
		case cueLeave:
			d.combat, d.boss = false, false
			if t, ok := d.maps[d.mapKey]; ok {
				d.area = t
			}
		}
		return
	}
}

// Text 收一段路徑 A 或 B 的原文。
func (d *MusicDirector) Text(s string) {
	if d == nil {
		return
	}
	for _, c := range d.texts {
		if !strings.Contains(s, c.pattern) {
			continue
		}
		switch c.kind {
		case cueBoss:
			if d.combat {
				d.boss = true
			}
		case cueEnd:
			d.combat, d.boss = false, false
			d.oneShot = c.track
			d.serial++
		}
		return
	}
}

// MapKey 收目前地圖鍵（AutoMap.Key）；換到有對應曲目的地圖時換區域曲（戰鬥中先記下）。
func (d *MusicDirector) MapKey(k string) {
	if d == nil || k == d.mapKey {
		return
	}
	d.mapKey = k
	if t, ok := d.maps[k]; ok {
		d.area = t
	}
}

// OneShotDone 由播放端在短曲播完時呼叫。
func (d *MusicDirector) OneShotDone() {
	if d != nil {
		d.oneShot = ""
	}
}

// Want 回目前該播的曲目與短曲序號（序號變了表示要從頭播短曲）；空字串表示維持靜音。
func (d *MusicDirector) Want() (track string, oneShot bool, serial int) {
	if d == nil {
		return "", false, 0
	}
	if d.oneShot != "" {
		return d.oneShot, true, d.serial
	}
	if d.combat && d.boss && d.bossTrack != "" {
		return d.bossTrack, false, d.serial
	}
	if d.combat && d.battleTrack != "" {
		return d.battleTrack, false, d.serial
	}
	return d.area, false, d.serial
}
