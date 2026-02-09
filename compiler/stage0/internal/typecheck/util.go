package typecheck

import (
	"strconv"
	"strings"

	"voxlang/internal/ast"
)

func (t Type) String() string {
	switch t.K {
	case TyUnit:
		return "()"
	case TyBool:
		return "bool"
	case TyI8:
		return "i8"
	case TyU8:
		return "u8"
	case TyI16:
		return "i16"
	case TyU16:
		return "u16"
	case TyI32:
		return "i32"
	case TyU32:
		return "u32"
	case TyI64:
		return "i64"
	case TyU64:
		return "u64"
	case TyISize:
		return "isize"
	case TyUSize:
		return "usize"
	case TyString:
		return "String"
	case TyUntypedInt:
		return "untyped-int"
	case TyRange:
		if t.Base == nil {
			return "@range(<bad>)"
		}
		return "@range(" + strconv.FormatInt(t.Lo, 10) + "..=" + strconv.FormatInt(t.Hi, 10) + ") " + t.Base.String()
	case TyStruct:
		return t.Name
	case TyEnum:
		return t.Name
	case TyVec:
		if t.Elem == nil {
			return "Vec[<bad>]"
		}
		return "Vec[" + t.Elem.String() + "]"
	case TyParam:
		return t.Name
	default:
		return "<bad>"
	}
}

func sameModPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameType(a, b Type) bool {
	if a.K == TyBad || b.K == TyBad {
		return false
	}
	// Resolve untyped int to concrete only via constraints before comparing.
	if a.K == TyUntypedInt || b.K == TyUntypedInt {
		return false
	}
	if a.K != b.K {
		return false
	}
	if a.K == TyRange {
		if a.Base == nil || b.Base == nil {
			return false
		}
		return a.Lo == b.Lo && a.Hi == b.Hi && sameType(*a.Base, *b.Base)
	}
	if a.K == TyStruct || a.K == TyEnum {
		return a.Name == b.Name
	}
	if a.K == TyVec {
		if a.Elem == nil || b.Elem == nil {
			return false
		}
		return sameType(*a.Elem, *b.Elem)
	}
	if a.K == TyParam {
		return a.Name == b.Name
	}
	return true
}

// assignableTo reports whether a value of type got can be used in a context
// that expects want.
//
// Stage0 v0 rule: @range(lo..=hi) Base is a subtype of Base (widening only).
func assignableTo(want, got Type) bool {
	if sameType(want, got) {
		return true
	}
	// Allow range -> base (widening).
	if isIntType(want) && got.K == TyRange && got.Base != nil && sameType(want, *got.Base) {
		return true
	}
	return false
}

// stripRange returns the underlying integer type when t is a range type.
func stripRange(t Type) Type {
	if t.K != TyRange || t.Base == nil {
		return t
	}
	return *t.Base
}

func isRangeOf(t Type, base Kind) bool {
	return t.K == TyRange && t.Base != nil && t.Base.K == base
}

func isIntType(t Type) bool {
	switch t.K {
	case TyI8, TyU8, TyI16, TyU16, TyI32, TyU32, TyI64, TyU64, TyISize, TyUSize:
		return true
	default:
		return false
	}
}

func isIntLikeType(t Type) bool {
	if t.K == TyUntypedInt {
		return true
	}
	if t.K == TyRange {
		return t.Base != nil && isIntType(*t.Base)
	}
	return isIntType(t)
}

func isSignedIntType(t Type) bool { return t.K == TyI8 || t.K == TyI16 || t.K == TyI32 || t.K == TyI64 || t.K == TyISize }

func isUnsignedIntType(t Type) bool {
	return t.K == TyU8 || t.K == TyU16 || t.K == TyU32 || t.K == TyU64 || t.K == TyUSize
}

func intBitWidth(t Type) int {
	switch t.K {
	case TyI8, TyU8:
		return 8
	case TyI16, TyU16:
		return 16
	case TyI32, TyU32:
		return 32
	case TyI64, TyU64, TyISize:
		return 64
	case TyUSize:
		return 64 // stage0 v0: usize is fixed to 64-bit
	default:
		return 0
	}
}

// intMinMax returns inclusive min/max bounds for an integer type.
// max is returned as unsigned to cover u64.
func intMinMax(t Type) (min int64, max uint64, ok bool) {
	switch t.K {
	case TyI8:
		return -128, 127, true
	case TyU8:
		return 0, 255, true
	case TyI16:
		return -32768, 32767, true
	case TyU16:
		return 0, 65535, true
	case TyI32:
		return -2147483648, 2147483647, true
	case TyU32:
		return 0, 4294967295, true
	case TyI64:
		return -9223372036854775808, 9223372036854775807, true
	case TyU64:
		return 0, 18446744073709551615, true
	case TyISize:
		return -9223372036854775808, 9223372036854775807, true
	case TyUSize:
		return 0, 18446744073709551615, true
	default:
		return 0, 0, false
	}
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
