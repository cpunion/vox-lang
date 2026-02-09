package lexer

import (
	"unicode"

	"voxlang/internal/source"
)

func Lex(file *source.File) []Token {
	lx := &lexer{file: file, input: file.Input}
	for {
		lx.skipSpaceAndComments()
		start := lx.pos
		if lx.pos >= len(lx.input) {
			lx.emit(TokenEOF, "", start, start)
			break
		}
		ch := lx.peek()
		switch {
		case isIdentStart(ch):
			lx.lexIdentOrKeyword()
		case isDigit(ch):
			lx.lexInt()
		default:
			lx.lexPunct()
		}
	}
	return lx.tokens
}

type lexer struct {
	file   *source.File
	input  string
	pos    int
	tokens []Token
}

func (lx *lexer) peek() byte { return lx.input[lx.pos] }

func (lx *lexer) next() byte {
	ch := lx.input[lx.pos]
	lx.pos++
	return ch
}

func (lx *lexer) emit(k Kind, lex string, start, end int) {
	lx.tokens = append(lx.tokens, Token{
		Kind:   k,
		Lexeme: lex,
		Span:   source.Span{File: lx.file, Start: start, End: end},
	})
}

func (lx *lexer) skipSpaceAndComments() {
	for lx.pos < len(lx.input) {
		ch := lx.input[lx.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			lx.pos++
			continue
		}
		// line comment
		if ch == '/' && lx.pos+1 < len(lx.input) && lx.input[lx.pos+1] == '/' {
			lx.pos += 2
			for lx.pos < len(lx.input) && lx.input[lx.pos] != '\n' {
				lx.pos++
			}
			continue
		}
		return
	}
}

func (lx *lexer) lexIdentOrKeyword() {
	start := lx.pos
	lx.pos++
	for lx.pos < len(lx.input) && isIdentContinue(lx.input[lx.pos]) {
		lx.pos++
	}
	lex := lx.input[start:lx.pos]
	switch lex {
	case "fn":
		lx.emit(TokenFn, lex, start, lx.pos)
	case "struct":
		lx.emit(TokenStruct, lex, start, lx.pos)
	case "enum":
		lx.emit(TokenEnum, lex, start, lx.pos)
	case "let":
		lx.emit(TokenLet, lex, start, lx.pos)
	case "mut":
		lx.emit(TokenMut, lex, start, lx.pos)
	case "return":
		lx.emit(TokenReturn, lex, start, lx.pos)
	case "if":
		lx.emit(TokenIf, lex, start, lx.pos)
	case "else":
		lx.emit(TokenElse, lex, start, lx.pos)
	case "true":
		lx.emit(TokenTrue, lex, start, lx.pos)
	case "false":
		lx.emit(TokenFalse, lex, start, lx.pos)
	case "type":
		lx.emit(TokenType, lex, start, lx.pos)
	case "const":
		lx.emit(TokenConst, lex, start, lx.pos)
	case "static":
		lx.emit(TokenStatic, lex, start, lx.pos)
	case "import":
		lx.emit(TokenImport, lex, start, lx.pos)
	case "pub":
		lx.emit(TokenPub, lex, start, lx.pos)
	case "match":
		lx.emit(TokenMatch, lex, start, lx.pos)
	case "as":
		lx.emit(TokenAs, lex, start, lx.pos)
	case "from":
		lx.emit(TokenFrom, lex, start, lx.pos)
	case "for":
		lx.emit(TokenFor, lex, start, lx.pos)
	case "trait":
		lx.emit(TokenTrait, lex, start, lx.pos)
	case "impl":
		lx.emit(TokenImpl, lex, start, lx.pos)
	case "while":
		lx.emit(TokenWhile, lex, start, lx.pos)
	case "break":
		lx.emit(TokenBreak, lex, start, lx.pos)
	case "continue":
		lx.emit(TokenContinue, lex, start, lx.pos)
	default:
		lx.emit(TokenIdent, lex, start, lx.pos)
	}
}

func (lx *lexer) lexInt() {
	start := lx.pos
	for lx.pos < len(lx.input) && isDigit(lx.input[lx.pos]) {
		lx.pos++
	}
	lx.emit(TokenInt, lx.input[start:lx.pos], start, lx.pos)
}

func (lx *lexer) lexString() {
	start := lx.pos
	lx.pos++ // opening "
	for lx.pos < len(lx.input) {
		ch := lx.next()
		if ch == '"' {
			lx.emit(TokenString, lx.input[start:lx.pos], start, lx.pos)
			return
		}
		if ch == '\\' && lx.pos < len(lx.input) {
			// skip escaped char
			lx.pos++
		}
	}
	// unterminated
	lx.emit(TokenBad, lx.input[start:lx.pos], start, lx.pos)
}

