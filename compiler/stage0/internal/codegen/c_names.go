package codegen

import (
	"fmt"
	"strings"

	"voxlang/internal/ir"
)

func cType(t ir.Type) string {
	switch t.K {
	case ir.TUnit:
		return "void"
	case ir.TBool:
		return "bool"
	case ir.TI32:
		return "int32_t"
	case ir.TI64:
		return "int64_t"
	case ir.TString:
		return "const char*"
	case ir.TStruct:
		return cStructTypeName(t.Name)
	case ir.TEnum:
		return cEnumTypeName(t.Name)
	case ir.TVec:
		return "vox_vec"
	default:
		return "void"
	}
}

func cFnName(name string) string { return "vox_fn_" + cMangle(name) }

func cLabelName(name string) string { return "vox_blk_" + cIdent(name) }

func cStructTypeName(name string) string { return "vox_struct_" + cMangle(name) }

func cEnumTypeName(name string) string { return "vox_enum_" + cMangle(name) }

func cIdent(s string) string {
	// Best-effort sanitization: keep [A-Za-z0-9_], map others to '_'.
	// This keeps IR readable (may contain "::") while emitting valid C symbols.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
		if !ok {
			ch = '_'
		}
		if i == 0 && (ch >= '0' && ch <= '9') {
			b.WriteByte('_')
		}
		b.WriteByte(ch)
	}
	out := b.String()
	if out == "" {
		return "_"
	}
	return out
}

func cMangle(s string) string {
	// Collision-free mangling for qualified names.
	//
	// We keep [A-Za-z0-9] as-is for readability, and hex-escape everything else,
	// including '_' so that separators like "::" and "." cannot collide with user
	// identifiers that contain underscores.
	//
	// Always prefix with 'm' so the result is a valid C identifier.
	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteByte('m')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if ok {
			b.WriteByte(ch)
			continue
		}
		fmt.Fprintf(&b, "_%02x", ch)
	}
	return b.String()
}

func cParamName(i int, name string) string {
	_ = name
	return fmt.Sprintf("p%d", i)
}

func cTempName(id int) string { return fmt.Sprintf("t%d", id) }

func cSlotName(id int) string { return fmt.Sprintf("v%d", id) }
