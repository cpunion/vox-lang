package ast

import "voxlang/internal/source"

type Program struct {
	Imports []*ImportDecl
	Types   []*TypeAliasDecl
	Consts  []*ConstDecl
	Structs []*StructDecl
	Enums   []*EnumDecl
	Traits  []*TraitDecl
	Impls   []*ImplDecl
	Funcs   []*FuncDecl
}

type Visibility uint8

const (
	VisPrivate Visibility = iota
	VisPub
	VisCrate
	VisSuper
)

func (v Visibility) IsPub() bool { return v != VisPrivate }

type ImportDecl struct {
	Path  string // string literal content, unquoted
	Alias string // optional; when empty, defaults to last path segment
	Names []ImportName
	Span  source.Span
}

type ImportName struct {
	Name  string
	Alias string // optional local alias
	Span  source.Span
}

type TypeAliasDecl struct {
	Vis  Visibility
	Pub  bool
	Name string
	Type Type
	Span source.Span
}

type ConstDecl struct {
	Vis  Visibility
	Pub  bool
	Name string
	Type Type
	Expr Expr
	Span source.Span
}

type FuncDecl struct {
	Vis        Visibility
	Pub        bool
	Name       string
	TypeParams []string // generic type parameters, e.g. fn id[T](x: T) -> T
	Params     []Param
	Ret        Type
	Body       *BlockStmt
	Span       source.Span
}

type StructDecl struct {
	Vis        Visibility
	Pub        bool
	Name       string
	TypeParams []string // generic type parameters, e.g. struct Pair[T] { ... }
	Fields     []StructField
	Span       source.Span
}

type StructField struct {
	Vis  Visibility
	Pub  bool
	Name string
	Type Type
	Span source.Span
}

type EnumDecl struct {
	Vis        Visibility
	Pub        bool
	Name       string
	TypeParams []string // generic type parameters, e.g. enum Option[T] { ... }
	Variants   []EnumVariant
	Span       source.Span
}

type EnumVariant struct {
	Name   string
	Fields []Type // tuple-like payload, arity 0..N
	Span   source.Span
}

type TraitDecl struct {
	Vis     Visibility
	Pub     bool
	Name    string
	Methods []TraitMethodSig
	Span    source.Span
}

type TraitMethodSig struct {
	Name   string
	Params []Param
	Ret    Type
	Span   source.Span
}

type ImplDecl struct {
	Trait   Type
	ForType Type
	Methods []*FuncDecl
	Span    source.Span
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
	Parts []string
	Args  []Type // optional, for future generics
	S     source.Span
}

func (*NamedType) typeNode()           {}
func (t *NamedType) Span() source.Span { return t.S }

type UnitType struct {
	S source.Span
}

func (*UnitType) typeNode()           {}
func (t *UnitType) Span() source.Span { return t.S }

// RangeType is a refined integer type: `@range(lo..=hi) Base`.
//
// Stage0 v0 restriction: Base must be i32 or i64; bounds are non-negative integer literals.
type RangeType struct {
	Lo   int64
	Hi   int64
	Base Type
	S    source.Span
}

func (*RangeType) typeNode()           {}
func (t *RangeType) Span() source.Span { return t.S }

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

type FieldAssignStmt struct {
	Recv  string // stage0: only `ident.field = expr;`
	Field string
	Expr  Expr
	S     source.Span
}

func (*FieldAssignStmt) stmtNode()           {}
func (s *FieldAssignStmt) Span() source.Span { return s.S }

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

type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
	S    source.Span
}

func (*WhileStmt) stmtNode()           {}
func (s *WhileStmt) Span() source.Span { return s.S }

type BreakStmt struct {
	S source.Span
}

func (*BreakStmt) stmtNode()           {}
func (s *BreakStmt) Span() source.Span { return s.S }

type ContinueStmt struct {
	S source.Span
}

func (*ContinueStmt) stmtNode()           {}
func (s *ContinueStmt) Span() source.Span { return s.S }

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

// DotExpr is an enum-variant shorthand when the enum type is known from context.
// Examples: `.None`, `.Some(1)` (as CallExpr with callee DotExpr).
type DotExpr struct {
	Name string
	S    source.Span
}

