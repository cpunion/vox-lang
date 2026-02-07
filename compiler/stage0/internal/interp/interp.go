package interp

import (
	"fmt"
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
)

type Value struct {
	K ValueKind
	I int64
	B bool
	S string
}

func unit() Value { return Value{K: VUnit} }

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
		if strings.HasPrefix(name, "test_") {
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
		rt.frame()[s.Name] = v
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
		fr[s.Name] = v
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
