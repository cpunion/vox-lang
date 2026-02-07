package codegen

import (
	"fmt"
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
		return fmt.Sprintf("%d", x.V)
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
