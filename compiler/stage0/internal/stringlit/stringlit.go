package stringlit

import (
	"fmt"
	"strconv"
	"strings"
)

// Decode decodes a Vox string literal token text.
//
// Supported forms:
// - regular: "..."
// - multiline: """..."""
//
// Multiline rules:
// - normalize newlines to \n
// - if content starts with a newline, drop exactly one
// - unindent by minimum leading spaces across non-empty lines
// - tabs in indentation are rejected
// - then process escapes: \\ \" \n \r \t
func Decode(lit string) (string, error) {
	if strings.HasPrefix(lit, `"""`) {
		return decodeTriple(lit)
	}
	return strconv.Unquote(lit)
}

func decodeTriple(lit string) (string, error) {
	if len(lit) < 6 || !strings.HasPrefix(lit, `"""`) || !strings.HasSuffix(lit, `"""`) {
		return "", fmt.Errorf("invalid multiline string literal")
	}

	body := lit[3 : len(lit)-3]
	body = normalizeNewlines(body)
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	lines := strings.Split(body, "\n")
	// If closing quotes are on their own line, keep content ergonomic by
	// dropping the synthetic trailing empty line.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	minIndent := -1
	for _, ln := range lines {
		nonWS := firstNonSpaceTab(ln)
		if nonWS < 0 {
			continue
		}
		if strings.ContainsRune(ln[:nonWS], '\t') {
			return "", fmt.Errorf("tab indentation is not allowed in multiline string")
		}
		if minIndent < 0 || nonWS < minIndent {
			minIndent = nonWS
		}
	}
	if minIndent < 0 {
		return "", nil
	}

	for i, ln := range lines {
		nonWS := firstNonSpaceTab(ln)
		if nonWS < 0 {
			lines[i] = ""
			continue
		}
		if len(ln) < minIndent {
			lines[i] = ""
			continue
		}
		lines[i] = ln[minIndent:]
	}
	raw := strings.Join(lines, "\n")
	return unescape(raw)
}

func normalizeNewlines(s string) string {
	// Keep this small and deterministic: normalize \r\n and \r to \n.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func firstNonSpaceTab(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return i
		}
	}
	return -1
}

func unescape(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("invalid escape at end of string")
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		default:
			return "", fmt.Errorf("invalid escape sequence")
		}
	}
	return b.String(), nil
}
