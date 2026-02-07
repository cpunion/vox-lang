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
	// CallTargets stores the resolved function name (possibly qualified, e.g. "dep::foo")
	// for each call expression.
	CallTargets map[*ast.CallExpr]string
}

type FuncSig struct {
	Params []Type
	Ret    Type
}

func Check(prog *ast.Program) (*CheckedProgram, *diag.Bag) {
	c := &checker{
		prog:      prog,
		diags:     &diag.Bag{},
		funcSigs:  map[string]FuncSig{},
		exprTypes: map[ast.Expr]Type{},
		callTgts:  map[*ast.CallExpr]string{},
	}
	c.collectFuncSigs()
	c.checkAll()
	return &CheckedProgram{Prog: prog, FuncSigs: c.funcSigs, ExprTypes: c.exprTypes, CallTargets: c.callTgts}, c.diags
}

type checker struct {
	prog      *ast.Program
	diags     *diag.Bag
	funcSigs  map[string]FuncSig
	exprTypes map[ast.Expr]Type
	callTgts  map[*ast.CallExpr]string

	curFn *ast.FuncDecl
	scope []map[string]varInfo
}

type varInfo struct {
	ty      Type
	mutable bool
}

func (c *checker) collectFuncSigs() {
	// Builtins (stage0): keep minimal and stable.
	c.funcSigs["assert"] = FuncSig{Params: []Type{{K: TyBool}}, Ret: Type{K: TyUnit}}

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
		var target string
		switch cal := e.Callee.(type) {
		case *ast.IdentExpr:
			// Unqualified call: resolve within the current package first.
			target = cal.Name
			if _, ok := c.funcSigs[target]; !ok {
				pkg := names.PackageFromFileName(c.curFn.Span.File.Name)
				if pkg != "" {
					if _, ok := c.funcSigs[pkg+"::"+cal.Name]; ok {
						target = pkg + "::" + cal.Name
					}
				}
			}
		case *ast.PathExpr:
			if len(cal.Parts) != 2 {
				c.errorAt(e.S, "stage0 only supports `pkg::fn(...)` calls for qualified names")
				return c.setExprType(ex, Type{K: TyBad})
			}
			target = strings.Join(cal.Parts, "::")
		default:
			c.errorAt(e.S, "callee must be an identifier (stage0)")
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
