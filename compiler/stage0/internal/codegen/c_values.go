package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"voxlang/internal/ir"
)

func cValue(v ir.Value) string {
	switch x := v.(type) {
	case *ir.Temp:
		return cTempName(x.ID)
	case *ir.ParamRef:
		return fmt.Sprintf("p%d", x.Index)
	case *ir.Slot:
		return cSlotName(x.ID)
	case *ir.ConstInt:
		return cIntConst(x)
	case *ir.ConstBool:
		if x.V {
			return "true"
		}
		return "false"
	case *ir.ConstStr:
		return cStringLit(x.S)
	default:
		return "0"
	}
}

func cIntConst(x *ir.ConstInt) string {
	if x == nil {
		return "0"
	}
	// Use *C(n) macros to keep constants well-typed across platforms.
	// (Even though stage0 currently restricts literals to i64 range.)
	var s string
	switch x.Ty.K {
	case ir.TI8:
		s = strconv.FormatInt(int64(int8(x.Bits)), 10)
	case ir.TI16:
		s = strconv.FormatInt(int64(int16(x.Bits)), 10)
	case ir.TI32:
		s = strconv.FormatInt(int64(int32(x.Bits)), 10)
	case ir.TI64:
		s = strconv.FormatInt(int64(x.Bits), 10)
	default:
		s = strconv.FormatUint(x.Bits, 10)
	}
	switch x.Ty.K {
	case ir.TI8:
		return "INT8_C(" + s + ")"
	case ir.TU8:
		return "UINT8_C(" + s + ")"
	case ir.TI16:
		return "INT16_C(" + s + ")"
	case ir.TU16:
		return "UINT16_C(" + s + ")"
	case ir.TI32:
		return "INT32_C(" + s + ")"
	case ir.TU32:
		return "UINT32_C(" + s + ")"
	case ir.TI64:
		return "INT64_C(" + s + ")"
	case ir.TU64, ir.TUSize:
		return "UINT64_C(" + s + ")"
	default:
		return s
	}
}

func cStringLit(s string) string {
	// Emit a valid C string literal.
	// Keep it ASCII-ish; escape control chars and quotes/backslashes.
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if ch < 0x20 {
				// generic hex escape
				fmt.Fprintf(&b, "\\x%02x", ch)
			} else {
				b.WriteByte(ch)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func stringToCOp(op string) string {
	switch op {
	case "add":
		return "+"
	case "sub":
		return "-"
	case "mul":
		return "*"
	case "div":
		return "/"
	case "mod":
		return "%"
	default:
		return op
	}
}

func cCmpOp(op ir.CmpKind) string {
	switch op {
	case ir.CmpLt:
		return "<"
	case ir.CmpLe:
		return "<="
	case ir.CmpGt:
		return ">"
	case ir.CmpGe:
		return ">="
	case ir.CmpEq:
		return "=="
	case ir.CmpNe:
		return "!="
	default:
		return "=="
	}
}
