package ir

import (
	"fmt"
	"sort"
	"strings"
)

type TypeKind int

const (
	TBad TypeKind = iota
	TUnit
	TBool
	TI32
	TI64
)

type Type struct {
	K TypeKind
}

func (t Type) String() string {
	switch t.K {
	case TUnit:
		return "unit"
	case TBool:
		return "bool"
	case TI32:
		return "i32"
	case TI64:
		return "i64"
	default:
		return "<bad>"
	}
}

type Program struct {
	Funcs map[string]*Func
}

type Func struct {
	Name   string
	Params []Param
	Ret    Type
	Blocks []*Block
}

type Param struct {
	Name string
	Ty   Type
}

type Block struct {
	Name  string
	Instr []Instr
	Term  Term
}

type Instr interface {
	instrNode()
	fmtString() string
}

type Term interface {
	termNode()
	fmtString() string
}

// Values
type Value interface {
	valueNode()
	fmtString() string
}

// ParamRef refers to a function parameter by index.
// It formats as %pN in IR text.
type ParamRef struct {
	Index int
}

func (*ParamRef) valueNode() {}
func (p *ParamRef) fmtString() string {
	return fmt.Sprintf("%%p%d", p.Index)
}

type Temp struct {
	ID int
}

func (*Temp) valueNode() {}
func (t *Temp) fmtString() string {
	return fmt.Sprintf("%%t%d", t.ID)
}

type Slot struct {
	ID int
}

func (*Slot) valueNode() {}
func (s *Slot) fmtString() string {
	return fmt.Sprintf("$v%d", s.ID)
}

type ConstInt struct {
	Ty Type
	V  int64
}

func (*ConstInt) valueNode() {}
func (c *ConstInt) fmtString() string {
	return fmt.Sprintf("%d", c.V)
}

type ConstBool struct {
	V bool
}

func (*ConstBool) valueNode() {}
func (c *ConstBool) fmtString() string {
	if c.V {
		return "true"
	}
	return "false"
}

// Instructions
type Const struct {
	Dst *Temp
	Ty  Type
	Val Value // ConstInt/ConstBool only
}

func (*Const) instrNode() {}
func (i *Const) fmtString() string {
	return fmt.Sprintf("%s = const %s %s", i.Dst.fmtString(), i.Ty.String(), i.Val.fmtString())
}

type BinOpKind string

const (
	OpAdd BinOpKind = "add"
	OpSub BinOpKind = "sub"
	OpMul BinOpKind = "mul"
	OpDiv BinOpKind = "div"
	OpMod BinOpKind = "mod"
)

type BinOp struct {
	Dst *Temp
	Op  BinOpKind
	Ty  Type
	A   Value
	B   Value
}

func (*BinOp) instrNode() {}
func (i *BinOp) fmtString() string {
	return fmt.Sprintf("%s = %s %s %s %s", i.Dst.fmtString(), string(i.Op), i.Ty.String(), i.A.fmtString(), i.B.fmtString())
}

type CmpKind string

const (
	CmpLt CmpKind = "cmp_lt"
	CmpLe CmpKind = "cmp_le"
	CmpGt CmpKind = "cmp_gt"
	CmpGe CmpKind = "cmp_ge"
	CmpEq CmpKind = "cmp_eq"
	CmpNe CmpKind = "cmp_ne"
)

type Cmp struct {
	Dst *Temp
	Op  CmpKind
	Ty  Type // operand type
	A   Value
	B   Value
}

func (*Cmp) instrNode() {}
func (i *Cmp) fmtString() string {
	return fmt.Sprintf("%s = %s %s %s %s", i.Dst.fmtString(), string(i.Op), i.Ty.String(), i.A.fmtString(), i.B.fmtString())
}

type And struct {
	Dst *Temp
	A   Value
	B   Value
}

func (*And) instrNode() {}
func (i *And) fmtString() string {
	return fmt.Sprintf("%s = and %s %s", i.Dst.fmtString(), i.A.fmtString(), i.B.fmtString())
}

