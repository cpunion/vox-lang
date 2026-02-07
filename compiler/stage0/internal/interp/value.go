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
	P *Value           // VEnum payload (nil when no payload)
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
		if v.P != nil {
			p := cloneValue(*v.P)
			out.P = &p
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
