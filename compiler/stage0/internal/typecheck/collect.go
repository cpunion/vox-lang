package typecheck

import (
	"voxlang/internal/names"
)

func (c *checker) collectNominalSigs() {
	// First pass: register all nominal type names (struct + enum) so bodies can reference each other.
	for _, st := range c.prog.Structs {
		if st == nil || st.Span.File == nil {
			continue
		}
		pkg, mod, _ := names.SplitOwnerAndModule(st.Span.File.Name)
		qname := names.QualifyFunc(st.Span.File.Name, st.Name)
		if _, exists := c.enumSigs[qname]; exists {
			c.errorAt(st.Span, "duplicate nominal type name (enum already exists): "+qname)
			continue
		}
		if _, exists := c.structSigs[qname]; exists {
			c.errorAt(st.Span, "duplicate struct: "+qname)
			continue
		}
		c.structSigs[qname] = StructSig{
			Name:       qname,
			Pub:        st.Pub,
			OwnerPkg:   pkg,
			OwnerMod:   mod,
			FieldIndex: map[string]int{},
		}
	}
	for _, en := range c.prog.Enums {
		if en == nil || en.Span.File == nil {
			continue
		}
		pkg, mod, _ := names.SplitOwnerAndModule(en.Span.File.Name)
		qname := names.QualifyFunc(en.Span.File.Name, en.Name)
		if _, exists := c.structSigs[qname]; exists {
			c.errorAt(en.Span, "duplicate nominal type name (struct already exists): "+qname)
			continue
		}
		if _, exists := c.enumSigs[qname]; exists {
			c.errorAt(en.Span, "duplicate enum: "+qname)
			continue
		}
		c.enumSigs[qname] = EnumSig{
			Name:         qname,
			Pub:          en.Pub,
			OwnerPkg:     pkg,
			OwnerMod:     mod,
			VariantIndex: map[string]int{},
		}
	}
}

func (c *checker) fillStructSigs() {
	for _, st := range c.prog.Structs {
		if st == nil || st.Span.File == nil {
			continue
		}
		qname := names.QualifyFunc(st.Span.File.Name, st.Name)
		sig := c.structSigs[qname]
		sig.Fields = nil
		sig.FieldIndex = map[string]int{}
		for _, f := range st.Fields {
			if _, exists := sig.FieldIndex[f.Name]; exists {
				c.errorAt(f.Span, "duplicate field: "+f.Name)
				continue
			}
			fty := c.typeFromAstInFile(f.Type, st.Span.File)
			sig.FieldIndex[f.Name] = len(sig.Fields)
			sig.Fields = append(sig.Fields, StructFieldSig{Pub: f.Pub, Name: f.Name, Ty: fty})
		}
		c.structSigs[qname] = sig
	}
}

func (c *checker) fillEnumSigs() {
	for _, en := range c.prog.Enums {
		if en == nil || en.Span.File == nil {
			continue
		}
		qname := names.QualifyFunc(en.Span.File.Name, en.Name)
		sig := c.enumSigs[qname]
		sig.Variants = nil
		sig.VariantIndex = map[string]int{}

		for _, v := range en.Variants {
			if _, exists := sig.VariantIndex[v.Name]; exists {
				c.errorAt(v.Span, "duplicate variant: "+v.Name)
				continue
			}
			fields := []Type{}
			for _, ft := range v.Fields {
				fty := c.typeFromAstInFile(ft, en.Span.File)
				fields = append(fields, fty)
			}
			sig.VariantIndex[v.Name] = len(sig.Variants)
			sig.Variants = append(sig.Variants, EnumVariantSig{Name: v.Name, Fields: fields})
		}
		c.enumSigs[qname] = sig
	}
}

func (c *checker) collectFuncSigs() {
	// Builtins (stage0): keep minimal and stable.
	// Higher-level helpers (assert/testing/etc.) should live in stdlib Vox sources.
	c.funcSigs["panic"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["print"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}

	for _, fn := range c.prog.Funcs {
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		c.funcDecls[qname] = fn
		if _, exists := c.funcSigs[qname]; exists {
			c.errorAt(fn.Span, "duplicate function: "+qname)
			continue
		}
		sig := FuncSig{}
		pkg, mod, _ := names.SplitOwnerAndModule(fn.Span.File.Name)
		sig.Pub = fn.Pub
		sig.OwnerPkg = pkg
		sig.OwnerMod = mod
		sig.TypeParams = append([]string{}, fn.TypeParams...)
		// Allow type params in the signature for generic functions.
		if len(fn.TypeParams) > 0 {
			c.curTyVars = map[string]bool{}
			for _, tp := range fn.TypeParams {
				c.curTyVars[tp] = true
			}
		} else {
			c.curTyVars = nil
		}
		for _, p := range fn.Params {
			sig.Params = append(sig.Params, c.typeFromAstInFile(p.Type, fn.Span.File))
		}
		sig.Ret = c.typeFromAstInFile(fn.Ret, fn.Span.File)
		c.curTyVars = nil
		c.funcSigs[qname] = sig
	}
	// main presence check (stage0)
	if _, ok := c.funcSigs["main"]; !ok {
		// not a hard error for library compilation, but stage0 expects main
	}
}
