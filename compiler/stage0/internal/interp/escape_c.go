package interp

import (
	"fmt"
	"strings"
)

func escapeC(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
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
			if ch >= 0x20 && ch <= 0x7e {
				b.WriteByte(ch)
			} else {
				fmt.Fprintf(&b, "\\x%02x", ch)
			}
		}
	}
	return b.String()
}
