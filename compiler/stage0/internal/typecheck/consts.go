package typecheck

import (
	"fmt"
	"strconv"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

func (c *checker) collectConstSigs() {
	for _, cd := range c.prog.Consts {
		if cd == nil || cd.Span.File == nil {
			continue
		}
		if strings.HasPrefix(cd.Name, "__") {
			c.errorAt(cd.Span, "reserved const name: "+cd.Name)
			continue
		}
		qname := names.QualifyFunc(cd.Span.File.Name, cd.Name)
		if _, exists := c.constSigs[qname]; exists {
			c.errorAt(cd.Span, "duplicate const: "+qname)
			continue
		}
		pkg, mod, _ := names.SplitOwnerAndModule(cd.Span.File.Name)
		ty := c.typeFromAstInFile(cd.Type, cd.Span.File)
		c.constSigs[qname] = ConstSig{Pub: cd.Pub, OwnerPkg: pkg, OwnerMod: mod, Ty: ty}
		c.constDecls[qname] = cd
	}
}

func (c *checker) evalAllConsts() {
	for q := range c.constSigs {
		cd := c.constDecls[q]
		var at *source.File
		if cd != nil {
			at = cd.Span.File
		}
		_ = c.evalConst(q, at)
	}
}

func (c *checker) evalConst(qname string, at *source.File) ConstValue {
	if v, ok := c.constVals[qname]; ok {
		return v
	}
	if c.constBusy[qname] {
		c.errorAt(source.Span{File: at}, "const cycle: "+qname)
		return ConstValue{K: ConstBad}
	}
	cd := c.constDecls[qname]
	if cd == nil {
		c.errorAt(source.Span{File: at}, "internal error: missing const decl: "+qname)
		return ConstValue{K: ConstBad}
	}
	c.constBusy[qname] = true
	declTy := c.constSigs[qname].Ty
	v, vty := c.evalConstExpr(cd.Expr, cd.Span.File)
	if v.K == ConstBad || vty.K == TyBad || declTy.K == TyBad {
		c.constBusy[qname] = false
		c.constVals[qname] = ConstValue{K: ConstBad}
		return ConstValue{K: ConstBad}
	}

	// Coerce untyped int to declared int type.
	if vty.K == TyUntypedInt && (declTy.K == TyI32 || declTy.K == TyI64) {
		if declTy.K == TyI32 && (v.I64 < -2147483648 || v.I64 > 2147483647) {
			c.errorAt(cd.Span, "const integer out of range for i32")
			v = ConstValue{K: ConstBad}
		}
		vty = declTy
	}

	if !sameType(declTy, vty) {
		c.errorAt(cd.Span, fmt.Sprintf("const init type mismatch: expected %s, got %s", declTy.String(), vty.String()))
		v = ConstValue{K: ConstBad}
	}

	c.constBusy[qname] = false
	c.constVals[qname] = v
	return v
}

func (c *checker) lookupConstName(file *source.File, name string, s source.Span) (qname string, sig ConstSig, val ConstValue, ok bool) {
	if file == nil {
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	// Named import.
	if m := c.namedConsts[file]; m != nil {
		if tgt := m[name]; tgt != "" {
			sig, ok2 := c.constSigs[tgt]
			if !ok2 {
				c.errorAt(s, "unknown const: "+tgt)
				return "", ConstSig{}, ConstValue{K: ConstBad}, false
			}
			if !c.canAccess(file, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
				c.errorAt(s, "const is private: "+tgt)
				return "", ConstSig{}, ConstValue{K: ConstBad}, false
			}
			return tgt, sig, c.evalConst(tgt, file), true
		}
	}
	// Local module const only (no root-module fallback in v0).
	q := names.QualifyFunc(file.Name, name)
	sig, ok2 := c.constSigs[q]
	if !ok2 {
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	return q, sig, c.evalConst(q, file), true
}

func (c *checker) lookupConstByParts(file *source.File, parts []string, s source.Span) (qname string, sig ConstSig, val ConstValue, ok bool) {
	if file == nil || len(parts) < 2 {
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	alias := parts[0]
	extraMods := parts[1 : len(parts)-1]
	name := parts[len(parts)-1]

	m := c.imports[file]
	tgt, ok2 := m[alias]
	if !ok2 {
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	mod := append(append([]string{}, tgt.Mod...), extraMods...)
	q := names.QualifyParts(tgt.Pkg, mod, name)
	sig, ok3 := c.constSigs[q]
	if !ok3 {
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	if !c.canAccess(file, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
		c.errorAt(s, "const is private: "+q)
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	return q, sig, c.evalConst(q, file), true
}

func (c *checker) evalConstExpr(ex ast.Expr, file *source.File) (ConstValue, Type) {
	switch e := ex.(type) {
	case *ast.IntLit:
		v, err := strconv.ParseInt(e.Text, 10, 64)
		if err != nil {
			c.errorAt(e.S, "invalid integer literal")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		return ConstValue{K: ConstInt, I64: v}, Type{K: TyUntypedInt}
	case *ast.BoolLit:
		return ConstValue{K: ConstBool, B: e.Value}, Type{K: TyBool}
	case *ast.StringLit:
		s, err := strconv.Unquote(e.Text)
		if err != nil {
			c.errorAt(e.S, "invalid string literal")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		return ConstValue{K: ConstStr, S: s}, Type{K: TyString}
	case *ast.IdentExpr:
		q, sig, v, ok := c.lookupConstName(file, e.Name, e.S)
		_ = q
		if !ok {
			c.errorAt(e.S, "const expression: unknown identifier: "+e.Name)
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		if v.K == ConstBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		return v, sig.Ty
	case *ast.MemberExpr:
		parts, ok := calleeParts(ex)
		if !ok {
			c.errorAt(e.S, "const expression: unsupported member")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		_, sig, v, ok2 := c.lookupConstByParts(file, parts, e.S)
		if !ok2 {
			c.errorAt(e.S, "const expression: unknown const")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		if v.K == ConstBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		return v, sig.Ty
	case *ast.UnaryExpr:
		v, ty := c.evalConstExpr(e.Expr, file)
		if v.K == ConstBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		switch e.Op {
		case "-":
			if ty.K != TyUntypedInt && ty.K != TyI32 && ty.K != TyI64 {
				c.errorAt(e.S, "const expression: unary - expects int")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: -v.I64}, Type{K: TyUntypedInt}
		case "!":
			if ty.K != TyBool || v.K != ConstBool {
				c.errorAt(e.S, "const expression: unary ! expects bool")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstBool, B: !v.B}, Type{K: TyBool}
		default:
			c.errorAt(e.S, "const expression: unsupported unary op: "+e.Op)
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
	case *ast.AsExpr:
		// const casts (stage0 v0): i32 <-> i64.
		v, ty := c.evalConstExpr(e.Expr, file)
		if v.K == ConstBad || ty.K == TyBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		to := c.typeFromAstInFile(e.Ty, file)
		toBase := stripRange(to)
		if toBase.K != TyI32 && toBase.K != TyI64 {
			c.errorAt(e.S, "const expression: cast target must be i32/i64 or @range(..) i32/i64")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		// Only integer consts are supported.
		if ty.K != TyUntypedInt && ty.K != TyI32 && ty.K != TyI64 {
			c.errorAt(e.S, "const expression: cast expects int")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		// Current const int payload always lives in I64.
		if toBase.K == TyI32 {
			if v.I64 < -2147483648 || v.I64 > 2147483647 {
				c.errorAt(e.S, "const expression: i64 to i32 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			// Range bounds check (if any).
			if to.K == TyRange {
				if v.I64 < to.Lo || v.I64 > to.Hi {
					c.errorAt(e.S, "const expression: range check failed")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				return ConstValue{K: ConstInt, I64: v.I64}, to
			}
			return ConstValue{K: ConstInt, I64: v.I64}, Type{K: TyI32}
		}
		if to.K == TyRange {
			if v.I64 < to.Lo || v.I64 > to.Hi {
				c.errorAt(e.S, "const expression: range check failed")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: v.I64}, to
		}
		return ConstValue{K: ConstInt, I64: v.I64}, Type{K: TyI64}
	case *ast.BinaryExpr:
		// Short-circuit for && and ||.
		if e.Op == "&&" || e.Op == "||" {
			lv, lty := c.evalConstExpr(e.Left, file)
			if lv.K == ConstBad || lty.K != TyBool {
				c.errorAt(e.S, "const expression: &&/|| expects bool")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			if e.Op == "&&" && !lv.B {
				return ConstValue{K: ConstBool, B: false}, Type{K: TyBool}
			}
			if e.Op == "||" && lv.B {
				return ConstValue{K: ConstBool, B: true}, Type{K: TyBool}
			}
			rv, rty := c.evalConstExpr(e.Right, file)
			if rv.K == ConstBad || rty.K != TyBool {
				c.errorAt(e.S, "const expression: &&/|| expects bool")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			if e.Op == "&&" {
				return ConstValue{K: ConstBool, B: lv.B && rv.B}, Type{K: TyBool}
			}
			return ConstValue{K: ConstBool, B: lv.B || rv.B}, Type{K: TyBool}
		}

		lv, lty := c.evalConstExpr(e.Left, file)
		rv, rty := c.evalConstExpr(e.Right, file)
		if lv.K == ConstBad || rv.K == ConstBad || lty.K == TyBad || rty.K == TyBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}

		// int ops
		isInt := func(t Type) bool { return t.K == TyUntypedInt || t.K == TyI32 || t.K == TyI64 }
		if e.Op == "+" || e.Op == "-" || e.Op == "*" || e.Op == "/" || e.Op == "%" {
			if !isInt(lty) || !isInt(rty) {
				c.errorAt(e.S, "const expression: arithmetic expects ints")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			var out int64
			switch e.Op {
			case "+":
				out = lv.I64 + rv.I64
			case "-":
				out = lv.I64 - rv.I64
			case "*":
				out = lv.I64 * rv.I64
			case "/":
				if rv.I64 == 0 {
					c.errorAt(e.S, "const expression: division by zero")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				out = lv.I64 / rv.I64
			case "%":
				if rv.I64 == 0 {
					c.errorAt(e.S, "const expression: division by zero")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				out = lv.I64 % rv.I64
			}
			return ConstValue{K: ConstInt, I64: out}, Type{K: TyUntypedInt}
		}

		// comparisons
		if e.Op == "<" || e.Op == "<=" || e.Op == ">" || e.Op == ">=" {
			if !isInt(lty) || !isInt(rty) {
				c.errorAt(e.S, "const expression: comparison expects ints")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			var b bool
			switch e.Op {
			case "<":
				b = lv.I64 < rv.I64
			case "<=":
				b = lv.I64 <= rv.I64
			case ">":
				b = lv.I64 > rv.I64
			default:
				b = lv.I64 >= rv.I64
			}
			return ConstValue{K: ConstBool, B: b}, Type{K: TyBool}
		}

		// equality
		if e.Op == "==" || e.Op == "!=" {
			var b bool
			switch {
			case isInt(lty) && isInt(rty):
				b = lv.I64 == rv.I64
			case lty.K == TyBool && rty.K == TyBool && lv.K == ConstBool && rv.K == ConstBool:
				b = lv.B == rv.B
			case lty.K == TyString && rty.K == TyString && lv.K == ConstStr && rv.K == ConstStr:
				b = lv.S == rv.S
			default:
				c.errorAt(e.S, "const expression: ==/!= type mismatch")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			if e.Op == "!=" {
				b = !b
			}
			return ConstValue{K: ConstBool, B: b}, Type{K: TyBool}
		}

		c.errorAt(e.S, "const expression: unsupported binary op: "+e.Op)
		return ConstValue{K: ConstBad}, Type{K: TyBad}
	case *ast.IfExpr:
		cv, cty := c.evalConstExpr(e.Cond, file)
		if cv.K == ConstBad || cty.K != TyBool {
			c.errorAt(e.S, "const expression: if cond must be bool")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		if cv.B {
			return c.evalConstExpr(e.Then, file)
		}
		return c.evalConstExpr(e.Else, file)
	default:
		c.errorAt(ex.Span(), "const expression: unsupported form")
		return ConstValue{K: ConstBad}, Type{K: TyBad}
	}
}
