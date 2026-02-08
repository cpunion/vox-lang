package interp

import (
	"fmt"

	"voxlang/internal/typecheck"
)

func stripRange(t typecheck.Type) typecheck.Type {
	if t.K == typecheck.TyRange && t.Base != nil {
		return *t.Base
	}
	return t
}

func isIntType(t typecheck.Type) bool {
	switch t.K {
	case typecheck.TyI8, typecheck.TyU8, typecheck.TyI32, typecheck.TyU32, typecheck.TyI64, typecheck.TyU64, typecheck.TyUSize, typecheck.TyUntypedInt:
		return true
	default:
		return false
	}
}

func isSignedIntType(t typecheck.Type) bool {
	switch t.K {
	case typecheck.TyI8, typecheck.TyI32, typecheck.TyI64, typecheck.TyUntypedInt:
		return true
	default:
		return false
	}
}

func intBitWidth(t typecheck.Type) int {
	switch t.K {
	case typecheck.TyI8, typecheck.TyU8:
		return 8
	case typecheck.TyI32, typecheck.TyU32:
		return 32
	case typecheck.TyI64, typecheck.TyU64, typecheck.TyUSize, typecheck.TyUntypedInt:
		return 64
	default:
		return 0
	}
}

func truncBits(u uint64, w int) uint64 {
	if w >= 64 {
		return u
	}
	if w <= 0 {
		return u
	}
	return u & ((uint64(1) << uint64(w)) - 1)
}

func truncInt(u uint64, t typecheck.Type) uint64 {
	w := intBitWidth(t)
	return truncBits(u, w)
}

func intSigned(bits uint64, t typecheck.Type) int64 {
	w := intBitWidth(t)
	bits = truncBits(bits, w)
	switch t.K {
	case typecheck.TyI8:
		return int64(int8(bits))
	case typecheck.TyI32:
		return int64(int32(bits))
	default:
		// i64/untyped
		return int64(bits)
	}
}

func intMinSigned(t typecheck.Type) int64 {
	switch t.K {
	case typecheck.TyI8:
		return -128
	case typecheck.TyI32:
		return -2147483648
	default:
		// i64/untyped
		return -9223372036854775808
	}
}

func intMaxSigned(t typecheck.Type) int64 {
	switch t.K {
	case typecheck.TyI8:
		return 127
	case typecheck.TyI32:
		return 2147483647
	default:
		// i64/untyped
		return 9223372036854775807
	}
}

func intMaxUnsigned(t typecheck.Type) uint64 {
	switch t.K {
	case typecheck.TyU8:
		return 255
	case typecheck.TyU32:
		return 4294967295
	default:
		// u64/usize
		return 18446744073709551615
	}
}

func castIntChecked(bits uint64, from typecheck.Type, to typecheck.Type) (uint64, error) {
	from = stripRange(from)
	to = stripRange(to)
	if !isIntType(from) || !isIntType(to) {
		return 0, fmt.Errorf("unsupported cast")
	}
	fromW := intBitWidth(from)
	toW := intBitWidth(to)
	xBits := truncBits(bits, fromW)

	msg := "int cast overflow"
	if from.K == typecheck.TyI64 && to.K == typecheck.TyI32 {
		msg = "i64 to i32 overflow"
	}

	// Range check.
	if isSignedIntType(from) {
		x := intSigned(xBits, from)
		if isSignedIntType(to) {
			if x < intMinSigned(to) || x > intMaxSigned(to) {
				return 0, fmt.Errorf("%s", msg)
			}
			return truncBits(uint64(x), toW), nil
		}
		// to unsigned
		if x < 0 || uint64(x) > intMaxUnsigned(to) {
			return 0, fmt.Errorf("%s", msg)
		}
		return truncBits(uint64(x), toW), nil
	}

	// from unsigned
	u := truncBits(xBits, fromW)
	if isSignedIntType(to) {
		if u > uint64(intMaxSigned(to)) {
			return 0, fmt.Errorf("%s", msg)
		}
		return truncBits(u, toW), nil
	}
	if u > intMaxUnsigned(to) {
		return 0, fmt.Errorf("%s", msg)
	}
	return truncBits(u, toW), nil
}

func rangeContainsBits(bits uint64, rng typecheck.Type) bool {
	if rng.K != typecheck.TyRange || rng.Base == nil {
		return false
	}
	base := stripRange(*rng.Base)
	if !isIntType(base) {
		return false
	}
	w := intBitWidth(base)
	b := truncBits(bits, w)
	if isSignedIntType(base) {
		x := intSigned(b, base)
		return x >= rng.Lo && x <= rng.Hi
	}
	if rng.Lo < 0 || rng.Hi < 0 {
		return false
	}
	u := b
	return u >= uint64(rng.Lo) && u <= uint64(rng.Hi)
}
