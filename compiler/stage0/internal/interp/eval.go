package interp

import (
	"fmt"
	"strconv"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/typecheck"
)

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
		// Keep runtime semantics aligned with IR generation (Go-like unquoting).
		s, err := strconv.Unquote(e.Text)
		if err != nil {
			return unit(), fmt.Errorf("invalid string literal")
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
	case *ast.AsExpr:
		v, err := rt.evalExpr(e.Expr)
		if err != nil {
			return unit(), err
		}
		from := rt.prog.ExprTypes[e.Expr]
		to := rt.prog.ExprTypes[ex]
		// Stage0 v0: i32 <-> i64.
		if (from.K != typecheck.TyI32 && from.K != typecheck.TyI64) || (to.K != typecheck.TyI32 && to.K != typecheck.TyI64) {
			return unit(), fmt.Errorf("unsupported cast")
		}
		if from.K == to.K {
			return v, nil
		}
		if to.K == typecheck.TyI64 {
			return Value{K: VInt, I: v.I}, nil
		}
		// to i32: bounds check then keep as int
		if v.I < -2147483648 || v.I > 2147483647 {
			return unit(), fmt.Errorf("i64 to i32 overflow")
		}
		return Value{K: VInt, I: v.I}, nil
	case *ast.BinaryExpr:
		l, err := rt.evalExpr(e.Left)
		if err != nil {
			return unit(), err
		}

		// Short-circuit logical ops.
		// These must not evaluate RHS when the result is already determined by LHS.
		if e.Op == "&&" || e.Op == "||" {
			if l.K != VBool {
				return unit(), fmt.Errorf("logical op expects bool")
			}
			if e.Op == "&&" {
				if !l.B {
					return Value{K: VBool, B: false}, nil
				}
				r, err := rt.evalExpr(e.Right)
				if err != nil {
					return unit(), err
				}
				if r.K != VBool {
					return unit(), fmt.Errorf("logical op expects bool")
				}
				return Value{K: VBool, B: r.B}, nil
			}
			// "||"
			if l.B {
				return Value{K: VBool, B: true}, nil
			}
			r, err := rt.evalExpr(e.Right)
			if err != nil {
				return unit(), err
			}
			if r.K != VBool {
				return unit(), fmt.Errorf("logical op expects bool")
			}
			return Value{K: VBool, B: r.B}, nil
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
		default:
			return unit(), fmt.Errorf("unknown op: %s", e.Op)
		}
		return unit(), fmt.Errorf("unreachable")
	case *ast.IfExpr:
		cv, err := rt.evalExpr(e.Cond)
		if err != nil {
			return unit(), err
		}
		if cv.K != VBool {
			return unit(), fmt.Errorf("if condition must be bool")
		}
		if cv.B {
			return rt.evalExpr(e.Then)
		}
		return rt.evalExpr(e.Else)
	case *ast.BlockExpr:
		rt.pushFrame()
		for _, st := range e.Stmts {
			if _, err := rt.evalStmt(st); err != nil {
				rt.popFrame()
				return unit(), err
			}
		}
		if e.Tail == nil {
			rt.popFrame()
			return unit(), nil
		}
		v, err := rt.evalExpr(e.Tail)
		rt.popFrame()
		return v, err
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
				if vc.RecvName != "" {
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
				}

				// Field-place receiver: ident.field
				mem, ok := vc.Recv.(*ast.MemberExpr)
				if !ok {
					return unit(), fmt.Errorf("Vec.push receiver must be a place")
				}
				base, ok := mem.Recv.(*ast.IdentExpr)
				if !ok {
					return unit(), fmt.Errorf("Vec.push receiver must be a direct field of a local variable")
				}
				fr, ok := rt.lookup(base.Name)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", base.Name)
				}
				sv := fr[base.Name]
				if sv.K != VStruct {
					return unit(), fmt.Errorf("Vec.push field receiver must be a struct")
				}
				fv, ok := sv.M[mem.Name]
				if !ok {
					return unit(), fmt.Errorf("unknown field: %s", mem.Name)
				}
				if fv.K != VVec {
					return unit(), fmt.Errorf("Vec.push requires vec receiver")
				}
				fv.A = append(fv.A, cloneValue(v))
				sv.M[mem.Name] = fv
				fr[base.Name] = sv
				return unit(), nil
			case typecheck.VecCallLen:
				var recv Value
				if vc.Recv != nil {
					rv, err := rt.evalExpr(vc.Recv)
					if err != nil {
						return unit(), err
					}
					recv = rv
				} else {
					fr, ok := rt.lookup(vc.RecvName)
					if !ok {
						return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
					}
					recv = fr[vc.RecvName]
				}
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
				var recv Value
				if vc.Recv != nil {
					rv, err := rt.evalExpr(vc.Recv)
					if err != nil {
						return unit(), err
					}
					recv = rv
				} else {
					fr, ok := rt.lookup(vc.RecvName)
					if !ok {
						return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
					}
					recv = fr[vc.RecvName]
				}
				if recv.K != VVec {
					return unit(), fmt.Errorf("Vec.get requires vec receiver")
				}
				if idx < 0 || idx >= len(recv.A) {
					return unit(), fmt.Errorf("Vec.get index out of bounds")
				}
				return cloneValue(recv.A[idx]), nil
			case typecheck.VecCallJoin:
				if len(e.Args) != 1 {
					return unit(), fmt.Errorf("Vec.join expects 1 arg")
				}
				sepV, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				if sepV.K != VString {
					return unit(), fmt.Errorf("Vec.join separator must be string")
				}
				var recv Value
				if vc.Recv != nil {
					rv, err := rt.evalExpr(vc.Recv)
					if err != nil {
						return unit(), err
					}
					recv = rv
				} else {
					fr, ok := rt.lookup(vc.RecvName)
					if !ok {
						return unit(), fmt.Errorf("unknown variable: %s", vc.RecvName)
					}
					recv = fr[vc.RecvName]
				}
				if recv.K != VVec {
					return unit(), fmt.Errorf("Vec.join requires vec receiver")
				}
				// Stage0: Vec.join is only supported for Vec[String].
				var b strings.Builder
				for i, it := range recv.A {
					if it.K != VString {
						return unit(), fmt.Errorf("Vec.join expects Vec[String]")
					}
					if i > 0 {
						b.WriteString(sepV.S)
					}
					b.WriteString(it.S)
				}
				return Value{K: VString, S: b.String()}, nil
			default:
				return unit(), fmt.Errorf("unsupported vec call")
			}
		}

		if sc, ok := rt.prog.StrCalls[e]; ok {
			var recv Value
			if sc.Recv != nil {
				rv, err := rt.evalExpr(sc.Recv)
				if err != nil {
					return unit(), err
				}
				recv = rv
			} else {
				fr, ok := rt.lookup(sc.RecvName)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", sc.RecvName)
				}
				recv = fr[sc.RecvName]
			}
			if recv.K != VString {
				return unit(), fmt.Errorf("String method requires string receiver")
			}
			switch sc.Kind {
			case typecheck.StrCallLen:
				return Value{K: VInt, I: int64(len(recv.S))}, nil
			case typecheck.StrCallByteAt:
				if len(e.Args) != 1 {
					return unit(), fmt.Errorf("String.byte_at expects 1 arg")
				}
				idxV, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				if idxV.K != VInt {
					return unit(), fmt.Errorf("String.byte_at index must be int")
				}
				idx := int(idxV.I)
				if idx < 0 || idx >= len(recv.S) {
					return unit(), fmt.Errorf("String.byte_at index out of bounds")
				}
				return Value{K: VInt, I: int64(recv.S[idx])}, nil
			case typecheck.StrCallSlice:
				if len(e.Args) != 2 {
					return unit(), fmt.Errorf("String.slice expects 2 args")
				}
				sv, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				ev, err := rt.evalExpr(e.Args[1])
				if err != nil {
					return unit(), err
				}
				if sv.K != VInt || ev.K != VInt {
					return unit(), fmt.Errorf("String.slice indices must be int")
				}
				start := int(sv.I)
				end := int(ev.I)
				if start < 0 || end < start || end > len(recv.S) {
					return unit(), fmt.Errorf("String.slice index out of bounds")
				}
				return Value{K: VString, S: recv.S[start:end]}, nil
			case typecheck.StrCallConcat:
				if len(e.Args) != 1 {
					return unit(), fmt.Errorf("String.concat expects 1 arg")
				}
				ov, err := rt.evalExpr(e.Args[0])
				if err != nil {
					return unit(), err
				}
				if ov.K != VString {
					return unit(), fmt.Errorf("String.concat arg must be string")
				}
				return Value{K: VString, S: recv.S + ov.S}, nil
			case typecheck.StrCallEscapeC:
				if len(e.Args) != 0 {
					return unit(), fmt.Errorf("String.escape_c expects 0 args")
				}
				return Value{K: VString, S: escapeC(recv.S)}, nil
			default:
				return unit(), fmt.Errorf("unsupported string call")
			}
		}

		if ts, ok := rt.prog.ToStrCalls[e]; ok {
			var recv Value
			if ts.Recv != nil {
				rv, err := rt.evalExpr(ts.Recv)
				if err != nil {
					return unit(), err
				}
				recv = rv
			} else {
				fr, ok := rt.lookup(ts.RecvName)
				if !ok {
					return unit(), fmt.Errorf("unknown variable: %s", ts.RecvName)
				}
				recv = fr[ts.RecvName]
			}
			switch ts.Kind {
			case typecheck.ToStrI32, typecheck.ToStrI64:
				if recv.K != VInt {
					return unit(), fmt.Errorf("to_string expects int receiver")
				}
				return Value{K: VString, S: strconv.FormatInt(recv.I, 10)}, nil
			case typecheck.ToStrBool:
				if recv.K != VBool {
					return unit(), fmt.Errorf("to_string expects bool receiver")
				}
				if recv.B {
					return Value{K: VString, S: "true"}, nil
				}
				return Value{K: VString, S: "false"}, nil
			default:
				return unit(), fmt.Errorf("unsupported to_string call")
			}
		}

		if ctor, ok := rt.prog.EnumCtors[e]; ok {
			if len(ctor.Fields) == 0 {
				return Value{K: VEnum, E: ctor.Enum.Name, T: ctor.Tag}, nil
			}
			if len(e.Args) != len(ctor.Fields) {
				return unit(), fmt.Errorf("enum constructor expects %d args", len(ctor.Fields))
			}
			payload := make([]Value, 0, len(e.Args))
			for _, a := range e.Args {
				v, err := rt.evalExpr(a)
				if err != nil {
					return unit(), err
				}
				payload = append(payload, cloneValue(v))
			}
			return Value{K: VEnum, E: ctor.Enum.Name, T: ctor.Tag, P: payload}, nil
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
			// Unit enum variant value: `Enum.Variant`.
			if cu, ok := rt.prog.EnumUnitVariants[e]; ok {
				return Value{K: VEnum, E: cu.Enum.Name, T: cu.Tag}, nil
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
		case *ast.DotExpr:
			// Unit enum variant shorthand: `.Variant`.
			if cu, ok := rt.prog.EnumUnitVariants[e]; ok {
				return Value{K: VEnum, E: cu.Enum.Name, T: cu.Tag}, nil
			}
			return unit(), fmt.Errorf("unresolved unit enum variant shorthand")
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
		// Enum metadata is only needed when matching enum variant patterns.
		var es typecheck.EnumSig
		hasEnumSig := false
		if sv.K == VEnum {
			ety := rt.prog.ExprTypes[e.Scrutinee]
			if ety.K != typecheck.TyEnum {
				ety = typecheck.Type{K: typecheck.TyEnum, Name: sv.E}
			}
			sig, ok := rt.prog.EnumSigs[ety.Name]
			if !ok {
				return unit(), fmt.Errorf("unknown enum: %s", ety.Name)
			}
			es = sig
			hasEnumSig = true
		}
		for _, arm := range e.Arms {
			switch p := arm.Pat.(type) {
			case *ast.WildPat:
				return rt.evalExpr(arm.Expr)
			case *ast.BindPat:
				// Bind pattern always matches and binds the scrutinee to Name.
				rt.pushFrame()
				rt.frame()[p.Name] = cloneValue(sv)
				v, err := rt.evalExpr(arm.Expr)
				rt.popFrame()
				return v, err
			case *ast.IntPat:
				if sv.K != VInt {
					continue
				}
				// typechecker guaranteed parseability
				var n int64
				for i := 0; i < len(p.Text); i++ {
					n = n*10 + int64(p.Text[i]-'0')
				}
				if sv.I != n {
					continue
				}
				return rt.evalExpr(arm.Expr)
			case *ast.StrPat:
				if sv.K != VString {
					continue
				}
				s, err := strconv.Unquote(p.Text)
				if err != nil {
					return unit(), fmt.Errorf("invalid string pattern")
				}
				if sv.S != s {
					continue
				}
				return rt.evalExpr(arm.Expr)
			case *ast.VariantPat:
				if sv.K != VEnum {
					continue
				}
				if !hasEnumSig {
					return unit(), fmt.Errorf("internal: missing enum sig for match")
				}
				tag, ok := es.VariantIndex[p.Variant]
				if !ok {
					return unit(), fmt.Errorf("unknown variant: %s", p.Variant)
				}
				if sv.T != tag {
					continue
				}
				rt.pushFrame()
				for i := 0; i < len(p.Binds); i++ {
					if i < len(sv.P) {
						rt.frame()[p.Binds[i]] = cloneValue(sv.P[i])
					} else {
						rt.frame()[p.Binds[i]] = unit()
					}
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
