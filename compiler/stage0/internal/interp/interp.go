package interp

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

type ValueKind int

const (
	VUnit ValueKind = iota
	VBool
	VInt
	VString
	VStruct
	VEnum
	VVec
)

type Value struct {
	K ValueKind
	I int64
	B bool
	S string
	M map[string]Value // VStruct fields
	E string           // VEnum qualified enum name
	T int              // VEnum tag
	P *Value           // VEnum payload (nil when no payload)
	A []Value          // VVec elements
}

func unit() Value { return Value{K: VUnit} }

func cloneValue(v Value) Value {
	if v.K != VStruct {
		if v.K != VEnum {
			if v.K != VVec {
				return v
			}
			out := Value{K: VVec, A: make([]Value, 0, len(v.A))}
			for _, e := range v.A {
				out.A = append(out.A, cloneValue(e))
			}
			return out
		}
		out := Value{K: VEnum, E: v.E, T: v.T}
		if v.P != nil {
			p := cloneValue(*v.P)
			out.P = &p
		}
		return out
	}
	out := Value{K: VStruct, M: map[string]Value{}}
	for k, fv := range v.M {
		out.M[k] = cloneValue(fv)
	}
	return out
}

func derefOrUnit(v *Value) Value {
	if v == nil {
		return unit()
	}
	return *v
}

func cloneValuePtr(v *Value) *Value {
	if v == nil {
		return nil
	}
	c := cloneValue(*v)
	return &c
}

type Runtime struct {
	prog  *typecheck.CheckedProgram
	funcs map[string]*ast.FuncDecl
	stack []map[string]Value
}

func RunMain(p *typecheck.CheckedProgram) (string, error) {
	rt := &Runtime{prog: p, funcs: map[string]*ast.FuncDecl{}}
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	mainFn, ok := rt.funcs["main"]
	if !ok {
		return "", fmt.Errorf("missing main")
	}
	if len(mainFn.Params) != 0 {
		return "", fmt.Errorf("main must have no parameters (stage0)")
	}
	v, err := rt.call("main", nil)
	if err != nil {
		return "", err
	}
	switch v.K {
	case VUnit:
		return "", nil
	case VInt:
		return fmt.Sprintf("%d", v.I), nil
	case VBool:
		if v.B {
			return "true", nil
		}
		return "false", nil
	case VString:
		return v.S, nil
	default:
		return "", nil
	}
}

func RunTests(p *typecheck.CheckedProgram) (string, error) {
	rt := &Runtime{prog: p, funcs: map[string]*ast.FuncDecl{}}
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	// Discover tests by naming convention.
	var testNames []string
	for name := range rt.funcs {
		fn := rt.funcs[name]
		if fn == nil || fn.Span.File == nil {
			continue
		}
		if !isTestFile(fn.Span.File.Name) {
			continue
		}
		if strings.HasPrefix(fn.Name, "test_") {
			testNames = append(testNames, name)
		}
	}
	sort.Strings(testNames)
	var log strings.Builder
	failed := 0
	for _, name := range testNames {
		sig, ok := p.FuncSigs[name]
		if !ok || len(sig.Params) != 0 || sig.Ret.K != typecheck.TyUnit {
			failed++
			fmt.Fprintf(&log, "[FAIL] %s: invalid test signature (expected fn %s() -> ())\n", name, name)
			continue
		}
		_, err := rt.call(name, nil)
		if err != nil {
			failed++
			fmt.Fprintf(&log, "[FAIL] %s: %v\n", name, err)
		} else {
			fmt.Fprintf(&log, "[OK] %s\n", name)
		}
	}
	if len(testNames) == 0 {
		log.WriteString("[test] no tests found\n")
	} else {
		fmt.Fprintf(&log, "[test] %d passed, %d failed\n", len(testNames)-failed, failed)
	}
	if failed != 0 {
		return log.String(), fmt.Errorf("%d test(s) failed", failed)
	}
	return log.String(), nil
}

func isTestFile(name string) bool {
	// Stage0 convention:
	// - tests/**/*.vox
	// - src/**/*_test.vox
	name = filepath.ToSlash(name)
	if strings.HasPrefix(name, "tests/") {
		return true
	}
	return strings.HasSuffix(name, "_test.vox")
}

