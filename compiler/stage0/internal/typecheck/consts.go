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
		c.constSigs[qname] = ConstSig{Vis: cd.Vis, Pub: cd.Pub, OwnerPkg: pkg, OwnerMod: mod, Ty: ty}
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
	v, vty := c.evalConstExprWithExpected(cd.Expr, cd.Span.File, declTy)
	if v.K == ConstBad || vty.K == TyBad || declTy.K == TyBad {
		c.constBusy[qname] = false
		c.constVals[qname] = ConstValue{K: ConstBad}
		return ConstValue{K: ConstBad}
	}

	// Coerce untyped int to declared int type.
	if vty.K == TyUntypedInt && isIntType(declTy) {
		min, max, ok := intMinMax(declTy)
		if !ok {
			c.errorAt(cd.Span, "internal error: bad int type for const coercion")
			v = ConstValue{K: ConstBad}
			vty = Type{K: TyBad}
		}
		if isUnsignedIntType(declTy) {
			if v.I64 < 0 || uint64(v.I64) > max {
				c.errorAt(cd.Span, "const integer out of range for "+declTy.String())
				v = ConstValue{K: ConstBad}
			}
		} else {
			if v.I64 < min || v.I64 > int64(max) {
				c.errorAt(cd.Span, "const integer out of range for "+declTy.String())
				v = ConstValue{K: ConstBad}
			}
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
			if !c.canAccess(file, sig.OwnerPkg, sig.OwnerMod, sig.Vis) {
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
	if !c.canAccess(file, sig.OwnerPkg, sig.OwnerMod, sig.Vis) {
		c.errorAt(s, "const is private: "+q)
		return "", ConstSig{}, ConstValue{K: ConstBad}, false
	}
	return q, sig, c.evalConst(q, file), true
}

func constTruncBits(u uint64, w int) uint64 {
	if w <= 0 || w >= 64 {
		return u
	}
	return u & ((uint64(1) << uint64(w)) - 1)
}

func constBitWidth(t Type) int {
	if t.K == TyUntypedInt {
		return 64
	}
	return intBitWidth(t)
}

func constIntSigned(bits uint64, t Type) int64 {
	bt := stripRange(t)
	w := constBitWidth(bt)
	bits = constTruncBits(bits, w)
	switch bt.K {
	case TyI8:
		return int64(int8(bits))
	case TyI16:
		return int64(int16(bits))
	case TyI32:
		return int64(int32(bits))
	default:
		return int64(bits)
	}
}

func constIntMinSigned(t Type) int64 {
	switch stripRange(t).K {
	case TyI8:
		return -128
	case TyI16:
		return -32768
	case TyI32:
		return -2147483648
	default:
		return -9223372036854775808
	}
}

func constUintBitsForType(v int64, t Type) (uint64, bool) {
	bt := stripRange(t)
	if bt.K == TyUntypedInt || isSignedIntType(bt) {
		return constTruncBits(uint64(v), constBitWidth(bt)), true
	}
	if !isUnsignedIntType(bt) {
		return 0, false
	}
	return constTruncBits(uint64(v), constBitWidth(bt)), true
}

func constIntValueFromBits(bits uint64, t Type) (int64, bool) {
	bt := stripRange(t)
	if bt.K == TyUntypedInt || isSignedIntType(bt) {
		return constIntSigned(bits, bt), true
	}
	if !isUnsignedIntType(bt) {
		return 0, false
	}
	u := constTruncBits(bits, constBitWidth(bt))
	return int64(u), true
}

func constCastIntChecked(v int64, from, to Type) (int64, bool) {
	from = stripRange(from)
	to = stripRange(to)
	if !isIntType(from) || !isIntType(to) {
		return 0, false
	}

	fromBits := constTruncBits(uint64(v), constBitWidth(from))
	toW := constBitWidth(to)
	toMin, toMax, ok := intMinMax(to)
	if !ok {
		return 0, false
	}

	if from.K == TyUntypedInt || isSignedIntType(from) {
		x := constIntSigned(fromBits, from)
		if isSignedIntType(to) {
			if x < toMin || x > int64(toMax) {
				return 0, false
			}
			return int64(constTruncBits(uint64(x), toW)), true
		}
		if x < 0 || uint64(x) > toMax {
			return 0, false
		}
		return int64(constTruncBits(uint64(x), toW)), true
	}

	u := fromBits
	if isSignedIntType(to) {
		if u > uint64(int64(toMax)) {
			return 0, false
		}
		return int64(constTruncBits(u, toW)), true
	}
	if u > toMax {
		return 0, false
	}
	return int64(constTruncBits(u, toW)), true
}

func constRangeContains(v int64, rng Type) bool {
	if rng.K != TyRange || rng.Base == nil {
		return false
	}
	base := stripRange(*rng.Base)
	if !isIntType(base) {
		return false
	}
	bits := constTruncBits(uint64(v), constBitWidth(base))
	if isSignedIntType(base) {
		x := constIntSigned(bits, base)
		return x >= rng.Lo && x <= rng.Hi
	}
	if rng.Lo < 0 || rng.Hi < 0 {
		return false
	}
	u := bits
	return u >= uint64(rng.Lo) && u <= uint64(rng.Hi)
}

func constExpectedScalarInt(expected Type) (Type, bool) {
	if expected.K == TyBad || expected.K == TyRange {
		return Type{K: TyBad}, false
	}
	bt := stripRange(expected)
	if !isIntType(bt) {
		return Type{K: TyBad}, false
	}
	return bt, true
}

func constIsIntOrUntyped(t Type) bool {
	bt := stripRange(t)
	return bt.K == TyUntypedInt || isIntType(bt)
}

func constResolveIntType(expected, left, right Type) (Type, bool) {
	want, hasWant := constExpectedScalarInt(expected)
	l := stripRange(left)
	r := stripRange(right)
	if !constIsIntOrUntyped(l) || !constIsIntOrUntyped(r) {
		return Type{K: TyBad}, false
	}
	if l.K == TyUntypedInt && r.K == TyUntypedInt {
		if hasWant {
			return want, true
		}
		return Type{K: TyI64}, true
	}
	if l.K == TyUntypedInt {
		if hasWant {
			if !sameType(want, r) {
				return Type{K: TyBad}, false
			}
			return want, true
		}
		return r, true
	}
	if r.K == TyUntypedInt {
		if hasWant {
			if !sameType(want, l) {
				return Type{K: TyBad}, false
			}
			return want, true
		}
		return l, true
	}
	if !sameType(l, r) {
		return Type{K: TyBad}, false
	}
	if hasWant && !sameType(want, l) {
		return Type{K: TyBad}, false
	}
	return l, true
}

func (c *checker) evalConstExpr(ex ast.Expr, file *source.File) (ConstValue, Type) {
	return c.evalConstExprWithExpected(ex, file, Type{K: TyBad})
}

func (c *checker) evalConstExprWithExpected(ex ast.Expr, file *source.File, expected Type) (ConstValue, Type) {
	switch e := ex.(type) {
	case *ast.IntLit:
		u, err := strconv.ParseUint(e.Text, 10, 64)
		if err != nil {
			c.errorAt(e.S, "invalid integer literal")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		if want, ok := constExpectedScalarInt(expected); ok {
			min, max, ok2 := intMinMax(want)
			if !ok2 {
				c.errorAt(e.S, "const expression: bad int type")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			if isUnsignedIntType(want) {
				if u > max {
					c.errorAt(e.S, "const integer out of range for "+want.String())
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				return ConstValue{K: ConstInt, I64: int64(u)}, want
			}
			if u > max || int64(u) < min {
				c.errorAt(e.S, "const integer out of range for "+want.String())
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: int64(u)}, want
		}
		if u > uint64(^uint64(0)>>1) {
			c.errorAt(e.S, "invalid integer literal")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		return ConstValue{K: ConstInt, I64: int64(u)}, Type{K: TyUntypedInt}
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
		if e.Op == "!" {
			v, ty := c.evalConstExprWithExpected(e.Expr, file, Type{K: TyBool})
			if v.K == ConstBad || ty.K != TyBool || v.K != ConstBool {
				c.errorAt(e.S, "const expression: unary ! expects bool")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstBool, B: !v.B}, Type{K: TyBool}
		}
		v, ty := c.evalConstExprWithExpected(e.Expr, file, Type{K: TyBad})
		if v.K == ConstBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		switch e.Op {
		case "-":
			outTy, hasOutTy := constExpectedScalarInt(expected)
			if !hasOutTy {
				outTy = stripRange(ty)
			}
			if outTy.K == TyUntypedInt {
				return ConstValue{K: ConstInt, I64: -v.I64}, Type{K: TyUntypedInt}
			}
			if !isIntType(outTy) || !isSignedIntType(outTy) {
				c.errorAt(e.S, "const expression: unary - expects int")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			bits, okBits := constUintBitsForType(v.I64, outTy)
			if !okBits {
				c.errorAt(e.S, "const expression: const u64 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			w := constBitWidth(outTy)
			negBits := constTruncBits(uint64(-constIntSigned(bits, outTy)), w)
			out, okOut := constIntValueFromBits(negBits, outTy)
			if !okOut {
				c.errorAt(e.S, "const expression: const u64 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: out}, outTy
		default:
			c.errorAt(e.S, "const expression: unsupported unary op: "+e.Op)
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
	case *ast.AsExpr:
		// const casts (stage0 v0): int <-> int, and `@range(..) T`.
		to := c.typeFromAstInFile(e.Ty, file)
		toBase := stripRange(to)
		if !isIntType(toBase) {
			c.errorAt(e.S, "const expression: cast target must be int or @range(..) int")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		evalExpected := Type{K: TyBad}
		if _, isLit := e.Expr.(*ast.IntLit); isLit && isUnsignedIntType(toBase) {
			evalExpected = toBase
		}
		v, ty := c.evalConstExprWithExpected(e.Expr, file, evalExpected)
		if v.K == ConstBad || ty.K == TyBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		// Only integer consts are supported.
		fromBase := stripRange(ty)
		if fromBase.K != TyUntypedInt && !isIntType(fromBase) {
			c.errorAt(e.S, "const expression: cast expects int")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}

		// Current const int payload always lives in I64; treat untyped as i64.
		if fromBase.K == TyUntypedInt {
			fromBase = Type{K: TyI64}
		}

		if _, _, ok := intMinMax(toBase); !ok {
			c.errorAt(e.S, "const expression: bad int type in cast")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		msg := "const expression: int cast overflow"
		if fromBase.K == TyI64 && toBase.K == TyI32 {
			msg = "const expression: i64 to i32 overflow"
		}

		out, okCast := constCastIntChecked(v.I64, fromBase, toBase)
		if !okCast {
			c.errorAt(e.S, msg)
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}

		// Range bounds check (if any).
		if to.K == TyRange {
			if !constRangeContains(out, to) {
				c.errorAt(e.S, "const expression: range check failed")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: out}, to
		}
		return ConstValue{K: ConstInt, I64: out}, toBase
	case *ast.BinaryExpr:
		// Short-circuit for && and ||.
		if e.Op == "&&" || e.Op == "||" {
			lv, lty := c.evalConstExprWithExpected(e.Left, file, Type{K: TyBool})
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
			rv, rty := c.evalConstExprWithExpected(e.Right, file, Type{K: TyBool})
			if rv.K == ConstBad || rty.K != TyBool {
				c.errorAt(e.S, "const expression: &&/|| expects bool")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			if e.Op == "&&" {
				return ConstValue{K: ConstBool, B: lv.B && rv.B}, Type{K: TyBool}
			}
			return ConstValue{K: ConstBool, B: lv.B || rv.B}, Type{K: TyBool}
		}

		want, hasWant := constExpectedScalarInt(expected)
		leftExpected := Type{K: TyBad}
		if hasWant {
			leftExpected = want
		}
		lv, lty := c.evalConstExprWithExpected(e.Left, file, leftExpected)
		rightExpected := Type{K: TyBad}
		lb := stripRange(lty)
		if isIntType(lb) {
			rightExpected = lb
		} else if hasWant {
			rightExpected = want
		}
		rv, rty := c.evalConstExprWithExpected(e.Right, file, rightExpected)
		if lv.K == ConstBad || rv.K == ConstBad || lty.K == TyBad || rty.K == TyBad {
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}

		// int ops
		if e.Op == "+" || e.Op == "-" || e.Op == "*" || e.Op == "/" || e.Op == "%" || e.Op == "&" || e.Op == "|" || e.Op == "^" || e.Op == "<<" || e.Op == ">>" {
			if !constIsIntOrUntyped(lty) || !constIsIntOrUntyped(rty) {
				c.errorAt(e.S, "const expression: arithmetic expects ints")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			outTy, okTy := constResolveIntType(expected, lty, rty)
			if !okTy {
				c.errorAt(e.S, "const expression: arithmetic expects same integer type")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			w := constBitWidth(outTy)
			ua, okA := constUintBitsForType(lv.I64, outTy)
			ub, okB := constUintBitsForType(rv.I64, outTy)
			if !okA || !okB {
				c.errorAt(e.S, "const expression: const u64 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			var outBits uint64
			switch e.Op {
			case "+":
				outBits = constTruncBits(ua+ub, w)
			case "-":
				outBits = constTruncBits(ua-ub, w)
			case "*":
				outBits = constTruncBits(ua*ub, w)
			case "/":
				if ub == 0 {
					c.errorAt(e.S, "const expression: division by zero")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				if isSignedIntType(outTy) || outTy.K == TyUntypedInt {
					a := constIntSigned(ua, outTy)
					b := constIntSigned(ub, outTy)
					if b == -1 && a == constIntMinSigned(outTy) {
						c.errorAt(e.S, "const expression: division overflow")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(uint64(a/b), w)
				} else {
					outBits = constTruncBits(ua/ub, w)
				}
			case "%":
				if ub == 0 {
					c.errorAt(e.S, "const expression: division by zero")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				if isSignedIntType(outTy) || outTy.K == TyUntypedInt {
					a := constIntSigned(ua, outTy)
					b := constIntSigned(ub, outTy)
					if b == -1 && a == constIntMinSigned(outTy) {
						c.errorAt(e.S, "const expression: division overflow")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(uint64(a%b), w)
				} else {
					outBits = constTruncBits(ua%ub, w)
				}
			case "&":
				outBits = constTruncBits(ua&ub, w)
			case "|":
				outBits = constTruncBits(ua|ub, w)
			case "^":
				outBits = constTruncBits(ua^ub, w)
			case "<<":
				if isSignedIntType(outTy) || outTy.K == TyUntypedInt {
					n := constIntSigned(ub, outTy)
					if n < 0 || n >= int64(w) {
						c.errorAt(e.S, "const expression: shift count out of range")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(ua<<uint(n), w)
				} else {
					if ub >= uint64(w) {
						c.errorAt(e.S, "const expression: shift count out of range")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(ua<<uint(ub), w)
				}
			case ">>":
				if isSignedIntType(outTy) || outTy.K == TyUntypedInt {
					n := constIntSigned(ub, outTy)
					if n < 0 || n >= int64(w) {
						c.errorAt(e.S, "const expression: shift count out of range")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(uint64(constIntSigned(ua, outTy)>>uint(n)), w)
				} else {
					if ub >= uint64(w) {
						c.errorAt(e.S, "const expression: shift count out of range")
						return ConstValue{K: ConstBad}, Type{K: TyBad}
					}
					outBits = constTruncBits(ua>>uint(ub), w)
				}
			}
			out, okOut := constIntValueFromBits(outBits, outTy)
			if !okOut {
				c.errorAt(e.S, "const expression: const u64 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			return ConstValue{K: ConstInt, I64: out}, outTy
		}

		// comparisons
		if e.Op == "<" || e.Op == "<=" || e.Op == ">" || e.Op == ">=" {
			if !constIsIntOrUntyped(lty) || !constIsIntOrUntyped(rty) {
				c.errorAt(e.S, "const expression: comparison expects ints")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			cmpTy, okTy := constResolveIntType(expected, lty, rty)
			if !okTy {
				c.errorAt(e.S, "const expression: comparison expects same integer type")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			ua, okA := constUintBitsForType(lv.I64, cmpTy)
			ub, okB := constUintBitsForType(rv.I64, cmpTy)
			if !okA || !okB {
				c.errorAt(e.S, "const expression: const u64 overflow")
				return ConstValue{K: ConstBad}, Type{K: TyBad}
			}
			var b bool
			if isSignedIntType(cmpTy) || cmpTy.K == TyUntypedInt {
				a := constIntSigned(ua, cmpTy)
				r := constIntSigned(ub, cmpTy)
				switch e.Op {
				case "<":
					b = a < r
				case "<=":
					b = a <= r
				case ">":
					b = a > r
				default:
					b = a >= r
				}
			} else {
				switch e.Op {
				case "<":
					b = ua < ub
				case "<=":
					b = ua <= ub
				case ">":
					b = ua > ub
				default:
					b = ua >= ub
				}
			}
			return ConstValue{K: ConstBool, B: b}, Type{K: TyBool}
		}

		// equality
		if e.Op == "==" || e.Op == "!=" {
			var b bool
			switch {
			case constIsIntOrUntyped(lty) && constIsIntOrUntyped(rty):
				eqTy, okTy := constResolveIntType(Type{K: TyBad}, lty, rty)
				if !okTy {
					c.errorAt(e.S, "const expression: ==/!= type mismatch")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				ua, okA := constUintBitsForType(lv.I64, eqTy)
				ub, okB := constUintBitsForType(rv.I64, eqTy)
				if !okA || !okB {
					c.errorAt(e.S, "const expression: const u64 overflow")
					return ConstValue{K: ConstBad}, Type{K: TyBad}
				}
				b = ua == ub
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
		cv, cty := c.evalConstExprWithExpected(e.Cond, file, Type{K: TyBool})
		if cv.K == ConstBad || cty.K != TyBool {
			c.errorAt(e.S, "const expression: if cond must be bool")
			return ConstValue{K: ConstBad}, Type{K: TyBad}
		}
		if cv.B {
			return c.evalConstExprWithExpected(e.Then, file, expected)
		}
		return c.evalConstExprWithExpected(e.Else, file, expected)
	default:
		c.errorAt(ex.Span(), "const expression: unsupported form")
		return ConstValue{K: ConstBad}, Type{K: TyBad}
	}
}
