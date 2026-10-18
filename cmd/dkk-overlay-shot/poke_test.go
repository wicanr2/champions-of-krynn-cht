package main

import "testing"

func TestParseLinPokes(t *testing.T) {
	p, err := parseLinPokes("lin:18E54@20=0A;18E53@10=05 06")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 2 || p[0].at != 10 || p[0].addr != 0x18E53 || len(p[0].data) != 2 || p[1].addr != 0x18E54 || p[1].data[0] != 0x0A {
		t.Fatalf("%+v", p)
	}
	for _, bad := range []string{"18E53=05", "18E53@x=05", "zz@1=05", "18E53@1=", "18E53@1=GG"} {
		if _, err := parseLinPokes(bad); err == nil {
			t.Fatalf("%q 應報錯", bad)
		}
	}
}
