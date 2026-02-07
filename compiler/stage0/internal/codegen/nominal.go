package codegen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"voxlang/internal/ir"
)

func emitNominalTypes(out *bytes.Buffer, p *ir.Program) error {
	if p == nil {
		return nil
	}

	// Collect all nominal type names. In stage0 these are always by-value.
	// That means we must be able to emit them as plain C typedefs in a strict
	// dependency order (no cycles).
	all := map[string]string{} // name -> "struct"|"enum"
	if p.Structs != nil {
		for name := range p.Structs {
			all[name] = "struct"
		}
	}
	if p.Enums != nil {
		for name := range p.Enums {
			if prev, ok := all[name]; ok && prev != "enum" {
				return fmt.Errorf("nominal name collision: %s is both %s and enum", name, prev)
			}
			all[name] = "enum"
		}
	}
	if len(all) == 0 {
		return nil
	}

	deps := map[string]map[string]struct{}{}
	indeg := map[string]int{}
	for name := range all {
		deps[name] = map[string]struct{}{}
		indeg[name] = 0
	}

	addDep := func(from, to string) error {
		if from == to {
			return fmt.Errorf("cyclic nominal type by-value dependency: %s depends on itself", from)
		}
		if _, ok := all[to]; !ok {
			return fmt.Errorf("unknown nominal type reference: %s -> %s", from, to)
		}
		// Emit dependencies first: if `from` references `to` by-value, then `to` must be emitted before `from`.
		// Model the topo edge as: to -> from.
		if _, ok := deps[to][from]; ok {
			return nil
		}
		deps[to][from] = struct{}{}
		indeg[from]++
		return nil
	}

	// struct deps
	for name, st := range p.Structs {
		if st == nil {
			continue
		}
		for _, f := range st.Fields {
			switch f.Ty.K {
			case ir.TStruct, ir.TEnum:
				if err := addDep(name, f.Ty.Name); err != nil {
					return err
				}
			}
		}
	}

	// enum deps (payload types)
	for name, en := range p.Enums {
		if en == nil {
			continue
		}
		for _, v := range en.Variants {
			for _, ft := range v.Fields {
				switch ft.K {
				case ir.TStruct, ir.TEnum:
					if err := addDep(name, ft.Name); err != nil {
						return err
					}
				}
			}
		}
	}

	// Kahn topo sort with deterministic selection.
	var ready []string
	for name, d := range indeg {
		if d == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(all))
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		order = append(order, n)

		targets := make([]string, 0, len(deps[n]))
		for m := range deps[n] {
			targets = append(targets, m)
		}
		sort.Strings(targets)
		for _, m := range targets {
			indeg[m]--
			if indeg[m] == 0 {
				ready = append(ready, m)
			}
		}
		sort.Strings(ready)
	}

	if len(order) != len(all) {
		var remain []string
		for name, d := range indeg {
			if d > 0 {
				remain = append(remain, name)
			}
		}
		sort.Strings(remain)
		return fmt.Errorf("cyclic nominal types by-value dependency: %s", strings.Join(remain, ", "))
	}

	for _, name := range order {
		switch all[name] {
		case "struct":
			st := p.Structs[name]
			if st == nil {
				return fmt.Errorf("nil struct def: %s", name)
			}
			emitStructTypedef(out, st)
		case "enum":
			en := p.Enums[name]
			if en == nil {
				return fmt.Errorf("nil enum def: %s", name)
			}
			emitEnumTypedef(out, en)
		default:
			return fmt.Errorf("unknown nominal kind: %s", all[name])
		}
		out.WriteString("\n")
	}
	return nil
}

func emitStructTypedef(out *bytes.Buffer, st *ir.Struct) {
	out.WriteString("typedef struct {\n")
	for _, f := range st.Fields {
		out.WriteString("  ")
		out.WriteString(cType(f.Ty))
		out.WriteByte(' ')
		out.WriteString(cIdent(f.Name))
		out.WriteString(";\n")
	}
	out.WriteString("} ")
	out.WriteString(cStructTypeName(st.Name))
	out.WriteString(";\n")
}

func emitEnumTypedef(out *bytes.Buffer, en *ir.Enum) {
	hasPayload := false
	for _, v := range en.Variants {
		if len(v.Fields) != 0 {
			hasPayload = true
			break
		}
	}

	// Tagged-union layout:
	//   { int32_t tag; union { struct { T _0; U _1; } Variant; struct { uint8_t _; } Unit; ... } payload; }
	// Using per-variant structs keeps the emitted initializers and payload reads simple.
	out.WriteString("typedef struct {\n")
	out.WriteString("  int32_t tag;\n")
	if hasPayload {
		out.WriteString("  union {\n")
		for _, v := range en.Variants {
			out.WriteString("    struct {\n")
			if len(v.Fields) != 0 {
				for i, fty := range v.Fields {
					out.WriteString("      ")
					out.WriteString(cType(fty))
					out.WriteByte(' ')
					out.WriteString(fmt.Sprintf("_%d", i))
					out.WriteString(";\n")
				}
			} else {
				// Dummy field for unit variants so `.payload.V = { ._ = 0 }` is always valid C.
				out.WriteString("      uint8_t _;\n")
			}
			out.WriteString("    } ")
			out.WriteString(cIdent(v.Name))
			out.WriteString(";\n")
		}
		out.WriteString("  } payload;\n")
	}
	out.WriteString("} ")
	out.WriteString(cEnumTypeName(en.Name))
	out.WriteString(";\n")
}
