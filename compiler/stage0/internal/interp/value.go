package interp

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
	P []Value          // VEnum payload fields (nil/empty when no payload)
	A []Value          // VVec elements
}

func unit() Value { return Value{K: VUnit} }

func cloneValue(v Value) Value {
	switch v.K {
	case VStruct:
		out := Value{K: VStruct, M: map[string]Value{}}
		for k, fv := range v.M {
			out.M[k] = cloneValue(fv)
		}
		return out
	case VEnum:
		out := Value{K: VEnum, E: v.E, T: v.T}
		if len(v.P) != 0 {
			out.P = make([]Value, 0, len(v.P))
			for _, pv := range v.P {
				out.P = append(out.P, cloneValue(pv))
			}
		}
		return out
	case VVec:
		out := Value{K: VVec, A: make([]Value, 0, len(v.A))}
		for _, e := range v.A {
			out.A = append(out.A, cloneValue(e))
		}
		return out
	default:
		return v
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
		if len(a.P) != len(b.P) {
			return false
		}
		for i := 0; i < len(a.P); i++ {
			if !valueEq(a.P[i], b.P[i]) {
				return false
			}
		}
		return true
	case VVec:
		// Not needed for stage0 yet; keep it conservative.
		return false
	default:
		return false
	}
}
