package main

import (
	"testing"

	"github.com/wicanr2/dosgolem/oracle"
)

func TestFormatRegsExact(t *testing.T) {
	r := oracle.Regs{AX: 0x5000, BX: 0x0100, CX: 0xA440, DX: 0x0204,
		SI: 0x1234, DI: 0x5678, BP: 0x9ABC, SP: 0x8000,
		DS: 0x3500, ES: 0xA0C8, SS: 0x4000, CS: 0x1F7A, IP: 0x0004, Flags: 0x0246}
	want := "REGS AX=5000 BX=0100 CX=A440 DX=0204 SI=1234 DI=5678 " +
		"BP=9ABC SP=8000 DS=3500 ES=A0C8 SS=4000 CS=1F7A IP=0004 FLAGS=0246"
	if got := formatRegs(r); got != want {
		t.Fatalf("formatRegs:\n got %q\nwant %q", got, want)
	}
}

func TestParseWatchMem(t *testing.T) {
	s, err := parseWatchMem("ES:BP:64")
	if err != nil || s == nil || s.segReg != "ES" || s.offReg != "BP" || s.length != 64 {
		t.Fatalf("valid spec: %+v err=%v", s, err)
	}
	r := oracle.Regs{ES: 0xA0C8, BP: 0x1234}
	if seg, off := s.resolve(r); seg != 0xA0C8 || off != 0x1234 {
		t.Fatalf("resolve = %04X:%04X", seg, off)
	}
	if s, err := parseWatchMem(""); err != nil || s != nil {
		t.Fatalf("empty spec: %+v err=%v", s, err)
	}
	lit, err := parseWatchMem("CS:#F7A1:4")
	if err != nil || lit == nil || lit.offReg != "" || lit.literal != 0xF7A1 {
		t.Fatalf("literal spec: %+v err=%v", lit, err)
	}
	if seg, off := lit.resolve(oracle.Regs{CS: 0x2000}); seg != 0x2000 || off != 0xF7A1 {
		t.Fatalf("literal resolve = %04X:%04X", seg, off)
	}
	multi, err := parseWatchMems("ES:BP:64,CS:#F780:48")
	if err != nil || len(multi) != 2 {
		t.Fatalf("multi spec: %v err=%v", multi, err)
	}
	for _, bad := range []string{"XX:BP:64", "ES:ZZ:64", "ES:BP:0", "ES:BP:4097", "ES:BP", "ES:BP:x"} {
		if _, err := parseWatchMem(bad); err == nil {
			t.Fatalf("bad spec %q accepted", bad)
		}
	}
}
