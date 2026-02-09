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
	K  ValueKind
	I  uint64 // integer raw bits (interpreted by static type at use site)
	B  bool
	S  string
	St *StructData // VStruct fields (copy-on-write container)
	E  string      // VEnum qualified enum name
	T  int         // VEnum tag
	P  []Value     // VEnum payload fields (nil/empty when no payload)
	Vc *VecData    // VVec elements (copy-on-write container)
}

type StructData struct {
	shared bool
	Fields map[string]Value
}

type VecData struct {
	shared bool
	Elems  []Value
}

func unit() Value { return Value{K: VUnit} }

func newStructValue(fields map[string]Value) Value {
	return Value{K: VStruct, St: &StructData{shared: false, Fields: fields}}
}

func newVecValue(elems []Value) Value {
	return Value{K: VVec, Vc: &VecData{shared: false, Elems: elems}}
}

func ensureStructUnique(v Value) Value {
	if v.K != VStruct || v.St == nil || !v.St.shared {
		return v
	}
	next := make(map[string]Value, len(v.St.Fields))
	for k, fv := range v.St.Fields {
		next[k] = cloneValue(fv)
	}
	return newStructValue(next)
}

func ensureVecUnique(v Value) Value {
	if v.K != VVec || v.Vc == nil || !v.Vc.shared {
		return v
	}
	next := make([]Value, 0, len(v.Vc.Elems))
	for _, e := range v.Vc.Elems {
		next = append(next, cloneValue(e))
	}
	return newVecValue(next)
}

func cloneValue(v Value) Value {
	switch v.K {
	case VStruct:
		if v.St != nil {
			v.St.shared = true
		}
		return v
	case VEnum:
		// Enum payload is immutable in stage0; shallow copy is enough.
		return v
	case VVec:
		if v.Vc != nil {
			v.Vc.shared = true
		}
		return v
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
