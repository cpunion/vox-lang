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
			merged.Types = append(merged.Types, prog.Types...)
			merged.Structs = append(merged.Structs, prog.Structs...)
			merged.Enums = append(merged.Enums, prog.Enums...)
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

		// Optional visibility modifier.
		if p.match(lexer.TokenPub) {
			switch {
			case p.match(lexer.TokenType):
				td := p.parseTypeAliasDecl()
				if td != nil {
					td.Pub = true
					prog.Types = append(prog.Types, td)
				}
			case p.match(lexer.TokenStruct):
				st := p.parseStructDecl()
				if st != nil {
					st.Pub = true
					prog.Structs = append(prog.Structs, st)
				}
			case p.match(lexer.TokenEnum):
				en := p.parseEnumDecl()
				if en != nil {
					en.Pub = true
					prog.Enums = append(prog.Enums, en)
				}
			case p.match(lexer.TokenFn):
				fn := p.parseFuncDecl()
				if fn != nil {
					fn.Pub = true
					prog.Funcs = append(prog.Funcs, fn)
				}
			default:
				p.errorHere("expected `type`, `fn`, `struct`, or `enum` after `pub`")
				p.advance()
			}
			continue
		}

		if p.match(lexer.TokenType) {
			td := p.parseTypeAliasDecl()
			if td != nil {
				prog.Types = append(prog.Types, td)
			}
			continue
		}
		if p.match(lexer.TokenStruct) {
			st := p.parseStructDecl()
			if st != nil {
				prog.Structs = append(prog.Structs, st)
			}
			continue
		}
		if p.match(lexer.TokenEnum) {
			en := p.parseEnumDecl()
			if en != nil {
				prog.Enums = append(prog.Enums, en)
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
		p.errorHere("expected `import`, `type`, `struct`, `enum`, or `fn`")
		p.advance()
	}
	return prog
}

func (p *Parser) parseTypeAliasDecl() *ast.TypeAliasDecl {
	startTok := p.prev() // `type`
	nameTok := p.expect(lexer.TokenIdent, "expected type name")
	if nameTok.Kind != lexer.TokenIdent {
		return nil
	}
	p.expect(lexer.TokenEq, "expected `=` in type alias")
	ty := p.parseType()
	endTok := p.prev()
	// Optional semicolon.
	if p.match(lexer.TokenSemicolon) {
		endTok = p.prev()
	}
	return &ast.TypeAliasDecl{Name: nameTok.Lexeme, Type: ty, Span: joinSpan(startTok.Span, endTok.Span)}
}

func (p *Parser) parseImportDecl() *ast.ImportDecl {
	start := p.prev()
	var path string
	alias := ""
	var names []ast.ImportName

	// Named import: import { a, b as c } from "path"
	if p.match(lexer.TokenLBrace) {
		for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
			id := p.expect(lexer.TokenIdent, "expected imported name")
			local := ""
			if p.match(lexer.TokenAs) {
				a := p.expect(lexer.TokenIdent, "expected alias after `as`")
				if a.Kind == lexer.TokenIdent {
					local = a.Lexeme
				}
			}
			if id.Kind == lexer.TokenIdent {
				names = append(names, ast.ImportName{Name: id.Lexeme, Alias: local, Span: joinSpan(id.Span, id.Span)})
			}
			if p.match(lexer.TokenComma) {
				continue
			}
			break
		}
		p.expect(lexer.TokenRBrace, "expected `}` to end import list")
		pathTok := p.expect(lexer.TokenFrom, "expected `from` in named import")
		_ = pathTok
		strTok := p.expect(lexer.TokenString, "expected string literal import path")
		if strTok.Kind != lexer.TokenString {
			return nil
		}
		path = unquote(strTok.Lexeme)
	} else {
		pathTok := p.expect(lexer.TokenString, "expected string literal import path")
		if pathTok.Kind != lexer.TokenString {
			return nil
		}
		path = unquote(pathTok.Lexeme)
		if p.match(lexer.TokenAs) {
			id := p.expect(lexer.TokenIdent, "expected alias after `as`")
			if id.Kind == lexer.TokenIdent {
				alias = id.Lexeme
			}
		}
	}
	// Optional semicolon: if absent, next token must start a new top-level item.
	endTok := p.peek()
	if p.match(lexer.TokenSemicolon) {
		endTok = p.prev()
	} else {
		switch p.peek().Kind {
		case lexer.TokenPub, lexer.TokenType, lexer.TokenFn, lexer.TokenStruct, lexer.TokenEnum, lexer.TokenImport, lexer.TokenEOF:
			// ok
		default:
			p.errorHere("expected `;` or next top-level item after import")
		}
	}
	return &ast.ImportDecl{Path: path, Alias: alias, Names: names, Span: joinSpan(start.Span, endTok.Span)}
}

