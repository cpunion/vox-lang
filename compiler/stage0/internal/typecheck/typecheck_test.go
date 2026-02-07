package typecheck

import (
	"strings"
	"testing"

	"voxlang/internal/ast"
	"voxlang/internal/parser"
	"voxlang/internal/source"
	"voxlang/internal/stdlib"
)

func TestUntypedIntConstraint(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { let x: i32 = 1; return x; }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestQualifiedCallRequiresImport(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn main() -> i32 { return dep.one(); }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected typecheck diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "unknown module qualifier: dep (did you forget `import \"dep\"`?)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing import diagnostic, got: %+v", tdiags.Items)
	}
}

func TestImportAliasResolvesCallTarget(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "mathlib" as m
fn main() -> i32 { return m.one(); }`),
		source.NewFile("mathlib/src/mathlib.vox", `pub fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	var call *ast.CallExpr
	for _, fn := range prog.Funcs {
		if fn.Name != "main" {
			continue
		}
		// return m.one();
		ret := fn.Body.Stmts[0].(*ast.ReturnStmt)
		call = ret.Expr.(*ast.CallExpr)
	}
	if call == nil {
		t.Fatalf("missing call expr")
	}
	if got := checked.CallTargets[call]; got != "mathlib::one" {
		t.Fatalf("expected call target mathlib::one, got %q", got)
	}
}

func TestBreakContinueOutsideLoop(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
		want string
	}{
		{name: "break", src: `fn main() -> i32 { break; return 0; }`, want: "`break` outside of loop"},
		{name: "continue", src: `fn main() -> i32 { continue; return 0; }`, want: "`continue` outside of loop"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := source.NewFile("src/main.vox", tt.src)
			prog, pdiags := parser.Parse(f)
			if pdiags != nil && len(pdiags.Items) > 0 {
				t.Fatalf("parse diags: %+v", pdiags.Items)
			}
			_, tdiags := Check(prog, Options{})
			if tdiags == nil || len(tdiags.Items) == 0 {
				t.Fatalf("expected diagnostics")
			}
			found := false
			for _, it := range tdiags.Items {
				if it.Msg == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q, got: %+v", tt.want, tdiags.Items)
			}
		})
	}
}

func TestWhileConditionMustBeBool(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn main() -> i32 { while 1 { } return 0; }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "while condition must be bool" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected while condition diag, got: %+v", tdiags.Items)
	}
}

