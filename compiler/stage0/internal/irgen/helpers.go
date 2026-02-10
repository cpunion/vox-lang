package irgen

import "voxlang/internal/stringlit"

func parseUint64(text string) uint64 {
	var n uint64
	for i := 0; i < len(text); i++ {
		n = n*10 + uint64(text[i]-'0')
	}
	return n
}

func unquoteUnescape(lit string) (string, error) {
	return stringlit.Decode(lit)
}