func (p *Parser) parseStructDecl() *ast.StructDecl {
	startTok := p.prev() // `struct`
	nameTok := p.expect(lexer.TokenIdent, "expected struct name")
	if nameTok.Kind != lexer.TokenIdent {
		return nil
	}
	lbrace := p.expect(lexer.TokenLBrace, "expected `{` after struct name")
	if lbrace.Kind != lexer.TokenLBrace {
		return nil
	}
	fields := []ast.StructField{}
	for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
		fpub := p.match(lexer.TokenPub)
		fname := p.expect(lexer.TokenIdent, "expected field name")
		p.expect(lexer.TokenColon, "expected `:` after field name")
		ty := p.parseType()
		fields = append(fields, ast.StructField{Pub: fpub, Name: fname.Lexeme, Type: ty, Span: joinSpan(fname.Span, ty.Span())})
		if p.match(lexer.TokenComma) {
			// allow trailing comma before }
			continue
		}
		break
	}
	rbrace := p.expect(lexer.TokenRBrace, "expected `}`")
	endTok := rbrace
	_ = p.match(lexer.TokenSemicolon) // optional
	if p.prev().Kind == lexer.TokenSemicolon {
		endTok = p.prev()
	}
	return &ast.StructDecl{Name: nameTok.Lexeme, Fields: fields, Span: joinSpan(startTok.Span, endTok.Span)}
}

func (p *Parser) parseEnumDecl() *ast.EnumDecl {
	startTok := p.prev() // `enum`
	nameTok := p.expect(lexer.TokenIdent, "expected enum name")
	if nameTok.Kind != lexer.TokenIdent {
		return nil
	}
	lbrace := p.expect(lexer.TokenLBrace, "expected `{` after enum name")
	if lbrace.Kind != lexer.TokenLBrace {
		return nil
	}
	variants := []ast.EnumVariant{}
	for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
		vname := p.expect(lexer.TokenIdent, "expected variant name")
		fields := []ast.Type{}
		end := vname.Span
		if p.match(lexer.TokenLParen) {
			if !p.at(lexer.TokenRParen) {
				for {
					ty := p.parseType()
					fields = append(fields, ty)
					end = ty.Span()
					if p.match(lexer.TokenComma) {
						continue
					}
					break
				}
			}
			rp := p.expect(lexer.TokenRParen, "expected `)`")
			end = rp.Span
		}
		variants = append(variants, ast.EnumVariant{Name: vname.Lexeme, Fields: fields, Span: joinSpan(vname.Span, end)})
		if p.match(lexer.TokenComma) {
			// allow trailing comma before }
			continue
		}
		break
	}
	rbrace := p.expect(lexer.TokenRBrace, "expected `}`")
	endTok := rbrace
	_ = p.match(lexer.TokenSemicolon) // optional
	if p.prev().Kind == lexer.TokenSemicolon {
		endTok = p.prev()
	}
	return &ast.EnumDecl{Name: nameTok.Lexeme, Variants: variants, Span: joinSpan(startTok.Span, endTok.Span)}
}

