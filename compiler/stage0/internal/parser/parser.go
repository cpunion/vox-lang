package parser

import (
	"strconv"

	"voxlang/internal/ast"
	"voxlang/internal/diag"
	"voxlang/internal/lexer"
	"voxlang/internal/source"
)

type Parser struct {
	file  *source.File
	toks  []lexer.Token
	pos   int
	diags *diag.Bag
}

func Parse(file *source.File) (*ast.Program, *diag.Bag) {
	toks := lexer.Lex(file)
	p := &Parser{file: file, toks: toks, diags: &diag.Bag{}}
	return p.parseProgram(), p.diags
}

func ParseFiles(files []*source.File) (*ast.Program, *diag.Bag) {
	merged := &ast.Program{}
	diags := &diag.Bag{}
	for _, f := range files {
		prog, d := Parse(f)
		if prog != nil {
			merged.Imports = append(merged.Imports, prog.Imports...)
			merged.Funcs = append(merged.Funcs, prog.Funcs...)
		}
		if d != nil && len(d.Items) > 0 {
			diags.Items = append(diags.Items, d.Items...)
		}
	}
	if len(diags.Items) > 0 {
		return merged, diags
	}
	return merged, nil
}

func (p *Parser) parseProgram() *ast.Program {
	prog := &ast.Program{}
	for !p.at(lexer.TokenEOF) {
		if p.match(lexer.TokenImport) {
			imp := p.parseImportDecl()
			if imp != nil {
				prog.Imports = append(prog.Imports, imp)
			}
			continue
		}
		if p.match(lexer.TokenFn) {
			fn := p.parseFuncDecl()
			if fn != nil {
				prog.Funcs = append(prog.Funcs, fn)
			}
			continue
		}
		p.errorHere("expected `import` or `fn`")
		p.advance()
	}
	return prog
}

func (p *Parser) parseImportDecl() *ast.ImportDecl {
	start := p.prev()
	pathTok := p.expect(lexer.TokenString, "expected string literal import path")
	if pathTok.Kind != lexer.TokenString {
		return nil
	}
	path := unquote(pathTok.Lexeme)
	alias := ""
	if p.match(lexer.TokenAs) {
		id := p.expect(lexer.TokenIdent, "expected alias after `as`")
		if id.Kind == lexer.TokenIdent {
			alias = id.Lexeme
		}
	}
	// Optional semicolon: if absent, next token must start a new top-level item.
	endTok := p.peek()
	if p.match(lexer.TokenSemicolon) {
		endTok = p.prev()
	} else {
		switch p.peek().Kind {
		case lexer.TokenFn, lexer.TokenImport, lexer.TokenEOF:
			// ok
		default:
			p.errorHere("expected `;` or next top-level item after import")
		}
	}
	return &ast.ImportDecl{Path: path, Alias: alias, Span: joinSpan(start.Span, endTok.Span)}
}

func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	startTok := p.prev()
	nameTok := p.expect(lexer.TokenIdent, "expected function name")
	if nameTok.Kind != lexer.TokenIdent {
		return nil
	}
	p.expect(lexer.TokenLParen, "expected `(`")
	var params []ast.Param
	if !p.at(lexer.TokenRParen) {
		for {
			paramName := p.expect(lexer.TokenIdent, "expected parameter name")
			p.expect(lexer.TokenColon, "expected `:` after parameter name")
			ty := p.parseType()
			params = append(params, ast.Param{Name: paramName.Lexeme, Type: ty, Span: joinSpan(paramName.Span, ty.Span())})
			if p.match(lexer.TokenComma) {
				continue
			}
			break
		}
	}
	p.expect(lexer.TokenRParen, "expected `)`")

	var ret ast.Type = &ast.UnitType{S: p.prev().Span}
	if p.match(lexer.TokenArrow) {
		ret = p.parseType()
	}
	body := p.parseBlock()
	if body == nil {
		return nil
	}
	return &ast.FuncDecl{
		Name:   nameTok.Lexeme,
		Params: params,
		Ret:    ret,
		Body:   body,
		Span:   joinSpan(startTok.Span, body.S),
	}
}

