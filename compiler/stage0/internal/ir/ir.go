package ir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TypeKind int

const (
	TBad TypeKind = iota
	TUnit
	TBool
	TI32
	TI64
	TString
	TStruct
	TEnum
	TVec
)

type Type struct {
	K TypeKind
	// Name is set when K == TStruct or K == TEnum (qualified name).
	Name string
	// Elem is set when K == TVec.
	Elem *Type
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
	case TString:
		return "str"
	case TStruct:
		return "struct(" + t.Name + ")"
	case TEnum:
		return "enum(" + t.Name + ")"
	case TVec:
		if t.Elem == nil {
			return "vec(<bad>)"
		}
		return "vec(" + t.Elem.String() + ")"
	default:
		return "<bad>"
	}
}

type Program struct {
	Structs map[string]*Struct
	Enums   map[string]*Enum
	Funcs   map[string]*Func
}

type Struct struct {
	Name   string
	Fields []StructField
}

type StructField struct {
	Name string
	Ty   Type
}

type Enum struct {
	Name         string
	Variants     []EnumVariant
	VariantIndex map[string]int
}

type EnumVariant struct {
	Name   string
	Fields []Type // empty for unit variants
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

type ConstStr struct {
	S string // unescaped content
}

func (*ConstStr) valueNode() {}
func (c *ConstStr) fmtString() string {
	// Keep IR round-trippable and readable.
	return strconv.Quote(c.S)
}

// Instructions
type Const struct {
	Dst *Temp
	Ty  Type
	Val Value // ConstInt/ConstBool/ConstStr only
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

type StructInit struct {
	Dst    *Temp
	Ty     Type // struct type
	Fields []StructInitField
}

type StructInitField struct {
	Name string
	Val  Value
}

func (*StructInit) instrNode() {}
func (i *StructInit) fmtString() string {
	var sb strings.Builder
	sb.WriteString(i.Dst.fmtString())
	sb.WriteString(" = struct_init ")
	sb.WriteString(i.Ty.String())
	sb.WriteByte(' ')
	sb.WriteByte('{')
	for j, f := range i.Fields {
		if j > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.Name)
		sb.WriteString(": ")
		sb.WriteString(f.Val.fmtString())
	}
	sb.WriteByte('}')
	return sb.String()
}

type FieldGet struct {
	Dst   *Temp
	Ty    Type
	Recv  Value
	Field string
}

func (*FieldGet) instrNode() {}
func (i *FieldGet) fmtString() string {
	return fmt.Sprintf("%s = field_get %s %s .%s", i.Dst.fmtString(), i.Ty.String(), i.Recv.fmtString(), i.Field)
}

type StoreField struct {
	Slot  *Slot
	Field string
	Val   Value
}

func (*StoreField) instrNode() {}
func (i *StoreField) fmtString() string {
	return fmt.Sprintf("store_field %s .%s %s", i.Slot.fmtString(), i.Field, i.Val.fmtString())
}

type EnumInit struct {
	Dst     *Temp
	Ty      Type   // enum type
	Variant string // variant name
	Payload []Value // empty for unit variants
}

func (*EnumInit) instrNode() {}
func (i *EnumInit) fmtString() string {
	var sb strings.Builder
	sb.WriteString(i.Dst.fmtString())
	sb.WriteString(" = enum_init ")
	sb.WriteString(i.Ty.String())
	sb.WriteByte(' ')
	sb.WriteString(i.Variant)
	if len(i.Payload) != 0 {
		sb.WriteByte('(')
		for idx, v := range i.Payload {
			if idx != 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(v.fmtString())
		}
		sb.WriteByte(')')
	}
	return sb.String()
}

type EnumTag struct {
	Dst  *Temp
	Recv Value
}

func (*EnumTag) instrNode() {}
func (i *EnumTag) fmtString() string {
	return fmt.Sprintf("%s = enum_tag %s", i.Dst.fmtString(), i.Recv.fmtString())
}

type EnumPayload struct {
	Dst     *Temp
	Ty      Type
	Recv    Value
	Variant string
	Index   int
}

func (*EnumPayload) instrNode() {}
func (i *EnumPayload) fmtString() string {
	return fmt.Sprintf("%s = enum_payload %s %s %s %d", i.Dst.fmtString(), i.Ty.String(), i.Recv.fmtString(), i.Variant, i.Index)
}

type VecNew struct {
	Dst  *Temp
	Ty   Type // vec type
	Elem Type // element type
}

func (*VecNew) instrNode() {}
func (i *VecNew) fmtString() string {
	return fmt.Sprintf("%s = vec_new %s", i.Dst.fmtString(), i.Ty.String())
}

type VecPush struct {
	Recv *Slot // mutated in-place
	Elem Type
	Val  Value
}

func (*VecPush) instrNode() {}
func (i *VecPush) fmtString() string {
	return fmt.Sprintf("vec_push %s %s", i.Recv.fmtString(), i.Val.fmtString())
}

type VecLen struct {
	Dst  *Temp
	Recv *Slot
}

func (*VecLen) instrNode() {}
func (i *VecLen) fmtString() string {
	return fmt.Sprintf("%s = vec_len %s", i.Dst.fmtString(), i.Recv.fmtString())
}

type VecGet struct {
	Dst  *Temp
	Ty   Type
	Recv *Slot
	Idx  Value
}

func (*VecGet) instrNode() {}
func (i *VecGet) fmtString() string {
	return fmt.Sprintf("%s = vec_get %s %s %s", i.Dst.fmtString(), i.Ty.String(), i.Recv.fmtString(), i.Idx.fmtString())
}

// String intrinsics (stage0)
type StrLen struct {
	Dst  *Temp
	Recv Value
}

func (*StrLen) instrNode() {}
func (i *StrLen) fmtString() string {
	return fmt.Sprintf("%s = str_len %s", i.Dst.fmtString(), i.Recv.fmtString())
}

type StrByteAt struct {
	Dst  *Temp
	Recv Value
	Idx  Value
}

func (*StrByteAt) instrNode() {}
func (i *StrByteAt) fmtString() string {
	return fmt.Sprintf("%s = str_byte_at %s %s", i.Dst.fmtString(), i.Recv.fmtString(), i.Idx.fmtString())
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
	if p == nil {
		return sb.String()
	}

	if len(p.Structs) > 0 {
		snames := make([]string, 0, len(p.Structs))
		for name := range p.Structs {
			snames = append(snames, name)
		}
		sort.Strings(snames)
		for _, name := range snames {
			st := p.Structs[name]
			sb.WriteString("struct ")
			sb.WriteString(st.Name)
			sb.WriteString(" { ")
			for i, f := range st.Fields {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(f.Name)
				sb.WriteString(": ")
				sb.WriteString(f.Ty.String())
			}
			sb.WriteString(" }\n")
		}
	}

	if len(p.Enums) > 0 {
		enames := make([]string, 0, len(p.Enums))
		for name := range p.Enums {
			enames = append(enames, name)
		}
		sort.Strings(enames)
		for _, name := range enames {
			en := p.Enums[name]
			sb.WriteString("enum ")
			sb.WriteString(en.Name)
			sb.WriteString(" { ")
			for i, v := range en.Variants {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(v.Name)
				if len(v.Fields) != 0 {
					sb.WriteByte('(')
					for j, ft := range v.Fields {
						if j != 0 {
							sb.WriteString(", ")
						}
						sb.WriteString(ft.String())
					}
					sb.WriteByte(')')
				}
			}
			sb.WriteString(" }\n")
		}
	}

	if len(p.Funcs) == 0 {
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
