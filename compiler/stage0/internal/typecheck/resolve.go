package typecheck

import (
	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

func (c *checker) typeFromAstInFile(t ast.Type, file *source.File) Type {
	switch tt := t.(type) {
	case *ast.UnitType:
		return Type{K: TyUnit}
	case *ast.RangeType:
		base := c.typeFromAstInFile(tt.Base, file)
		if !isIntType(base) {
			c.errorAt(tt.S, "range base type must be an integer type (stage0)")
			return Type{K: TyBad}
		}
		if tt.Lo > tt.Hi {
			c.errorAt(tt.S, "invalid range: lo > hi")
			return Type{K: TyBad}
		}
		// Bounds must fit the base integer type.
		min, max, ok := intMinMax(base)
		if !ok {
			c.errorAt(tt.S, "invalid range base type")
			return Type{K: TyBad}
		}
		// Stage0 v0: range bounds are i64 syntax, so unsigned ranges are limited to i64 bounds.
		if isUnsignedIntType(base) && tt.Lo < 0 {
			c.errorAt(tt.S, "range bounds out of base type")
			return Type{K: TyBad}
		}
		if tt.Lo < min || uint64(tt.Hi) > max {
			c.errorAt(tt.S, "range bounds out of base type")
			return Type{K: TyBad}
		}
		return Type{K: TyRange, Base: &base, Lo: tt.Lo, Hi: tt.Hi}
	case *ast.NamedType:
		if len(tt.Parts) == 0 {
			c.errorAt(tt.S, "missing type name")
			return Type{K: TyBad}
		}
		// Qualified types: a.b.C
		if len(tt.Parts) > 1 {
			if file == nil {
				c.errorAt(tt.S, "unknown type")
				return Type{K: TyBad}
			}
			alias := tt.Parts[0]
			extraMods := tt.Parts[1 : len(tt.Parts)-1]
			name := tt.Parts[len(tt.Parts)-1]

			m := c.imports[file]
			tgt, ok := m[alias]
			if !ok {
				c.errorAt(tt.S, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
				return Type{K: TyBad}
			}
			mod := append(append([]string{}, tgt.Mod...), extraMods...)
			q := names.QualifyParts(tgt.Pkg, mod, name)
			if ss, ok := c.structSigs[q]; ok {
				if !c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
					c.errorAt(tt.S, "type is private: "+q)
					return Type{K: TyBad}
				}
				return Type{K: TyStruct, Name: q}
			}
			if es, ok := c.enumSigs[q]; ok {
				if !c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Pub) {
					c.errorAt(tt.S, "type is private: "+q)
					return Type{K: TyBad}
				}
				return Type{K: TyEnum, Name: q}
			}
			if _, ok := c.typeAliases[q]; ok {
				return c.resolveTypeAliasType(q, file, tt.S)
			}
			c.errorAt(tt.S, "unknown type: "+q)
			return Type{K: TyBad}
		}

		// Single-segment types: builtins or local/root nominal types.
		name := tt.Parts[0]
		if c.curTyVars != nil && c.curTyVars[name] {
			return Type{K: TyParam, Name: name}
		}
		switch name {
		case "i8":
			return Type{K: TyI8}
		case "u8":
			return Type{K: TyU8}
		case "i16":
			return Type{K: TyI16}
		case "u16":
			return Type{K: TyU16}
		case "i32":
			return Type{K: TyI32}
		case "u32":
			return Type{K: TyU32}
		case "i64":
			return Type{K: TyI64}
		case "u64":
			return Type{K: TyU64}
		case "isize":
			return Type{K: TyISize}
		case "usize":
			return Type{K: TyUSize}
		case "bool":
			return Type{K: TyBool}
		case "String":
			return Type{K: TyString}
		case "Vec":
			if len(tt.Args) != 1 {
				c.errorAt(tt.S, "Vec expects 1 type argument")
				return Type{K: TyBad}
			}
			elem := c.typeFromAstInFile(tt.Args[0], file)
			return Type{K: TyVec, Elem: &elem}
		default:
			if file != nil {
				private := ""
				pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
				q1 := names.QualifyParts(pkg, mod, name)
				if ss, ok := c.structSigs[q1]; ok {
					if c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
						return Type{K: TyStruct, Name: q1}
					}
					private = q1
				}
				if es, ok := c.enumSigs[q1]; ok {
					if c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Pub) {
						return Type{K: TyEnum, Name: q1}
					}
					if private == "" {
						private = q1
					}
				}
				if tm := c.namedTypes[file]; tm != nil {
					if ty := tm[name]; ty.K != TyBad {
						return ty
					}
				}
				if _, ok := c.typeAliases[q1]; ok {
					return c.resolveTypeAliasType(q1, file, tt.S)
				}
				q2 := names.QualifyParts(pkg, nil, name)
				if ss, ok := c.structSigs[q2]; ok {
					if c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
						return Type{K: TyStruct, Name: q2}
					}
					if private == "" {
						private = q2
					}
				}
				if es, ok := c.enumSigs[q2]; ok {
					if c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Pub) {
						return Type{K: TyEnum, Name: q2}
					}
					if private == "" {
						private = q2
					}
				}
				if _, ok := c.typeAliases[q2]; ok {
					return c.resolveTypeAliasType(q2, file, tt.S)
				}
				if private != "" {
					c.errorAt(tt.S, "type is private: "+private)
					return Type{K: TyBad}
				}
			}
			c.errorAt(tt.S, "unknown type: "+name)
			return Type{K: TyBad}
		}
	default:
		c.errorAt(t.Span(), "unsupported type")
		return Type{K: TyBad}
	}
}