func (p *Parser) parseBlock() *ast.BlockStmt {
	lbrace := p.expect(lexer.TokenLBrace, "expected `{`")
	if lbrace.Kind != lexer.TokenLBrace {
		return nil
	}
	stmts := []ast.Stmt{}
	for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
		st := p.parseStmt()
		if st != nil {
			stmts = append(stmts, st)
		} else {
			p.advance()
		}
	}
	rbrace := p.expect(lexer.TokenRBrace, "expected `}`")
	return &ast.BlockStmt{Stmts: stmts, S: joinSpan(lbrace.Span, rbrace.Span)}
}

func (p *Parser) parseStmt() ast.Stmt {
	switch p.peek().Kind {
	case lexer.TokenLet:
		return p.parseLet()
	case lexer.TokenReturn:
		return p.parseReturn()
	case lexer.TokenIf:
		return p.parseIf()
	case lexer.TokenWhile:
		return p.parseWhile()
	case lexer.TokenBreak:
		return p.parseBreak()
	case lexer.TokenContinue:
		return p.parseContinue()
	case lexer.TokenLBrace:
		return p.parseBlock()
	case lexer.TokenIdent:
		// assignment or expr stmt
		if p.peekN(1).Kind == lexer.TokenEq {
			return p.parseAssign()
		}
	}
	// expr stmt
	ex := p.parseExpr(0)
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.ExprStmt{Expr: ex, S: joinSpan(ex.Span(), semi.Span)}
}

func (p *Parser) parseLet() ast.Stmt {
	letTok := p.expect(lexer.TokenLet, "expected `let`")
	mutable := p.match(lexer.TokenMut)
	nameTok := p.expect(lexer.TokenIdent, "expected binding name")
	var ann ast.Type
	if p.match(lexer.TokenColon) {
		ann = p.parseType()
	}
	var init ast.Expr
	if p.match(lexer.TokenEq) {
		init = p.parseExpr(0)
	}
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.LetStmt{
		Mutable: mutable,
		Name:    nameTok.Lexeme,
		AnnType: ann,
		Init:    init,
		S:       joinSpan(letTok.Span, semi.Span),
	}
}

func (p *Parser) parseAssign() ast.Stmt {
	nameTok := p.expect(lexer.TokenIdent, "expected name")
	eq := p.expect(lexer.TokenEq, "expected `=`")
	_ = eq
	ex := p.parseExpr(0)
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.AssignStmt{Name: nameTok.Lexeme, Expr: ex, S: joinSpan(nameTok.Span, semi.Span)}
}

func (p *Parser) parseReturn() ast.Stmt {
	retTok := p.expect(lexer.TokenReturn, "expected `return`")
	var ex ast.Expr
	if !p.at(lexer.TokenSemicolon) {
		ex = p.parseExpr(0)
	}
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.ReturnStmt{Expr: ex, S: joinSpan(retTok.Span, semi.Span)}
}

func (p *Parser) parseIf() ast.Stmt {
	ifTok := p.expect(lexer.TokenIf, "expected `if`")
	cond := p.parseExpr(0)
	thenBlk := p.parseBlock()
	if thenBlk == nil {
		return nil
	}
	var elseSt ast.Stmt
	if p.match(lexer.TokenElse) {
		if p.at(lexer.TokenIf) {
			elseSt = p.parseIf()
		} else if p.at(lexer.TokenLBrace) {
			elseSt = p.parseBlock()
		} else {
			p.errorHere("expected `if` or `{` after else")
		}
	}
	endSpan := thenBlk.S
	if elseSt != nil {
		endSpan = elseSt.Span()
	}
	return &ast.IfStmt{Cond: cond, Then: thenBlk, Else: elseSt, S: joinSpan(ifTok.Span, endSpan)}
}