func TestIfExprTypechecks(t *testing.T) {
	for _, tt := range []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "ok",
			src: `fn main() -> i32 {
  let x: i32 = if true { 1 } else { 2 };
  return x;
}`,
		},
		{
			name:    "mismatch",
			wantErr: "if branch type mismatch",
			src: `fn main() -> i32 {
  let x: i32 = if true { 1 } else { false };
  return x;
}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := source.NewFile("src/main.vox", tt.src)
			prog, pdiags := parser.Parse(f)
			if pdiags != nil && len(pdiags.Items) > 0 {
				t.Fatalf("parse diags: %+v", pdiags.Items)
			}
			_, tdiags := Check(prog, Options{})
			if tt.wantErr == "" {
				if tdiags != nil && len(tdiags.Items) > 0 {
					t.Fatalf("type diags: %+v", tdiags.Items)
				}
				return
			}
			if tdiags == nil || len(tdiags.Items) == 0 {
				t.Fatalf("expected diagnostics")
			}
			found := false
			for _, it := range tdiags.Items {
				if strings.Contains(it.Msg, tt.wantErr) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected diag containing %q, got: %+v", tt.wantErr, tdiags.Items)
			}
		})
	}
}

func TestModuleQualifierDoesNotOverrideLocalValue(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "dep"
fn main() -> i32 {
  let dep: i32 = 0;
  return dep.one();
}`),
		source.NewFile("dep/src/dep.vox", `fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "member calls on values are not supported yet" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected member-call-on-value diag, got: %+v", tdiags.Items)
	}
}

func TestGenericFuncInferenceAndMonomorphization(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn id[T](x: T) -> T { return x; }
fn main() -> i32 { return id(1); }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	// Find call expr: id(1)
	var call *ast.CallExpr
	for _, fn := range prog.Funcs {
		if fn.Name != "main" {
			continue
		}
		ret := fn.Body.Stmts[0].(*ast.ReturnStmt)
		call = ret.Expr.(*ast.CallExpr)
	}
	if call == nil {
		t.Fatalf("missing call expr")
	}
	tgt := checked.CallTargets[call]
	if !strings.HasPrefix(tgt, "id$") {
		t.Fatalf("expected monomorphized call target starting with id$, got %q", tgt)
	}
	if _, ok := checked.FuncSigs[tgt]; !ok {
		t.Fatalf("expected instantiated function sig %q to exist", tgt)
	}
}

func TestStructFieldReadWrite(t *testing.T) {
	f := source.NewFile("src/main.vox", `struct Point { x: i32, y: i32 }
fn main() -> i32 {
  let mut p: Point = Point { x: 1, y: 2 };
  let a: i32 = p.x;
  p.x = a + 1;
  return p.x + p.y;
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestEnumCtorAndMatch(t *testing.T) {
	f := source.NewFile("src/main.vox", `enum E { A(i32), B(String), None }
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
	stdFiles, err := stdlib.Files()
	if err != nil {
		t.Fatal(err)
	}
	prog, pdiags := parser.ParseFiles(append(stdFiles, f))
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestEnumEqualityAgainstUnitVariant(t *testing.T) {
	for _, tt := range []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "ok",
			src: `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  let b: bool = x == E.None;
  if b { return 1; } else { return 0; }
}`,
		},
		{
			name:    "reject_payload_ctor_compare",
			wantErr: "enum equality is only supported against unit variants in stage0",
			src: `enum E { A(i32), None }
fn main() -> i32 {
  let x: E = E.A(1);
  let b: bool = x == E.A(2);
  if b { return 1; } else { return 0; }
}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := source.NewFile("src/main.vox", tt.src)
			prog, pdiags := parser.Parse(f)
			if pdiags != nil && len(pdiags.Items) > 0 {
				t.Fatalf("parse diags: %+v", pdiags.Items)
			}
			_, tdiags := Check(prog, Options{})
			if tt.wantErr == "" {
				if tdiags != nil && len(tdiags.Items) > 0 {
					t.Fatalf("type diags: %+v", tdiags.Items)
				}
				return
			}
			if tdiags == nil || len(tdiags.Items) == 0 {
				t.Fatalf("expected diagnostics")
			}
			found := false
			for _, it := range tdiags.Items {
				if it.Msg == tt.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q, got: %+v", tt.wantErr, tdiags.Items)
			}
		})
	}
}

func TestEnumMultiPayload(t *testing.T) {
	f := source.NewFile("src/main.vox", `enum E { Pair(i32, i32), None }
fn main() -> i32 {
  let x: E = E.Pair(40, 2);
  return match x {
    E.Pair(a, b) => a + b,
    E.None => 0,
  };
}`)
	stdFiles, err := stdlib.Files()
	if err != nil {
		t.Fatal(err)
	}
	prog, pdiags := parser.ParseFiles(append(stdFiles, f))
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestPubVisibilityForCrossModuleAccess(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "a"
fn main() -> i32 {
  // calling a private function must fail
  return a.secret();
}`),
		source.NewFile("src/a/a.vox", `fn secret() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "function is private: a::secret" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected private function diagnostic, got: %+v", tdiags.Items)
	}
}

