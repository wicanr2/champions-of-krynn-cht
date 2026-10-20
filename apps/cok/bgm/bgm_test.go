package bgm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/apps/cok"
)

// 兩首小曲：loop（循環 0.5–1.0 秒）、hit（短曲 0.2 秒）。
func writeMusicFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	note := "0 20 01\n0 23 01\n0 40 10\n0 43 00\n0 60 F0\n0 63 F0\n0 80 77\n0 83 77\n0 A0 98\n0 B0 31\n0.4 B0 11\n"
	os.WriteFile(filepath.Join(dir, "loop.opl"), []byte(note+"0.5 B0 31\n0.9 B0 11\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "hit.opl"), []byte(note), 0o644)
	os.WriteFile(filepath.Join(dir, "music.tsv"), []byte("key\tloop\tloop_start\tloop_end\tlength\tgain\nloop\t1\t0.5\t1.0\t1.0\t1\nhit\t0\t0\t0\t0.2\t1\n"), 0o644)
	return dir
}

func musicDirector(t *testing.T) *cok.MusicDirector {
	t.Helper()
	d, err := cok.LoadMusicCues([]byte("kind\tpattern\ttrack\nmenu\tMove Encamp\tloop\nend\tThe party has won\thit\n"))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func energy(b []float32) float64 {
	var e float64
	for _, v := range b {
		e += float64(v * v)
	}
	return e
}

func TestMusicPlayerSilentUntilCue(t *testing.T) {
	p, err := Load(writeMusicFixture(t), musicDirector(t))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]float32, 4800)
	p.Mix(buf)
	if energy(buf) != 0 {
		t.Fatal("沒有選曲時不得混入任何樣本")
	}
}

func TestMusicPlayerPlaysLoopsAndOneShot(t *testing.T) {
	d := musicDirector(t)
	p, err := Load(writeMusicFixture(t), d)
	if err != nil {
		t.Fatal(err)
	}
	d.Menu("Move Encamp")
	buf := make([]float32, 2*48000*2) // 2 秒
	p.Mix(buf)
	if energy(buf) == 0 {
		t.Fatal("選曲後要有聲音")
	}
	if p.cur == nil || p.cur.key != "loop" || p.pos < 0.5 || p.pos >= 1.0 {
		t.Fatalf("2 秒後應在循環區間內：%v %.3f", p.cur, p.pos)
	}
	d.Text("The party has won.")
	p.Mix(make([]float32, 2*48000*6/10)) // 淡出 0.5 秒後開始短曲（0.6 秒時正在播）
	if p.cur == nil || p.cur.key != "hit" {
		t.Fatalf("要切到短曲：%v", p.cur)
	}
	p.Mix(make([]float32, 2*48000/2)) // 短曲 0.2 秒播完，回到區域曲
	if p.cur == nil || p.cur.key != "loop" {
		t.Fatalf("短曲播完要回循環曲：%v", p.cur)
	}
}

func TestMusicPlayerToggleOffIsSilent(t *testing.T) {
	d := musicDirector(t)
	p, err := Load(writeMusicFixture(t), d)
	if err != nil {
		t.Fatal(err)
	}
	d.Menu("Move Encamp")
	p.Toggle()
	buf := make([]float32, 2*48000)
	p.Mix(buf)
	if energy(buf) != 0 {
		t.Fatal("配樂關閉時不得混入任何樣本")
	}
	var nilPlayer *Player
	nilPlayer.Mix(buf)
	nilPlayer.Toggle()
}

func TestMusicTSVRejectsBadLoop(t *testing.T) {
	dir := writeMusicFixture(t)
	os.WriteFile(filepath.Join(dir, "music.tsv"), []byte("key\tloop\tloop_start\tloop_end\tlength\tgain\nloop\t1\t1.0\t0.5\t1.0\t1\n"), 0o644)
	if _, err := Load(dir, musicDirector(t)); err == nil {
		t.Fatal("循環點顛倒要報錯")
	}
}
