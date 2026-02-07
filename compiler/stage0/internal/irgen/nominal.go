package irgen

import (
	"sort"

	"voxlang/internal/ir"
)

func (g *gen) genNominalDefs() error {
	if g.p == nil {
		return nil
	}

	// Stable ordering for reproducible output.
	var snames []string
	for name := range g.p.StructSigs {
		snames = append(snames, name)
	}
	sort.Strings(snames)
	for _, name := range snames {
		ss := g.p.StructSigs[name]
		st := &ir.Struct{Name: ss.Name}
		for _, f := range ss.Fields {
			ty, err := g.irTypeFromChecked(f.Ty)
			if err != nil {
				return err
			}
			st.Fields = append(st.Fields, ir.StructField{Name: f.Name, Ty: ty})
		}
		g.out.Structs[st.Name] = st
	}

	var enames []string
	for name := range g.p.EnumSigs {
		enames = append(enames, name)
	}
	sort.Strings(enames)
	for _, name := range enames {
		es := g.p.EnumSigs[name]
		en := &ir.Enum{Name: es.Name, VariantIndex: map[string]int{}}
		for i, v := range es.Variants {
			fields := make([]ir.Type, 0, len(v.Fields))
			for _, ft := range v.Fields {
				pty, err := g.irTypeFromChecked(ft)
				if err != nil {
					return err
				}
				fields = append(fields, pty)
			}
			en.VariantIndex[v.Name] = i
			en.Variants = append(en.Variants, ir.EnumVariant{Name: v.Name, Fields: fields})
		}
		g.out.Enums[en.Name] = en
	}
	return nil
}
