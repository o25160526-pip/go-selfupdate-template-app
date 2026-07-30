package version

import "testing"

func TestDisplayTagRoundTrip(t *testing.T) {
	cases := []string{"1.26.0729.1930", "1.26.0101.0005", "1.27.1231.2359"}
	for _, tc := range cases {
		v, err := Parse(tc)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(v.Tag())
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != tc {
			t.Fatalf("roundtrip %s => %s => %s", tc, v.Tag(), got.String())
		}
	}
}
func TestCompare(t *testing.T) {
	a := MustParse("1.26.0729.1930")
	b := MustParse("1.26.0730.0000")
	c := MustParse("1.27.0101.0000")
	if a.Compare(b) >= 0 || b.Compare(c) >= 0 || c.Compare(a) <= 0 {
		t.Fatal("comparison order invalid")
	}
}
func TestRejectInvalid(t *testing.T) {
	for _, s := range []string{"1.26.0230.0000", "v1.26.99999999", "1.26.0729"} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("expected error for %s", s)
		}
	}
}
func FuzzParseRoundTrip(f *testing.F) {
	f.Add("1.26.0729.1930")
	f.Fuzz(func(t *testing.T, s string) {
		v, err := Parse(s)
		if err != nil {
			return
		}
		v2, err := Parse(v.String())
		if err != nil || v.Compare(v2) != 0 {
			t.Fatalf("roundtrip failed: %q", s)
		}
	})
}
