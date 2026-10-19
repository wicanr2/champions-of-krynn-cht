package cok

import "testing"

const musicCuesFixture = "kind\tpattern\ttrack\n" +
	"map\t1/20\toccupied\n" +
	"menu\tInn Hall Commandant\toutpost\n" +
	"menu\tMove Encamp\toverland\n" +
	"menu\tSave View Magic Rest\tcamp\n" +
	"leave\tMove Area\t-\n" +
	"battle\tMove View Aim\tbattle\n" +
	"boss\tDRAGON\tbattle-boss\n" +
	"end\tThe party has won\tvictory\n"

func TestMusicDirectorFlow(t *testing.T) {
	d, err := LoadMusicCues([]byte(musicCuesFixture))
	if err != nil {
		t.Fatal(err)
	}
	want := func(step, track string, one bool) {
		t.Helper()
		if got, o, _ := d.Want(); got != track || o != one {
			t.Fatalf("%s：要 %q（短曲 %v），得 %q（%v）", step, track, one, got, o)
		}
	}
	want("開機", "", false)
	d.Menu("Inn Hall Commandant Armoury Next Leave")
	want("屯墾區", "outpost", false)
	d.Menu("Move Encamp")
	want("大地圖", "overland", false)
	d.MapKey("1/20")
	want("進斯拉托", "occupied", false)
	d.Text("DRAGON") // 戰鬥外的名字不算強敵
	d.Menu("Move View Aim Use Done")
	want("戰鬥", "battle", false)
	d.Text("RED DRAGON")
	want("強敵", "battle-boss", false)
	d.Menu("Move View Aim Use Done") // 下一名角色的回合不重設強敵
	want("強敵持續", "battle-boss", false)
	_, _, s0 := d.Want()
	d.Text("The party has won.")
	want("勝利", "victory", true)
	if _, _, s1 := d.Want(); s1 != s0+1 {
		t.Fatalf("短曲序號要加一：%d → %d", s0, s1)
	}
	d.OneShotDone()
	want("勝利後回區域曲", "occupied", false)
	d.Menu("Save View Magic Rest Alter Fix Exit")
	want("紮營", "camp", false)
	d.Menu("Move Area Cast View Encamp Search Look")
	want("離開營地回地圖曲", "occupied", false)
	d.MapKey("9/99") // 沒有對應曲目的地圖沿用目前區域曲
	want("未知地圖", "occupied", false)
}

func TestMusicDirectorNil(t *testing.T) {
	var d *MusicDirector
	d.Menu("Move Encamp")
	d.Text("x")
	d.MapKey("1/20")
	d.OneShotDone()
	if got, _, _ := d.Want(); got != "" {
		t.Fatalf("nil 要靜音：%q", got)
	}
}

func TestMusicCuesRejectUnknownKind(t *testing.T) {
	if _, err := LoadMusicCues([]byte("kind\tpattern\ttrack\nfoo\tx\ty\n")); err == nil {
		t.Fatal("未知 kind 要報錯")
	}
}
