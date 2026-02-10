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
		// Single-segment types: builtins, type params, or local/root nominal types.
		name := tt.Parts[0]
		if c.curTyVars != nil && c.curTyVars[name] {
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "type parameter does not accept type arguments: "+name)
				return Type{K: TyBad}
			}
			return Type{K: TyParam, Name: name}
		}
		switch name {
		case "i8":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: i8")
				return Type{K: TyBad}
			}
			return Type{K: TyI8}
		case "u8":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: u8")
				return Type{K: TyBad}
			}
			return Type{K: TyU8}
		case "i16":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: i16")
				return Type{K: TyBad}
			}
			return Type{K: TyI16}
		case "u16":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: u16")
				return Type{K: TyBad}
			}
			return Type{K: TyU16}
		case "i32":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: i32")
				return Type{K: TyBad}
			}
			return Type{K: TyI32}
		case "u32":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: u32")
				return Type{K: TyBad}
			}
			return Type{K: TyU32}
		case "i64":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: i64")
				return Type{K: TyBad}
			}
			return Type{K: TyI64}
		case "u64":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: u64")
				return Type{K: TyBad}
			}
			return Type{K: TyU64}
		case "isize":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: isize")
				return Type{K: TyBad}
			}
			return Type{K: TyISize}
		case "usize":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: usize")
				return Type{K: TyBad}
			}
			return Type{K: TyUSize}
		case "bool":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: bool")
				return Type{K: TyBad}
			}
			return Type{K: TyBool}
		case "String":
			if len(tt.Args) != 0 {
				c.errorAt(tt.S, "non-generic type does not accept type arguments: String")
				return Type{K: TyBad}
			}
			return Type{K: TyString}
		case "Vec":
			if len(tt.Args) != 1 {
				c.errorAt(tt.S, "Vec expects 1 type argument")
				return Type{K: TyBad}
			}
			elem := c.typeFromAstInFile(tt.Args[0], file)
			return Type{K: TyVec, Elem: &elem}
		default:
			return c.resolveNamedNominalType(file, tt.Parts, tt.Args, tt.S)
		}
	default:
		c.errorAt(t.Span(), "unsupported type")
		return Type{K: TyBad}
	}
}

func (c *checker) resolveNamedNominalType(file *source.File, parts []string, args []ast.Type, s source.Span) Type {
	if len(parts) == 0 {
		c.errorAt(s, "missing type name")
		return Type{K: TyBad}
	}
	if len(parts) > 1 {
		if file == nil {
			c.errorAt(s, "unknown type")
			return Type{K: TyBad}
		}
		alias := parts[0]
		extraMods := parts[1 : len(parts)-1]
		name := parts[len(parts)-1]
		m := c.imports[file]
		tgt, ok := m[alias]
		if !ok {
			c.errorAt(s, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
			return Type{K: TyBad}
		}
		mod := append(append([]string{}, tgt.Mod...), extraMods...)
		q := names.QualifyParts(tgt.Pkg, mod, name)
		if ty, _, found, private := c.resolveStructTypeByQName(file, q, args, s); found {
			if private {
				c.errorAt(s, "type is private: "+q)
				return Type{K: TyBad}
			}
			return ty
		}
		if ty, _, found, private := c.resolveEnumTypeByQName(file, q, args, s); found {
			if private {
				c.errorAt(s, "type is private: "+q)
				return Type{K: TyBad}
			}
			return ty
		}
		if _, ok := c.typeAliases[q]; ok {
			if len(args) != 0 {
				c.errorAt(s, "type alias does not accept type arguments: "+q)
				return Type{K: TyBad}
			}
			return c.resolveTypeAliasType(q, file, s)
		}
		c.errorAt(s, "unknown type: "+q)
		return Type{K: TyBad}
	}

	name := parts[0]
	if file != nil {
		private := ""
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, name)
		if ty, _, found, blocked := c.resolveStructTypeByQName(file, q1, args, s); found {
			if blocked {
				private = q1
			} else {
				return ty
			}
		}
		if ty, _, found, blocked := c.resolveEnumTypeByQName(file, q1, args, s); found {
			if blocked {
				if private == "" {
					private = q1
				}
			} else {
				return ty
			}
		}
		if tm := c.namedTypes[file]; tm != nil {
			if ty := tm[name]; ty.K == TyStruct {
				if out, _, found, blocked := c.resolveStructTypeByQName(file, ty.Name, args, s); found {
					if blocked {
						if private == "" {
							private = ty.Name
						}
					} else {
						return out
					}
				}
			} else if ty.K == TyEnum {
				if out, _, found, blocked := c.resolveEnumTypeByQName(file, ty.Name, args, s); found {
					if blocked {
						if private == "" {
							private = ty.Name
						}
					} else {
						return out
					}
				}
			} else if ty.K != TyBad && len(args) == 0 {
				return ty
			}
		}
		if _, ok := c.typeAliases[q1]; ok {
			if len(args) != 0 {
				c.errorAt(s, "type alias does not accept type arguments: "+q1)
				return Type{K: TyBad}
			}
			return c.resolveTypeAliasType(q1, file, s)
		}
		q2 := names.QualifyParts(pkg, nil, name)
		if ty, _, found, blocked := c.resolveStructTypeByQName(file, q2, args, s); found {
			if blocked {
				if private == "" {
					private = q2
				}
			} else {
				return ty
			}
		}
		if ty, _, found, blocked := c.resolveEnumTypeByQName(file, q2, args, s); found {
			if blocked {
				if private == "" {
					private = q2
				}
			} else {
				return ty
			}
		}
		if _, ok := c.typeAliases[q2]; ok {
			if len(args) != 0 {
				c.errorAt(s, "type alias does not accept type arguments: "+q2)
				return Type{K: TyBad}
			}
			return c.resolveTypeAliasType(q2, file, s)
		}
		if private != "" {
			c.errorAt(s, "type is private: "+private)
			return Type{K: TyBad}
		}
	}
	c.errorAt(s, "unknown type: "+name)
	return Type{K: TyBad}
}

