package typecheck

import (
	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

type TypeAliasSig struct {
	Name     string // qualified name
	Pub      bool
	OwnerPkg string
	OwnerMod []string
	File     *source.File
	Target   ast.Type
}

func (c *checker) collectTypeAliasSigs() {
	for _, td := range c.prog.Types {
		if td == nil || td.Span.File == nil {
			continue
		}
		pkg, mod, _ := names.SplitOwnerAndModule(td.Span.File.Name)
		qname := names.QualifyFunc(td.Span.File.Name, td.Name)

		if _, exists := c.structSigs[qname]; exists {
			c.errorAt(td.Span, "duplicate nominal type name (struct already exists): "+qname)
			continue
		}
		if _, exists := c.enumSigs[qname]; exists {
			c.errorAt(td.Span, "duplicate nominal type name (enum already exists): "+qname)
			continue
		}
		if _, exists := c.typeAliases[qname]; exists {
			c.errorAt(td.Span, "duplicate type alias: "+qname)
			continue
		}

		c.typeAliases[qname] = TypeAliasSig{
			Name:     qname,
			Pub:      td.Pub,
			OwnerPkg: pkg,
			OwnerMod: mod,
			File:     td.Span.File,
			Target:   td.Type,
		}
	}
}

func (c *checker) resolveTypeAliasType(qname string, useFile *source.File, useSpan source.Span) Type {
	if ty, ok := c.typeAliasTy[qname]; ok {
		return ty
	}
	if c.typeAliasBusy[qname] {
		c.errorAt(useSpan, "cyclic type alias: "+qname)
		return Type{K: TyBad}
	}
	sig, ok := c.typeAliases[qname]
	if !ok {
		c.errorAt(useSpan, "unknown type: "+qname)
		return Type{K: TyBad}
	}

	// Visibility of the alias itself.
	if useFile != nil && sig.File != nil {
		if !c.canAccess(useFile, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
			c.errorAt(useSpan, "type is private: "+qname)
			return Type{K: TyBad}
		}
	}

	c.typeAliasBusy[qname] = true
	out := c.typeFromAstInFile(sig.Target, sig.File)
	c.typeAliasBusy[qname] = false
	c.typeAliasTy[qname] = out

	// Don't allow the alias to bypass underlying nominal-type privacy.
	if useFile != nil && out.K == TyStruct {
		if ss, ok := c.structSigs[out.Name]; ok {
			if !c.canAccess(useFile, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
				c.errorAt(useSpan, "type is private: "+out.Name)
				return Type{K: TyBad}
			}
		}
	}
	if useFile != nil && out.K == TyEnum {
		if es, ok := c.enumSigs[out.Name]; ok {
			if !c.canAccess(useFile, es.OwnerPkg, es.OwnerMod, es.Pub) {
				c.errorAt(useSpan, "type is private: "+out.Name)
				return Type{K: TyBad}
			}
		}
	}
	return out
}