func (c *checker) resolveStructByParts(file *source.File, parts []string, s source.Span) (Type, StructSig, bool) {
	if len(parts) == 0 {
		c.errorAt(s, "missing struct name")
		return Type{K: TyBad}, StructSig{}, false
	}
	if len(parts) == 1 {
		private := ""
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if ss, ok := c.structSigs[q1]; ok {
			if c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
				return Type{K: TyStruct, Name: q1}, ss, true
			}
			private = q1
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if ss, ok := c.structSigs[q2]; ok {
			if c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
				return Type{K: TyStruct, Name: q2}, ss, true
			}
			if private == "" {
				private = q2
			}
		}
		if file != nil {
			if tm := c.namedTypes[file]; tm != nil {
				if ty := tm[parts[0]]; ty.K == TyStruct {
					if ss, ok := c.structSigs[ty.Name]; ok {
						return ty, ss, true
					}
				}
			}
		}
		if private != "" {
			c.errorAt(s, "type is private: "+private)
			return Type{K: TyBad}, StructSig{}, false
		}
		c.errorAt(s, "unknown type: "+parts[0])
		return Type{K: TyBad}, StructSig{}, false
	}

	alias := parts[0]
	extraMods := parts[1 : len(parts)-1]
	name := parts[len(parts)-1]

	m := c.imports[file]
	tgt, ok := m[alias]
	if !ok {
		c.errorAt(s, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
		return Type{K: TyBad}, StructSig{}, false
	}
	mod := append(append([]string{}, tgt.Mod...), extraMods...)
	q := names.QualifyParts(tgt.Pkg, mod, name)
	ss, ok := c.structSigs[q]
	if !ok {
		c.errorAt(s, "unknown type: "+q)
		return Type{K: TyBad}, StructSig{}, false
	}
	if !c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
		c.errorAt(s, "type is private: "+q)
		return Type{K: TyBad}, StructSig{}, false
	}
	return Type{K: TyStruct, Name: q}, ss, true
}

func (c *checker) findEnumByParts(file *source.File, parts []string) (Type, EnumSig, bool) {
	if len(parts) == 0 {
		return Type{K: TyBad}, EnumSig{}, false
	}
	if len(parts) == 1 {
		if file != nil {
			if tm := c.namedTypes[file]; tm != nil {
				if ty := tm[parts[0]]; ty.K == TyEnum {
					if es, ok := c.enumSigs[ty.Name]; ok {
						return ty, es, true
					}
				}
			}
		}
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if es, ok := c.enumSigs[q1]; ok {
			return Type{K: TyEnum, Name: q1}, es, true
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if es, ok := c.enumSigs[q2]; ok {
			return Type{K: TyEnum, Name: q2}, es, true
		}
		return Type{K: TyBad}, EnumSig{}, false
	}

	alias := parts[0]
	extraMods := parts[1 : len(parts)-1]
	name := parts[len(parts)-1]

	m := c.imports[file]
	tgt, ok := m[alias]
	if !ok {
		return Type{K: TyBad}, EnumSig{}, false
	}
	mod := append(append([]string{}, tgt.Mod...), extraMods...)
	q := names.QualifyParts(tgt.Pkg, mod, name)
	es, ok := c.enumSigs[q]
	if !ok {
		return Type{K: TyBad}, EnumSig{}, false
	}
	return Type{K: TyEnum, Name: q}, es, true
}

func (c *checker) resolveEnumByParts(file *source.File, parts []string, s source.Span) (Type, EnumSig, bool) {
	if len(parts) == 0 {
		c.errorAt(s, "missing enum name")
		return Type{K: TyBad}, EnumSig{}, false
	}
	if ty, es, found := c.findEnumByParts(file, parts); found {
		if c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Pub) {
			return ty, es, true
		}
		c.errorAt(s, "type is private: "+ty.Name)
		return Type{K: TyBad}, EnumSig{}, false
	}

	if len(parts) == 1 {
		c.errorAt(s, "unknown type: "+parts[0])
		return Type{K: TyBad}, EnumSig{}, false
	}
	alias := parts[0]
	if file != nil {
		m := c.imports[file]
		if _, ok := m[alias]; !ok {
			c.errorAt(s, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
			return Type{K: TyBad}, EnumSig{}, false
		}
	}
	c.errorAt(s, "unknown type")
	return Type{K: TyBad}, EnumSig{}, false
}
