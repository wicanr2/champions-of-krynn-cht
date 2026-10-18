package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// linPoke 是「跑到第 at 步把線性位址 addr 起的位元組換成 data」（主專案 G6 跨區盤點跳格用）。
type linPoke struct {
	at   uint64
	addr uint32
	data []byte
}

// parseLinPokes 解 `-poke`：`<線性位址 hex>@<步數>=<hex bytes>`，分號分隔；依步數排序。
func parseLinPokes(spec string) ([]linPoke, error) {
	var out []linPoke
	for _, item := range strings.Split(spec, ";") {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}
		lhs, hexs, ok := strings.Cut(item, "=")
		where, when, ok2 := strings.Cut(lhs, "@")
		if !ok || !ok2 {
			return nil, fmt.Errorf("-poke 的 %q 須為 位址@步數=bytes", item)
		}
		addr, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(where), "lin:"), 16, 20)
		if err != nil {
			return nil, fmt.Errorf("-poke 的 %q 位址看不懂：%w", item, err)
		}
		at, err := strconv.ParseUint(strings.TrimSpace(when), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("-poke 的 %q 步數看不懂：%w", item, err)
		}
		var data []byte
		for _, b := range strings.Fields(strings.ReplaceAll(hexs, ",", " ")) {
			v, err := strconv.ParseUint(b, 16, 8)
			if err != nil {
				return nil, fmt.Errorf("-poke 的 %q 有不是 hex 的位元組 %q", item, b)
			}
			data = append(data, byte(v))
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("-poke 的 %q 沒有要寫的位元組", item)
		}
		out = append(out, linPoke{at: at, addr: uint32(addr), data: data})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out, nil
}
