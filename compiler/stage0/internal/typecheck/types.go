package typecheck

import (
	"fmt"
	"strconv"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/diag"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

type Kind int

const (
	TyBad Kind = iota
	TyUnit
	TyBool
	TyI32
	TyI64
	TyString
	TyUntypedInt
)

type Type struct {
	K Kind
}

func (t Type) String() string {
	switch t.K {
	case TyUnit:
		return "()"
	case TyBool:
		return "bool"
	case TyI32:
		return "i32"
	case TyI64:
		return "i64"
	case TyString:
		return "String"
	case TyUntypedInt:
		return "untyped-int"
	default:
		return "<bad>"
	}
}

type CheckedProgram struct {
	Prog      *ast.Program
	FuncSigs  map[string]FuncSig
	ExprTypes map[ast.Expr]Type
	// CallTargets stores the resolved function name (possibly qualified, e.g. "dep::foo").
	// Note: Vox surface syntax may use `dep.foo(...)`; it still resolves to `dep::foo` internally.
	// for each call expression.
	CallTargets map[*ast.CallExpr]string
}

type FuncSig struct {
	Params []Type
	Ret    Type
}

type Options struct {
	// AllowedPkgs is the set of importable package names in the current build.
	// When nil, imports are accepted without validation.
	AllowedPkgs map[string]bool
	// LocalModules is the set of importable local modules (paths like "utils" or "utils/io").
	// When nil, local module imports are accepted without validation.
	LocalModules map[string]bool
	// LocalModulesByPkg validates local module imports per package.
	// Key is dependency package name; root package uses "".
	// When nil, LocalModules is used as a fallback.
	LocalModulesByPkg map[string]map[string]bool
}

func Check(prog *ast.Program, opts Options) (*CheckedProgram, *diag.Bag) {
	c := &checker{
		prog:      prog,
		diags:     &diag.Bag{},
		funcSigs:  map[string]FuncSig{},
		exprTypes: map[ast.Expr]Type{},
		callTgts:  map[*ast.CallExpr]string{},
		opts:      opts,
		imports:   map[*source.File]map[string]importTarget{},
	}
	c.collectFuncSigs()
	c.collectImports()
	c.checkAll()
	return &CheckedProgram{Prog: prog, FuncSigs: c.funcSigs, ExprTypes: c.exprTypes, CallTargets: c.callTgts}, c.diags
}

type checker struct {
	prog      *ast.Program
	diags     *diag.Bag
	funcSigs  map[string]FuncSig
	exprTypes map[ast.Expr]Type
	callTgts  map[*ast.CallExpr]string

	curFn     *ast.FuncDecl
	scope     []map[string]varInfo
	loopDepth int

	opts    Options
	imports map[*source.File]map[string]importTarget // file -> qualifier -> target
}

type varInfo struct {
	ty      Type
	mutable bool
}

type importTarget struct {
	Pkg string   // dependency package name; empty for local modules
	Mod []string // base module path segments
}

func (c *checker) collectFuncSigs() {
	// Builtins (stage0): keep minimal and stable.
	c.funcSigs["assert"] = FuncSig{Params: []Type{{K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert"] = FuncSig{Params: []Type{{K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_i32"] = FuncSig{Params: []Type{{K: TyI32}, {K: TyI32}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_i64"] = FuncSig{Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_bool"] = FuncSig{Params: []Type{{K: TyBool}, {K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_str"] = FuncSig{Params: []Type{{K: TyString}, {K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::fail"] = FuncSig{Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}

	for _, fn := range c.prog.Funcs {
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		if _, exists := c.funcSigs[qname]; exists {
			c.errorAt(fn.Span, "duplicate function: "+qname)
			continue
		}
		sig := FuncSig{}
		for _, p := range fn.Params {
			sig.Params = append(sig.Params, c.typeFromAst(p.Type))
		}
		sig.Ret = c.typeFromAst(fn.Ret)
		c.funcSigs[qname] = sig
	}
	// main presence check (stage0)
	if _, ok := c.funcSigs["main"]; !ok {
		// not a hard error for library compilation, but stage0 expects main
	}
}

func (c *checker) checkAll() {
	for _, fn := range c.prog.Funcs {
		c.curFn = fn
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		sig := c.funcSigs[qname]
		c.pushScope()
		for i, p := range fn.Params {
			c.scopeTop()[p.Name] = varInfo{ty: sig.Params[i], mutable: false}
		}
		c.checkBlock(fn.Body, sig.Ret)
		c.popScope()
	}
}

func (c *checker) collectImports() {
	for _, imp := range c.prog.Imports {
		if imp == nil || imp.Span.File == nil {
			continue
		}

		path := imp.Path
		alias := imp.Alias

		var tgt importTarget
		if c.opts.AllowedPkgs != nil && c.opts.AllowedPkgs[path] {
			// dependency root import
			tgt = importTarget{Pkg: path, Mod: nil}
			if alias == "" {
				alias = path
			}
		} else {
			// local module import
			if !c.isKnownLocalModule(imp.Span.File.Name, path) {
				c.errorAt(imp.Span, "unknown local module: "+path)
				continue
			}
			tgt = importTarget{Pkg: "", Mod: splitModPath(path)}
			if alias == "" {
				alias = defaultImportAlias(path)
			}
		}

		m := c.imports[imp.Span.File]
		if m == nil {
			m = map[string]importTarget{}
			c.imports[imp.Span.File] = m
		}
		if _, exists := m[alias]; exists {
			c.errorAt(imp.Span, "duplicate import alias: "+alias)
			continue
		}
		m[alias] = tgt
	}
}

func (c *checker) checkBlock(b *ast.BlockStmt, expectedRet Type) {
	c.pushScope()
	for _, st := range b.Stmts {
		c.checkStmt(st, expectedRet)
	}
	c.popScope()
}

func (c *checker) checkStmt(st ast.Stmt, expectedRet Type) {
	switch s := st.(type) {
	case *ast.BlockStmt:
		c.checkBlock(s, expectedRet)
	case *ast.LetStmt:
		var ann Type
		if s.AnnType != nil {
			ann = c.typeFromAst(s.AnnType)
		}
		var initTy Type
		if s.Init != nil {
			initTy = c.checkExpr(s.Init, ann)
			if initTy.K == TyUntypedInt && ann.K == TyBad {
				initTy = Type{K: TyI64}
			}
			if ann.K != TyBad && !sameType(ann, initTy) {
				c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", ann.String(), initTy.String()))
			}
		} else {
			if ann.K == TyBad {
				c.errorAt(s.S, "let binding requires a type annotation or initializer")
				ann = Type{K: TyBad}
			}
			initTy = ann
		}
		c.scopeTop()[s.Name] = varInfo{ty: chooseType(ann, initTy), mutable: s.Mutable}
	case *ast.AssignStmt:
		vi, ok := c.lookupVar(s.Name)
		if !ok {
			c.errorAt(s.S, "unknown variable: "+s.Name)
			return
		}
		if !vi.mutable {
			c.errorAt(s.S, "cannot assign to immutable variable: "+s.Name)
			return
		}
		rhs := c.checkExpr(s.Expr, vi.ty)
		if !sameType(vi.ty, rhs) {
			c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", vi.ty.String(), rhs.String()))
		}
	case *ast.ReturnStmt:
		var ty Type
		if s.Expr == nil {
			ty = Type{K: TyUnit}
		} else {
			ty = c.checkExpr(s.Expr, expectedRet)
		}
		if !sameType(expectedRet, ty) {
			c.errorAt(s.S, fmt.Sprintf("return type mismatch: expected %s, got %s", expectedRet.String(), ty.String()))
		}
	case *ast.IfStmt:
		condTy := c.checkExpr(s.Cond, Type{K: TyBool})
		if condTy.K != TyBool {
			c.errorAt(s.Cond.Span(), "if condition must be bool")
		}
		c.checkBlock(s.Then, expectedRet)
		if s.Else != nil {
			c.checkStmt(s.Else, expectedRet)
		}
	case *ast.WhileStmt:
		condTy := c.checkExpr(s.Cond, Type{K: TyBool})
		if condTy.K != TyBool {
			c.errorAt(s.Cond.Span(), "while condition must be bool")
		}
		c.loopDepth++
		c.checkBlock(s.Body, expectedRet)
		c.loopDepth--
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			c.errorAt(s.S, "`break` outside of loop")
		}
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			c.errorAt(s.S, "`continue` outside of loop")
		}
	case *ast.ExprStmt:
		_ = c.checkExpr(s.Expr, Type{K: TyBad})
	default:
		c.errorAt(st.Span(), "unsupported statement")
	}
}

func (c *checker) checkExpr(ex ast.Expr, expected Type) Type {
	switch e := ex.(type) {
	case *ast.IntLit:
		v, err := strconv.ParseInt(e.Text, 10, 64)
		if err != nil {
			c.errorAt(e.S, "invalid integer literal")
			return c.setExprType(ex, Type{K: TyBad})
		}
		// Constrain by expected type if it's an int.
		if expected.K == TyI32 {
			if v < -2147483648 || v > 2147483647 {
				c.errorAt(e.S, "integer literal out of range for i32")
				return c.setExprType(ex, Type{K: TyBad})
			}
			return c.setExprType(ex, expected)
		}
		if expected.K == TyI64 {
			return c.setExprType(ex, expected)
		}
		return c.setExprType(ex, Type{K: TyUntypedInt})
	case *ast.StringLit:
		return c.setExprType(ex, Type{K: TyString})
	case *ast.BoolLit:
		return c.setExprType(ex, Type{K: TyBool})
	case *ast.IdentExpr:
		if vi, ok := c.lookupVar(e.Name); ok {
			return c.setExprType(ex, vi.ty)
		}
		// function name is allowed as callee only
		if _, ok := c.funcSigs[e.Name]; ok {
			return c.setExprType(ex, Type{K: TyBad})
		}
		c.errorAt(e.S, "unknown identifier: "+e.Name)
		return c.setExprType(ex, Type{K: TyBad})
	case *ast.UnaryExpr:
		switch e.Op {
		case "-":
			ty := c.checkExpr(e.Expr, Type{K: TyI64})
			ty = c.forceIntType(e.Expr, ty, expected)
			return c.setExprType(ex, ty)
		case "!":
			ty := c.checkExpr(e.Expr, Type{K: TyBool})
			if ty.K != TyBool {
				c.errorAt(e.S, "operator ! expects bool")
			}
			return c.setExprType(ex, Type{K: TyBool})
		default:
			c.errorAt(e.S, "unknown unary operator: "+e.Op)
			return c.setExprType(ex, Type{K: TyBad})
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case "+", "-", "*", "/", "%":
			l := c.checkExpr(e.Left, expected)
			r := c.checkExpr(e.Right, l)
			l = c.forceIntType(e.Left, l, expected)
			r = c.forceIntType(e.Right, r, l)
			if !sameType(l, r) {
				c.errorAt(e.S, "binary integer ops require same type")
				return c.setExprType(ex, Type{K: TyBad})
			}
			return c.setExprType(ex, l)
		case "<", "<=", ">", ">=":
			l := c.checkExpr(e.Left, Type{K: TyI64})
			r := c.checkExpr(e.Right, l)
			l = c.forceIntType(e.Left, l, Type{K: TyI64})
			r = c.forceIntType(e.Right, r, l)
			if !sameType(l, r) {
				c.errorAt(e.S, "comparison requires same integer type")
			}
			return c.setExprType(ex, Type{K: TyBool})
		case "==", "!=":
			l := c.checkExpr(e.Left, Type{K: TyBad})
			r := c.checkExpr(e.Right, l)
			if l.K == TyUntypedInt || r.K == TyUntypedInt {
				// default to i64
				l = c.forceIntType(e.Left, l, Type{K: TyI64})
				r = c.forceIntType(e.Right, r, l)
			}
			if !sameType(l, r) {
				c.errorAt(e.S, "equality requires same type")
			}
			return c.setExprType(ex, Type{K: TyBool})
		case "&&", "||":
			l := c.checkExpr(e.Left, Type{K: TyBool})
			r := c.checkExpr(e.Right, Type{K: TyBool})
			if l.K != TyBool || r.K != TyBool {
				c.errorAt(e.S, "logical operators require bool")
			}
			return c.setExprType(ex, Type{K: TyBool})
		default:
			c.errorAt(e.S, "unknown operator: "+e.Op)
			return c.setExprType(ex, Type{K: TyBad})
		}
	case *ast.CallExpr:
		parts, ok := calleeParts(e.Callee)
		if !ok || len(parts) == 0 {
			c.errorAt(e.S, "callee must be an identifier or member path (stage0)")
			return c.setExprType(ex, Type{K: TyBad})
		}

		target := ""
		if len(parts) == 1 {
			name := parts[0]
			if _, ok := c.funcSigs[name]; ok {
				target = name // builtins
			} else {
				pkg, mod, _ := names.SplitOwnerAndModule(c.curFn.Span.File.Name)
				q := names.QualifyParts(pkg, mod, name)
				if _, ok := c.funcSigs[q]; ok {
					target = q
				} else {
					// fallback: root module of the same package
					q2 := names.QualifyParts(pkg, nil, name)
					if _, ok := c.funcSigs[q2]; ok {
						target = q2
					}
				}
			}
		} else {
			// Qualified call: first segment must be an imported alias.
			qualParts := parts[:len(parts)-1]
			member := parts[len(parts)-1]
			alias := qualParts[0]
			extraMods := qualParts[1:]

			if _, ok := c.lookupVar(alias); ok {
				c.errorAt(e.S, "member calls on values are not supported yet")
				return c.setExprType(ex, Type{K: TyBad})
			}

			if c.curFn.Span.File == nil {
				c.errorAt(e.S, "internal error: missing file for import resolution")
				return c.setExprType(ex, Type{K: TyBad})
			}
			m := c.imports[c.curFn.Span.File]
			tgt, ok := m[alias]
			if !ok {
				c.errorAt(e.S, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
				return c.setExprType(ex, Type{K: TyBad})
			}
			mod := append(append([]string{}, tgt.Mod...), extraMods...)
			target = names.QualifyParts(tgt.Pkg, mod, member)
		}

		if target == "" {
			c.errorAt(e.S, "unknown function")
			return c.setExprType(ex, Type{K: TyBad})
		}
		sig, ok := c.funcSigs[target]
		if !ok {
			c.errorAt(e.S, "unknown function: "+target)
			return c.setExprType(ex, Type{K: TyBad})
		}
		c.callTgts[e] = target
		if len(e.Args) != len(sig.Params) {
			c.errorAt(e.S, fmt.Sprintf("wrong number of arguments: expected %d, got %d", len(sig.Params), len(e.Args)))
			return c.setExprType(ex, Type{K: TyBad})
		}
		for i, a := range e.Args {
			at := c.checkExpr(a, sig.Params[i])
			if !sameType(sig.Params[i], at) {
				c.errorAt(a.Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", sig.Params[i].String(), at.String()))
			}
		}
		return c.setExprType(ex, sig.Ret)
	default:
		c.errorAt(ex.Span(), "unsupported expression")
		return c.setExprType(ex, Type{K: TyBad})
	}
}

func (c *checker) forceIntType(ex ast.Expr, got Type, want Type) Type {
	if got.K == TyUntypedInt {
		// untyped int defaults to "want" if want is concrete int, else i64.
		if want.K == TyI32 || want.K == TyI64 {
			return want
		}
		return Type{K: TyI64}
	}
	if got.K == TyI32 || got.K == TyI64 {
		return got
	}
	return got
}

func (c *checker) typeFromAst(t ast.Type) Type {
	switch tt := t.(type) {
	case *ast.UnitType:
		return Type{K: TyUnit}
	case *ast.NamedType:
		switch tt.Name {
		case "i32":
			return Type{K: TyI32}
		case "i64":
			return Type{K: TyI64}
		case "bool":
			return Type{K: TyBool}
		case "String":
			return Type{K: TyString}
		default:
			c.errorAt(tt.S, "unknown type: "+tt.Name)
			return Type{K: TyBad}
		}
	default:
		c.errorAt(t.Span(), "unsupported type")
		return Type{K: TyBad}
	}
}

func (c *checker) setExprType(ex ast.Expr, t Type) Type {
	c.exprTypes[ex] = t
	return t
}

func (c *checker) pushScope() { c.scope = append(c.scope, map[string]varInfo{}) }
func (c *checker) popScope()  { c.scope = c.scope[:len(c.scope)-1] }
func (c *checker) scopeTop() map[string]varInfo {
	return c.scope[len(c.scope)-1]
}

func (c *checker) lookupVar(name string) (varInfo, bool) {
	for i := len(c.scope) - 1; i >= 0; i-- {
		if v, ok := c.scope[i][name]; ok {
			return v, true
		}
	}
	return varInfo{}, false
}

func (c *checker) errorAt(s source.Span, msg string) {
	fn, line, col := s.LocStart()
	c.diags.Add(fn, line, col, msg)
}

func sameType(a, b Type) bool {
	if a.K == TyBad || b.K == TyBad {
		return false
	}
	// Resolve untyped int to concrete only via constraints before comparing.
	if a.K == TyUntypedInt || b.K == TyUntypedInt {
		return false
	}
	return a.K == b.K
}

func chooseType(ann, init Type) Type {
	if ann.K != TyBad {
		return ann
	}
	return init
}

func defaultImportAlias(path string) string {
	// For stage0, dependency package import paths are simple names like "mathlib".
	// If we later support nested module paths, use the last segment.
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func splitModPath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return out
}

func calleeParts(ex ast.Expr) ([]string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return []string{e.Name}, true
	case *ast.MemberExpr:
		p, ok := calleeParts(e.Recv)
		if !ok {
			return nil, false
		}
		return append(p, e.Name), true
	default:
		return nil, false
	}
}

func (c *checker) isKnownLocalModule(fileName string, importPath string) bool {
	// If no validation info is provided, accept.
	if c.opts.LocalModulesByPkg == nil && c.opts.LocalModules == nil {
		return true
	}
	pkg, _, _ := names.SplitOwnerAndModule(fileName)
	if c.opts.LocalModulesByPkg != nil {
		m := c.opts.LocalModulesByPkg[pkg]
		if m == nil {
			// No data for this package; accept to avoid spurious failures.
			return true
		}
		return m[importPath]
	}
	return c.opts.LocalModules[importPath]
}
