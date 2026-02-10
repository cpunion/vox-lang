package typecheck

import (
	"voxlang/internal/ast"
	"voxlang/internal/source"
)

func typeHasParam(t Type) bool {
	switch t.K {
	case TyParam:
		return true
	case TyVec:
		return t.Elem != nil && typeHasParam(*t.Elem)
	case TyRange:
		return t.Base != nil && typeHasParam(*t.Base)
	default:
		return false
	}
}

func (c *checker) instantiateGenericStruct(qname string, file *source.File, args []ast.Type, at source.Span) (Type, StructSig, bool) {
	gs, ok := c.genericStructSigs[qname]
	if !ok {
		return Type{K: TyBad}, StructSig{}, false
	}
	if len(args) != len(gs.TypeParams) {
		c.errorAt(at, "wrong number of type arguments: expected "+itoa(len(gs.TypeParams))+", got "+itoa(len(args)))
		return Type{K: TyBad}, StructSig{}, false
	}

	subs := map[string]Type{}
	for i, tp := range gs.TypeParams {
		argTy := c.typeFromAstInFile(args[i], file)
		if argTy.K == TyBad {
			return Type{K: TyBad}, StructSig{}, false
		}
		if typeHasParam(argTy) {
			c.errorAt(at, "generic nominal type arguments must be concrete in stage0")
			return Type{K: TyBad}, StructSig{}, false
		}
		subs[tp] = argTy
	}

	instName := qname + "$" + instSuffix(gs.TypeParams, subs)
	if ss, exists := c.structSigs[instName]; exists {
		return Type{K: TyStruct, Name: instName}, ss, true
	}

	ss := StructSig{
		Name:       instName,
		Vis:        gs.Vis,
		Pub:        gs.Pub,
		OwnerPkg:   gs.OwnerPkg,
		OwnerMod:   append([]string{}, gs.OwnerMod...),
		FieldIndex: map[string]int{},
	}
	for _, f := range gs.Fields {
		fty := substType(f.Ty, subs)
		if typeHasParam(fty) {
			c.errorAt(at, "could not fully resolve generic struct fields: "+qname)
			return Type{K: TyBad}, StructSig{}, false
		}
		ss.FieldIndex[f.Name] = len(ss.Fields)
		ss.Fields = append(ss.Fields, StructFieldSig{
			Vis:  f.Vis,
			Pub:  f.Pub,
			Name: f.Name,
			Ty:   fty,
		})
	}
	c.structSigs[instName] = ss
	return Type{K: TyStruct, Name: instName}, ss, true
}

func (c *checker) instantiateGenericEnum(qname string, file *source.File, args []ast.Type, at source.Span) (Type, EnumSig, bool) {
	ge, ok := c.genericEnumSigs[qname]
	if !ok {
		return Type{K: TyBad}, EnumSig{}, false
	}
	if len(args) != len(ge.TypeParams) {
		c.errorAt(at, "wrong number of type arguments: expected "+itoa(len(ge.TypeParams))+", got "+itoa(len(args)))
		return Type{K: TyBad}, EnumSig{}, false
	}

	subs := map[string]Type{}
	for i, tp := range ge.TypeParams {
		argTy := c.typeFromAstInFile(args[i], file)
		if argTy.K == TyBad {
			return Type{K: TyBad}, EnumSig{}, false
		}
		if typeHasParam(argTy) {
			c.errorAt(at, "generic nominal type arguments must be concrete in stage0")
			return Type{K: TyBad}, EnumSig{}, false
		}
		subs[tp] = argTy
	}

	instName := qname + "$" + instSuffix(ge.TypeParams, subs)
	if es, exists := c.enumSigs[instName]; exists {
		return Type{K: TyEnum, Name: instName}, es, true
	}

	es := EnumSig{
		Name:         instName,
		Vis:          ge.Vis,
		Pub:          ge.Pub,
		OwnerPkg:     ge.OwnerPkg,
		OwnerMod:     append([]string{}, ge.OwnerMod...),
		VariantIndex: map[string]int{},
	}
	for _, v := range ge.Variants {
		fields := []Type{}
		for _, f := range v.Fields {
			fty := substType(f, subs)
			if typeHasParam(fty) {
				c.errorAt(at, "could not fully resolve generic enum variants: "+qname)
				return Type{K: TyBad}, EnumSig{}, false
			}
			fields = append(fields, fty)
		}
		es.VariantIndex[v.Name] = len(es.Variants)
		es.Variants = append(es.Variants, EnumVariantSig{Name: v.Name, Fields: fields})
	}
	c.enumSigs[instName] = es
	return Type{K: TyEnum, Name: instName}, es, true
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := "0123456789"
	out := ""
	n := v
	for n > 0 {
		d := n % 10
		out = digits[d:d+1] + out
		n = n / 10
	}
	return out
}
