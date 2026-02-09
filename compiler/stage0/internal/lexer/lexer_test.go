package lexer

import (
	"strings"
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

func TestLexKeywordsAndPunct(t *testing.T) {
	f := source.NewFile("test.vox", `
// keywords
fn let mut return if else true false type const static import pub match as from while break continue
// punct/op
(){}[] , ; : :: . .. ..= @ + - * / % ! = == != < <= << > >= >> & && | || ^ ->
// string
"a\nb\t\"c\"\\"
`)
	toks := Lex(f)
	if len(toks) == 0 || toks[len(toks)-1].Kind != TokenEOF {
		t.Fatalf("expected EOF token")
	}
	ks := kindsNoEOF(toks)

	// Spot-check: first line keyword sequence.
	wantPrefix := []Kind{
		TokenFn, TokenLet, TokenMut, TokenReturn, TokenIf, TokenElse,
		TokenTrue, TokenFalse, TokenType, TokenConst, TokenStatic, TokenImport,
		TokenPub, TokenMatch, TokenAs, TokenFrom, TokenWhile, TokenBreak, TokenContinue,
	}
	if len(ks) < len(wantPrefix) {
		t.Fatalf("too few tokens: %d", len(ks))
	}
	for i := range wantPrefix {
		if ks[i] != wantPrefix[i] {
			t.Fatalf("token %d: expected %v, got %v", i, wantPrefix[i], ks[i])
		}
	}

	// String token should include quotes (lexer preserves raw lexeme).
	var sawString bool
	for _, tk := range toks {
		if tk.Kind == TokenString {
			sawString = true
			if !strings.HasPrefix(tk.Lexeme, "\"") || !strings.HasSuffix(tk.Lexeme, "\"") {
				t.Fatalf("expected string token to include quotes, got %q", tk.Lexeme)
			}
		}
	}
	if !sawString {
		t.Fatalf("expected to see a string token")
	}
}

func TestLexSingleAmpersandIsToken(t *testing.T) {
	f := source.NewFile("test.vox", `&`)
	toks := Lex(f)
	if len(toks) < 2 {
		t.Fatalf("expected at least 2 tokens")
	}
	if toks[0].Kind != TokenAmp {
		t.Fatalf("expected TokenAmp for single &, got %v", toks[0].Kind)
	}
}

func TestLexSinglePipeIsToken(t *testing.T) {
	f := source.NewFile("test.vox", `|`)
	toks := Lex(f)
	if len(toks) < 2 {
		t.Fatalf("expected at least 2 tokens")
	}
	if toks[0].Kind != TokenPipe {
		t.Fatalf("expected TokenPipe for single |, got %v", toks[0].Kind)
	}
}

func TestLexShiftAndCaretTokens(t *testing.T) {
	f := source.NewFile("test.vox", `<< >> ^`)
	toks := Lex(f)
	if len(toks) < 4 {
		t.Fatalf("expected at least 4 tokens")
	}
	if toks[0].Kind != TokenLtLt {
		t.Fatalf("expected TokenLtLt, got %v", toks[0].Kind)
	}
	if toks[1].Kind != TokenGtGt {
		t.Fatalf("expected TokenGtGt, got %v", toks[1].Kind)
	}
	if toks[2].Kind != TokenCaret {
		t.Fatalf("expected TokenCaret, got %v", toks[2].Kind)
	}
}

func kindsNoEOF(toks []Token) []Kind {
	out := make([]Kind, 0, len(toks))
	for _, tk := range toks {
		if tk.Kind == TokenEOF {
			break
		}
		out = append(out, tk.Kind)
	}
	return out
}
