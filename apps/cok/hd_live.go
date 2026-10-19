package cok

import "github.com/wicanr2/dosgolem/xlate"

// HDLive 沿用 Live 的配樂、印字與遊戲攔截，僅在 Compose 增加圖面。
// live.go 的修改尚待寫入範圍確認，獨立收據入口先使用此包裝。
type HDLive struct {
	*Live
	*HDLayers
}

func NewHDLive(font *xlate.Font, ui Lookuper, ecl *Catalog) *HDLive {
	return &HDLive{Live: NewLive(font, ui, ecl), HDLayers: &HDLayers{}}
}

func (l *HDLive) AddHD(dir string) error {
	items, err := LoadHDTheme(dir)
	if err != nil {
		return err
	}
	layers, count, err := LoadHDLayers(dir)
	if err != nil {
		return err
	}
	// 全部資產核對成功後才登記，不留下部分主題。
	for _, it := range items {
		l.Layer.Watch(it.Watcher())
	}
	l.HDLayers = layers
	l.HDItems += len(items) + count
	return nil
}

func (l *HDLive) Frame(idx []byte, pal [256][3]uint8) {
	l.HDLayers.Frame(idx, pal)
	l.Live.Frame(idx, pal)
}

func (l *HDLive) Compose(scale int, overlay bool, missing func(rune)) []byte {
	out := l.Live.Compose(scale, false, missing)
	if out == nil || !overlay {
		return out
	}
	l.HDLayers.Draw(out, l.indexed, scale)
	l.Layer.Draw(out, scale, missing)
	if l.Journal.Open() {
		l.Journal.Draw(out, scale, l.Font, missing)
	}
	if l.Guide.Open() {
		l.Guide.Draw(out, scale, l.Font, missing)
	}
	if l.Map.Open() {
		l.Map.Draw(out, scale, l.Font, missing)
	}
	return out
}

// HDAllowed 供命令列收據列出已匹配的原版像素範圍，獨立預期另由 Pillow 核對。
func (l *HDLive) HDAllowed(x, y int) bool {
	h := l.HDLayers
	if h == nil {
		return false
	}
	if h.combat != nil {
		for _, m := range h.combat.matches {
			if x >= m.x && x < m.x+24 && y >= m.y && y < m.y+24 {
				return true
			}
		}
	}
	if h.actors != nil {
		for _, m := range h.actors.matches {
			if m.conflict {
				continue
			}
			p := &h.actors.patterns[m.pattern]
			for _, pt := range p.ink {
				if x == m.x+pt.x && y == m.y+pt.y {
					return true
				}
			}
		}
	}
	if h.encounters != nil {
		for _, n := range h.encounters.matches {
			it := &h.encounters.items[n]
			for _, pt := range it.ink {
				if x == it.x+pt.x && y == it.y+pt.y {
					return true
				}
			}
		}
	}
	return false
}
