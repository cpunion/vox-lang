package typecheck

import (
	"fmt"
	"strconv"

	"voxlang/internal/ast"
	"voxlang/internal/names"
)

func (c *checker) checkAll() {
	// Note: stage0 generic functions are monomorphized on demand. We only typecheck
	// concrete (non-generic) functions and instantiated clones.
	for i := 0; i < len(c.prog.Funcs); i++ {
		fn := c.prog.Funcs[i]
		if fn == nil || fn.Span.File == nil {
			continue
		}
		if len(fn.TypeParams) != 0 {
			continue
		}
		c.curFn = fn
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		sig := c.funcSigs[qname]
		c.pushScope()
		for i, p := range fn.Params {
			c.scopeTop()[p.Name] = varInfo{ty: sig.Params[i], mutable: false}
		}
		c.checkBlock(fn.Body, sig.Ret)
		c.popScope()

		c.materializePendingInstantiations()
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
			ann = c.typeFromAstInFile(s.AnnType, c.curFn.Span.File)
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
		final := chooseType(ann, initTy)
		c.scopeTop()[s.Name] = varInfo{ty: final, mutable: s.Mutable}
		c.letTypes[s] = final
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
	case *ast.FieldAssignStmt:
		vi, ok := c.lookupVar(s.Recv)
		if !ok {
			c.errorAt(s.S, "unknown variable: "+s.Recv)
			return
		}
		if !vi.mutable {
			c.errorAt(s.S, "cannot assign to immutable variable: "+s.Recv)
			return
		}
		if vi.ty.K != TyStruct {
			c.errorAt(s.S, "field assignment requires a struct receiver")
			return
		}
		ss, ok := c.structSigs[vi.ty.Name]
		if !ok {
			c.errorAt(s.S, "unknown struct type: "+vi.ty.Name)
			return
		}
		idx, ok := ss.FieldIndex[s.Field]
		if !ok {
			c.errorAt(s.S, "unknown field: "+s.Field)
			return
		}
		want := ss.Fields[idx].Ty
		got := c.checkExpr(s.Expr, want)
		if !sameType(want, got) {
			c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", want.String(), got.String()))
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
			want := expected
			if want.K != TyI32 && want.K != TyI64 {
				want = Type{K: TyI64}
			}
			ty := c.checkExpr(e.Expr, want)
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
			// Stage0 backend only supports equality on a small set of primitives.
			// (Struct/enum/vec equality would need dedicated lowering.)
			switch l.K {
			case TyBad, TyBool, TyI32, TyI64, TyString:
				// ok
			default:
				c.errorAt(e.S, "equality is only supported for bool/i32/i64/String in stage0")
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
		// Intrinsic method calls on values (stage0 subset).
		//
		// We only treat a callee as a value method call when the receiver expression
		// is rooted in a local variable. This avoids stealing syntax from enum
		// constructors like `Option.Some(1)`.
		if me, ok := e.Callee.(*ast.MemberExpr); ok {
			if ty, handled := c.tryIntrinsicMethodCall(ex, e, me); handled {
				return ty
			}
		}

		// Vec constructor: `Vec()` with expected type `Vec[T]`.
		if cal, ok := e.Callee.(*ast.IdentExpr); ok && cal.Name == "Vec" && expected.K == TyVec {
			if len(e.Args) != 0 {
				c.errorAt(e.S, "Vec() expects 0 args")
				return c.setExprType(ex, Type{K: TyBad})
			}
			if expected.Elem == nil || expected.Elem.K == TyBad {
				c.errorAt(e.S, "cannot infer Vec element type")
				return c.setExprType(ex, Type{K: TyBad})
			}
			c.vecCalls[e] = VecCallTarget{Kind: VecCallNew, Elem: *expected.Elem}
			return c.setExprType(ex, expected)
		}

		parts, ok := calleeParts(e.Callee)
		if !ok || len(parts) == 0 {
			c.errorAt(e.S, "callee must be an identifier or member path (stage0)")
			return c.setExprType(ex, Type{K: TyBad})
		}

		// Enum constructor: `Enum.Variant(...)` (including qualified types like `dep.Option.Some(...)`).
		if len(parts) >= 2 {
			alias := parts[0]
				if vi, ok := c.lookupVar(alias); ok {
					// Vec methods on local vars: v.push(...), v.len(), v.get(i)
					if len(parts) == 2 && vi.ty.K == TyVec && vi.ty.Elem != nil {
						method := parts[1]
						switch method {
					case "push":
						if len(e.Args) != 1 {
							c.errorAt(e.S, "Vec.push expects 1 arg")
							return c.setExprType(ex, Type{K: TyBad})
						}
						if !vi.mutable {
							c.errorAt(e.S, "cannot call push on immutable variable: "+alias)
							return c.setExprType(ex, Type{K: TyBad})
						}
						at := c.checkExpr(e.Args[0], *vi.ty.Elem)
						if !sameType(*vi.ty.Elem, at) {
							c.errorAt(e.Args[0].Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", vi.ty.Elem.String(), at.String()))
						}
						c.vecCalls[e] = VecCallTarget{Kind: VecCallPush, RecvName: alias, Elem: *vi.ty.Elem}
						return c.setExprType(ex, Type{K: TyUnit})
					case "len":
						if len(e.Args) != 0 {
							c.errorAt(e.S, "Vec.len expects 0 args")
							return c.setExprType(ex, Type{K: TyBad})
						}
						c.vecCalls[e] = VecCallTarget{Kind: VecCallLen, RecvName: alias, Elem: *vi.ty.Elem}
						return c.setExprType(ex, Type{K: TyI32})
					case "get":
						if len(e.Args) != 1 {
							c.errorAt(e.S, "Vec.get expects 1 arg")
							return c.setExprType(ex, Type{K: TyBad})
						}
						idxTy := c.checkExpr(e.Args[0], Type{K: TyI32})
						if idxTy.K != TyI32 {
							c.errorAt(e.Args[0].Span(), "Vec.get index must be i32")
						}
						c.vecCalls[e] = VecCallTarget{Kind: VecCallGet, RecvName: alias, Elem: *vi.ty.Elem}
						return c.setExprType(ex, *vi.ty.Elem)
					default:
						c.errorAt(e.S, "unknown Vec method: "+method)
						return c.setExprType(ex, Type{K: TyBad})
					}
				}

				// String methods on local vars: s.len(), s.byte_at(i)
				if len(parts) == 2 && vi.ty.K == TyString {
					method := parts[1]
					switch method {
					case "len":
						if len(e.Args) != 0 {
							c.errorAt(e.S, "String.len expects 0 args")
							return c.setExprType(ex, Type{K: TyBad})
						}
						c.strCalls[e] = StrCallTarget{Kind: StrCallLen, RecvName: alias}
						return c.setExprType(ex, Type{K: TyI32})
					case "byte_at":
						if len(e.Args) != 1 {
							c.errorAt(e.S, "String.byte_at expects 1 arg")
							return c.setExprType(ex, Type{K: TyBad})
						}
						idxTy := c.checkExpr(e.Args[0], Type{K: TyI32})
						if idxTy.K != TyI32 {
							c.errorAt(e.Args[0].Span(), "String.byte_at index must be i32")
						}
						c.strCalls[e] = StrCallTarget{Kind: StrCallByteAt, RecvName: alias}
						return c.setExprType(ex, Type{K: TyI32})
					case "slice":
						if len(e.Args) != 2 {
							c.errorAt(e.S, "String.slice expects 2 args")
							return c.setExprType(ex, Type{K: TyBad})
						}
						sTy := c.checkExpr(e.Args[0], Type{K: TyI32})
						eTy := c.checkExpr(e.Args[1], Type{K: TyI32})
						if sTy.K != TyI32 || eTy.K != TyI32 {
							c.errorAt(e.S, "String.slice indices must be i32")
							return c.setExprType(ex, Type{K: TyBad})
						}
						c.strCalls[e] = StrCallTarget{Kind: StrCallSlice, RecvName: alias}
						return c.setExprType(ex, Type{K: TyString})
					default:
						c.errorAt(e.S, "unknown String method: "+method)
						return c.setExprType(ex, Type{K: TyBad})
					}
				}

				c.errorAt(e.S, "member calls on values are not supported yet")
				return c.setExprType(ex, Type{K: TyBad})
			}
			if c.curFn == nil || c.curFn.Span.File == nil {
				c.errorAt(e.S, "internal error: missing file for call resolution")
				return c.setExprType(ex, Type{K: TyBad})
			}
			ety, es, found := c.findEnumByParts(c.curFn.Span.File, parts[:len(parts)-1])
			if found {
				if !c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
					c.errorAt(e.S, "type is private: "+ety.Name)
					return c.setExprType(ex, Type{K: TyBad})
				}
				varName := parts[len(parts)-1]
				vidx, vok := es.VariantIndex[varName]
				if !vok {
					c.errorAt(e.S, "unknown variant: "+varName)
					return c.setExprType(ex, Type{K: TyBad})
				}
				vs := es.Variants[vidx]
				if len(e.Args) != len(vs.Fields) {
					c.errorAt(e.S, fmt.Sprintf("wrong number of arguments: expected %d, got %d", len(vs.Fields), len(e.Args)))
					return c.setExprType(ex, Type{K: TyBad})
				}
				for i, a := range e.Args {
					at := c.checkExpr(a, vs.Fields[i])
					if !sameType(vs.Fields[i], at) {
						c.errorAt(a.Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", vs.Fields[i].String(), at.String()))
					}
				}
				fields := make([]Type, 0, len(vs.Fields))
				fields = append(fields, vs.Fields...)
				c.enumCtors[e] = EnumCtorTarget{Enum: ety, Variant: varName, Tag: vidx, Fields: fields}
				return c.setExprType(ex, ety)
			}
		}

		target := ""
		if len(parts) == 1 {
			name := parts[0]
			pkg, mod, _ := names.SplitOwnerAndModule(c.curFn.Span.File.Name)
			// 1) current module
			q := names.QualifyParts(pkg, mod, name)
			if _, ok := c.funcSigs[q]; ok {
				target = q
			} else {
				// 2) named imports for this file
				if c.curFn.Span.File != nil {
					if m := c.namedFuncs[c.curFn.Span.File]; m != nil {
						if tgt, ok := m[name]; ok {
							target = tgt
						}
					}
				}
				// 3) fallback: root module of the same package
				if target == "" {
					q2 := names.QualifyParts(pkg, nil, name)
					if _, ok := c.funcSigs[q2]; ok {
						target = q2
					}
				}
				// 4) global root (builtins live here)
				if target == "" {
					if _, ok := c.funcSigs[name]; ok {
						target = name
					}
				}
				// 5) implicit prelude: std/prelude
				if target == "" {
					q3 := names.QualifyParts("", []string{"std", "prelude"}, name)
					if _, ok := c.funcSigs[q3]; ok {
						target = q3
					}
				}
			}
		} else {
			// Qualified call: first segment must be an imported alias.
			qualParts := parts[:len(parts)-1]
			member := parts[len(parts)-1]
			alias := qualParts[0]
			extraMods := qualParts[1:]

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
		// Generic function instantiation (stage0 minimal): monomorphize on demand.
		if instTarget, instSig, ok := c.maybeInstantiateCall(e, target, sig, expected); ok {
			target = instTarget
			sig = instSig
		} else if len(e.TypeArgs) > 0 {
			c.errorAt(e.S, "type arguments provided for non-generic function: "+target)
			return c.setExprType(ex, Type{K: TyBad})
		}
		if c.curFn != nil && c.curFn.Span.File != nil && !c.canAccess(c.curFn.Span.File, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
			c.errorAt(e.S, "function is private: "+target)
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
	case *ast.MemberExpr:
		// Unit enum variant: `Enum.Variant` (including qualified enum types).
		if c.curFn != nil && c.curFn.Span.File != nil {
			parts, ok := calleeParts(ex)
			if ok && len(parts) >= 2 {
				alias := parts[0]
				if _, ok := c.lookupVar(alias); !ok {
					ety, es, found := c.findEnumByParts(c.curFn.Span.File, parts[:len(parts)-1])
					if found && !c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
						c.errorAt(e.S, "type is private: "+ety.Name)
						return c.setExprType(ex, Type{K: TyBad})
					}
					if found && c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
						vname := parts[len(parts)-1]
						vidx, vok := es.VariantIndex[vname]
						if vok && len(es.Variants[vidx].Fields) == 0 {
							c.enumUnits[e] = EnumCtorTarget{Enum: ety, Variant: vname, Tag: vidx}
							return c.setExprType(ex, ety)
						}
					}
				}
			}
		}

		recvTy := c.checkExpr(e.Recv, Type{K: TyBad})
		if recvTy.K != TyStruct {
			c.errorAt(e.S, "member access requires a struct receiver")
			return c.setExprType(ex, Type{K: TyBad})
		}
		ss, ok := c.structSigs[recvTy.Name]
		if !ok {
			c.errorAt(e.S, "unknown struct type: "+recvTy.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		idx, ok := ss.FieldIndex[e.Name]
		if !ok {
			c.errorAt(e.S, "unknown field: "+e.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		if c.curFn != nil && c.curFn.Span.File != nil && !c.isSameModule(c.curFn.Span.File, ss.OwnerPkg, ss.OwnerMod) && !ss.Fields[idx].Pub {
			c.errorAt(e.S, "field is private: "+recvTy.Name+"."+e.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		return c.setExprType(ex, ss.Fields[idx].Ty)
	case *ast.StructLitExpr:
		if e.S.File == nil {
			c.errorAt(e.S, "internal error: missing file for struct literal")
			return c.setExprType(ex, Type{K: TyBad})
		}
		sty, ss, ok := c.resolveStructByParts(e.S.File, e.TypeParts, e.S)
		if !ok {
			return c.setExprType(ex, Type{K: TyBad})
		}
		if !c.isSameModule(e.S.File, ss.OwnerPkg, ss.OwnerMod) {
			for _, f := range ss.Fields {
				if !f.Pub {
					c.errorAt(e.S, "cannot construct struct "+sty.Name+": field "+f.Name+" is private")
					return c.setExprType(ex, Type{K: TyBad})
				}
			}
		}
		seen := map[string]bool{}
		for _, init := range e.Inits {
			if seen[init.Name] {
				c.errorAt(init.Span, "duplicate field init: "+init.Name)
				continue
			}
			seen[init.Name] = true
			idx, ok := ss.FieldIndex[init.Name]
			if !ok {
				c.errorAt(init.Span, "unknown field: "+init.Name)
				continue
			}
			if !c.isSameModule(e.S.File, ss.OwnerPkg, ss.OwnerMod) && !ss.Fields[idx].Pub {
				c.errorAt(init.Span, "field is private: "+sty.Name+"."+init.Name)
				continue
			}
			want := ss.Fields[idx].Ty
			got := c.checkExpr(init.Expr, want)
			if !sameType(want, got) {
				c.errorAt(init.Expr.Span(), fmt.Sprintf("field %s type mismatch: expected %s, got %s", init.Name, want.String(), got.String()))
			}
		}
		for _, f := range ss.Fields {
			if !seen[f.Name] {
				c.errorAt(e.S, "missing field init: "+f.Name)
			}
		}
		return c.setExprType(ex, sty)
	case *ast.MatchExpr:
		scrutTy := c.checkExpr(e.Scrutinee, Type{K: TyBad})
		if scrutTy.K != TyEnum {
			c.errorAt(e.S, "match scrutinee must be an enum (stage0)")
			return c.setExprType(ex, Type{K: TyBad})
		}
		esig, ok := c.enumSigs[scrutTy.Name]
		if !ok {
			c.errorAt(e.S, "unknown enum type: "+scrutTy.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}

		resultTy := expected
		seenVariants := map[string]bool{}
		hasWild := false

		for _, arm := range e.Arms {
			c.pushScope()
			switch p := arm.Pat.(type) {
			case *ast.WildPat:
				hasWild = true
			case *ast.VariantPat:
				if c.curFn == nil || c.curFn.Span.File == nil {
					c.errorAt(arm.S, "internal error: missing file for match")
					break
				}
				pty, psig, ok := c.resolveEnumByParts(c.curFn.Span.File, p.TypeParts, p.S)
				if ok && !sameType(scrutTy, pty) {
					c.errorAt(p.S, "pattern enum type does not match scrutinee")
				}
				vidx, vok := psig.VariantIndex[p.Variant]
				if !vok {
					c.errorAt(p.S, "unknown variant: "+p.Variant)
					break
				}
				if seenVariants[p.Variant] {
					c.errorAt(p.S, "duplicate match arm for variant: "+p.Variant)
				}
				seenVariants[p.Variant] = true
				v := psig.Variants[vidx]
				if len(p.Binds) != len(v.Fields) {
					c.errorAt(p.S, fmt.Sprintf("wrong number of binders: expected %d, got %d", len(v.Fields), len(p.Binds)))
				}
				for i := 0; i < len(p.Binds) && i < len(v.Fields); i++ {
					c.scopeTop()[p.Binds[i]] = varInfo{ty: v.Fields[i], mutable: false}
				}
			default:
				c.errorAt(arm.S, "unsupported pattern (stage0)")
			}

			armTy := c.checkExpr(arm.Expr, resultTy)
			if resultTy.K == TyBad {
				if armTy.K == TyUntypedInt {
					resultTy = Type{K: TyI64}
				} else {
					resultTy = armTy
				}
			} else if !sameType(resultTy, armTy) {
				c.errorAt(arm.S, fmt.Sprintf("match arm type mismatch: expected %s, got %s", resultTy.String(), armTy.String()))
			}
			c.popScope()
		}

		if !hasWild {
			for _, v := range esig.Variants {
				if !seenVariants[v.Name] {
					c.errorAt(e.S, "non-exhaustive match, missing variant: "+v.Name)
				}
			}
		}
		return c.setExprType(ex, resultTy)
	default:
		c.errorAt(ex.Span(), "unsupported expression")
		return c.setExprType(ex, Type{K: TyBad})
	}
}

func rootIdentName(ex ast.Expr) (string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return e.Name, true
	case *ast.MemberExpr:
		return rootIdentName(e.Recv)
	default:
		return "", false
	}
}

func (c *checker) tryIntrinsicMethodCall(ex ast.Expr, call *ast.CallExpr, me *ast.MemberExpr) (Type, bool) {
	root, ok := rootIdentName(me.Recv)
	if !ok {
		return Type{K: TyBad}, false
	}
	if _, ok := c.lookupVar(root); !ok {
		// Likely a type/module path (e.g. Enum.Variant(...)); let the normal call resolver handle it.
		return Type{K: TyBad}, false
	}

	recvTy := c.checkExpr(me.Recv, Type{K: TyBad})
	method := me.Name

	// Vec methods.
	if recvTy.K == TyVec && recvTy.Elem != nil {
		// Optimization: if receiver is a simple local variable, keep RecvName to allow
		// direct lowering to slot-based IR for len/get and required for push.
		recvName := ""
		if id, ok := me.Recv.(*ast.IdentExpr); ok {
			recvName = id.Name
		}

		switch method {
		case "push":
			if len(call.Args) != 1 {
				c.errorAt(call.S, "Vec.push expects 1 arg")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			if recvName == "" {
				c.errorAt(call.S, "Vec.push receiver must be a local variable in stage0")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			vi, ok := c.lookupVar(recvName)
			if !ok {
				c.errorAt(call.S, "unknown variable: "+recvName)
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			if !vi.mutable {
				c.errorAt(call.S, "cannot call push on immutable variable: "+recvName)
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			at := c.checkExpr(call.Args[0], *recvTy.Elem)
			if !sameType(*recvTy.Elem, at) {
				c.errorAt(call.Args[0].Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", recvTy.Elem.String(), at.String()))
			}
			c.vecCalls[call] = VecCallTarget{Kind: VecCallPush, RecvName: recvName, Recv: me.Recv, Elem: *recvTy.Elem}
			return c.setExprType(ex, Type{K: TyUnit}), true
		case "len":
			if len(call.Args) != 0 {
				c.errorAt(call.S, "Vec.len expects 0 args")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			c.vecCalls[call] = VecCallTarget{Kind: VecCallLen, RecvName: recvName, Recv: me.Recv, Elem: *recvTy.Elem}
			return c.setExprType(ex, Type{K: TyI32}), true
		case "get":
			if len(call.Args) != 1 {
				c.errorAt(call.S, "Vec.get expects 1 arg")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			idxTy := c.checkExpr(call.Args[0], Type{K: TyI32})
			if idxTy.K != TyI32 {
				c.errorAt(call.Args[0].Span(), "Vec.get index must be i32")
			}
			c.vecCalls[call] = VecCallTarget{Kind: VecCallGet, RecvName: recvName, Recv: me.Recv, Elem: *recvTy.Elem}
			return c.setExprType(ex, *recvTy.Elem), true
		}
	}

	// String methods.
	if recvTy.K == TyString {
		recvName := ""
		if id, ok := me.Recv.(*ast.IdentExpr); ok {
			recvName = id.Name
		}
		switch method {
		case "len":
			if len(call.Args) != 0 {
				c.errorAt(call.S, "String.len expects 0 args")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			c.strCalls[call] = StrCallTarget{Kind: StrCallLen, RecvName: recvName, Recv: me.Recv}
			return c.setExprType(ex, Type{K: TyI32}), true
		case "byte_at":
			if len(call.Args) != 1 {
				c.errorAt(call.S, "String.byte_at expects 1 arg")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			idxTy := c.checkExpr(call.Args[0], Type{K: TyI32})
			if idxTy.K != TyI32 {
				c.errorAt(call.Args[0].Span(), "String.byte_at index must be i32")
			}
			c.strCalls[call] = StrCallTarget{Kind: StrCallByteAt, RecvName: recvName, Recv: me.Recv}
			return c.setExprType(ex, Type{K: TyI32}), true
		case "slice":
			if len(call.Args) != 2 {
				c.errorAt(call.S, "String.slice expects 2 args")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			sTy := c.checkExpr(call.Args[0], Type{K: TyI32})
			eTy := c.checkExpr(call.Args[1], Type{K: TyI32})
			if sTy.K != TyI32 || eTy.K != TyI32 {
				c.errorAt(call.S, "String.slice indices must be i32")
				return c.setExprType(ex, Type{K: TyBad}), true
			}
			c.strCalls[call] = StrCallTarget{Kind: StrCallSlice, RecvName: recvName, Recv: me.Recv}
			return c.setExprType(ex, Type{K: TyString}), true
		}
	}

	return Type{K: TyBad}, false
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

func (c *checker) setExprType(ex ast.Expr, t Type) Type {
	c.exprTypes[ex] = t
	return t
}
