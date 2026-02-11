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

func TestParseFuncParamsAllowTrailingComma(t *testing.T) {
	f := source.NewFile("test.vox", `fn sum(a: i32, b: i32,) -> i32 { return a + b; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if len(prog.Funcs[0].Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(prog.Funcs[0].Params))
	}
}

func TestParseTraitMethodParamsAllowTrailingComma(t *testing.T) {
	f := source.NewFile("test.vox", `trait Add {
  fn add(x: i32, y: i32,) -> i32;
}
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Traits) != 1 {
		t.Fatalf("expected 1 trait, got %d", len(prog.Traits))
	}
	if len(prog.Traits[0].Methods) != 1 {
		t.Fatalf("expected 1 trait method, got %d", len(prog.Traits[0].Methods))
	}
	if len(prog.Traits[0].Methods[0].Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(prog.Traits[0].Methods[0].Params))
	}
}

func TestParseFuncTypeParamsAllowTrailingComma(t *testing.T) {
	f := source.NewFile("test.vox", `fn id[T,](x: T) -> T { return x; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	if len(prog.Funcs[0].TypeParams) != 1 || prog.Funcs[0].TypeParams[0] != "T" {
		t.Fatalf("unexpected type params: %#v", prog.Funcs[0].TypeParams)
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

func TestParseCompileErrorIntrinsicCall(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { @compile_error("boom"); return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) < 1 {
		t.Fatalf("expected at least 1 stmt")
	}
	es, ok := body.Stmts[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected first stmt expr, got %T", body.Stmts[0])
	}
	call, ok := es.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", es.Expr)
	}
	callee, ok := call.Callee.(*ast.IdentExpr)
	if !ok || callee.Name != "@compile_error" {
		t.Fatalf("expected @compile_error callee, got %T / %#v", call.Callee, call.Callee)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
	if _, ok := call.Args[0].(*ast.StringLit); !ok {
		t.Fatalf("expected string arg, got %T", call.Args[0])
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

func TestParseRangeTypeAliasDecl(t *testing.T) {
	f := source.NewFile("test.vox", `type Tiny = @range(0..=3) i32
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Types) != 1 {
		t.Fatalf("expected 1 type alias, got %d", len(prog.Types))
	}
	td := prog.Types[0]
	if td.Name != "Tiny" {
		t.Fatalf("unexpected alias name: %#v", td.Name)
	}
	rt, ok := td.Type.(*ast.RangeType)
	if !ok {
		t.Fatalf("expected RangeType, got %#v", td.Type)
	}
	if rt.Lo != 0 || rt.Hi != 3 {
		t.Fatalf("unexpected bounds: %#v", rt)
	}
	bt, ok := rt.Base.(*ast.NamedType)
	if !ok || len(bt.Parts) != 1 || bt.Parts[0] != "i32" {
		t.Fatalf("unexpected base type: %#v", rt.Base)
	}
}

func TestParseRangeTypeAliasDeclNegativeBounds(t *testing.T) {
	f := source.NewFile("test.vox", `type Small = @range(-5..=5) i32
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Types) != 1 {
		t.Fatalf("expected 1 type alias, got %d", len(prog.Types))
	}
	td := prog.Types[0]
	rt, ok := td.Type.(*ast.RangeType)
	if !ok {
		t.Fatalf("expected RangeType, got %#v", td.Type)
	}
	if rt.Lo != -5 || rt.Hi != 5 {
		t.Fatalf("unexpected bounds: %#v", rt)
	}
}

func TestParseRangeTypeAliasDeclISize(t *testing.T) {
	f := source.NewFile("test.vox", `type Tiny = @range(-3..=3) isize
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Types) != 1 {
		t.Fatalf("expected 1 type alias, got %d", len(prog.Types))
	}
	td := prog.Types[0]
	rt, ok := td.Type.(*ast.RangeType)
	if !ok {
		t.Fatalf("expected RangeType, got %#v", td.Type)
	}
	if rt.Lo != -3 || rt.Hi != 3 {
		t.Fatalf("unexpected bounds: %#v", rt)
	}
	bt, ok := rt.Base.(*ast.NamedType)
	if !ok || len(bt.Parts) != 1 || bt.Parts[0] != "isize" {
		t.Fatalf("unexpected base type: %#v", rt.Base)
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

func TestParseCompoundAssignStmt(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let mut x: i32 = 1;
  x += 2;
  return x;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) < 2 {
		t.Fatalf("expected at least 2 stmts")
	}
	as, ok := body.Stmts[1].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("expected assign stmt, got %T", body.Stmts[1])
	}
	be, ok := as.Expr.(*ast.BinaryExpr)
	if !ok || be.Op != "+" {
		t.Fatalf("expected desugared binary +, got %T", as.Expr)
	}
	lhs, ok := be.Left.(*ast.IdentExpr)
	if !ok || lhs.Name != "x" {
		t.Fatalf("expected lhs ident x, got %T", be.Left)
	}
}

