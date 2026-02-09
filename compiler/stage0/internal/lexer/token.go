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
	TokenEnum
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
	TokenFrom
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
	TokenDotDot   // ..
	TokenDotDotEq // ..=
	TokenAt       // @

	// Operators
	TokenPlus
	TokenPlusEq
	TokenMinus
	TokenMinusEq
	TokenStar
	TokenStarEq
	TokenSlash
	TokenSlashEq
	TokenPercent
	TokenPercentEq
	TokenBang
	TokenEq
	TokenEqEq
	TokenBangEq
	TokenLt
	TokenLtEq
	TokenLtLt
	TokenLtLtEq
	TokenGt
	TokenGtEq
	TokenGtGt
	TokenGtGtEq
	TokenAmp
	TokenAmpEq
	TokenCaret
	TokenCaretEq
	TokenAndAnd
	TokenPipe
	TokenPipeEq
	TokenOrOr
	TokenArrow
	TokenFatArrow
)

type Token struct {
	Kind   Kind
	Lexeme string
	Span   source.Span
}

func (t Token) Is(k Kind) bool { return t.Kind == k }
