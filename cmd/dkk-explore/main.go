// Command dkk-explore 是《Death Knights of Krynn》冒險畫面的閉迴路自動探索（death-knights-of-krynn-cht
// docs/re/010）：每走一步從路徑 A 的座標列（`x,y D hh:mm`）讀位置與方向，以深度優先把每格四個方向都試過；
// 路徑 B 印出符合 -until 的訊息時停下、再跑 -settle 步後存狀態。
//
// 對話處理：底列出現 `Yes No` 時依 -yes 決定（訊息符合就答 Y，其餘答 N）；`Press button…` 按 Enter；
// 冒險主選單（`Move Area …`）按 M 進移動模式。-fight 時戰鬥中按 Q 交給電腦。只觀測與送鍵；-poke 才改記憶體（例如起點座標）。
//
//	tools/go.sh run ./cmd/dkk-explore -state /st/x.state -root /orig -scratch /out/scratch \
//	  -font /font/dkk-eten.golemfnt -text-dir /text -until 'BERTIL|journal entry' -out /out
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wicanr2/dosgolem/apps/dkk"
	"github.com/wicanr2/dosgolem/internal/cpu"
	"github.com/wicanr2/dosgolem/internal/dos"
	"github.com/wicanr2/dosgolem/internal/machine"
	"github.com/wicanr2/dosgolem/internal/state"
	"github.com/wicanr2/dosgolem/xlate"
)

var coordRe = regexp.MustCompile(`^(\d+),(\d+) ([NESW]) \d+:\d+`)

type pos struct{ x, y int }

