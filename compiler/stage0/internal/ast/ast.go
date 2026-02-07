package ast

import "voxlang/internal/source"

type Program struct {
	Funcs []*FuncDecl
}

type FuncDecl struct {
	Name   string
	Params []Param
	Ret    Type
	Body   *BlockStmt
	Span   source.Span
}

type Param struct {
	Name string
	Type Type
	Span source.Span
}

// Type is a syntactic type.
type Type interface {
	typeNode()
	Span() source.Span
}

type NamedType struct {
	Name string
	Args []Type // optional, for future generics
	S    source.Span
}

func (*NamedType) typeNode()           {}
func (t *NamedType) Span() source.Span { return t.S }

type UnitType struct {
	S source.Span
}

func (*UnitType) typeNode()           {}
func (t *UnitType) Span() source.Span { return t.S }

// Stmt
type Stmt interface {
	stmtNode()
	Span() source.Span
}

type BlockStmt struct {
	Stmts []Stmt
	S     source.Span
}

func (*BlockStmt) stmtNode()           {}
func (s *BlockStmt) Span() source.Span { return s.S }

type LetStmt struct {
	Mutable bool
	Name    string
	AnnType Type // optional
	Init    Expr // optional
	S       source.Span
}

func (*LetStmt) stmtNode()           {}
func (s *LetStmt) Span() source.Span { return s.S }

type AssignStmt struct {
	Name string
	Expr Expr
	S    source.Span
}

func (*AssignStmt) stmtNode()           {}
func (s *AssignStmt) Span() source.Span { return s.S }

type ReturnStmt struct {
	Expr Expr // optional
	S    source.Span
}

func (*ReturnStmt) stmtNode()           {}
func (s *ReturnStmt) Span() source.Span { return s.S }

type IfStmt struct {
	Cond Expr
	Then *BlockStmt
	Else Stmt // either *BlockStmt or *IfStmt, optional
	S    source.Span
}

func (*IfStmt) stmtNode()           {}
func (s *IfStmt) Span() source.Span { return s.S }

type ExprStmt struct {
	Expr Expr
	S    source.Span
}

func (*ExprStmt) stmtNode()           {}
func (s *ExprStmt) Span() source.Span { return s.S }

// Expr
type Expr interface {
	exprNode()
	Span() source.Span
}

type IdentExpr struct {
	Name string
	S    source.Span
}

func (*IdentExpr) exprNode()           {}
func (e *IdentExpr) Span() source.Span { return e.S }

// PathExpr represents a qualified path like `dep::foo`.
// Stage0 uses it only for function calls into dependency packages.
type PathExpr struct {
	Parts []string
	S     source.Span
}

func (*PathExpr) exprNode()           {}
func (e *PathExpr) Span() source.Span { return e.S }

type IntLit struct {
	Text string
	S    source.Span
}

func (*IntLit) exprNode()           {}
func (e *IntLit) Span() source.Span { return e.S }

type StringLit struct {
	Text string
	S    source.Span
}

func (*StringLit) exprNode()           {}
func (e *StringLit) Span() source.Span { return e.S }

type BoolLit struct {
	Value bool
	S     source.Span
}

func (*BoolLit) exprNode()           {}
func (e *BoolLit) Span() source.Span { return e.S }

type UnaryExpr struct {
	Op   string
	Expr Expr
	S    source.Span
}

func (*UnaryExpr) exprNode()           {}
func (e *UnaryExpr) Span() source.Span { return e.S }

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
	S     source.Span
}

func (*BinaryExpr) exprNode()           {}
func (e *BinaryExpr) Span() source.Span { return e.S }

type CallExpr struct {
	Callee Expr
	Args   []Expr
	S      source.Span
}

func (*CallExpr) exprNode()           {}
func (e *CallExpr) Span() source.Span { return e.S }
