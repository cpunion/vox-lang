package typecheck

import (
	"strings"

	"voxlang/internal/ast"
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
		if c.hasNominalName(qname) {
			c.errorAt(st.Span, "duplicate struct: "+qname)
			continue
		}
		if len(st.TypeParams) == 0 {
			c.structSigs[qname] = StructSig{
				Name:       qname,
				Vis:        st.Vis,
				Pub:        st.Pub,
				OwnerPkg:   pkg,
				OwnerMod:   mod,
				FieldIndex: map[string]int{},
			}
		} else {
			c.genericStructSigs[qname] = GenericStructSig{
				Name:       qname,
				Vis:        st.Vis,
				Pub:        st.Pub,
				OwnerPkg:   pkg,
				OwnerMod:   mod,
				TypeParams: append([]string{}, st.TypeParams...),
				FieldIndex: map[string]int{},
			}
		}
	}
	for _, en := range c.prog.Enums {
		if en == nil || en.Span.File == nil {
			continue
		}
		pkg, mod, _ := names.SplitOwnerAndModule(en.Span.File.Name)
		qname := names.QualifyFunc(en.Span.File.Name, en.Name)
		if c.hasNominalName(qname) {
			c.errorAt(en.Span, "duplicate enum: "+qname)
			continue
		}
		if len(en.TypeParams) == 0 {
			c.enumSigs[qname] = EnumSig{
				Name:         qname,
				Vis:          en.Vis,
				Pub:          en.Pub,
				OwnerPkg:     pkg,
				OwnerMod:     mod,
				VariantIndex: map[string]int{},
			}
		} else {
			c.genericEnumSigs[qname] = GenericEnumSig{
				Name:         qname,
				Vis:          en.Vis,
				Pub:          en.Pub,
				OwnerPkg:     pkg,
				OwnerMod:     mod,
				TypeParams:   append([]string{}, en.TypeParams...),
				VariantIndex: map[string]int{},
			}
		}
	}
}

func (c *checker) fillStructSigs() {
	for _, st := range c.prog.Structs {
		if st == nil || st.Span.File == nil {
			continue
		}
		qname := names.QualifyFunc(st.Span.File.Name, st.Name)
		prevTyVars := c.curTyVars
		if len(st.TypeParams) > 0 {
			c.curTyVars = map[string]bool{}
			for _, tp := range st.TypeParams {
				c.curTyVars[tp] = true
			}
		} else {
			c.curTyVars = nil
		}
		if len(st.TypeParams) == 0 {
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
				sig.Fields = append(sig.Fields, StructFieldSig{Vis: f.Vis, Pub: f.Pub, Name: f.Name, Ty: fty})
			}
			c.structSigs[qname] = sig
		} else {
			sig := c.genericStructSigs[qname]
			sig.Fields = nil
			sig.FieldIndex = map[string]int{}
			for _, f := range st.Fields {
				if _, exists := sig.FieldIndex[f.Name]; exists {
					c.errorAt(f.Span, "duplicate field: "+f.Name)
					continue
				}
				fty := c.typeFromAstInFile(f.Type, st.Span.File)
				sig.FieldIndex[f.Name] = len(sig.Fields)
				sig.Fields = append(sig.Fields, StructFieldSig{Vis: f.Vis, Pub: f.Pub, Name: f.Name, Ty: fty})
			}
			c.genericStructSigs[qname] = sig
		}
		c.curTyVars = prevTyVars
	}
}

func (c *checker) fillEnumSigs() {
	for _, en := range c.prog.Enums {
		if en == nil || en.Span.File == nil {
			continue
		}
		qname := names.QualifyFunc(en.Span.File.Name, en.Name)
		prevTyVars := c.curTyVars
		if len(en.TypeParams) > 0 {
			c.curTyVars = map[string]bool{}
			for _, tp := range en.TypeParams {
				c.curTyVars[tp] = true
			}
		} else {
			c.curTyVars = nil
		}
		if len(en.TypeParams) == 0 {
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
		} else {
			sig := c.genericEnumSigs[qname]
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
			c.genericEnumSigs[qname] = sig
		}
		c.curTyVars = prevTyVars
	}
}