func (rt *Runtime) call(name string, args []Value) (Value, error) {
	if v, ok, err := rt.callBuiltin(name, args); ok {
		return v, err
	}
	fn, ok := rt.funcs[name]
	if !ok {
		return unit(), fmt.Errorf("unknown function: %s", name)
	}
	rt.pushFrame()
	for i, p := range fn.Params {
		rt.frame()[p.Name] = args[i]
	}
	v, err := rt.evalBlock(fn.Body)
	rt.popFrame()
	if err != nil {
		if r, ok := err.(returnSignal); ok {
			return r.V, nil
		}
		return unit(), err
	}
	return v, nil
}

type returnSignal struct{ V Value }

func (r returnSignal) Error() string { return "return" }

type breakSignal struct{}

func (b breakSignal) Error() string { return "break" }

type continueSignal struct{}

func (c continueSignal) Error() string { return "continue" }

func (rt *Runtime) evalBlock(b *ast.BlockStmt) (Value, error) {
	rt.pushFrame()
	for _, st := range b.Stmts {
		if _, err := rt.evalStmt(st); err != nil {
			rt.popFrame()
			return unit(), err
		}
	}
	rt.popFrame()
	return unit(), nil
}

func (rt *Runtime) evalStmt(st ast.Stmt) (Value, error) {
	switch s := st.(type) {
	case *ast.BlockStmt:
		return rt.evalBlock(s)
	case *ast.LetStmt:
		v := unit()
		if s.Init != nil {
			var err error
			v, err = rt.evalExpr(s.Init)
			if err != nil {
				return unit(), err
			}
		}
		rt.frame()[s.Name] = cloneValue(v)
		return unit(), nil
	case *ast.AssignStmt:
		v, err := rt.evalExpr(s.Expr)
		if err != nil {
			return unit(), err
		}
		fr, ok := rt.lookup(s.Name)
		if !ok {
			return unit(), fmt.Errorf("unknown variable: %s", s.Name)
		}
		fr[s.Name] = cloneValue(v)
		return unit(), nil
	case *ast.FieldAssignStmt:
		v, err := rt.evalExpr(s.Expr)
		if err != nil {
			return unit(), err
		}
		fr, ok := rt.lookup(s.Recv)
		if !ok {
			return unit(), fmt.Errorf("unknown variable: %s", s.Recv)
		}
		recv := fr[s.Recv]
		if recv.K != VStruct || recv.M == nil {
			return unit(), fmt.Errorf("field assignment requires struct receiver")
		}
		recv.M[s.Field] = cloneValue(v)
		fr[s.Recv] = recv
		return unit(), nil
	case *ast.ReturnStmt:
		v := unit()
		if s.Expr != nil {
			var err error
			v, err = rt.evalExpr(s.Expr)
			if err != nil {
				return unit(), err
			}
		}
		return unit(), returnSignal{V: v}
	case *ast.IfStmt:
		cond, err := rt.evalExpr(s.Cond)
		if err != nil {
			return unit(), err
		}
		if cond.K != VBool {
			return unit(), fmt.Errorf("if condition is not bool")
		}
		if cond.B {
			return rt.evalBlock(s.Then)
		}
		if s.Else != nil {
			return rt.evalStmt(s.Else)
		}
		return unit(), nil
	case *ast.WhileStmt:
		for {
			cond, err := rt.evalExpr(s.Cond)
			if err != nil {
				return unit(), err
			}
			if cond.K != VBool {
				return unit(), fmt.Errorf("while condition is not bool")
			}
			if !cond.B {
				return unit(), nil
			}
			_, err = rt.evalBlock(s.Body)
			if err != nil {
				switch err.(type) {
				case breakSignal:
					return unit(), nil
				case continueSignal:
					continue
				default:
					return unit(), err
				}
			}
		}
	case *ast.BreakStmt:
		return unit(), breakSignal{}
	case *ast.ContinueStmt:
		return unit(), continueSignal{}
	case *ast.ExprStmt:
		_, err := rt.evalExpr(s.Expr)
		return unit(), err
	default:
		return unit(), fmt.Errorf("unsupported statement")
	}
}

