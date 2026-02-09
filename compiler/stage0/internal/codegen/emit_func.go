package codegen

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

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
			case *ir.IntCastChecked:
				tempTypes[i.Dst.ID] = i.To
			case *ir.IntCast:
				tempTypes[i.Dst.ID] = i.To
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
			case *ir.VecStrJoin:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.StrLen:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TI32}
			case *ir.StrByteAt:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TI32}
			case *ir.StrSlice:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.StrConcat:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.StrEscapeC:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.I32ToStr:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.I64ToStr:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.U64ToStr:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
			case *ir.BoolToStr:
				tempTypes[i.Dst.ID] = ir.Type{K: ir.TString}
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
		if isIntType(i.Ty) {
			return emitIntBinOp(out, i)
		}
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
	case *ir.IntCastChecked:
		return emitIntCastChecked(out, i)
	case *ir.IntCast:
		return emitIntCast(out, i)
	case *ir.RangeCheckInt:
		return emitRangeCheckInt(out, i)
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
		vidx := tag
		if vidx < 0 || vidx >= len(en.Variants) {
			return fmt.Errorf("invalid enum variant index: %s.%s", i.Ty.Name, i.Variant)
		}
		v := en.Variants[vidx]
		if len(i.Payload) != len(v.Fields) {
			return fmt.Errorf("enum_init payload arity mismatch: %s.%s expects %d fields, got %d", i.Ty.Name, i.Variant, len(v.Fields), len(i.Payload))
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
			if len(v.Fields) != 0 {
				hasPayloadUnion = true
				break
			}
		}
		if hasPayloadUnion {
			out.WriteString(", .payload.")
			out.WriteString(cIdent(i.Variant))
			out.WriteString(" = {")
			if len(i.Payload) != 0 {
				for idx, pv := range i.Payload {
					if idx != 0 {
						out.WriteString(", ")
					}
					out.WriteString(fmt.Sprintf("._%d = ", idx))
					out.WriteString(cValue(pv))
				}
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
		out.WriteString(fmt.Sprintf("._%d;\n", i.Index))
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
	case *ir.VecStrJoin:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_vec_str_join(&")
		out.WriteString(cSlotName(i.Recv.ID))
		out.WriteString(", ")
		out.WriteString(cValue(i.Sep))
		out.WriteString(");\n")
		return nil
	case *ir.StrLen:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_str_len(")
		out.WriteString(cValue(i.Recv))
		out.WriteString(");\n")
		return nil
	case *ir.StrByteAt:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_str_byte_at(")
		out.WriteString(cValue(i.Recv))
		out.WriteString(", ")
		out.WriteString(cValue(i.Idx))
		out.WriteString(");\n")
		return nil
	case *ir.StrSlice:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_str_slice(")
		out.WriteString(cValue(i.Recv))
		out.WriteString(", ")
		out.WriteString(cValue(i.Start))
		out.WriteString(", ")
		out.WriteString(cValue(i.End))
		out.WriteString(");\n")
		return nil
	case *ir.StrConcat:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_str_concat(")
		out.WriteString(cValue(i.A))
		out.WriteString(", ")
		out.WriteString(cValue(i.B))
		out.WriteString(");\n")
		return nil
	case *ir.StrEscapeC:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_str_escape_c(")
		out.WriteString(cValue(i.Recv))
		out.WriteString(");\n")
		return nil
	case *ir.I32ToStr:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_i32_to_string(")
		out.WriteString(cValue(i.V))
		out.WriteString(");\n")
		return nil
	case *ir.I64ToStr:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_i64_to_string(")
		out.WriteString(cValue(i.V))
		out.WriteString(");\n")
		return nil
	case *ir.U64ToStr:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_u64_to_string(")
		out.WriteString(cValue(i.V))
		out.WriteString(");\n")
		return nil
	case *ir.BoolToStr:
		out.WriteString("  ")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(" = vox_bool_to_string(")
		out.WriteString(cValue(i.V))
		out.WriteString(");\n")
		return nil
	case *ir.Call:
		// Stage0 builtins.
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
		if i.Name == "__args" {
			if len(i.Args) != 0 {
				return fmt.Errorf("__args expects 0 args")
			}
			if i.Ret.K == ir.TUnit {
				return fmt.Errorf("__args must return a value")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = vox_builtin_args();\n")
			return nil
		}
		if i.Name == "__exe_path" {
			if len(i.Args) != 0 {
				return fmt.Errorf("__exe_path expects 0 args")
			}
			if i.Ret.K == ir.TUnit {
				return fmt.Errorf("__exe_path must return a value")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = vox_builtin_exe_path();\n")
			return nil
		}
		if i.Name == "__read_file" {
			if len(i.Args) != 1 {
				return fmt.Errorf("__read_file expects 1 arg")
			}
			if i.Ret.K == ir.TUnit {
				return fmt.Errorf("__read_file must return a value")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = vox_builtin_read_file(")
			out.WriteString(cValue(i.Args[0]))
			out.WriteString(");\n")
			return nil
		}
		if i.Name == "__write_file" {
			if len(i.Args) != 2 {
				return fmt.Errorf("__write_file expects 2 args")
			}
			out.WriteString("  vox_builtin_write_file(")
			out.WriteString(cValue(i.Args[0]))
			out.WriteString(", ")
			out.WriteString(cValue(i.Args[1]))
			out.WriteString(");\n")
			return nil
		}
		if i.Name == "__exec" {
			if len(i.Args) != 1 {
				return fmt.Errorf("__exec expects 1 arg")
			}
			if i.Ret.K == ir.TUnit {
				return fmt.Errorf("__exec must return a value")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = vox_builtin_exec(")
			out.WriteString(cValue(i.Args[0]))
			out.WriteString(");\n")
			return nil
		}
		if i.Name == "__walk_vox_files" {
			if len(i.Args) != 1 {
				return fmt.Errorf("__walk_vox_files expects 1 arg")
			}
			if i.Ret.K == ir.TUnit {
				return fmt.Errorf("__walk_vox_files must return a value")
			}
			out.WriteString("  ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = vox_builtin_walk_vox_files(")
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

func isIntType(t ir.Type) bool {
	switch t.K {
	case ir.TI8, ir.TU8, ir.TI16, ir.TU16, ir.TI32, ir.TU32, ir.TI64, ir.TU64, ir.TISize, ir.TUSize:
		return true
	default:
		return false
	}
}

func intBits(k ir.TypeKind) int {
	switch k {
	case ir.TI8, ir.TU8:
		return 8
	case ir.TI16, ir.TU16:
		return 16
	case ir.TI32, ir.TU32:
		return 32
	case ir.TI64, ir.TU64, ir.TISize, ir.TUSize:
		return 64
	default:
		return 0
	}
}

func intSigned(k ir.TypeKind) bool {
	switch k {
	case ir.TI8, ir.TI16, ir.TI32, ir.TI64, ir.TISize:
		return true
	default:
		return false
	}
}

func intMinMacro(k ir.TypeKind) string {
	switch k {
	case ir.TI8:
		return "INT8_MIN"
	case ir.TI16:
		return "INT16_MIN"
	case ir.TI32:
		return "INT32_MIN"
	case ir.TI64, ir.TISize:
		return "INT64_MIN"
	default:
		return "INT64_MIN"
	}
}

func intMaxMacro(k ir.TypeKind) string {
	switch k {
	case ir.TI8:
		return "INT8_MAX"
	case ir.TI16:
		return "INT16_MAX"
	case ir.TI32:
		return "INT32_MAX"
	case ir.TI64, ir.TISize:
		return "INT64_MAX"
	default:
		return "INT64_MAX"
	}
}

func uintMaxMacro(k ir.TypeKind) string {
	switch k {
	case ir.TU8:
		return "UINT8_MAX"
	case ir.TU16:
		return "UINT16_MAX"
	case ir.TU32:
		return "UINT32_MAX"
	case ir.TU64, ir.TUSize:
		return "UINT64_MAX"
	default:
		return "UINT64_MAX"
	}
}

func cUnsignedType(t ir.Type) (string, error) {
	switch t.K {
	case ir.TI8, ir.TU8:
		return "uint8_t", nil
	case ir.TI16, ir.TU16:
		return "uint16_t", nil
	case ir.TI32, ir.TU32:
		return "uint32_t", nil
	case ir.TI64, ir.TU64, ir.TISize, ir.TUSize:
		return "uint64_t", nil
	default:
		return "", fmt.Errorf("not an int type: %s", t.String())
	}
}

func emitIntBinOp(out *bytes.Buffer, i *ir.BinOp) error {
	if i == nil {
		return fmt.Errorf("nil binop")
	}
	if !isIntType(i.Ty) {
		return fmt.Errorf("int binop requires int type")
	}
	switch i.Op {
	case ir.OpAdd, ir.OpSub, ir.OpMul:
		// Wrapping semantics for all integer types, without signed overflow UB.
		ut, err := cUnsignedType(i.Ty)
		if err != nil {
			return err
		}
		out.WriteString("  {\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _a; memcpy(&_a, &")
		out.WriteString(cValue(i.A))
		out.WriteString(", sizeof(_a));\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _b; memcpy(&_b, &")
		out.WriteString(cValue(i.B))
		out.WriteString(", sizeof(_b));\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _r = _a ")
		out.WriteString(stringToCOp(string(i.Op)))
		out.WriteString(" _b;\n")
		out.WriteString("    memcpy(&")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(", &_r, sizeof(_r));\n")
		out.WriteString("  }\n")
		return nil
	case ir.OpBitAnd, ir.OpBitOr, ir.OpBitXor:
		// Bitwise ops operate on raw bits for all integer types.
		ut, err := cUnsignedType(i.Ty)
		if err != nil {
			return err
		}
		out.WriteString("  {\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _a; memcpy(&_a, &")
		out.WriteString(cValue(i.A))
		out.WriteString(", sizeof(_a));\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _b; memcpy(&_b, &")
		out.WriteString(cValue(i.B))
		out.WriteString(", sizeof(_b));\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _r = _a ")
		out.WriteString(stringToCOp(string(i.Op)))
		out.WriteString(" _b;\n")
		out.WriteString("    memcpy(&")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(", &_r, sizeof(_r));\n")
		out.WriteString("  }\n")
		return nil
	case ir.OpShl, ir.OpShr:
		// Shift ops on raw bits; panic on invalid shift counts.
		bits := intBits(i.Ty.K)
		if bits == 0 {
			return fmt.Errorf("unsupported int type for shift: %s", i.Ty.String())
		}
		ut, err := cUnsignedType(i.Ty)
		if err != nil {
			return err
		}
		out.WriteString("  {\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _a; memcpy(&_a, &")
		out.WriteString(cValue(i.A))
		out.WriteString(", sizeof(_a));\n")
		if intSigned(i.Ty.K) {
			out.WriteString("    int64_t _sh = (int64_t)")
			out.WriteString(cValue(i.B))
			out.WriteString(";\n")
			out.WriteString("    if (_sh < 0 || _sh >= ")
			out.WriteString(strconv.Itoa(bits))
			out.WriteString(") { vox_builtin_panic(\"shift count out of range\"); }\n")
			out.WriteString("    uint64_t _n = (uint64_t)_sh;\n")
		} else {
			out.WriteString("    uint64_t _n = (uint64_t)")
			out.WriteString(cValue(i.B))
			out.WriteString(";\n")
			out.WriteString("    if (_n >= ")
			out.WriteString(strconv.Itoa(bits))
			out.WriteString(") { vox_builtin_panic(\"shift count out of range\"); }\n")
		}
		if i.Op == ir.OpShl {
			out.WriteString("    ")
			out.WriteString(ut)
			out.WriteString(" _r = (")
			out.WriteString(ut)
			out.WriteString(")(_a << _n);\n")
		} else if intSigned(i.Ty.K) {
			out.WriteString("    ")
			out.WriteString(ut)
			out.WriteString(" _r;\n")
			out.WriteString("    if (_n == 0) {\n")
			out.WriteString("      _r = _a;\n")
			out.WriteString("    } else if ((_a & ((")
			out.WriteString(ut)
			out.WriteString(")1 << ")
			out.WriteString(strconv.Itoa(bits - 1))
			out.WriteString(")) != 0) {\n")
			out.WriteString("      ")
			out.WriteString(ut)
			out.WriteString(" _m = ~(((")
			out.WriteString(ut)
			out.WriteString(")~(")
			out.WriteString(ut)
			out.WriteString(")0) >> _n);\n")
			out.WriteString("      _r = (")
			out.WriteString(ut)
			out.WriteString(")((_a >> _n) | _m);\n")
			out.WriteString("    } else {\n")
			out.WriteString("      _r = (")
			out.WriteString(ut)
			out.WriteString(")(_a >> _n);\n")
			out.WriteString("    }\n")
		} else {
			out.WriteString("    ")
			out.WriteString(ut)
			out.WriteString(" _r = (")
			out.WriteString(ut)
			out.WriteString(")(_a >> _n);\n")
		}
		out.WriteString("    memcpy(&")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(", &_r, sizeof(_r));\n")
		out.WriteString("  }\n")
		return nil
	case ir.OpDiv, ir.OpMod:
		// Division/modulo: check divide-by-zero; signed MIN/-1 overflows.
		if intSigned(i.Ty.K) {
			out.WriteString("  {\n")
			out.WriteString("    int64_t _a = (int64_t)")
			out.WriteString(cValue(i.A))
			out.WriteString(";\n")
			out.WriteString("    int64_t _b = (int64_t)")
			out.WriteString(cValue(i.B))
			out.WriteString(";\n")
			out.WriteString("    if (_b == 0) { vox_builtin_panic(\"division by zero\"); }\n")
			out.WriteString("    if (_a == (int64_t)")
			out.WriteString(intMinMacro(i.Ty.K))
			out.WriteString(" && _b == -1) { vox_builtin_panic(\"division overflow\"); }\n")
			out.WriteString("    int64_t _r = _a ")
			if i.Op == ir.OpDiv {
				out.WriteString("/")
			} else {
				out.WriteString("%")
			}
			out.WriteString(" _b;\n")
			out.WriteString("    ")
			out.WriteString(cTempName(i.Dst.ID))
			out.WriteString(" = (")
			out.WriteString(cType(i.Ty))
			out.WriteString(")_r;\n")
			out.WriteString("  }\n")
			return nil
		}

		// Unsigned division/modulo.
		ut, err := cUnsignedType(i.Ty)
		if err != nil {
			return err
		}
		out.WriteString("  {\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _a; memcpy(&_a, &")
		out.WriteString(cValue(i.A))
		out.WriteString(", sizeof(_a));\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _b; memcpy(&_b, &")
		out.WriteString(cValue(i.B))
		out.WriteString(", sizeof(_b));\n")
		out.WriteString("    if (_b == 0) { vox_builtin_panic(\"division by zero\"); }\n")
		out.WriteString("    ")
		out.WriteString(ut)
		out.WriteString(" _r = _a ")
		if i.Op == ir.OpDiv {
			out.WriteString("/")
		} else {
			out.WriteString("%")
		}
		out.WriteString(" _b;\n")
		out.WriteString("    memcpy(&")
		out.WriteString(cTempName(i.Dst.ID))
		out.WriteString(", &_r, sizeof(_r));\n")
		out.WriteString("  }\n")
		return nil
	default:
		return fmt.Errorf("unsupported int binop: %s", i.Op)
	}
}

func intSignedMin(bits int) int64 {
	switch bits {
	case 8:
		return -128
	case 16:
		return -32768
	case 32:
		return -2147483648
	case 64:
		return -9223372036854775808
	default:
		return 0
	}
}

func intSignedMax(bits int) int64 {
	switch bits {
	case 8:
		return 127
	case 16:
		return 32767
	case 32:
		return 2147483647
	case 64:
		return 9223372036854775807
	default:
		return 0
	}
}

func intUnsignedMax(bits int) uint64 {
	switch bits {
	case 8:
		return 255
	case 16:
		return 65535
	case 32:
		return 4294967295
	case 64:
		return 18446744073709551615
	default:
		return 0
	}
}

func emitIntCastChecked(out *bytes.Buffer, i *ir.IntCastChecked) error {
	if i == nil {
		return fmt.Errorf("nil cast")
	}
	if !isIntType(i.From) || !isIntType(i.To) {
		return fmt.Errorf("int_cast_checked requires integer types")
	}
	fromBits := intBits(i.From.K)
	toBits := intBits(i.To.K)
	if fromBits == 0 || toBits == 0 {
		return fmt.Errorf("unsupported int type in cast")
	}

	msg := "int cast overflow"
	if i.From.K == ir.TI64 && i.To.K == ir.TI32 {
		msg = "i64 to i32 overflow"
	}

	out.WriteString("  {\n")
	if intSigned(i.From.K) {
		out.WriteString("    int64_t _x = (int64_t)")
		out.WriteString(cValue(i.V))
		out.WriteString(";\n")
		if intSigned(i.To.K) {
			// signed -> signed: only narrowing can overflow.
			if fromBits > toBits {
				out.WriteString("    if (_x < (int64_t)")
				out.WriteString(intMinMacro(i.To.K))
				out.WriteString(" || _x > (int64_t)")
				out.WriteString(intMaxMacro(i.To.K))
				out.WriteString(") { vox_builtin_panic(\"")
				out.WriteString(msg)
				out.WriteString("\"); }\n")
			}
		} else {
			// signed -> unsigned: always reject negative; upper bound only matters when narrowing.
			out.WriteString("    if (_x < 0")
			if toBits < 64 {
				out.WriteString(" || (uint64_t)_x > (uint64_t)")
				out.WriteString(uintMaxMacro(i.To.K))
			}
			out.WriteString(") { vox_builtin_panic(\"")
			out.WriteString(msg)
			out.WriteString("\"); }\n")
		}
	} else {
		out.WriteString("    uint64_t _x = (uint64_t)")
		out.WriteString(cValue(i.V))
		out.WriteString(";\n")
		if intSigned(i.To.K) {
			// unsigned -> signed: always need an upper bound check.
			out.WriteString("    if (_x > (uint64_t)")
			out.WriteString(intMaxMacro(i.To.K))
			out.WriteString(") { vox_builtin_panic(\"")
			out.WriteString(msg)
			out.WriteString("\"); }\n")
		} else {
			// unsigned -> unsigned: only narrowing can overflow.
			if fromBits > toBits {
				out.WriteString("    if (_x > (uint64_t)")
				out.WriteString(uintMaxMacro(i.To.K))
				out.WriteString(") { vox_builtin_panic(\"")
				out.WriteString(msg)
				out.WriteString("\"); }\n")
			}
		}
	}
	out.WriteString("    ")
	out.WriteString(cTempName(i.Dst.ID))
	out.WriteString(" = (")
	out.WriteString(cType(i.To))
	out.WriteString(")")
	out.WriteString("_x")
	out.WriteString(";\n")
	out.WriteString("  }\n")
	return nil
}

func emitIntCast(out *bytes.Buffer, i *ir.IntCast) error {
	if i == nil {
		return fmt.Errorf("nil cast")
	}
	if !isIntType(i.From) || !isIntType(i.To) {
		return fmt.Errorf("int_cast requires integer types")
	}
	// Stage0 currently does not generate wrapping casts in IR; keep this as a plain cast for now.
	out.WriteString("  ")
	out.WriteString(cTempName(i.Dst.ID))
	out.WriteString(" = (")
	out.WriteString(cType(i.To))
	out.WriteString(")")
	out.WriteString(cValue(i.V))
	out.WriteString(";\n")
	return nil
}

func emitRangeCheckInt(out *bytes.Buffer, i *ir.RangeCheckInt) error {
	if i == nil {
		return fmt.Errorf("nil range check")
	}
	if !isIntType(i.Ty) {
		return fmt.Errorf("range_check requires integer type")
	}
	out.WriteString("  {\n")
	if intSigned(i.Ty.K) {
		out.WriteString("    int64_t _x = (int64_t)")
		out.WriteString(cValue(i.V))
		out.WriteString(";\n")
		out.WriteString("    if (_x < (int64_t)")
		out.WriteString(strconv.FormatInt(i.Lo, 10))
		out.WriteString(" || _x > (int64_t)")
		out.WriteString(strconv.FormatInt(i.Hi, 10))
		out.WriteString(") { vox_builtin_panic(\"range check failed\"); }\n")
	} else {
		out.WriteString("    uint64_t _x = (uint64_t)")
		out.WriteString(cValue(i.V))
		out.WriteString(";\n")
		out.WriteString("    if (_x < (uint64_t)")
		out.WriteString(strconv.FormatInt(i.Lo, 10))
		out.WriteString(" || _x > (uint64_t)")
		out.WriteString(strconv.FormatInt(i.Hi, 10))
		out.WriteString(") { vox_builtin_panic(\"range check failed\"); }\n")
	}
	out.WriteString("  }\n")
	return nil
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