func (c *checker) hasNominalName(qname string) bool {
	if _, exists := c.structSigs[qname]; exists {
		return true
	}
	if _, exists := c.genericStructSigs[qname]; exists {
		return true
	}
	if _, exists := c.enumSigs[qname]; exists {
		return true
	}
	if _, exists := c.genericEnumSigs[qname]; exists {
		return true
	}
	return false
}

func (c *checker) collectFuncSigs() {
	// Builtins (stage0): keep minimal and stable.
	// Higher-level helpers (assert/testing/etc.) should live in stdlib Vox sources.
	c.funcSigs["panic"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["print"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}
	// Tooling/stdlib support builtins (stage0): used to bootstrap stage1 toolchain.
	// These are intentionally low-level; prefer std wrappers (e.g. std/fs, std/process).
	c.funcSigs["__read_file"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyString}}
	c.funcSigs["__write_file"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}, {K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__path_exists"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyBool}}
	c.funcSigs["__mkdir_p"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__exec"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyI32}}
	c.funcSigs["__args"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: nil, Ret: Type{K: TyVec, Elem: &Type{K: TyString}}}
	c.funcSigs["__walk_vox_files"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}}, Ret: Type{K: TyVec, Elem: &Type{K: TyString}}}
	c.funcSigs["__exe_path"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: nil, Ret: Type{K: TyString}}
	c.funcSigs["__now_ns"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: nil, Ret: Type{K: TyI64}}
	// Stage1 std/sync intrinsics (i32 MVP).
	c.funcSigs["__mutex_i32_new"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI32}}, Ret: Type{K: TyI64}}
	c.funcSigs["__mutex_i32_load"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI32}}
	c.funcSigs["__mutex_i32_store"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI32}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__atomic_i32_new"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI32}}, Ret: Type{K: TyI64}}
	c.funcSigs["__atomic_i32_load"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI32}}
	c.funcSigs["__atomic_i32_store"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI32}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__atomic_i32_fetch_add"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI32}}, Ret: Type{K: TyI32}}
	c.funcSigs["__atomic_i32_swap"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI32}}, Ret: Type{K: TyI32}}
	// Stage1 std/sync intrinsics (i64 generic storage backend).
	c.funcSigs["__mutex_i64_new"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI64}}
	c.funcSigs["__mutex_i64_load"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI64}}
	c.funcSigs["__mutex_i64_store"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__atomic_i64_new"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI64}}
	c.funcSigs["__atomic_i64_load"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyI64}}
	c.funcSigs["__atomic_i64_store"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyUnit}}
	c.funcSigs["__atomic_i64_fetch_add"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyI64}}
	c.funcSigs["__atomic_i64_swap"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI64}}, Ret: Type{K: TyI64}}
	// Stage1 std/io network intrinsics (minimal TCP).
	c.funcSigs["__tcp_connect"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyString}, {K: TyI32}}, Ret: Type{K: TyI64}}
	c.funcSigs["__tcp_send"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyString}}, Ret: Type{K: TyI32}}
	c.funcSigs["__tcp_recv"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}, {K: TyI32}}, Ret: Type{K: TyString}}
	c.funcSigs["__tcp_close"] = FuncSig{Vis: ast.VisPub, Pub: true, OwnerPkg: "", OwnerMod: nil, Params: []Type{{K: TyI64}}, Ret: Type{K: TyUnit}}

	for _, fn := range c.prog.Funcs {
		if strings.HasPrefix(fn.Name, "__") {
			c.errorAt(fn.Span, "reserved function name: "+fn.Name)
			continue
		}
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		c.funcDecls[qname] = fn
		if _, exists := c.funcSigs[qname]; exists {
			c.errorAt(fn.Span, "duplicate function: "+qname)
			continue
		}
		sig := FuncSig{}
		pkg, mod, _ := names.SplitOwnerAndModule(fn.Span.File.Name)
		sig.Vis = fn.Vis
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
