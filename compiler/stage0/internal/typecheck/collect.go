package typecheck

import (
	"voxlang/internal/names"
)

func (c *checker) collectStructSigs() {
	// First pass: register all struct names so field types can reference other structs.
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

	// Second pass: fill fields.
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

func (c *checker) collectEnumSigs() {
	// First pass: register all enum names so payload types can reference other nominal types.
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

	// Second pass: fill variants.
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
			if len(v.Fields) > 1 {
				c.errorAt(v.Span, "stage0 enum payload arity > 1 is not supported yet")
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
	c.funcSigs["assert"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_i32"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyI32}, {K: TyI32}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_i64"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_bool"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyBool}, {K: TyBool}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::assert_eq_str"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyString}, {K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["std.testing::fail"] = FuncSig{Pub: true, OwnerPkg: "", OwnerMod: []string{"std", "testing"}, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}

	for _, fn := range c.prog.Funcs {
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		if _, exists := c.funcSigs[qname]; exists {
			c.errorAt(fn.Span, "duplicate function: "+qname)
			continue
		}
		sig := FuncSig{}
		pkg, mod, _ := names.SplitOwnerAndModule(fn.Span.File.Name)
		sig.Pub = fn.Pub
		sig.OwnerPkg = pkg
		sig.OwnerMod = mod
		for _, p := range fn.Params {
			sig.Params = append(sig.Params, c.typeFromAstInFile(p.Type, fn.Span.File))
		}
		sig.Ret = c.typeFromAstInFile(fn.Ret, fn.Span.File)
		c.funcSigs[qname] = sig
	}
	// main presence check (stage0)
	if _, ok := c.funcSigs["main"]; !ok {
		// not a hard error for library compilation, but stage0 expects main
	}
}
