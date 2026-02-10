package parser

import (
	"testing"

	"voxlang/internal/ast"
	"voxlang/internal/source"
)

func TestParseTopLevelRecoveryAfterGarbage(t *testing.T) {
	f := source.NewFile("test.vox", `bogus fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if prog == nil {
		t.Fatalf("expected program")
	}
	if diags == nil || len(diags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	if len(prog.Funcs) != 1 || prog.Funcs[0].Name != "main" {
		t.Fatalf("expected to still parse main, got funcs: %v", len(prog.Funcs))
	}
}

func TestParseImportErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "missing_string", src: `import utils fn main() -> i32 { return 0; }`, want: "expected string literal import path"},
		{name: "missing_alias", src: `import "utils" as fn main() -> i32 { return 0; }`, want: "expected alias after `as`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := source.NewFile("test.vox", tc.src)
			_, diags := Parse(f)
			if diags == nil || len(diags.Items) == 0 {
				t.Fatalf("expected diagnostics")
			}
			found := false
			for _, it := range diags.Items {
				if it.Msg == tc.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q, got: %+v", tc.want, diags.Items)
			}
		})
	}
}

func TestParseStmtRecoveryAfterBadToken(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let x: i32 = 1;
  @;
  return x;
}`)
	prog, diags := Parse(f)
	if prog == nil || len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 function")
	}
	if diags == nil || len(diags.Items) == 0 {
		t.Fatalf("expected diagnostics from bad token")
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) < 2 {
		t.Fatalf("expected at least 2 stmts, got %d", len(body.Stmts))
	}
	// Ensure we recovered enough to parse return x;
	ret, ok := body.Stmts[len(body.Stmts)-1].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected last stmt return, got %T", body.Stmts[len(body.Stmts)-1])
	}
	id, ok := ret.Expr.(*ast.IdentExpr)
	if !ok || id.Name != "x" {
		t.Fatalf("expected return x, got %T", ret.Expr)
	}
}

func TestParsePubRestrictedErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unknown_scope",
			src:  `pub(foo) fn main() -> i32 { return 0; }`,
			want: "expected `crate` or `super` in `pub(...)`",
		},
		{
			name: "missing_rparen",
			src:  `pub(crate fn main() -> i32 { return 0; }`,
			want: "expected `)` after `pub(...)`",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := source.NewFile("test.vox", tc.src)
			_, diags := Parse(f)
			if diags == nil || len(diags.Items) == 0 {
				t.Fatalf("expected diagnostics")
			}
			found := false
			for _, it := range diags.Items {
				if it.Msg == tc.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q, got: %+v", tc.want, diags.Items)
			}
		})
	}
}