func (rt *Runtime) evalExpr(ex ast.Expr) (Value, error) {
	switch e := ex.(type) {
	case *ast.IntLit:
		// typechecker guaranteed parseability
		var n int64
		for i := 0; i < len(e.Text); i++ {
			n = n*10 + int64(e.Text[i]-'0')
		}
		return Value{K: VInt, I: n}, nil
	case *ast.StringLit:
		// includes quotes; keep it simple for stage0
		s := e.Text
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return Value{K: VString, S: s}, nil
	case *ast.BoolLit:
		return Value{K: VBool, B: e.Value}, nil
	case *ast.IdentExpr:
		v, ok := rt.lookupValue(e.Name)
		if !ok {
			return unit(), fmt.Errorf("unknown identifier: %s", e.Name)
		}
		return v, nil
	case *ast.UnaryExpr:
		v, err := rt.evalExpr(e.Expr)
		if err != nil {
			return unit(), err
		}
		switch e.Op {
		case "-":
			if v.K != VInt {
				return unit(), fmt.Errorf("unary - expects int")
			}
			return Value{K: VInt, I: -v.I}, nil
		case "!":
			if v.K != VBool {
				return unit(), fmt.Errorf("unary ! expects bool")
			}
			return Value{K: VBool, B: !v.B}, nil
		default:
			return unit(), fmt.Errorf("unknown unary op: %s", e.Op)
		}
	case *ast.BinaryExpr:
		l, err := rt.evalExpr(e.Left)
		if err != nil {
			return unit(), err
		}
		r, err := rt.evalExpr(e.Right)
		if err != nil {
			return unit(), err
		}
		switch e.Op {
		case "+", "-", "*", "/", "%", "<", "<=", ">", ">=":
			if l.K != VInt || r.K != VInt {
				return unit(), fmt.Errorf("binary op %s expects ints", e.Op)
			}
			switch e.Op {
			case "+":
				return Value{K: VInt, I: l.I + r.I}, nil
			case "-":
				return Value{K: VInt, I: l.I - r.I}, nil
			case "*":
				return Value{K: VInt, I: l.I * r.I}, nil
			case "/":
				return Value{K: VInt, I: l.I / r.I}, nil
			case "%":
				return Value{K: VInt, I: l.I % r.I}, nil
			case "<":
				return Value{K: VBool, B: l.I < r.I}, nil
			case "<=":
				return Value{K: VBool, B: l.I <= r.I}, nil
			case ">":
				return Value{K: VBool, B: l.I > r.I}, nil
			case ">=":
				return Value{K: VBool, B: l.I >= r.I}, nil
			}
		case "==", "!=":
			eq := valueEq(l, r)
			if e.Op == "!=" {
				eq = !eq
			}
			return Value{K: VBool, B: eq}, nil
		case "&&", "||":
			if l.K != VBool || r.K != VBool {
				return unit(), fmt.Errorf("logical op expects bool")
			}
			if e.Op == "&&" {
				return Value{K: VBool, B: l.B && r.B}, nil
			}
			return Value{K: VBool, B: l.B || r.B}, nil
		default:
			return unit(), fmt.Errorf("unknown op: %s", e.Op)
		}
		return unit(), fmt.Errorf("unreachable")
	case *ast.CallExpr:
		if vc, ok := rt.prog.VecCalls[e]; ok {
			switch vc.Kind {
			case typecheck.VecCallNew:
				return Value{K: VVec, A: nil}, nil
			case typecheck.VecCallPush:
				if len(e.Args) != 1 {
					return unit(), fmt.Errorf("Vec.push expects 1 arg")
				}
				v, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				fr, ok := rt.lookup(vc.RecvName)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
				}
				recv := fr[vc.RecvName]
				if recv.K != VVec {
					return unit(), fmt.Errorf("Vec.push requires vec receiver")
				}
				recv.A = append(recv.A, cloneValue(v))
				fr[vc.RecvName] = recv
				return unit(), nil
			case typecheck.VecCallLen:
				fr, ok := rt.lookup(vc.RecvName)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
				}
				recv := fr[vc.RecvName]
				if recv.K != VVec {
					return unit(), fmt.Errorf("Vec.len requires vec receiver")
				}
				return Value{K: VInt, I: int64(len(recv.A))}, nil
			case typecheck.VecCallGet:
				if len(e.Args) != 1 {
					return unit(), fmt.Errorf("Vec.get expects 1 arg")
				}
				idxV, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				if idxV.K != VInt {
					return unit(), fmt.Errorf("Vec.get index must be int")
				}
				idx := int(idxV.I)
				fr, ok := rt.lookup(vc.RecvName)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
				}
				recv := fr[vc.RecvName]
				if recv.K != VVec {
					return unit(), fmt.Errorf("Vec.get requires vec receiver")
				}
				if idx < 0 || idx >= len(recv.A) {
					return unit(), fmt.Errorf("Vec.get index out of bounds")
				}
				return cloneValue(recv.A[idx]), nil
			default:
				return unit(), fmt.Errorf("unsupported vec call")
			}
		}

		if ctor, ok := rt.prog.EnumCtors[e]; ok {
			if ctor.Payload.K == typecheck.TyUnit {
				return Value{K: VEnum, E: ctor.Enum.Name, T: ctor.Tag}, nil
			}
			if len(e.Args) != 1 {
				return unit(), fmt.Errorf("enum constructor expects 1 arg")
			}
			v, err := rt.evalExpr(e.Args[0])
			if err != nil {
				return unit(), err
			}
			p := cloneValue(v)
			return Value{K: VEnum, E: ctor.Enum.Name, T: ctor.Tag, P: &p}, nil
		}

		target := rt.prog.CallTargets[e]
		if target == "" {
			// Fallback (should not happen once typechecker fills CallTargets).
			switch cal := e.Callee.(type) {
			case *ast.IdentExpr:
				target = cal.Name
			case *ast.MemberExpr:
				parts, ok := interpCalleeParts(cal)
				if !ok || len(parts) == 0 {
					return unit(), fmt.Errorf("callee must be ident or member path (stage0)")
				}
				if len(parts) == 1 {
					target = parts[0]
				} else {
					qual := strings.Join(parts[:len(parts)-1], ".")
					target = qual + "::" + parts[len(parts)-1]
				}
			default:
				return unit(), fmt.Errorf("callee must be ident or member path (stage0)")
			}
		}
		args := make([]Value, 0, len(e.Args))
		for _, a := range e.Args {
			v, err := rt.evalExpr(a)
			if err != nil {
				return unit(), err
			}
			args = append(args, v)
		}
		return rt.call(target, args)
	case *ast.MemberExpr:
		if ty, ok := rt.prog.ExprTypes[ex]; ok && ty.K == typecheck.TyEnum {
			es, ok := rt.prog.EnumSigs[ty.Name]
			if !ok {
				return unit(), fmt.Errorf("unknown enum: %s", ty.Name)
			}
			tag, ok := es.VariantIndex[e.Name]
			if !ok {
				return unit(), fmt.Errorf("unknown variant: %s", e.Name)
			}
			if len(es.Variants[tag].Fields) != 0 {
				return unit(), fmt.Errorf("non-unit variant requires constructor call")
			}
			return Value{K: VEnum, E: ty.Name, T: tag}, nil
		}

		recv, err := rt.evalExpr(e.Recv)
		if err != nil {
			return unit(), err
		}
		if recv.K != VStruct || recv.M == nil {
			return unit(), fmt.Errorf("member access requires struct receiver")
		}
		v, ok := recv.M[e.Name]
		if !ok {
			return unit(), fmt.Errorf("unknown field: %s", e.Name)
		}
		return v, nil
	case *ast.StructLitExpr:
		m := map[string]Value{}
		for _, init := range e.Inits {
			v, err := rt.evalExpr(init.Expr)
			if err != nil {
				return unit(), err
			}
			m[init.Name] = cloneValue(v)
		}
		return Value{K: VStruct, M: m}, nil
	case *ast.MatchExpr:
		sv, err := rt.evalExpr(e.Scrutinee)
		if err != nil {
			return unit(), err
		}
		if sv.K != VEnum {
			return unit(), fmt.Errorf("match scrutinee is not enum")
		}
		ety := rt.prog.ExprTypes[e.Scrutinee]
		if ety.K != typecheck.TyEnum {
			ety = typecheck.Type{K: typecheck.TyEnum, Name: sv.E}
		}
		es, ok := rt.prog.EnumSigs[ety.Name]
		if !ok {
			return unit(), fmt.Errorf("unknown enum: %s", ety.Name)
		}
		for _, arm := range e.Arms {
			switch p := arm.Pat.(type) {
			case *ast.WildPat:
				return rt.evalExpr(arm.Expr)
			case *ast.VariantPat:
				tag, ok := es.VariantIndex[p.Variant]
				if !ok {
					return unit(), fmt.Errorf("unknown variant: %s", p.Variant)
				}
				if sv.T != tag {
					continue
				}
				rt.pushFrame()
				if len(p.Binds) == 1 {
					rt.frame()[p.Binds[0]] = derefOrUnit(cloneValuePtr(sv.P))
				}
				v, err := rt.evalExpr(arm.Expr)
				rt.popFrame()
				return v, err
			default:
				return unit(), fmt.Errorf("unsupported pattern")
			}
		}
		return unit(), fmt.Errorf("non-exhaustive match")
	default:
		return unit(), fmt.Errorf("unsupported expr")
	}
}