func (c *checker) resolveStructTypeByQName(file *source.File, qname string, args []ast.Type, at source.Span) (Type, StructSig, bool, bool) {
	if ss, ok := c.structSigs[qname]; ok {
		if file != nil && !c.canAccess(file, ss.OwnerPkg, ss.OwnerMod, ss.Vis) {
			return Type{K: TyBad}, StructSig{}, true, true
		}
		if len(args) != 0 {
			c.errorAt(at, "non-generic type does not accept type arguments: "+qname)
			return Type{K: TyBad}, StructSig{}, true, false
		}
		return Type{K: TyStruct, Name: qname}, ss, true, false
	}
	if gs, ok := c.genericStructSigs[qname]; ok {
		if file != nil && !c.canAccess(file, gs.OwnerPkg, gs.OwnerMod, gs.Vis) {
			return Type{K: TyBad}, StructSig{}, true, true
		}
		if len(args) == 0 {
			c.errorAt(at, "generic type requires type arguments: "+qname)
			return Type{K: TyBad}, StructSig{}, true, false
		}
		ty, ss, ok := c.instantiateGenericStruct(qname, file, args, at)
		if !ok {
			return Type{K: TyBad}, StructSig{}, true, false
		}
		return ty, ss, true, false
	}
	return Type{K: TyBad}, StructSig{}, false, false
}

func (c *checker) resolveEnumTypeByQName(file *source.File, qname string, args []ast.Type, at source.Span) (Type, EnumSig, bool, bool) {
	if es, ok := c.enumSigs[qname]; ok {
		if file != nil && !c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Vis) {
			return Type{K: TyBad}, EnumSig{}, true, true
		}
		if len(args) != 0 {
			c.errorAt(at, "non-generic type does not accept type arguments: "+qname)
			return Type{K: TyBad}, EnumSig{}, true, false
		}
		return Type{K: TyEnum, Name: qname}, es, true, false
	}
	if ge, ok := c.genericEnumSigs[qname]; ok {
		if file != nil && !c.canAccess(file, ge.OwnerPkg, ge.OwnerMod, ge.Vis) {
			return Type{K: TyBad}, EnumSig{}, true, true
		}
		if len(args) == 0 {
			c.errorAt(at, "generic type requires type arguments: "+qname)
			return Type{K: TyBad}, EnumSig{}, true, false
		}
		ty, es, ok := c.instantiateGenericEnum(qname, file, args, at)
		if !ok {
			return Type{K: TyBad}, EnumSig{}, true, false
		}
		return ty, es, true, false
	}
	return Type{K: TyBad}, EnumSig{}, false, false
}

