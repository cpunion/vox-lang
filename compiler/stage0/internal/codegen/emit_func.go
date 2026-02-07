package codegen

import (
	"bytes"
	"fmt"
	"sort"

	"voxlang/internal/ir"
)

func emitFunc(out *bytes.Buffer, p *ir.Program, f *ir.Func) error {
	// Collect locals (slots + temps)
	slotTypes := map[int]ir.Type{}
	tempTypes := map[int]ir.Type{}
	for _, b := range f.Blocks {
		for _, ins := range b.Instr {
			switch i := ins.(type) {
			case *ir.SlotDecl:
				slotTypes[i.Slot.ID] = i.Ty
			case *ir.Const:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.BinOp:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.Cmp:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TBool}
			case *ir.And:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TBool}
			case *ir.Or:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TBool}
			case *ir.Not:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TBool}
			case *ir.Load:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.StructInit:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.FieldGet:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.EnumInit:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.EnumTag:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TI32}
			case *ir.EnumPayload:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.VecNew:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.VecLen:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TI32}
			case *ir.VecGet:
				tempTypes[i.Dst.ID] = i.Ty
			case *ir.Call:
				if i.Ret.K != ir.TUnit && i.Dst != nil {
					tempTypes[i.Dst.ID] = i.Ret
				}
			}
		}
	}

	out.WriteString("static ")
	out.WriteString(cType(f.Ret))
	out.WriteByte(' ')
	out.WriteString(cFnName(f.Name))
	out.WriteByte('(')
	for i, pa := range f.Params {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(cType(pa.Ty))
		out.WriteByte(' ')
		out.WriteString(cParamName(i, pa.Name))
	}
	out.WriteString(") {\n")

	// Declare slots
	slotIDs := make([]int, 0, len(slotTypes))
	for id := range slotTypes {
		slotIDs = append(slotIDs, id)
	}
	sort.Ints(slotIDs)
	for _, id := range slotIDs {
		out.WriteString("  ")
		out.WriteString(cType(slotTypes[id]))
		out.WriteByte(' ')
		out.WriteString(cSlotName(id))
		out.WriteString(";\n")
	}

	// Declare temps
	tempIDs := make([]int, 0, len(tempTypes))
	for id := range tempTypes {
		tempIDs = append(tempIDs, id)
	}
	sort.Ints(tempIDs)
	for _, id := range tempIDs {
		out.WriteString("  ")
		out.WriteString(cType(tempTypes[id]))
		out.WriteByte(' ')
		out.WriteString(cTempName(id))
		out.WriteString(";\n")
	}

	if len(slotIDs) > 0 || len(tempIDs) > 0 {
		out.WriteString("\n")
	}

	// Emit blocks
	for _, b := range f.Blocks {
		out.WriteString(cLabelName(b.Name))
		out.WriteString(":\n")
		for _, ins := range b.Instr {
			if err := emitInstr(out, p, ins); err != nil {
				return err
			}
		}
		if b.Term == nil {
			return fmt.Errorf("block %s missing terminator", b.Name)
		}
		if err := emitTerm(out, b.Term); err != nil {
			return err
		}
		out.WriteString("\n")
	}

	out.WriteString("}\n")
	return nil
}