func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	startTok := p.prev()
	nameTok := p.expect(lexer.TokenIdent, "expected function name")
	if nameTok.Kind != lexer.TokenIdent {
		return nil
	}
	// Optional generic params: fn id[T](...)
	var typeParams []string
	if p.match(lexer.TokenLBracket) {
		if !p.at(lexer.TokenRBracket) {
			for {
				id := p.expect(lexer.TokenIdent, "expected type parameter name")
				if id.Kind == lexer.TokenIdent {
					typeParams = append(typeParams, id.Lexeme)
				}
				if p.match(lexer.TokenComma) {
					continue
				}
				break
			}
		}
		p.expect(lexer.TokenRBracket, "expected `]` to end type parameters")
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
		Name:       nameTok.Lexeme,
		TypeParams: typeParams,
		Params:     params,
		Ret:        ret,
		Body:       body,
		Span:       joinSpan(startTok.Span, body.S),
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
		if p.peekN(1).Kind == lexer.TokenDot && p.peekN(2).Kind == lexer.TokenIdent && p.peekN(3).Kind == lexer.TokenEq {
			return p.parseFieldAssign()
		}
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

func (p *Parser) parseFieldAssign() ast.Stmt {
	recvTok := p.expect(lexer.TokenIdent, "expected name")
	p.expect(lexer.TokenDot, "expected `.`")
	fieldTok := p.expect(lexer.TokenIdent, "expected field name")
	p.expect(lexer.TokenEq, "expected `=`")
	ex := p.parseExpr(0)
	semi := p.expect(lexer.TokenSemicolon, "expected `;`")
	return &ast.FieldAssignStmt{
		Recv:  recvTok.Lexeme,
		Field: fieldTok.Lexeme,
		Expr:  ex,
		S:     joinSpan(recvTok.Span, semi.Span),
	}
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
	cond := p.parseExprNoStructLit(0)
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
	cond := p.parseExprNoStructLit(0)
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
	parts := []string{}
	endSpan := nameTok.Span
	if nameTok.Kind == lexer.TokenIdent {
		parts = append(parts, nameTok.Lexeme)
	}
	for p.match(lexer.TokenDot) {
		id := p.expect(lexer.TokenIdent, "expected identifier after `.` in type path")
		endSpan = id.Span
		if id.Kind == lexer.TokenIdent {
			parts = append(parts, id.Lexeme)
		}
	}
	t := &ast.NamedType{Parts: parts, S: joinSpan(nameTok.Span, endSpan)}
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
func (p *Parser) parseExpr(minPrec int) ast.Expr { return p.parseExprWith(minPrec, true) }

// parseExprNoStructLit parses an expression but does not treat a following `{ ... }` as a struct literal.
// This is used in control-flow contexts (if/while/match) where `{` is expected to start a block/arm list.
func (p *Parser) parseExprNoStructLit(minPrec int) ast.Expr { return p.parseExprWith(minPrec, false) }

func (p *Parser) parseExprWith(minPrec int, allowStructLit bool) ast.Expr {
	left := p.parsePrefixWith(allowStructLit)
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
		right := p.parseExprWith(nextMin, allowStructLit)
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right, S: joinSpan(left.Span(), right.Span())}
		_ = opTok
	}
	return left
}