// dirs：N E S W 的位移（Gold Box：y 往南增加）。
var delta = [4]pos{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

const dirNames = "NESW"

type machineCPU struct{ m *machine.Machine }

func (c machineCPU) Read8(a uint32) uint8   { return c.m.Peek8(a) }
func (c machineCPU) Read16(a uint32) uint16 { return c.m.Peek16(a) }
func (c machineCPU) Steps() uint64          { return c.m.Steps }
func (c machineCPU) CS() uint16             { return c.m.CPU.Seg[cpu.CS] }
func (c machineCPU) IP() uint16             { return c.m.CPU.IP }
func (c machineCPU) SS() uint16             { return c.m.CPU.Seg[cpu.SS] }
func (c machineCPU) SP() uint16             { return c.m.CPU.R[cpu.SP] }
func (c machineCPU) DS() uint16             { return c.m.CPU.Seg[cpu.DS] }

func main() {
	statePath := flag.String("state", "", "probe 狀態檔（冒險畫面）")
	root := flag.String("root", "", "原版素材目錄（唯讀）")
	scratch := flag.String("scratch", "", "可寫暫存層")
	fontPath := flag.String("font", "", "16×16 GOLEMFNT")
	textDir := flag.String("text-dir", "", "譯文目錄")
	until := flag.String("until", "", "路徑 B 訊息符合此正規式就停")
	yes := flag.String("yes", "", "Yes/No 提問前的路徑 B 訊息符合此正規式時答 Y（預設一律 N）")
	reset := flag.String("reset", `CLIMB|DESCEND|UPPER|LOWER`, "路徑 B 訊息符合時視為換層，清空已造訪紀錄（上下樓座標會重疊）")
	settle := flag.Uint64("settle", 300_000_000, "停下後再跑幾步才存狀態")
	every := flag.Uint64("every", 30_000_000, "送鍵後至少等幾步才判讀")
	maxActions := flag.Int("max-actions", 600, "最多送幾個鍵")
	pokeSpec := flag.String("poke", "", "依步數改記憶體 `線性位址hex@步數=hex bytes`（例：起點座標 18E53@步數=05；主專案小地圖觸發格盤點）")
	fight := flag.Bool("fight", false, "遇到戰鬥時按 Q 交給電腦控制繼續探索（預設停下）")
	outDir := flag.String("out", "", "輸出目錄（end.state、inv.tsv、log）")
	picsDir := flag.String("pics-dir", "", "視窗圖 (24,24) 88×88 或大圖 (8,8) 304×120 出現沒見過、且連續兩次取樣不變的內容時，把整張色號畫面存成 <步數>.bin（HD 抽樣，主專案規格 010）")
	flag.Parse()
	if *statePath == "" || *root == "" || *fontPath == "" || *textDir == "" || *outDir == "" || *until == "" {
		fail(fmt.Errorf("state、root、font、text-dir、until、out 均為必填"))
	}
	untilRe := regexp.MustCompile(*until)
	resetRe := regexp.MustCompile(*reset)
	var yesRe *regexp.Regexp
	if *yes != "" {
		yesRe = regexp.MustCompile(*yes)
	}
	font, err := xlate.LoadFont(*fontPath)
	if err != nil {
		fail(err)
	}
	m := machine.New()
	d := dos.New(m, *root)
	d.Install()
	d.Scratch = *scratch
	if err := state.Load(*statePath, m, d); err != nil {
		fail(err)
	}
	ui, ecl, err := dkk.LoadTextDir(*textDir)
	if err != nil {
		fail(err)
	}
	live := dkk.NewLive(font, ui, ecl)
	live.Inventory = true
	live.Arm()
	cpuView := machineCPU{m}
	m.ObserveVideoWrites(func(w machine.VideoWrite) { live.VideoWrite(w.Offset) })
	m.ObserveModeChanges(func(machine.ModeChange) { live.ModeChange() })
	m.SetOnFrame(func() { live.Frame(m.Indexed(), m.FramePalette()) })

	logf, err := os.Create(filepath.Join(*outDir, "explore.log"))
	if err != nil {
		fail(err)
	}
	defer logf.Close()
	say := func(f string, a ...any) { fmt.Fprintf(logf, "#%d "+f+"\n", append([]any{m.Steps}, a...)...) }

	// 探索狀態
	visited := map[pos]bool{}
	tried := map[pos]int{} // 每格已試過的方向位元
	open := map[pos]int{}  // 每格已知可通行的方向位元
	var cur pos
	dir, known := 0, false
	var plan []string // 待送的鍵
	var pending *move
	seen := 0 // 已讀過的事件數
	lastKey := m.Steps
	actions := 0
	var lastB, lastC string
	stall, lastBCount, nB := 0, 0, 0
	var stallPos pos
	unstick := []string{"Return", "Esc", " "}
	lastCIdx := -1
	stopAt := uint64(0)
	pics := picSampler{seen: map[[32]byte]bool{}}

	push := func(name string) {
		if len(name) == 1 {
			k, _ := dos.KeyForRune(rune(name[0]))
			d.PushKey(k)
		} else if !d.PushKeyNamed(name) {
			fail(fmt.Errorf("鍵名 %q", name))
		}
		lastKey = m.Steps
		actions++
		say("送 %s", name)
	}
	turnTo := func(nd int) []string {
		switch (nd - dir + 4) % 4 {
		case 1:
			return []string{"Right"}
		case 2:
			return []string{"Down"}
		case 3:
			return []string{"Left"}
		}
		return nil
	}

	pokes, err := dkk.ParseLinPokes(*pokeSpec)
	if err != nil {
		fail(err)
	}
	for !d.Exited {
		if err := live.BeforeStep(cpuView); err != nil {
			fail(err)
		}
		for len(pokes) > 0 && m.Steps >= pokes[0].At {
			for i, b := range pokes[0].Data {
				m.Write8(pokes[0].Addr+uint32(i), b)
			}
			pokes = pokes[1:]
		}
		if err := m.Step(); err != nil {
			fail(err)
		}
		if stopAt != 0 {
			if m.Steps >= stopAt {
				break
			}
			continue
		}
		if live.Inv == nil {
			continue
		}
		// 讀新事件
		for ; seen < len(live.Inv.Events); seen++ {
			e := live.Inv.Events[seen]
			switch e.Path {
			case "A":
				if c := coordRe.FindStringSubmatch(e.Text); c != nil {
					x, _ := strconv.Atoi(c[1])
					y, _ := strconv.Atoi(c[2])
					cur, dir, known = pos{x, y}, strings.IndexByte(dirNames, c[3][0]), true
				}
			case "B":
				if t := strings.TrimSpace(e.Text); t != "" {
					lastB = t
					nB++
					say("B %d,%d %c %s", cur.x, cur.y, dirNames[dir], t)
					if resetRe.MatchString(t) {
						say("換層，清空探索紀錄")
						visited, tried, open, plan, pending = map[pos]bool{}, map[pos]int{}, map[pos]int{}, nil, nil
					}
					if untilRe.MatchString(t) {
						say("符合 -until，停下")
						stopAt = m.Steps + *settle
					}
				}
			case "C":
				if strings.TrimSpace(e.Text) != "" { // 補位空白不算；逐字累加，判讀時再讀全文
					lastCIdx = seen
				}
			}
		}
		if *picsDir != "" && m.Steps%1_000_000 == 0 {
			pics.sample(m.Indexed(), m.Steps, *picsDir, say)
		}
		if stopAt != 0 || m.Steps-lastKey < *every || m.Steps%1_000_000 != 0 {
			continue
		}
		if _, pendingKey := m.PeekKey(); pendingKey {
			continue
		}
		if lastCIdx >= 0 {
			lastC = strings.Join(strings.Fields(live.Inv.Events[lastCIdx].Text), " ")
			lastCIdx = -1
		}
		if actions >= *maxActions {
			say("達到 -max-actions")
			break
		}
		// 對話
		switch {
		case strings.Contains(lastC, "Yes No"):
			if yesRe != nil && yesRe.MatchString(lastB) {
				push("Y")
			} else {
				push("N")
			}
			lastC = ""
			continue
		case strings.HasPrefix(lastC, "Press button"), strings.HasPrefix(lastC, "press"):
			push("Return")
			lastC = ""
			continue
		case strings.HasPrefix(lastC, "Move Area"):
			push("M")
			lastC = ""
			continue
		case inCombat(lastC) && *fight:
			push("Q") // 這名角色交給電腦控制；每名角色輪到時各按一次
			lastC = ""
			continue
		case inCombat(lastC):
			say("進入戰鬥，停下")
			stopAt = m.Steps + 1
			continue
		case lastC != "" && lastC != "Exit":
			// 其他選單（神殿、商店、事件抉擇）：選不會開打、能離開的選項，按它的熱鍵（首字母）
			if k := preferredOption(lastC); k != "" {
				push(k)
				lastC = ""
				continue
			}
		}
		// 卡住偵測：連續動作後位置不變、也沒有新訊息，依序試 Enter、Esc、空白
		if known && cur == stallPos && nB == lastBCount {
			stall++
		} else {
			stall = 0
		}
		stallPos, lastBCount = cur, nB
		if stall >= 12 { // 一格試四個方向（轉向＋前進）最多 8 個動作不會移動
			k := unstick[(stall-12)%len(unstick)]
			say("卡住（選單 %q），試 %q", lastC, k)
			plan, pending = nil, nil
			if k == " " {
				kk, _ := dos.KeyForRune(' ')
				d.PushKey(kk)
				lastKey = m.Steps
				actions++
			} else {
				push(k)
			}
			continue
		}
		if !known { // 狀態檔起點的座標列早已印好：先進移動模式、轉一次向讓它重印
			if actions == 0 {
				push("M")
			} else {
				push("Left")
			}
			continue
		}
		// 上一步移動的結果：走到預期的相鄰格才記通道；原地不動是牆；其他（傳送、事件移位）只算試過。
		if pending != nil {
			to := pos{pending.from.x + delta[pending.d].x, pending.from.y + delta[pending.d].y}
			if cur == to {
				open[pending.from] |= 1 << pending.d
				open[cur] |= 1 << ((pending.d + 2) % 4)
			}
			pending = nil
		}
		visited[cur] = true
		if len(plan) > 0 {
			k := plan[0]
			plan = plan[1:]
			if k == "Up" {
				pending = &move{cur, dir}
			}
			push(k)
			continue
		}
		next := firstStep(cur, dir, open, tried)
		if next < 0 {
			say("探索完畢（已造訪 %d 格）", len(visited))
			break
		}
		tried[cur] |= 1 << next
		plan = append(turnTo(next), "Up")
	}
	say("結束：位置 %v 方向 %c，造訪 %d 格，送鍵 %d", cur, dirNames[dir], len(visited), actions)
	if err := state.Save(filepath.Join(*outDir, "end.state"), m, d); err != nil {
		fail(err)
	}
	ovr, _ := os.ReadFile(filepath.Join(*root, "GAME.OVR"))
	if err := live.Inv.WriteTSV(filepath.Join(*outDir, "inv.tsv"), ovr); err != nil {
		fail(err)
	}
	fmt.Printf("steps=%d pos=%v dir=%c visited=%d actions=%d stopped=%v\n", m.Steps, cur, dirNames[dir], len(visited), actions, stopAt != 0)
}

// preferredOption 回選單中「不開打、能離開」選項的熱鍵；沒有回空字串。
func preferredOption(menu string) string {
	words := strings.Fields(menu)
	for _, want := range []string{"Wait", "Leave", "Continue", "Done", "Exit", "No"} {
		for _, w := range words {
			if w == want {
				return w[:1]
			}
		}
	}
	return ""
}

type move struct {
	from pos
	d    int
}

// firstStep 回下一步要走的方向：目前格有未試方向就試（從面向開始順時針）；否則沿已知通道廣度優先
// 找最近有未試方向的格子，回第一步方向；都沒有回 -1。
func firstStep(cur pos, dir int, open, tried map[pos]int) int {
	for i := 0; i < 4; i++ {
		if nd := (dir + i) % 4; tried[cur]&(1<<nd) == 0 {
			return nd
		}
	}
	type node struct {
		p     pos
		first int
	}
	seen := map[pos]bool{cur: true}
	q := []node{{cur, -1}}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		if n.first >= 0 && tried[n.p] != 0xF {
			return n.first
		}
		for d := 0; d < 4; d++ {
			if open[n.p]&(1<<d) == 0 {
				continue
			}
			np := pos{n.p.x + delta[d].x, n.p.y + delta[d].y}
			if seen[np] {
				continue
			}
			seen[np] = true
			f := n.first
			if f < 0 {
				f = d
			}
			q = append(q, node{np, f})
		}
	}
	return -1
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "dkk-explore：", err)
	os.Exit(1)
}