func TestPubVisibilityForStructTypeAndField(t *testing.T) {
	// 1) private type cannot be constructed from another module
	{
		files := []*source.File{
			source.NewFile("src/main.vox", `import "a"
fn main() -> i32 {
  let s = a.S { x: 1 };
  return 0;
}`),
			source.NewFile("src/a/a.vox", `struct S { x: i32 }`),
		}
		prog, pdiags := parser.ParseFiles(files)
		if pdiags != nil && len(pdiags.Items) > 0 {
			t.Fatalf("parse diags: %+v", pdiags.Items)
		}
		_, tdiags := Check(prog, Options{})
		if tdiags == nil || len(tdiags.Items) == 0 {
			t.Fatalf("expected diagnostics")
		}
		found := false
		for _, it := range tdiags.Items {
			if it.Msg == "type is private: a::S" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected private type diagnostic, got: %+v", tdiags.Items)
		}
	}

	// 2) private field cannot be accessed from another module
	{
		files := []*source.File{
			source.NewFile("src/main.vox", `import "a"
fn main() -> i32 {
  let s = a.make();
  return s.y;
}`),
			source.NewFile("src/a/a.vox", `pub struct S { pub x: i32, y: i32 }
pub fn make() -> S { return S { x: 1, y: 2 }; }`),
		}
		prog, pdiags := parser.ParseFiles(files)
		if pdiags != nil && len(pdiags.Items) > 0 {
			t.Fatalf("parse diags: %+v", pdiags.Items)
		}
		_, tdiags := Check(prog, Options{})
		if tdiags == nil || len(tdiags.Items) == 0 {
			t.Fatalf("expected diagnostics")
		}
		found := false
		for _, it := range tdiags.Items {
			if it.Msg == "field is private: a::S.y" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected private field diagnostic, got: %+v", tdiags.Items)
		}
	}
}

func TestPubInterfaceCannotExposePrivateTypes(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "a"
fn main() -> i32 { return a.f(); }`),
		source.NewFile("src/a/a.vox", `struct Hidden { x: i32 }
pub fn f() -> Hidden { return Hidden { x: 1 }; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "public function a::f exposes private type: a::Hidden" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected private-in-public-interface diagnostic, got: %+v", tdiags.Items)
	}
}

func TestQualifiedTypePathInSignature(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import "a"
fn id(s: a.S) -> a.S { return s; }
fn main() -> i32 {
  let s: a.S = a.S { x: 1 };
  let t: a.S = id(s);
  return t.x;
}`),
		source.NewFile("src/a/a.vox", `pub struct S { pub x: i32 }
`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestNamedImportResolvesFunctionCall(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import { one as uno } from "dep"
fn main() -> i32 { return uno(); }`),
		source.NewFile("dep/src/dep.vox", `pub fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	checked, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
	var call *ast.CallExpr
	for _, fn := range prog.Funcs {
		if fn.Name != "main" {
			continue
		}
		ret := fn.Body.Stmts[0].(*ast.ReturnStmt)
		call = ret.Expr.(*ast.CallExpr)
	}
	if call == nil {
		t.Fatalf("missing call expr")
	}
	if got := checked.CallTargets[call]; got != "dep::one" {
		t.Fatalf("expected call target dep::one, got %q", got)
	}
}

func TestNamedImportRequiresPub(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import { one } from "dep"
fn main() -> i32 { return one(); }`),
		source.NewFile("dep/src/dep.vox", `fn one() -> i32 { return 1; }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags == nil || len(tdiags.Items) == 0 {
		t.Fatalf("expected diagnostics")
	}
	found := false
	for _, it := range tdiags.Items {
		if it.Msg == "function is private: dep::one" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected private function diagnostic, got: %+v", tdiags.Items)
	}
}

func TestNamedImportResolvesStructTypeInTypeAndLiteral(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import { S } from "dep"
fn main() -> i32 {
  let s: S = S { x: 1 };
  return s.x;
}`),
		source.NewFile("dep/src/dep.vox", `pub struct S { pub x: i32 }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestNamedImportResolvesEnumCtorAndMatch(t *testing.T) {
	files := []*source.File{
		source.NewFile("src/main.vox", `import { E } from "dep"
fn main() -> i32 {
  let x: E = E.A(41);
  return match x {
    E.A(v) => v + 1,
    E.None => 0,
  };
}`),
		source.NewFile("dep/src/dep.vox", `pub enum E { A(i32), None }`),
	}
	prog, pdiags := parser.ParseFiles(files)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}

func TestVecBasicTypecheck(t *testing.T) {
	f := source.NewFile("src/main.vox", `fn main() -> i32 {
  let mut v: Vec[i32] = Vec();
  v.push(41);
  return v.get(0) + v.len();
}`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog, Options{})
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}