func (*DotExpr) exprNode()           {}
func (e *DotExpr) Span() source.Span { return e.S }

type MemberExpr struct {
	Recv Expr
	Name string
	S    source.Span
}

func (*MemberExpr) exprNode()           {}
func (e *MemberExpr) Span() source.Span { return e.S }

// TypeAppExpr carries type arguments on a path expression before member/call/struct-literal disambiguation.
// Example source shape: `Option[i32].Some`.
type TypeAppExpr struct {
	Expr     Expr
	TypeArgs []Type
	S        source.Span
}

func (*TypeAppExpr) exprNode()           {}
func (e *TypeAppExpr) Span() source.Span { return e.S }

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

// AsExpr is a type cast:
//
//	expr as Type
//
// Stage0 v0 only supports numeric casts between i32 and i64 (with checked downcast).
type AsExpr struct {
	Expr Expr
	Ty   Type
	S    source.Span
}

func (*AsExpr) exprNode()           {}
func (e *AsExpr) Span() source.Span { return e.S }

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
	S     source.Span
}

func (*BinaryExpr) exprNode()           {}
func (e *BinaryExpr) Span() source.Span { return e.S }

type CallExpr struct {
	Callee   Expr
	TypeArgs []Type // optional explicit type args for generic calls: f[i32](...)
	Args     []Expr
	S        source.Span
}

func (*CallExpr) exprNode()           {}
func (e *CallExpr) Span() source.Span { return e.S }

type MatchExpr struct {
	Scrutinee Expr
	Arms      []MatchArm
	S         source.Span
}

type MatchArm struct {
	Pat  Pattern
	Expr Expr
	S    source.Span
}

func (*MatchExpr) exprNode()           {}
func (e *MatchExpr) Span() source.Span { return e.S }

// IfExpr is an expression-form if:
//
//	if cond { thenExpr } else { elseExpr }
//
// Branch bodies are expressions; they may be block expressions.
type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
	S    source.Span
}

func (*IfExpr) exprNode()           {}
func (e *IfExpr) Span() source.Span { return e.S }

// BlockExpr is an expression-form block:
//
//	{ stmt*; tailExpr }
//
// The tail expression is optional; when omitted the block's value is unit.
type BlockExpr struct {
	Stmts []Stmt
	Tail  Expr // optional
	S     source.Span
}

func (*BlockExpr) exprNode()           {}
func (e *BlockExpr) Span() source.Span { return e.S }

type StructLitExpr struct {
	TypeParts []string
	TypeArgs  []Type
	Inits     []FieldInit
	S         source.Span
}

type FieldInit struct {
	Name string
	Expr Expr
	Span source.Span
}

func (*StructLitExpr) exprNode()           {}
func (e *StructLitExpr) Span() source.Span { return e.S }

// Pattern (used by match)
type Pattern interface {
	patNode()
	Span() source.Span
}

type WildPat struct {
	S source.Span
}

func (*WildPat) patNode()            {}
func (p *WildPat) Span() source.Span { return p.S }

type VariantPat struct {
	TypeParts []string // enum type path segments
	Variant   string
	Args      []Pattern // payload patterns (arity 0..N)
	S         source.Span
}

func (*VariantPat) patNode()            {}
func (p *VariantPat) Span() source.Span { return p.S }

// BindPat always matches and binds the scrutinee to Name (like a named `_`).
type BindPat struct {
	Name string
	S    source.Span
}

func (*BindPat) patNode()            {}
func (p *BindPat) Span() source.Span { return p.S }

type IntPat struct {
	Text string
	S    source.Span
}

func (*IntPat) patNode()            {}
func (p *IntPat) Span() source.Span { return p.S }

type BoolPat struct {
	Value bool
	S     source.Span
}

func (*BoolPat) patNode()            {}
func (p *BoolPat) Span() source.Span { return p.S }

type StrPat struct {
	Text string
	S    source.Span
}

func (*StrPat) patNode()            {}
func (p *StrPat) Span() source.Span { return p.S }
