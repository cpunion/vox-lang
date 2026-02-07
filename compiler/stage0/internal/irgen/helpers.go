package irgen

import "strconv"

func parseInt64(text string) int64 {
	var n int64
	for i := 0; i < len(text); i++ {
		n = n*10 + int64(text[i]-'0')
	}
	return n
}

func unquoteUnescape(lit string) (string, error) {
	// Lexer keeps the full token lexeme including quotes; reuse Go-like unquoting.
	// This accepts standard escapes like \n, \t, \\, \", \r.
	return strconv.Unquote(lit)
}
