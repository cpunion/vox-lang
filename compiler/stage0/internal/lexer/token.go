package lexer

import "voxlang/internal/source"

type Kind int

const (
	TokenEOF Kind = iota
	TokenBad

	// Literals / identifiers
	TokenIdent
	TokenInt
	TokenString

	// Keywords
	TokenFn
	TokenStruct
	TokenLet
	TokenMut
	TokenReturn
	TokenIf
	TokenElse
	TokenTrue
	TokenFalse
	TokenType
	TokenConst
	TokenStatic
	TokenImport
	TokenPub
	TokenMatch
	TokenAs
	TokenWhile
	TokenBreak
	TokenContinue

	// Punct
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenComma
	TokenSemicolon
	TokenColon
	TokenColonColon
	TokenDot

	// Operators
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenBang
	TokenEq
	TokenEqEq
	TokenBangEq
	TokenLt
	TokenLtEq
	TokenGt
	TokenGtEq
	TokenAndAnd
	TokenOrOr
	TokenArrow
)

type Token struct {
	Kind   Kind
	Lexeme string
	Span   source.Span
}

func (t Token) Is(k Kind) bool { return t.Kind == k }