func TestParseCompoundFieldAssignStmt(t *testing.T) {
	f := source.NewFile("test.vox", `struct S { x: i32 }
fn main() -> i32 {
  let mut s: S = S { x: 1 };
  s.x <<= 1;
  return s.x;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	body := prog.Funcs[0].Body
	if len(body.Stmts) < 2 {
		t.Fatalf("expected at least 2 stmts")
	}
	as, ok := body.Stmts[1].(*ast.FieldAssignStmt)
	if !ok {
		t.Fatalf("expected field assign stmt, got %T", body.Stmts[1])
	}
	be, ok := as.Expr.(*ast.BinaryExpr)
	if !ok || be.Op != "<<" {
		t.Fatalf("expected desugared binary <<, got %T", as.Expr)
	}
	if _, ok := be.Left.(*ast.MemberExpr); !ok {
		t.Fatalf("expected lhs member expr, got %T", be.Left)
	}
}

func TestParseCompoundAssignAllOps(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 {
  let mut x: i32 = 0;
  x += 1;
  x -= 2;
  x *= 3;
  x /= 4;
  x %= 5;
  x &= 6;
  x |= 7;
  x ^= 8;
  x <<= 9;
  x >>= 10;
  return x;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	body := prog.Funcs[0].Body
	wantOps := []string{"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>"}
	for i, want := range wantOps {
		st := body.Stmts[1+i]
		as, ok := st.(*ast.AssignStmt)
		if !ok {
			t.Fatalf("stmt %d: expected assign stmt, got %T", i, st)
		}
		be, ok := as.Expr.(*ast.BinaryExpr)
		if !ok || be.Op != want {
			t.Fatalf("stmt %d: expected binary %q, got %T", i, want, as.Expr)
		}
		lhs, ok := be.Left.(*ast.IdentExpr)
		if !ok || lhs.Name != "x" {
			t.Fatalf("stmt %d: expected lhs ident x, got %T", i, be.Left)
		}
	}
}

func TestParseCompoundFieldAssignAllOps(t *testing.T) {
	f := source.NewFile("test.vox", `struct S { x: i32 }
fn main() -> i32 {
  let mut s: S = S { x: 0 };
  s.x += 1;
  s.x -= 2;
  s.x *= 3;
  s.x /= 4;
  s.x %= 5;
  s.x &= 6;
  s.x |= 7;
  s.x ^= 8;
  s.x <<= 9;
  s.x >>= 10;
  return s.x;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	body := prog.Funcs[0].Body
	wantOps := []string{"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>"}
	for i, want := range wantOps {
		st := body.Stmts[1+i]
		as, ok := st.(*ast.FieldAssignStmt)
		if !ok {
			t.Fatalf("stmt %d: expected field assign stmt, got %T", i, st)
		}
		be, ok := as.Expr.(*ast.BinaryExpr)
		if !ok || be.Op != want {
			t.Fatalf("stmt %d: expected binary %q, got %T", i, want, as.Expr)
		}
		if _, ok := be.Left.(*ast.MemberExpr); !ok {
			t.Fatalf("stmt %d: expected lhs member expr, got %T", i, be.Left)
		}
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

func TestParseMatchBoolPatterns(t *testing.T) {
	f := source.NewFile("test.vox", `fn main(b: bool) -> i32 {
  return match b {
    true => 1,
    false => 0,
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
	if prog.Structs[0].Vis != ast.VisPub {
		t.Fatalf("expected struct vis pub, got %v", prog.Structs[0].Vis)
	}
	if len(prog.Structs[0].Fields) != 2 || !prog.Structs[0].Fields[0].Pub || prog.Structs[0].Fields[1].Pub {
		t.Fatalf("expected struct fields to have pub flags")
	}
	if prog.Structs[0].Fields[0].Vis != ast.VisPub || prog.Structs[0].Fields[1].Vis != ast.VisPrivate {
		t.Fatalf("unexpected field vis flags: %#v", prog.Structs[0].Fields)
	}
	if len(prog.Enums) != 1 || !prog.Enums[0].Pub {
		t.Fatalf("expected 1 pub enum")
	}
	if prog.Enums[0].Vis != ast.VisPub {
		t.Fatalf("expected enum vis pub, got %v", prog.Enums[0].Vis)
	}
	if len(prog.Funcs) != 2 || !prog.Funcs[0].Pub {
		t.Fatalf("expected pub fn f and non-pub main")
	}
	if prog.Funcs[0].Vis != ast.VisPub {
		t.Fatalf("expected fn vis pub, got %v", prog.Funcs[0].Vis)
	}
}

func TestParseRestrictedPubDecls(t *testing.T) {
	f := source.NewFile("test.vox", `pub(crate) const N: i32 = 1
pub(super) fn f() -> i32 { return N; }
pub(crate) struct S { pub(super) x: i32, pub(crate) y: i32 }
fn main() -> i32 { return f(); }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Consts) != 1 || prog.Consts[0].Vis != ast.VisCrate || !prog.Consts[0].Pub {
		t.Fatalf("unexpected const vis: %#v", prog.Consts)
	}
	if len(prog.Funcs) != 2 || prog.Funcs[0].Vis != ast.VisSuper || !prog.Funcs[0].Pub {
		t.Fatalf("unexpected fn vis: %#v", prog.Funcs)
	}
	if len(prog.Structs) != 1 || prog.Structs[0].Vis != ast.VisCrate {
		t.Fatalf("unexpected struct vis: %#v", prog.Structs)
	}
	if len(prog.Structs[0].Fields) != 2 {
		t.Fatalf("unexpected struct fields: %#v", prog.Structs[0].Fields)
	}
	if prog.Structs[0].Fields[0].Vis != ast.VisSuper || prog.Structs[0].Fields[1].Vis != ast.VisCrate {
		t.Fatalf("unexpected field vis: %#v", prog.Structs[0].Fields)
	}
}

func TestParseTraitAndImplDecls(t *testing.T) {
	f := source.NewFile("test.vox", `pub trait Eq {
  fn eq(a: Self, b: Self) -> bool;
}
impl Eq for i32 {
  fn eq(a: i32, b: i32) -> bool { return a == b; }
}
fn main() -> i32 {
  return 0;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Traits) != 1 {
		t.Fatalf("expected 1 trait, got %d", len(prog.Traits))
	}
	if !prog.Traits[0].Pub || prog.Traits[0].Name != "Eq" {
		t.Fatalf("unexpected trait decl: %#v", prog.Traits[0])
	}
	if len(prog.Traits[0].Methods) != 1 || prog.Traits[0].Methods[0].Name != "eq" {
		t.Fatalf("unexpected trait methods: %#v", prog.Traits[0].Methods)
	}
	if len(prog.Impls) != 1 {
		t.Fatalf("expected 1 impl, got %d", len(prog.Impls))
	}
	if len(prog.Impls[0].Methods) != 1 || prog.Impls[0].Methods[0].Name != "eq" {
		t.Fatalf("unexpected impl methods: %#v", prog.Impls[0].Methods)
	}
}

func TestParseTraitDefaultMethodBodyCompat(t *testing.T) {
	f := source.NewFile("test.vox", `trait Show {
  fn show(x: Self) -> String { return "x"; }
}
struct I { v: i32 }
impl Show for I {}
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Traits) != 1 || len(prog.Traits[0].Methods) != 1 {
		t.Fatalf("unexpected trait parse result: %#v", prog.Traits)
	}
	if len(prog.Impls) != 1 {
		t.Fatalf("unexpected impl parse result: %#v", prog.Impls)
	}
}

func TestParseTraitSupertraitsCompat(t *testing.T) {
	f := source.NewFile("test.vox", `trait Child: A + B {
  fn go(x: Self) -> i32;
}
struct I { v: i32 }
impl Child for I { fn go(x: I) -> i32 { return 1; } }
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Traits) != 1 {
		t.Fatalf("expected 1 trait, got %d", len(prog.Traits))
	}
}

func TestParseTraitAssocTypeCompat(t *testing.T) {
	f := source.NewFile("test.vox", `trait Iter {
  type Item;
  fn next(x: Self) -> Self.Item;
}
struct I { v: i32 }
impl Iter for I {
  type Item = i32;
  fn next(x: I) -> i32 { return x.v; }
}
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Traits) != 1 || len(prog.Traits[0].Methods) != 1 {
		t.Fatalf("unexpected trait parse result: %#v", prog.Traits)
	}
	if len(prog.Impls) != 1 || len(prog.Impls[0].Methods) != 1 {
		t.Fatalf("unexpected impl parse result: %#v", prog.Impls)
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

func TestParseAsExprBindsTighterThanPlus(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 1 + 2 as i64; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.Funcs))
	}
	ret, ok := prog.Funcs[0].Body.Stmts[0].(*ast.ReturnStmt)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return stmt with expr")
	}
	be, ok := ret.Expr.(*ast.BinaryExpr)
	if !ok || be.Op != "+" {
		t.Fatalf("expected binary +, got %T", ret.Expr)
	}
	ae, ok := be.Right.(*ast.AsExpr)
	if !ok {
		t.Fatalf("expected right to be as-expr, got %T", be.Right)
	}
	lit, ok := ae.Expr.(*ast.IntLit)
	if !ok || lit.Text != "2" {
		t.Fatalf("expected cast expr to be int lit 2, got %T", ae.Expr)
	}
	nt, ok := ae.Ty.(*ast.NamedType)
	if !ok || len(nt.Parts) != 1 || nt.Parts[0] != "i64" {
		t.Fatalf("expected cast target i64, got %T %#v", ae.Ty, ae.Ty)
	}
}

func TestParsePrecedenceShiftLowerThanAdd(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 1 + 2 << 3; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	ret, ok := prog.Funcs[0].Body.Stmts[0].(*ast.ReturnStmt)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return stmt with expr")
	}
	sh, ok := ret.Expr.(*ast.BinaryExpr)
	if !ok || sh.Op != "<<" {
		t.Fatalf("expected top-level <<, got %T", ret.Expr)
	}
	add, ok := sh.Left.(*ast.BinaryExpr)
	if !ok || add.Op != "+" {
		t.Fatalf("expected left child +, got %T", sh.Left)
	}
	if _, ok := sh.Right.(*ast.IntLit); !ok {
		t.Fatalf("expected right child int, got %T", sh.Right)
	}
}

func TestParsePrecedenceBitwiseChain(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { return 1 | 2 ^ 3 & 4; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	ret, ok := prog.Funcs[0].Body.Stmts[0].(*ast.ReturnStmt)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return stmt with expr")
	}
	or, ok := ret.Expr.(*ast.BinaryExpr)
	if !ok || or.Op != "|" {
		t.Fatalf("expected top-level |, got %T", ret.Expr)
	}
	xor, ok := or.Right.(*ast.BinaryExpr)
	if !ok || xor.Op != "^" {
		t.Fatalf("expected rhs ^, got %T", or.Right)
	}
	and, ok := xor.Right.(*ast.BinaryExpr)
	if !ok || and.Op != "&" {
		t.Fatalf("expected rhs &, got %T", xor.Right)
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

func TestParseGenericTypeParamBoundsSyntax(t *testing.T) {
	f := source.NewFile("test.vox", `fn eq[T: Eq + Show](x: T) -> T { return x; }
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if fn.Name != "eq" || len(fn.TypeParams) != 1 || fn.TypeParams[0] != "T" {
		t.Fatalf("unexpected generic func decl: %#v", fn)
	}
}

func TestParseGenericWhereClauseSyntax(t *testing.T) {
	f := source.NewFile("test.vox", `fn eq[T](x: T, y: T) -> bool where T: Eq + Show { return true; }
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if fn.Name != "eq" || len(fn.TypeParams) != 1 || fn.TypeParams[0] != "T" {
		t.Fatalf("unexpected generic func decl: %#v", fn)
	}
}

func TestParseStructDeclTypeParams(t *testing.T) {
	f := source.NewFile("test.vox", `struct Pair[T] { a: T, b: T }
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(prog.Structs))
	}
	st := prog.Structs[0]
	if st.Name != "Pair" || len(st.TypeParams) != 1 || st.TypeParams[0] != "T" {
		t.Fatalf("unexpected struct type params: %#v", st)
	}
}

func TestParseEnumDeclTypeParams(t *testing.T) {
	f := source.NewFile("test.vox", `enum Option[T] { Some(T), None }
fn main() -> i32 { return 0; }`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	if len(prog.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(prog.Enums))
	}
	en := prog.Enums[0]
	if en.Name != "Option" || len(en.TypeParams) != 1 || en.TypeParams[0] != "T" {
		t.Fatalf("unexpected enum type params: %#v", en)
	}
}

func TestParseTypedPathGenericStructLit(t *testing.T) {
	f := source.NewFile("test.vox", `struct Pair[T] { a: T, b: T }
fn main() -> i32 {
  let _p = Pair[i32] { a: 1, b: 2 };
  return 0;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	mainFn := prog.Funcs[0]
	letSt, ok := mainFn.Body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected let stmt, got %T", mainFn.Body.Stmts[0])
	}
	lit, ok := letSt.Init.(*ast.StructLitExpr)
	if !ok {
		t.Fatalf("expected struct lit, got %T", letSt.Init)
	}
	if len(lit.TypeParts) != 1 || lit.TypeParts[0] != "Pair" {
		t.Fatalf("unexpected struct lit path: %#v", lit.TypeParts)
	}
	if len(lit.TypeArgs) != 1 {
		t.Fatalf("expected 1 struct type arg, got %d", len(lit.TypeArgs))
	}
}

func TestParseTypedPathGenericEnumCtor(t *testing.T) {
	f := source.NewFile("test.vox", `enum Option[T] { Some(T), None }
fn main() -> i32 {
  let _x = Option[i32].Some(7);
  return 0;
}`)
	prog, diags := Parse(f)
	if diags != nil && len(diags.Items) > 0 {
		t.Fatalf("unexpected diags: %+v", diags.Items)
	}
	mainFn := prog.Funcs[0]
	letSt, ok := mainFn.Body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected let stmt, got %T", mainFn.Body.Stmts[0])
	}
	call, ok := letSt.Init.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", letSt.Init)
	}
	if len(call.TypeArgs) != 0 {
		t.Fatalf("expected call.TypeArgs empty for typed path constructor, got %d", len(call.TypeArgs))
	}
	mem, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected member callee, got %T", call.Callee)
	}
	ta, ok := mem.Recv.(*ast.TypeAppExpr)
	if !ok {
		t.Fatalf("expected type-app receiver, got %T", mem.Recv)
	}
	if len(ta.TypeArgs) != 1 {
		t.Fatalf("expected 1 typed-path type arg, got %d", len(ta.TypeArgs))
	}
}