func (p *Parser) parsePrefixWith(allowStructLit bool) ast.Expr {
	tok := p.peek()
	switch tok.Kind {
	case lexer.TokenIdent:
		p.advance()
		ex := ast.Expr(&ast.IdentExpr{Name: tok.Lexeme, S: tok.Span})
		return p.parsePostfix(ex, allowStructLit)
	case lexer.TokenDot:
		// Enum variant shorthand: `.Variant` (or `.Variant(...)` as a call).
		dot := p.advance()
		id := p.expect(lexer.TokenIdent, "expected identifier after `.`")
		name := ""
		if id.Kind == lexer.TokenIdent {
			name = id.Lexeme
		}
		ex := ast.Expr(&ast.DotExpr{Name: name, S: joinSpan(dot.Span, id.Span)})
		return p.parsePostfix(ex, allowStructLit)
	case lexer.TokenInt:
		p.advance()
		return p.parsePostfix(&ast.IntLit{Text: tok.Lexeme, S: tok.Span}, allowStructLit)
	case lexer.TokenString:
		p.advance()
		return p.parsePostfix(&ast.StringLit{Text: tok.Lexeme, S: tok.Span}, allowStructLit)
	case lexer.TokenTrue, lexer.TokenFalse:
		p.advance()
		v := tok.Kind == lexer.TokenTrue
		return p.parsePostfix(&ast.BoolLit{Value: v, S: tok.Span}, allowStructLit)
	case lexer.TokenLParen:
		p.advance()
		ex := p.parseExprWith(0, allowStructLit)
		p.expect(lexer.TokenRParen, "expected `)`")
		return p.parsePostfix(ex, allowStructLit)
	case lexer.TokenMinus, lexer.TokenBang:
		p.advance()
		op := tok.Lexeme
		if op == "" {
			op = tokenOpString(tok.Kind)
		}
		ex := p.parseExprWith(7, allowStructLit)
		return p.parsePostfix(&ast.UnaryExpr{Op: op, Expr: ex, S: joinSpan(tok.Span, ex.Span())}, allowStructLit)
	case lexer.TokenMatch:
		p.advance()
		return p.parsePostfix(p.parseMatchExpr(tok), allowStructLit)
	case lexer.TokenIf:
		p.advance()
		return p.parsePostfix(p.parseIfExpr(tok), allowStructLit)
	case lexer.TokenLBrace:
		// Block expression: `{ stmt*; expr }` in expression position.
		p.advance()
		return p.parsePostfix(p.parseBlockExpr(tok), allowStructLit)
	default:
		p.errorHere("expected expression")
		p.advance()
		return &ast.IntLit{Text: "0", S: tok.Span}
	}
}

func (p *Parser) parseBlockExpr(lbrace lexer.Token) ast.Expr {
	stmts := []ast.Stmt{}
	var tail ast.Expr

	for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
		// Statement forms inside expression blocks.
		switch p.peek().Kind {
		case lexer.TokenLet, lexer.TokenIf, lexer.TokenWhile, lexer.TokenLBrace:
			st := p.parseStmt()
			if st != nil {
				stmts = append(stmts, st)
			} else {
				p.advance()
			}
			continue
		case lexer.TokenReturn, lexer.TokenBreak, lexer.TokenContinue:
			// These are legal statements generally, but block expressions are used as subexpressions.
			// Keep stage0 IR gen simple by rejecting top-level terminators here.
			p.errorHere("`return`/`break`/`continue` are not allowed in expression blocks (stage0)")
			p.advance()
			continue
		}

		// Expression: either an expr-stmt (`expr;`) or the tail (`expr` before `}`).
		ex := p.parseExprWith(0, true)
		if p.match(lexer.TokenSemicolon) {
			semi := p.prev()
			stmts = append(stmts, &ast.ExprStmt{Expr: ex, S: joinSpan(ex.Span(), semi.Span)})
			continue
		}
		tail = ex
		break
	}

	rb := p.expect(lexer.TokenRBrace, "expected `}` to end block expression")
	return &ast.BlockExpr{Stmts: stmts, Tail: tail, S: joinSpan(lbrace.Span, rb.Span)}
}

func (p *Parser) parseMatchExpr(matchTok lexer.Token) ast.Expr {
	scrut := p.parseExprNoStructLit(0)
	lb := p.expect(lexer.TokenLBrace, "expected `{` after match scrutinee")
	arms := []ast.MatchArm{}
	for !p.at(lexer.TokenRBrace) && !p.at(lexer.TokenEOF) {
		pat := p.parsePattern()
		arrow := p.expect(lexer.TokenFatArrow, "expected `=>` in match arm")
		ex := p.parseExprWith(0, true)
		end := ex.Span()
		if arrow.Kind == lexer.TokenFatArrow {
			end = ex.Span()
		}
		if p.match(lexer.TokenComma) {
			end = p.prev().Span
		}
		arms = append(arms, ast.MatchArm{Pat: pat, Expr: ex, S: joinSpan(pat.Span(), end)})
	}
	rb := p.expect(lexer.TokenRBrace, "expected `}` to end match")
	_ = lb
	return &ast.MatchExpr{Scrutinee: scrut, Arms: arms, S: joinSpan(matchTok.Span, rb.Span)}
}