func (c *checker) resolveStructByParts(file *source.File, parts []string, typeArgs []ast.Type, s source.Span) (Type, StructSig, bool) {
	if len(parts) == 0 {
		c.errorAt(s, "missing struct name")
		return Type{K: TyBad}, StructSig{}, false
	}
	if len(parts) == 1 {
		private := ""
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if ty, ss, found, blocked := c.resolveStructTypeByQName(file, q1, typeArgs, s); found {
			if blocked {
				private = q1
			} else {
				return ty, ss, true
			}
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if ty, ss, found, blocked := c.resolveStructTypeByQName(file, q2, typeArgs, s); found {
			if blocked {
				if private == "" {
					private = q2
				}
			} else {
				return ty, ss, true
			}
		}
		if file != nil {
			if tm := c.namedTypes[file]; tm != nil {
				if ty := tm[parts[0]]; ty.K == TyStruct {
					if out, ss, found, blocked := c.resolveStructTypeByQName(file, ty.Name, typeArgs, s); found {
						if blocked {
							if private == "" {
								private = ty.Name
							}
						} else {
							return out, ss, true
						}
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
	ty, ss, found, blocked := c.resolveStructTypeByQName(file, q, typeArgs, s)
	if !found {
		c.errorAt(s, "unknown type: "+q)
		return Type{K: TyBad}, StructSig{}, false
	}
	if blocked {
		c.errorAt(s, "type is private: "+q)
		return Type{K: TyBad}, StructSig{}, false
	}
	if ty.K == TyBad {
		return Type{K: TyBad}, StructSig{}, false
	}
	return ty, ss, true
}

func (c *checker) structBaseQNameByParts(file *source.File, parts []string) (string, bool) {
	if file == nil || len(parts) == 0 {
		return "", false
	}
	if len(parts) == 1 {
		if tm := c.namedTypes[file]; tm != nil {
			if ty := tm[parts[0]]; ty.K == TyStruct {
				return ty.Name, true
			}
		}
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if _, ok := c.structSigs[q1]; ok {
			return q1, true
		}
		if _, ok := c.genericStructSigs[q1]; ok {
			return q1, true
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if _, ok := c.structSigs[q2]; ok {
			return q2, true
		}
		if _, ok := c.genericStructSigs[q2]; ok {
			return q2, true
		}
		return "", false
	}
	alias := parts[0]
	extraMods := parts[1 : len(parts)-1]
	name := parts[len(parts)-1]
	m := c.imports[file]
	tgt, ok := m[alias]
	if !ok {
		return "", false
	}
	mod := append(append([]string{}, tgt.Mod...), extraMods...)
	q := names.QualifyParts(tgt.Pkg, mod, name)
	if _, ok := c.structSigs[q]; ok {
		return q, true
	}
	if _, ok := c.genericStructSigs[q]; ok {
		return q, true
	}
	return "", false
}

func (c *checker) enumBaseQNameByParts(file *source.File, parts []string) (string, bool) {
	if file == nil || len(parts) == 0 {
		return "", false
	}
	if len(parts) == 1 {
		if tm := c.namedTypes[file]; tm != nil {
			if ty := tm[parts[0]]; ty.K == TyEnum {
				return ty.Name, true
			}
		}
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if _, ok := c.enumSigs[q1]; ok {
			return q1, true
		}
		if _, ok := c.genericEnumSigs[q1]; ok {
			return q1, true
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if _, ok := c.enumSigs[q2]; ok {
			return q2, true
		}
		if _, ok := c.genericEnumSigs[q2]; ok {
			return q2, true
		}
		return "", false
	}
	alias := parts[0]
	extraMods := parts[1 : len(parts)-1]
	name := parts[len(parts)-1]
	m := c.imports[file]
	tgt, ok := m[alias]
	if !ok {
		return "", false
	}
	mod := append(append([]string{}, tgt.Mod...), extraMods...)
	q := names.QualifyParts(tgt.Pkg, mod, name)
	if _, ok := c.enumSigs[q]; ok {
		return q, true
	}
	if _, ok := c.genericEnumSigs[q]; ok {
		return q, true
	}
	return "", false
}

func (c *checker) findEnumByParts(file *source.File, parts []string, typeArgs []ast.Type, at source.Span) (Type, EnumSig, bool) {
	if len(parts) == 0 {
		return Type{K: TyBad}, EnumSig{}, false
	}
	if len(parts) == 1 {
		if file != nil {
			if tm := c.namedTypes[file]; tm != nil {
				if ty := tm[parts[0]]; ty.K == TyEnum {
					if out, es, found, blocked := c.resolveEnumTypeByQName(file, ty.Name, typeArgs, at); found && !blocked && out.K != TyBad {
						return out, es, true
					}
				}
			}
		}
		pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
		q1 := names.QualifyParts(pkg, mod, parts[0])
		if ty, es, found, blocked := c.resolveEnumTypeByQName(file, q1, typeArgs, at); found && !blocked && ty.K != TyBad {
			return ty, es, true
		}
		q2 := names.QualifyParts(pkg, nil, parts[0])
		if ty, es, found, blocked := c.resolveEnumTypeByQName(file, q2, typeArgs, at); found && !blocked && ty.K != TyBad {
			return ty, es, true
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
	ty, es, found, blocked := c.resolveEnumTypeByQName(file, q, typeArgs, at)
	if !found || blocked || ty.K == TyBad {
		return Type{K: TyBad}, EnumSig{}, false
	}
	return ty, es, true
}

func (c *checker) resolveEnumByParts(file *source.File, parts []string, typeArgs []ast.Type, s source.Span) (Type, EnumSig, bool) {
	if len(parts) == 0 {
		c.errorAt(s, "missing enum name")
		return Type{K: TyBad}, EnumSig{}, false
	}
	if ty, es, found := c.findEnumByParts(file, parts, typeArgs, s); found {
		if c.canAccess(file, es.OwnerPkg, es.OwnerMod, es.Vis) {
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
