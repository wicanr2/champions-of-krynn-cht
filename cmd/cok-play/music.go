package main

// 遊戲中配樂（主專案規格 014 §4）：第二顆 Nuked OPL 即時合成配樂的 OPL2 寫入序列，疊加在遊戲音訊上。
// 遊戲本身的 OPL 與喇叭聲道照常；配樂關閉時完全不混入，輸出與原版相同。

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/cok"
	"github.com/wicanr2/dosgolem/audio/mixer"
	"github.com/wicanr2/dosgolem/audio/nukedopl"
)

const (
	musicFade = 0.5  // 換曲淡出秒數
	musicGain = 1.0 // 整體音量；各曲增益在 music.tsv（對齊原版標題曲響度的 0.7 倍）
)

type oplWrite struct {
	at       float64
	reg, val uint8
}

type musicTrack struct {
	key                 string
	writes              []oplWrite
	loop                bool
	loopStart, loopEnd  float64
	length              float64
	gain                float64
}

// musicPlayer 依 MusicDirector 的選曲播放；Mix 在每次混音後呼叫。
type musicPlayer struct {
	tracks  map[string]*musicTrack
	dir     *cok.MusicDirector
	chip    *nukedopl.Chip
	cur     *musicTrack
	pos     float64 // 目前曲目的秒數
	idx     int     // 下一筆待寫入
	vol     float64 // 淡出中的音量（1 → 0）
	fading  bool
	serial  int
	enabled bool
	frame   [2]int16
}

// loadMusic 讀 dir/music.tsv（key、loop、loop_start、loop_end、length、gain）與各曲 <key>.opl。
func loadMusic(dir string, d *cok.MusicDirector) (*musicPlayer, error) {
	f, err := os.Open(filepath.Join(dir, "music.tsv"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	p := &musicPlayer{tracks: map[string]*musicTrack{}, dir: d, chip: nukedopl.New(mixer.SampleRate), vol: 1, enabled: true}
	sc := bufio.NewScanner(f)
	for ln := 1; sc.Scan(); ln++ {
		fs := strings.Split(sc.Text(), "\t")
		if ln == 1 || len(fs) == 1 && fs[0] == "" {
			continue
		}
		if len(fs) != 6 {
			return nil, fmt.Errorf("music.tsv:%d：要 6 欄", ln)
		}
		t := &musicTrack{key: fs[0], loop: fs[1] == "1"}
		var e1, e2, e3, e4 error
		t.loopStart, e1 = strconv.ParseFloat(fs[2], 64)
		t.loopEnd, e2 = strconv.ParseFloat(fs[3], 64)
		t.length, e3 = strconv.ParseFloat(fs[4], 64)
		t.gain, e4 = strconv.ParseFloat(fs[5], 64)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || t.gain <= 0 || t.length <= 0 || t.loop && !(t.loopStart < t.loopEnd && t.loopEnd <= t.length) {
			return nil, fmt.Errorf("music.tsv:%d：時間欄不合法", ln)
		}
		if t.writes, err = readOPL(filepath.Join(dir, t.key+".opl")); err != nil {
			return nil, err
		}
		p.tracks[t.key] = t
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(p.tracks) == 0 {
		return nil, fmt.Errorf("%s：music.tsv 沒有曲目", dir)
	}
	return p, nil
}

func readOPL(path string) ([]oplWrite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ws []oplWrite
	sc := bufio.NewScanner(f)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fs := strings.Fields(line)
		if len(fs) != 3 {
			return nil, fmt.Errorf("%s:%d：要三欄", path, ln)
		}
		at, err := strconv.ParseFloat(fs[0], 64)
		reg, err1 := strconv.ParseUint(fs[1], 16, 8)
		val, err2 := strconv.ParseUint(fs[2], 16, 8)
		if err != nil || err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s:%d：格式錯誤", path, ln)
		}
		if reg >= 0x02 && reg <= 0x04 {
			continue // 計時器暫存器不送合成器（同 audio/mixer）
		}
		ws = append(ws, oplWrite{at, uint8(reg), uint8(val)})
	}
	sort.SliceStable(ws, func(a, b int) bool { return ws[a].at < ws[b].at })
	return ws, sc.Err()
}

// start 從 at 秒開始播 t：重設合成器，把 at 之前的寫入立即套用以重建音色狀態。
func (p *musicPlayer) start(t *musicTrack, at float64) {
	p.chip.Reset(mixer.SampleRate)
	p.cur, p.pos, p.idx = t, at, 0
	if t == nil {
		return
	}
	for p.idx < len(t.writes) && t.writes[p.idx].at < at {
		w := t.writes[p.idx]
		if w.reg&0xF0 != 0xB0 { // 不重放 Key-On：循環點之前的音不延續
			p.chip.WriteReg(uint16(w.reg), w.val)
		}
		p.idx++
	}
}

// Mix 把配樂疊加到交錯立體聲 dst（float32，-1..1）。
func (p *musicPlayer) Mix(dst []float32) {
	if p == nil {
		return
	}
	want, oneShot, serial := p.dir.Want()
	if !p.enabled {
		want = ""
	}
	target := p.tracks[want]
	restart := oneShot && serial != p.serial
	if (target != p.cur || restart) && !p.fading {
		if p.cur == nil {
			p.vol = 1
			p.serial = serial
			p.start(target, 0)
		} else {
			p.fading = true
		}
	}
	for i := 0; i+1 < len(dst); i += 2 {
		if p.fading {
			p.vol -= 1 / (musicFade * mixer.SampleRate)
			if p.vol <= 0 {
				p.fading, p.vol, p.serial = false, 1, serial
				p.start(target, 0)
			}
		}
		t := p.cur
		if t == nil {
			continue
		}
		for p.idx < len(t.writes) && t.writes[p.idx].at <= p.pos {
			p.chip.WriteRegBuffered(uint16(t.writes[p.idx].reg), t.writes[p.idx].val)
			p.idx++
		}
		p.chip.GenerateStream(p.frame[:])
		g := musicGain * t.gain * p.vol / 32768
		dst[i] = clampF(dst[i] + float32(float64(p.frame[0])*g))
		dst[i+1] = clampF(dst[i+1] + float32(float64(p.frame[1])*g))
		p.pos += 1.0 / mixer.SampleRate
		switch {
		case t.loop && p.pos >= t.loopEnd:
			p.start(t, t.loopStart)
		case !t.loop && p.pos >= t.length:
			p.dir.OneShotDone()
			p.start(nil, 0)
			want, _, serial = p.dir.Want()
			target = p.tracks[want]
			if !p.enabled {
				target = nil
			}
			p.serial = serial
			p.start(target, 0)
		}
	}
}

// Toggle 切換配樂開關（切換列「配樂」、F4）；關閉時立即停止混入。
func (p *musicPlayer) Toggle() {
	if p == nil {
		return
	}
	p.enabled = !p.enabled
	if !p.enabled {
		p.fading = false
		p.start(nil, 0)
	}
}

func clampF(v float32) float32 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}