type Or struct {
	Dst *Temp
	A   Value
	B   Value
}

func (*Or) instrNode() {}
func (i *Or) fmtString() string {
	return fmt.Sprintf("%s = or %s %s", i.Dst.fmtString(), i.A.fmtString(), i.B.fmtString())
}

type Not struct {
	Dst *Temp
	A   Value
}

func (*Not) instrNode() {}
func (i *Not) fmtString() string {
	return fmt.Sprintf("%s = not %s", i.Dst.fmtString(), i.A.fmtString())
}

type SlotDecl struct {
	Slot *Slot
	Ty   Type
}

func (*SlotDecl) instrNode() {}
func (i *SlotDecl) fmtString() string {
	return fmt.Sprintf("%s = slot %s", i.Slot.fmtString(), i.Ty.String())
}

type Store struct {
	Slot *Slot
	Val  Value
}

func (*Store) instrNode() {}
func (i *Store) fmtString() string {
	return fmt.Sprintf("store %s %s", i.Slot.fmtString(), i.Val.fmtString())
}

type Load struct {
	Dst  *Temp
	Ty   Type
	Slot *Slot
}

func (*Load) instrNode() {}
func (i *Load) fmtString() string {
	return fmt.Sprintf("%s = load %s %s", i.Dst.fmtString(), i.Ty.String(), i.Slot.fmtString())
}

type Call struct {
	Dst  *Temp // optional when Ret is unit
	Ret  Type
	Name string
	Args []Value
}

func (*Call) instrNode() {}
func (i *Call) fmtString() string {
	var sb strings.Builder
	if i.Ret.K != TUnit {
		sb.WriteString(i.Dst.fmtString())
		sb.WriteString(" = ")
	}
	sb.WriteString("call ")
	sb.WriteString(i.Ret.String())
	sb.WriteByte(' ')
	sb.WriteString(i.Name)
	sb.WriteByte('(')
	for j, a := range i.Args {
		if j > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(a.fmtString())
	}
	sb.WriteByte(')')
	return sb.String()
}

// Terminators
type Ret struct {
	Val Value // optional for unit
}

func (*Ret) termNode() {}
func (t *Ret) fmtString() string {
	if t.Val == nil {
		return "ret"
	}
	return fmt.Sprintf("ret %s", t.Val.fmtString())
}

type Br struct {
	Target string
}

func (*Br) termNode()           {}
func (t *Br) fmtString() string { return fmt.Sprintf("br %s", t.Target) }

type CondBr struct {
	Cond Value
	Then string
	Else string
}

func (*CondBr) termNode() {}
func (t *CondBr) fmtString() string {
	return fmt.Sprintf("condbr %s %s %s", t.Cond.fmtString(), t.Then, t.Else)
}

func (p *Program) Format() string {
	var sb strings.Builder
	sb.WriteString("ir v0\n")
	if p == nil || len(p.Funcs) == 0 {
		return sb.String()
	}
	names := make([]string, 0, len(p.Funcs))
	for name := range p.Funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		f := p.Funcs[name]
		sb.WriteString("fn ")
		sb.WriteString(f.Name)
		sb.WriteByte('(')
		for i, p := range f.Params {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.Name)
			sb.WriteString(": ")
			sb.WriteString(p.Ty.String())
		}
		sb.WriteString(") -> ")
		sb.WriteString(f.Ret.String())
		sb.WriteByte('\n')
		for _, b := range f.Blocks {
			sb.WriteString("block ")
			sb.WriteString(b.Name)
			sb.WriteString(":\n")
			for _, ins := range b.Instr {
				sb.WriteString("  ")
				sb.WriteString(ins.fmtString())
				sb.WriteByte('\n')
			}
			if b.Term != nil {
				sb.WriteString("  ")
				sb.WriteString(b.Term.fmtString())
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}
