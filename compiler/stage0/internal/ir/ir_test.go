package ir

import (
	"strings"
	"testing"
)

func TestFormatIncludesStringConst(t *testing.T) {
	p := &Program{
		Funcs: map[string]*Func{
			"main": {
				Name: "main",
				Ret:  Type{K: TString},
				Blocks: []*Block{
					{
						Name: "entry",
						Instr: []Instr{
							&Const{
								Dst: &Temp{ID: 0},
								Ty:  Type{K: TString},
								Val: &ConstStr{S: "a\nb\t\"c\"\\"},
							},
						},
						Term: &Ret{Val: &Temp{ID: 0}},
					},
				},
			},
		},
	}
	got := p.Format()
	if wantSub := `const str "a\nb\t\"c\"\\"`; !strings.Contains(got, wantSub) {
		t.Fatalf("expected formatted IR to contain %q; got:\n%s", wantSub, got)
	}
}