func interpCalleeParts(ex ast.Expr) ([]string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return []string{e.Name}, true
	case *ast.MemberExpr:
		p, ok := interpCalleeParts(e.Recv)
		if !ok {
			return nil, false
		}
		return append(p, e.Name), true
	default:
		return nil, false
	}
}

func valueEq(a, b Value) bool {
	if a.K != b.K {
		return false
	}
	switch a.K {
	case VUnit:
		return true
	case VBool:
		return a.B == b.B
	case VInt:
		return a.I == b.I
	case VString:
		return a.S == b.S
	case VStruct:
		// Not needed for stage0 yet; keep it conservative.
		return false
	case VEnum:
		if a.E != b.E || a.T != b.T {
			return false
		}
		if a.P == nil && b.P == nil {
			return true
		}
		if a.P == nil || b.P == nil {
			return false
		}
		return valueEq(*a.P, *b.P)
	case VVec:
		// Not needed for stage0 yet; keep it conservative.
		return false
	default:
		return false
	}
}

func (rt *Runtime) callBuiltin(name string, args []Value) (Value, bool, error) {
	switch name {
	case "assert":
		if len(args) != 1 || args[0].K != VBool {
			return unit(), true, fmt.Errorf("assert expects (bool)")
		}
		if !args[0].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert":
		if len(args) != 1 || args[0].K != VBool {
			return unit(), true, fmt.Errorf("std.testing::assert expects (bool)")
		}
		if !args[0].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_i32", "std.testing::assert_eq_i64":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("%s expects (int, int)", name)
		}
		if args[0].I != args[1].I {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_bool":
		if len(args) != 2 || args[0].K != VBool || args[1].K != VBool {
			return unit(), true, fmt.Errorf("std.testing::assert_eq_bool expects (bool, bool)")
		}
		if args[0].B != args[1].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_str":
		if len(args) != 2 || args[0].K != VString || args[1].K != VString {
			return unit(), true, fmt.Errorf("std.testing::assert_eq_str expects (String, String)")
		}
		if args[0].S != args[1].S {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::fail":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("std.testing::fail expects (String)")
		}
		return unit(), true, fmt.Errorf("%s", args[0].S)
	default:
		return unit(), false, nil
	}
}

func (rt *Runtime) pushFrame() { rt.stack = append(rt.stack, map[string]Value{}) }
func (rt *Runtime) popFrame()  { rt.stack = rt.stack[:len(rt.stack)-1] }
func (rt *Runtime) frame() map[string]Value {
	return rt.stack[len(rt.stack)-1]
}

func (rt *Runtime) lookup(name string) (map[string]Value, bool) {
	for i := len(rt.stack) - 1; i >= 0; i-- {
		if _, ok := rt.stack[i][name]; ok {
			return rt.stack[i], true
		}
	}
	return nil, false
}

func (rt *Runtime) lookupValue(name string) (Value, bool) {
	if fr, ok := rt.lookup(name); ok {
		return fr[name], true
	}
	return unit(), false
}
