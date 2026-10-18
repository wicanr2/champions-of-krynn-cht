package dkk

import "testing"

func TestParseLinPokes(t *testing.T) {
	p, err := ParseLinPokes("lin:18E54@20=0A;18E53@10=05 06")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 2 || p[0].At != 10 || p[0].Addr != 0x18E53 || len(p[0].Data) != 2 || p[1].Addr != 0x18E54 || p[1].Data[0] != 0x0A {
		t.Fatalf("%+v", p)
	}
	for _, bad := range []string{"18E53=05", "18E53@x=05", "zz@1=05", "18E53@1=", "18E53@1=GG"} {
		if _, err := ParseLinPokes(bad); err == nil {
			t.Fatalf("%q 應報錯", bad)
		}
	}
}