func emitInstr(out *bytes.Buffer, p *ir.Program, ins ir.Instr) error {
	switch i := ins.(type) {
	case *ir.SlotDecl:
		// already declared as a C local
		return nil
	case *ir.Const:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Val))
		out.WriteString(";\n")
		return nil
	case *ir.BinOp:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.A))
		out.WriteByte(' ')
		out.WriteString(stringToCOp(string(i.Op)))
		out.WriteByte(' ')
		out.WriteString(cValue(i.B))
		out.WriteString(";\n")
		return nil
	case *ir.Cmp:
		if i.Ty.K == ir.TString {
			// Only equality is supported for strings in stage0 backend.
			if i.Op != ir.CmpEq && i.Op != ir.CmpNe {
				return fmt.Errorf("unsupported string comparison")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = (strcmp(")
			out.WriteString(cValue(i.A))
			out.WriteString(", ")
			out.WriteString(cValue(i.B))
			out.WriteString(") ")
			if i.Op == ir.CmpEq {
				out.WriteString("==")
			} else {
				out.WriteString("!=")
			}
			out.WriteString(" 0);\n")
			return nil
		}
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (")
		out.WriteString(cValue(i.A))
		out.WriteByte(' ')
		out.WriteString(cCmpOp(i.Op))
		out.WriteByte(' ')
		out.WriteString(cValue(i.B))
		out.WriteString(");\n")
		return nil
	case *ir.And:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (")
		out.WriteString(cValue(i.A))
		out.WriteString(" && ")
		out.WriteString(cValue(i.B))
		out.WriteString(");\n")
		return nil
	case *ir.Or:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (")
		out.WriteString(cValue(i.A))
		out.WriteString(" || ")
		out.WriteString(cValue(i.B))
		out.WriteString(");\n")
		return nil
	case *ir.Not:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (!")
		out.WriteString(cValue(i.A))
		out.WriteString(");\n")
		return nil
	case *ir.Store:
		out.WriteString("  ")
		out.WriteString(cSlotName(i.Slot.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Val))
		out.WriteString(";\n")
		return nil
	case *ir.Load:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cSlotName(i.Slot.ID))
		out.WriteString(";\n")
		return nil
	case *ir.StructInit:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (")
		out.WriteString(cType(i.Ty))
		out.WriteString("){")
		for j, f := range i.Fields {
			if j > 0 {
				out.WriteString(", ")
			}
			out.WriteByte('.')
			out.WriteString(cIdent(f.Name))
			out.WriteString(" = ")
			out.WriteString(cValue(f.Val))
		}
		out.WriteString("};\n")
		return nil
	case *ir.FieldGet:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Recv))
		out.WriteByte('.')
		out.WriteString(cIdent(i.Field))
		out.WriteString(";\n")
		return nil
	case *ir.StoreField:
		out.WriteString("  ")
		out.WriteString(cSlotName(i.Slot.ID))
		out.WriteByte('.')
		out.WriteString(cIdent(i.Field))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Val))
		out.WriteString(";\n")
		return nil
	case *ir.EnumInit:
		en, ok := p.Enums[i.Ty.Name]
		if !ok || en == nil {
			return fmt.Errorf("unknown enum: %s", i.Ty.Name)
		}
		tag, ok := en.VariantIndex[i.Variant]
		if !ok {
			return fmt.Errorf("unknown enum variant: %s.%s", i.Ty.Name, i.Variant)
		}
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = (")
		out.WriteString(cType(i.Ty))
		out.WriteString("){.tag = ")
		out.WriteString(fmt.Sprintf("%d", tag))
		// payload
		hasPayloadUnion := false
		for _, v := range en.Variants {
			if v.Payload != nil {
				hasPayloadUnion = true
				break
			}
		}
		if hasPayloadUnion {
			out.WriteString(", .payload.")
			out.WriteString(cIdent(i.Variant))
			out.WriteString(" = {")
			if i.Payload != nil {
				out.WriteString("._0 = ")
				out.WriteString(cValue(i.Payload))
			} else {
				out.WriteString("._ = 0")
			}
			out.WriteString("}")
		}
		out.WriteString("};\n")
		return nil
	case *ir.EnumTag:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Recv))
		out.WriteString(".tag;\n")
		return nil
	case *ir.EnumPayload:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = ")
		out.WriteString(cValue(i.Recv))
		out.WriteString(".payload.")
		out.WriteString(cIdent(i.Variant))
		out.WriteString("._0;\n")
		return nil
	case *ir.VecNew:
		if i.Ty.K != ir.TVec || i.Ty.Elem == nil {
			return fmt.Errorf("vec_new expects vec type")
		}
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_vec_new((int32_t)sizeof(")
		out.WriteString(cType(*i.Ty.Elem))
		out.WriteString("));\n")
		return nil
	case *ir.VecPush:
		out.WriteString("  vox_vec_push(&")
		out.WriteString(cSlotName(i.Recv.ID))
		out.WriteString(", &")
		out.WriteString(cValue(i.Val))
		out.WriteString(");\n")
		return nil
	case *ir.VecLen:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_vec_len(&")
		out.WriteString(cSlotName(i.Recv.ID))
		out.WriteString(");\n")
		return nil
	case *ir.VecGet:
		out.WriteString("  vox_vec_get(&")
		out.WriteString(cSlotName(i.Recv.ID))
		out.WriteString(", ")
		out.WriteString(cValue(i.Idx))
		out.WriteString(", &")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(");\n")
		return nil
	case *ir.Call:
		// Stage0 builtins: panic/print only.
		if i.Name == "panic" {
			if len(i.Args) != 1 {
				return fmt.Errorf("panic expects 1 arg")
			}
			out.WriteString("  vox_builtin_panic(")
			out.WriteString(cValue(i.Args[0]))
			out.WriteString(");\n")
			return nil
		}
		if i.Name == "print" {
			if len(i.Args) != 1 {
				return fmt.Errorf("print expects 1 arg")
			}
			out.WriteString("  vox_builtin_print(")
			out.WriteString(cValue(i.Args[0]))
			out.WriteString(");\n")
			return nil
		}
		out.WriteString("  ")
		if i.Ret.K != ir.TUnit {
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = ")
		}
		out.WriteString(cFnName(i.Name))
		out.WriteByte('(')
		for j, a := range i.Args {
			if j > 0 {
				out.WriteString(", ")
			}
			out.WriteString(cValue(a))
		}
		out.WriteString(");\n")
		return nil
	default:
		return fmt.Errorf("unsupported instr in codegen")
	}
}

func emitTerm(out *bytes.Buffer, t ir.Term) error {
	switch tt := t.(type) {
	case *ir.Ret:
		out.WriteString("  return")
		if tt.Val != nil {
			out.WriteByte(' ')
			out.WriteString(cValue(tt.Val))
		}
		out.WriteString(";\n")
		return nil
	case *ir.Br:
		out.WriteString("  goto ")
		out.WriteString(cLabelName(tt.Target))
		out.WriteString(";\n")
		return nil
	case *ir.CondBr:
		out.WriteString("  if (")
		out.WriteString(cValue(tt.Cond))
		out.WriteString(") goto ")
		out.WriteString(cLabelName(tt.Then))
		out.WriteString("; else goto ")
		out.WriteString(cLabelName(tt.Else))
		out.WriteString(";\n")
		return nil
	default:
		return fmt.Errorf("unsupported terminator")
	}
}
