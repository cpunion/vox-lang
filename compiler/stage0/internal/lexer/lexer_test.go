package lexer

import (
	"testing"

	"voxlang/internal/source"
)

func TestLexBasic(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 1 + 2; }`)
	toks := Lex(f)
	if len(toks) == 0 || toks[len(toks)-1].Kind != TokenEOF {
		t.Fatalf("expected EOF token")
	}
	if toks[0].Kind != TokenFn {
		t.Fatalf("expected first token fn, got %v", toks[0].Kind)
	}
}