func (p *Parser) parseIfExpr(ifTok lexer.Token) ast.Expr {
	// if <cond> { <expr> } else { <expr> }
	cond := p.parseExprNoStructLit(0)

	thenExpr, thenSpan := p.parseBracedExpr("expected `{` to start if-expression branch")
	if thenExpr == nil {
		thenExpr = &ast.IntLit{Text: "0", S: thenSpan}
	}

	p.expect(lexer.TokenElse, "expected `else` in if-expression")

	var elseExpr ast.Expr
	var endSpan source.Span
	if p.at(lexer.TokenIf) {
		t := p.advance()
		elseExpr = p.parseIfExpr(t)
		endSpan = elseExpr.Span()
	} else {
		elseExpr, endSpan = p.parseBracedExpr("expected `{` after else in if-expression")
		if elseExpr == nil {
			elseExpr = &ast.IntLit{Text: "0", S: endSpan}
		}
	}

	return &ast.IfExpr{Cond: cond, Then: thenExpr, Else: elseExpr, S: joinSpan(ifTok.Span, endSpan)}
}

func (p *Parser) parseBracedExpr(errMsg string) (ast.Expr, source.Span) {
	lb := p.expect(lexer.TokenLBrace, errMsg)
	if lb.Kind != lexer.TokenLBrace {
		return nil, lb.Span
	}
	if p.at(lexer.TokenRBrace) {
		p.errorHere("expected expression")
		rb := p.advance()
		return &ast.IntLit{Text: "0", S: joinSpan(lb.Span, rb.Span)}, joinSpan(lb.Span, rb.Span)
	}
	// Reuse block-expression parser for braced expression bodies.
	ex := p.parseBlockExpr(lb)
	return ex, ex.Span()
}

func (p *Parser) parsePattern() ast.Pattern {
	if p.at(lexer.TokenIdent) && p.peek().Lexeme == "_" {
		tok := p.advance()
		return &ast.WildPat{S: tok.Span}
	}
	if p.at(lexer.TokenInt) {
		tok := p.advance()
		return &ast.IntPat{Text: tok.Lexeme, S: tok.Span}
	}
	if p.at(lexer.TokenString) {
		tok := p.advance()
		return &ast.StrPat{Text: tok.Lexeme, S: tok.Span}
	}
	if p.at(lexer.TokenDot) {
		start := p.advance()
		id := p.expect(lexer.TokenIdent, "expected identifier after `.`")
		endSpan := id.Span
		binds := []string{}
		if p.match(lexer.TokenLParen) {
			if !p.at(lexer.TokenRParen) {
				for {
					b := p.expect(lexer.TokenIdent, "expected binder name")
					if b.Kind == lexer.TokenIdent {
						binds = append(binds, b.Lexeme)
					}
					if p.match(lexer.TokenComma) {
						continue
					}
					break
				}
			}
			rp := p.expect(lexer.TokenRParen, "expected `)`")
			endSpan = rp.Span
		}
		name := ""
		if id.Kind == lexer.TokenIdent {
			name = id.Lexeme
		}
		return &ast.VariantPat{TypeParts: nil, Variant: name, Binds: binds, S: joinSpan(start.Span, endSpan)}
	}
	start := p.peek()
	id := p.expect(lexer.TokenIdent, "expected pattern")
	parts := []string{}
	if id.Kind == lexer.TokenIdent {
		parts = append(parts, id.Lexeme)
	}
	endSpan := id.Span
	for p.match(lexer.TokenDot) {
		n := p.expect(lexer.TokenIdent, "expected identifier after `.`")
		if n.Kind == lexer.TokenIdent {
			parts = append(parts, n.Lexeme)
		}
		endSpan = n.Span
	}
	binds := []string{}
	if p.match(lexer.TokenLParen) {
		if !p.at(lexer.TokenRParen) {
			for {
				b := p.expect(lexer.TokenIdent, "expected binder name")
				if b.Kind == lexer.TokenIdent {
					binds = append(binds, b.Lexeme)
				}
				if p.match(lexer.TokenComma) {
					continue
				}
				break
			}
		}
		rp := p.expect(lexer.TokenRParen, "expected `)`")
		endSpan = rp.Span
	}
	if len(parts) < 2 {
		// Bind pattern: `name` always matches and binds the scrutinee to `name`.
		// `_` is handled earlier as WildPat.
		if len(binds) != 0 {
			p.errorAt(start.Span, "bind pattern does not take payload binders")
		}
		name := ""
		if len(parts) == 1 {
			name = parts[0]
		}
		return &ast.BindPat{Name: name, S: joinSpan(start.Span, endSpan)}
	}
	return &ast.VariantPat{TypeParts: parts[:len(parts)-1], Variant: parts[len(parts)-1], Binds: binds, S: joinSpan(start.Span, endSpan)}
}