// picSampler 收集探索途中出現過的視窗圖與大圖（HD 抽樣）：兩個矩形的色號雜湊連續兩次取樣相同才算畫完，
// 沒見過就存整張畫面。全黑不存。
type picSampler struct {
	seen map[[32]byte]bool
	last [32]byte
}

var picRects = [][4]int{{24, 24, 88, 88}, {8, 8, 304, 120}}

func (p *picSampler) sample(idx []byte, step uint64, dir string, say func(string, ...any)) {
	if len(idx) != 320*200 {
		return
	}
	h := sha256.New()
	nonBlack := false
	for _, r := range picRects {
		for y := r[1]; y < r[1]+r[3]; y++ {
			row := idx[y*320+r[0] : y*320+r[0]+r[2]]
			h.Write(row)
			for _, c := range row {
				nonBlack = nonBlack || c&15 != 0
			}
		}
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	stable := sum == p.last
	p.last = sum
	if !stable || !nonBlack || p.seen[sum] {
		return
	}
	p.seen[sum] = true
	name := filepath.Join(dir, fmt.Sprintf("%d.bin", step))
	if err := os.WriteFile(name, idx, 0o644); err != nil {
		fail(err)
	}
	say("存畫面 %s", name)
}

// inCombat 判斷戰鬥選單。選單分兩列印（`Move Aim … Quick` 與 `Delay … View Speed End`），
// 最後一段逐字可能只是第二列。
func inCombat(c string) bool {
	return strings.HasPrefix(c, "Move Aim") || strings.HasPrefix(c, "Delay ") && strings.HasSuffix(c, "Speed End")
}