func (p *Parser) parseWhile() ast.Stmt {
	whileTok := p.expect(lexer.TokenWhile, "expected `while`")
	cond := p.parseExpr(0)
	body := p.parseBlock()
	if body == nil {
		return nil
	}
	return &ast.WhileStmt{Cond: cond, Body: body, S: joinSpan(whileTok.Span, body.S)}
}

func (p *Parser) parseBreak() ast.Stmt {
	brTok := p.expect(lexer.TokenBreak, "expected `break`")
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.BreakStmt{S: joinSpan(brTok.Span, semi.Span)}
}

func (p *Parser) parseContinue() ast.Stmt {
	coTok := p.expect(lexer.TokenContinue, "expected `continue`")
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.ContinueStmt{S: joinSpan(coTok.Span, semi.Span)}
}

func (p *Parser) parseType() ast.Type {
	// unit
	if p.match(lexer.TokenLParen) {
		lp := p.prev()
		rp := p.expect(lexer.TokenRParen, "expected `)`")
		return &ast.UnitType{S: joinSpan(lp.Span, rp.Span)}
	}
	nameTok := p.expect(lexer.TokenIdent, "expected type name")
	t := &ast.NamedType{Name: nameTok.Lexeme, S: nameTok.Span}
	// optional generic args: Name[...]
	if p.match(lexer.TokenLBracket) {
		if !p.at(lexer.TokenRBracket) {
			for {
				arg := p.parseType()
				t.Args = append(t.Args, arg)
				if p.match(lexer.TokenComma) {
					continue
				}
				break
			}
		}
		rb := p.expect(lexer.TokenRBracket, "expected `]`")
		t.S = joinSpan(t.S, rb.Span)
	}
	return t
}

// Pratt parser
func (p *Parser) parseExpr(minPrec int) ast.Expr {
	left := p.parsePrefix()
	for {
		op, prec, rightAssoc := p.peekInfix()
		if prec < minPrec {
			break
		}
		opTok := p.advance()
		nextMin := prec + 1
		if rightAssoc {
			nextMin = prec
		}
		right := p.parseExpr(nextMin)
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right, S: joinSpan(left.Span(), right.Span())}
		_ = opTok
	}
	return left
}

func (p *Parser) parsePrefix() ast.Expr {
	tok := p.peek()
	switch tok.Kind {
	case lexer.TokenIdent:
		p.advance()
		ex := ast.Expr(&ast.IdentExpr{Name: tok.Lexeme, S: tok.Span})
		return p.parsePostfix(ex)
	case lexer.TokenInt:
		p.advance()
		return p.parsePostfix(&ast.IntLit{Text: tok.Lexeme, S: tok.Span})
	case lexer.TokenString:
		p.advance()
		return p.parsePostfix(&ast.StringLit{Text: tok.Lexeme, S: tok.Span})
	case lexer.TokenTrue, lexer.TokenFalse:
		p.advance()
		v := tok.Kind == lexer.TokenTrue
		return p.parsePostfix(&ast.BoolLit{Value: v, S: tok.Span})
	case lexer.TokenLParen:
		p.advance()
		ex := p.parseExpr(0)
		p.expect(lexer.TokenRParen, "expected `)`")
		return p.parsePostfix(ex)
	case lexer.TokenMinus, lexer.TokenBang:
		p.advance()
		op := tok.Lexeme
		if op == "" {
			op = tokenOpString(tok.Kind)
		}
		ex := p.parseExpr(7)
		return p.parsePostfix(&ast.UnaryExpr{Op: op, Expr: ex, S: joinSpan(tok.Span, ex.Span())})
	default:
		p.errorHere("expected expression")
		p.advance()
		return &ast.IntLit{Text: "0", S: tok.Span}
	}
}