func (lx *lexer) lexPunct() {
	start := lx.pos
	ch := lx.next()
	switch ch {
	case '(':
		lx.emit(TokenLParen, "(", start, lx.pos)
	case ')':
		lx.emit(TokenRParen, ")", start, lx.pos)
	case '{':
		lx.emit(TokenLBrace, "{", start, lx.pos)
	case '}':
		lx.emit(TokenRBrace, "}", start, lx.pos)
	case '[':
		lx.emit(TokenLBracket, "[", start, lx.pos)
	case ']':
		lx.emit(TokenRBracket, "]", start, lx.pos)
	case ',':
		lx.emit(TokenComma, ",", start, lx.pos)
	case ';':
		lx.emit(TokenSemicolon, ";", start, lx.pos)
	case ':':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == ':' {
			lx.pos++
			lx.emit(TokenColonColon, "::", start, lx.pos)
		} else {
			lx.emit(TokenColon, ":", start, lx.pos)
		}
	case '.':
		// .. or ..=
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '.' {
			lx.pos++
			if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
				lx.pos++
				lx.emit(TokenDotDotEq, "..=", start, lx.pos)
			} else {
				lx.emit(TokenDotDot, "..", start, lx.pos)
			}
		} else {
			lx.emit(TokenDot, ".", start, lx.pos)
		}
	case '@':
		lx.emit(TokenAt, "@", start, lx.pos)
	case '+':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenPlusEq, "+=", start, lx.pos)
		} else {
			lx.emit(TokenPlus, "+", start, lx.pos)
		}
	case '-':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '>' {
			lx.pos++
			lx.emit(TokenArrow, "->", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenMinusEq, "-=", start, lx.pos)
		} else {
			lx.emit(TokenMinus, "-", start, lx.pos)
		}
	case '*':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenStarEq, "*=", start, lx.pos)
		} else {
			lx.emit(TokenStar, "*", start, lx.pos)
		}
	case '/':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenSlashEq, "/=", start, lx.pos)
		} else {
			lx.emit(TokenSlash, "/", start, lx.pos)
		}
	case '%':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenPercentEq, "%=", start, lx.pos)
		} else {
			lx.emit(TokenPercent, "%", start, lx.pos)
		}
	case '!':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenBangEq, "!=", start, lx.pos)
		} else {
			lx.emit(TokenBang, "!", start, lx.pos)
		}
	case '=':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenEqEq, "==", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '>' {
			lx.pos++
			lx.emit(TokenFatArrow, "=>", start, lx.pos)
		} else {
			lx.emit(TokenEq, "=", start, lx.pos)
		}
	case '<':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenLtEq, "<=", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '<' {
			lx.pos++
			if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
				lx.pos++
				lx.emit(TokenLtLtEq, "<<=", start, lx.pos)
			} else {
				lx.emit(TokenLtLt, "<<", start, lx.pos)
			}
		} else {
			lx.emit(TokenLt, "<", start, lx.pos)
		}
	case '>':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenGtEq, ">=", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '>' {
			lx.pos++
			if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
				lx.pos++
				lx.emit(TokenGtGtEq, ">>=", start, lx.pos)
			} else {
				lx.emit(TokenGtGt, ">>", start, lx.pos)
			}
		} else {
			lx.emit(TokenGt, ">", start, lx.pos)
		}
	case '&':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '&' {
			lx.pos++
			lx.emit(TokenAndAnd, "&&", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenAmpEq, "&=", start, lx.pos)
		} else {
			lx.emit(TokenAmp, "&", start, lx.pos)
		}
	case '|':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '|' {
			lx.pos++
			lx.emit(TokenOrOr, "||", start, lx.pos)
		} else if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenPipeEq, "|=", start, lx.pos)
		} else {
			lx.emit(TokenPipe, "|", start, lx.pos)
		}
	case '^':
		if lx.pos < len(lx.input) && lx.input[lx.pos] == '=' {
			lx.pos++
			lx.emit(TokenCaretEq, "^=", start, lx.pos)
		} else {
			lx.emit(TokenCaret, "^", start, lx.pos)
		}
	case '"':
		lx.pos-- // back to opening
		lx.lexString()
	default:
		lx.emit(TokenBad, string(ch), start, lx.pos)
	}
}

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

func isIdentStart(ch byte) bool {
	return ch == '_' || unicode.IsLetter(rune(ch))
}

func isIdentContinue(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}
