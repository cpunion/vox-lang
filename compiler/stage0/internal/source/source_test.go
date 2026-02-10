package source

import "testing"

func TestLineColUnicodeColumns(t *testing.T) {
	f := NewFile("x.vox", "a中b\nxy\n")

	type tc struct {
		off      int
		wantLine int
		wantCol  int
	}
	// "a中b\n"
	// byte offsets: a(0), 中(1..3), b(4), \n(5)
	cases := []tc{
		{off: 0, wantLine: 1, wantCol: 1},
		{off: 1, wantLine: 1, wantCol: 2}, // at start of 中
		{off: 2, wantLine: 1, wantCol: 2}, // inside 中 bytes
		{off: 3, wantLine: 1, wantCol: 2}, // inside 中 bytes
		{off: 4, wantLine: 1, wantCol: 3}, // at b
		{off: 5, wantLine: 1, wantCol: 4}, // at newline
		{off: 6, wantLine: 2, wantCol: 1}, // next line start
		{off: 7, wantLine: 2, wantCol: 2},
	}
	for _, c := range cases {
		line, col := f.LineCol(c.off)
		if line != c.wantLine || col != c.wantCol {
			t.Fatalf("off=%d => (%d,%d), want (%d,%d)", c.off, line, col, c.wantLine, c.wantCol)
		}
	}
}