func (p *Parser) parsePostfix(ex ast.Expr) ast.Expr {
	for {
		if p.match(lexer.TokenDot) {
			id := p.expect(lexer.TokenIdent, "expected identifier after `.`")
			ex = &ast.MemberExpr{Recv: ex, Name: id.Lexeme, S: joinSpan(ex.Span(), id.Span)}
			continue
		}
		if p.match(lexer.TokenLParen) {
			var args []ast.Expr
			if !p.at(lexer.TokenRParen) {
				for {
					args = append(args, p.parseExpr(0))
					if p.match(lexer.TokenComma) {
						continue
					}
					break
				}
			}
			rp := p.expect(lexer.TokenRParen, "expected `)`")
			ex = &ast.CallExpr{Callee: ex, Args: args, S: joinSpan(ex.Span(), rp.Span)}
			continue
		}
		break
	}
	return ex
}

func (p *Parser) peekInfix() (op string, prec int, rightAssoc bool) {
	switch p.peek().Kind {
	case lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent:
		return tokenOpString(p.peek().Kind), 6, false
	case lexer.TokenPlus, lexer.TokenMinus:
		return tokenOpString(p.peek().Kind), 5, false
	case lexer.TokenLt, lexer.TokenLtEq, lexer.TokenGt, lexer.TokenGtEq:
		return tokenOpString(p.peek().Kind), 4, false
	case lexer.TokenEqEq, lexer.TokenBangEq:
		return tokenOpString(p.peek().Kind), 3, false
	case lexer.TokenAndAnd:
		return "&&", 2, false
	case lexer.TokenOrOr:
		return "||", 1, false
	default:
		return "", -1, false
	}
}

func tokenOpString(k lexer.Kind) string {
	switch k {
	case lexer.TokenPlus:
		return "+"
	case lexer.TokenMinus:
		return "-"
	case lexer.TokenStar:
		return "*"
	case lexer.TokenSlash:
		return "/"
	case lexer.TokenPercent:
		return "%"
	case lexer.TokenLt:
		return "<"
	case lexer.TokenLtEq:
		return "<="
	case lexer.TokenGt:
		return ">"
	case lexer.TokenGtEq:
		return ">="
	case lexer.TokenEqEq:
		return "=="
	case lexer.TokenBangEq:
		return "!="
	case lexer.TokenBang:
		return "!"
	default:
		return ""
	}
}

// helpers
func (p *Parser) peek() lexer.Token {
	if p.pos >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[p.pos]
}

func (p *Parser) peekN(n int) lexer.Token {
	i := p.pos + n
	if i >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[i]
}

func (p *Parser) prev() lexer.Token { return p.toks[p.pos-1] }

func (p *Parser) at(k lexer.Kind) bool { return p.peek().Kind == k }

func (p *Parser) match(k lexer.Kind) bool {
	if p.at(k) {
		p.pos++
		return true
	}
	return false
}

func (p *Parser) advance() lexer.Token {
	t := p.peek()
	if t.Kind != lexer.TokenEOF {
		p.pos++
	}
	return t
}

func (p *Parser) expect(k lexer.Kind, msg string) lexer.Token {
	if p.at(k) {
		return p.advance()
	}
	p.errorAt(p.peek().Span, msg)
	return p.peek()
}

func (p *Parser) errorHere(msg string) {
	p.errorAt(p.peek().Span, msg)
}

func (p *Parser) errorAt(s source.Span, msg string) {
	fn, line, col := s.LocStart()
	p.diags.Add(fn, line, col, msg)
}

func joinSpan(a source.Span, b source.Span) source.Span {
	if a.File == nil {
		return b
	}
	if b.File == nil {
		return a
	}
	start := a.Start
	if b.Start < start {
		start = b.Start
	}
	end := a.End
	if b.End > end {
		end = b.End
	}
	return source.Span{File: a.File, Start: start, End: end}
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// quick sanity: parseInt for potential future use
func parseInt64(text string) (int64, bool) {
	n, err := strconv.ParseInt(text, 10, 64)
	return n, err == nil
}
