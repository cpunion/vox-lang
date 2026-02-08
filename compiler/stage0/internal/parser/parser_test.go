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

func TestParseNamedImportDecl(t *testing.T) {
	f := source.NewFile("test.vox", `import { one as uno, two } from "dep"
fn main() -> i32 { return uno(); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(prog.Imports))
	}
	imp := prog.Imports[0]
	if imp.Path != "dep" || len(imp.Names) != 2 {
		t.Fatalf("unexpected import: %#v", imp)
	}
	if imp.Names[0].Name != "one" || imp.Names[0].Alias != "uno" {
		t.Fatalf("unexpected named import[0]: %#v", imp.Names[0])
	}
	if imp.Names[1].Name != "two" || imp.Names[1].Alias != "" {
		t.Fatalf("unexpected named import[1]: %#v", imp.Names[1])
	}
}

func TestParseTypeAliasDecl(t *testing.T) {
	f := source.NewFile("test.vox", `type I = i32
type V = Vec[i32]
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Types) != 2 {
		t.Fatalf("expected 2 type aliases, got %d", len(prog.Types))
	}
	if prog.Types[0].Name != "I" || prog.Types[1].Name != "V" {
		t.Fatalf("unexpected type aliases: %#v", prog.Types)
	}
}

func TestParseConstDecl(t *testing.T) {
	f := source.NewFile("test.vox", `const N: i32 = 10
pub const M: i64 = 20;
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Consts) != 2 {
		t.Fatalf("expected 2 const decls, got %d", len(prog.Consts))
	}
	if prog.Consts[0].Name != "N" || prog.Consts[1].Name != "M" || !prog.Consts[1].Pub {
		t.Fatalf("unexpected consts: %#v", prog.Consts)
	}
}

func TestParseUnionTypeDeclLowersToEnum(t *testing.T) {
	f := source.NewFile("test.vox", `type Value = I32: i32 | Str: String
fn main() -> i32 {
  let x: Value = .I32(1);
  return match x { .I32(v) => v, .Str(_s) => 0 };
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Enums) != 1 {
		t.Fatalf("expected 1 enum (from union type), got %d", len(prog.Enums))
	}
	if prog.Enums[0].Name != "Value" {
		t.Fatalf("expected enum Value, got %q", prog.Enums[0].Name)
	}
	if len(prog.Enums[0].Variants) != 2 || prog.Enums[0].Variants[0].Name != "I32" || prog.Enums[0].Variants[1].Name != "Str" {
		t.Fatalf("unexpected union variants: %#v", prog.Enums[0].Variants)
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

func TestParseEnumMultiPayload(t *testing.T) {
	f := source.NewFile("test.vox", `enum E { Pair(i32, i32), None }
fn main() -> i32 {
  let x: E = E.Pair(40, 2);
  return match x {
    E.Pair(a, b) => a + b,
    E.None => 0,
  };
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseBlockExprTailIfExpr(t *testing.T) {
	// `{ if ... else ... }` should be a valid block expression tail without extra parentheses.
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let x: i32 = { if true { 1 } else { 2 } };
  return x;
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseEnumCtorAndMatchShorthand(t *testing.T) {
	f := source.NewFile("test.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = .A(1);
  return match x {
    .A(v) => v,
    .None => 0,
  };
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseIfExprWithBlockBranches(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let x: i32 = if true {
    let a: i32 = 40;
    a + 2
  } else {
    0
  };
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

func TestParseMatchArmWithBlockExpr(t *testing.T) {
	f := source.NewFile("test.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  return match x {
    E.A(v) => {
      let y: i32 = v + 1;
      y
    },
    E.None => 0,
  };
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
}

func TestParseMatchBindPat(t *testing.T) {
	f := source.NewFile("test.vox", `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  return match x {
    v => 0,
  };
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseMatchIntAndStrPatterns(t *testing.T) {
	f1 := source.NewFile("test.vox", `fn main(x: i32) -> i32 {
  return match x {
    0 => 1,
    1 => 2,
    _ => 3,
  };
}`)
	_, diags := Parse(f1)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}

	f2 := source.NewFile("test.vox", `fn main(s: String) -> i32 {
  return match s {
    "a" => 1,
    _ => 0,
  };
}`)
	_, diags = Parse(f2)
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

func TestParseQualifiedTypePath(t *testing.T) {
	f := source.NewFile("test.vox", `import "a"
fn id(x: a.S) -> a.S { return x; }
fn main() -> i32 {
  let s: a.S = a.S { x: 1 };
  let t: a.S = id(s);
  return t.x;
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseVecMethods(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let mut v: Vec[i32] = Vec();
  v.push(1);
  return v.get(0) + v.len();
}`)
	_, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
}

func TestParseIfExpr(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let x: i32 = if true { 1 } else { 2 };
  return if x < 0 { 0 } else { x };
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) != 2 {
		t.Fatalf("expected 2 stmts, got %d", len(body.Stmts))
	}
	ls, ok := body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected let stmt, got %T", body.Stmts[0])
	}
	if _, ok := ls.Init.(*ast.IfExpr); !ok {
		t.Fatalf("expected if expr init, got %T", ls.Init)
	}
	ret := body.Stmts[1].(*ast.ReturnStmt)
	if _, ok := ret.Expr.(*ast.IfExpr); !ok {
		t.Fatalf("expected if expr return, got %T", ret.Expr)
	}
}

func TestParseGenericFuncAndExplicitTypeArgsCall(t *testing.T) {
	f := source.NewFile("test.vox", `fn id[T](x: T) -> T { return x; }
fn main() -> i32 { return id[i32](1); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(prog.Funcs))
	}
	id := prog.Funcs[0]
	if id.Name != "id" || len(id.TypeParams) != 1 || id.TypeParams[0] != "T" {
		t.Fatalf("unexpected generic func decl: %#v", id)
	}
	mainFn := prog.Funcs[1]
	ret := mainFn.Body.Stmts[0].(*ast.ReturnStmt)
	call := ret.Expr.(*ast.CallExpr)
	if len(call.TypeArgs) != 1 {
		t.Fatalf("expected 1 call type arg, got %d", len(call.TypeArgs))
	}
}