func (p *Parser) parsePostfix(ex ast.Expr, allowStructLit bool) ast.Expr {
	var pendingTypeArgs []ast.Type
	for {
		if allowStructLit && p.at(lexer.TokenLBrace) {
			parts, ok := exprPathParts(ex)
			if ok {
				// Struct literals share the `{ ... }` token with blocks, so we only parse them when the
				// caller has indicated it is safe (i.e. not in `if cond { ... }` / `match x { ... }`).
				ex = p.parseStructLit(parts, ex.Span())
				continue
			}
		}
		if p.match(lexer.TokenDot) {
			if len(pendingTypeArgs) != 0 {
				p.errorHere("type arguments must be followed by a call")
				pendingTypeArgs = nil
			}
			id := p.expect(lexer.TokenIdent, "expected identifier after `.`")
			ex = &ast.MemberExpr{Recv: ex, Name: id.Lexeme, S: joinSpan(ex.Span(), id.Span)}
			continue
		}
		if p.match(lexer.TokenLBracket) {
			if len(pendingTypeArgs) != 0 {
				p.errorHere("unexpected nested type argument list")
			}
			pendingTypeArgs = nil
			if !p.at(lexer.TokenRBracket) {
				for {
					ta := p.parseType()
					pendingTypeArgs = append(pendingTypeArgs, ta)
					if p.match(lexer.TokenComma) {
						continue
					}
					break
				}
			}
			p.expect(lexer.TokenRBracket, "expected `]` to end type arguments")
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
			ex = &ast.CallExpr{Callee: ex, TypeArgs: pendingTypeArgs, Args: args, S: joinSpan(ex.Span(), rp.Span)}
			pendingTypeArgs = nil
			continue
		}
		break
	}
	if len(pendingTypeArgs) != 0 {
		p.errorHere("type arguments must be followed by a call")
	}
	return ex
}

func (p *Parser) parseStructLit(typeParts []string, typeSpan source.Span) ast.Expr {
	lb := p.expect(lexer.TokenLBrace, "expected `{`")
	inits := []ast.FieldInit{}
	if !p.at(lexer.TokenRBrace) {
		for {
			fname := p.expect(lexer.TokenIdent, "expected field name")
			p.expect(lexer.TokenColon, "expected `:` after field name")
			val := p.parseExpr(0)
			inits = append(inits, ast.FieldInit{Name: fname.Lexeme, Expr: val, Span: joinSpan(fname.Span, val.Span())})
			if p.match(lexer.TokenComma) {
				// allow trailing comma
				if p.at(lexer.TokenRBrace) {
					break
				}
				continue
			}
			break
		}
	}
	rb := p.expect(lexer.TokenRBrace, "expected `}`")
	_ = lb
	return &ast.StructLitExpr{TypeParts: typeParts, Inits: inits, S: joinSpan(typeSpan, rb.Span)}
}

func exprPathParts(ex ast.Expr) ([]string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return []string{e.Name}, true
	case *ast.MemberExpr:
		p, ok := exprPathParts(e.Recv)
		if !ok {
			return nil, false
		}
		return append(p, e.Name), true
	default:
		return nil, false
	}
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
