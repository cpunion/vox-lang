package parser

import (
	"testing"

	"voxlang/internal/ast"
	"voxlang/internal/source"
)

func TestParseMain(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if prog.Funcs[0].Name != "main" {
		t.Fatalf("expected main, got %q", prog.Funcs[0].Name)
	}
}

func TestParsePathCall(t *testing.T) {
	f := source.NewFile("test.vox", `import "dep"
fn main() -> i32 { return dep.one(); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if len(prog.Imports) != 1 || prog.Imports[0].Path != "dep" {
		t.Fatalf("expected 1 import dep, got %#v", prog.Imports)
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(body.Stmts))
	}
	ret, ok := body.Stmts[0].(*ast.ReturnStmt)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return stmt with expr")
	}
	call, ok := ret.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", ret.Expr)
	}
	mem, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected member callee, got %T", call.Callee)
	}
	recv, ok := mem.Recv.(*ast.IdentExpr)
	if !ok {
		t.Fatalf("expected ident recv, got %T", mem.Recv)
	}
	if recv.Name != "dep" || mem.Name != "one" {
		t.Fatalf("unexpected member: %s.%s", recv.Name, mem.Name)
	}
}

func TestParseWhileBreakContinue(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let mut x: i32 = 0;
  while x < 10 {
    x = x + 1;
    if x == 5 { continue; }
    if x == 9 { break; }
  }
  return x;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
}

func TestParseStructDeclAndLitAndFieldAssign(t *testing.T) {
	f := source.NewFile("test.vox", `struct Point { x: i32, y: i32 }
fn main() -> i32 {
  let mut p: Point = Point { x: 1, y: 2 };
  let a: i32 = p.x;
  p.x = a + 1;
  return p.x + p.y;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	// struct decl count will be asserted once parser supports it
}

func TestParseEnumDeclAndCtorAndMatch(t *testing.T) {
	f := source.NewFile("test.vox", `enum E { A(i32), B(String), None }
fn main() -> i32 {
  // enum constructor call + match expression (payload types differ across variants)
  let x: E = E.B("hi");
  let ok: bool = match x {
    E.A(v) => v == 0,
    E.B(s) => s == "hi",
    E.None => false,
  };
  assert(ok);

  let y: E = E.A(41);
  return match y {
    E.A(v) => v + 1,
    E.B(s) => 0,
    E.None => 0,
  };
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParsePubDecls(t *testing.T) {
	f := source.NewFile("test.vox", `pub struct S { pub x: i32, y: i32 }
pub enum E { A(i32), None }
pub fn f() -> i32 { return 1; }
fn main() -> i32 { return f(); }
`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Structs) != 1 || !prog.Structs[0].Pub {
		t.Fatalf("expected 1 pub struct")
	}
	if len(prog.Structs[0].Fields) != 2 || !prog.Structs[0].Fields[0].Pub || prog.Structs[0].Fields[1].Pub {
		t.Fatalf("expected struct fields to have pub flags")
	}
	if len(prog.Enums) != 1 || !prog.Enums[0].Pub {
		t.Fatalf("expected 1 pub enum")
	}
	if len(prog.Funcs) != 2 || !prog.Funcs[0].Pub {
		t.Fatalf("expected pub fn f and non-pub main")
	}
}
