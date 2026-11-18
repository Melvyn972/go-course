package open5e

import "testing"

func TestCRValue(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"0", 0},
		{"1/8", 0.125},
		{"1/4", 0.25},
		{"1/2", 0.5},
		{"1", 1},
		{"2", 2},
	}
	for _, tc := range cases {
		if got := CRValue(tc.in); got != tc.want {
			t.Errorf("CRValue(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
