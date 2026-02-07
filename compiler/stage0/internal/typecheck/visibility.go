package typecheck

import (
	"voxlang/internal/names"
	"voxlang/internal/source"
)

func (c *checker) isSameModule(file *source.File, ownerPkg string, ownerMod []string) bool {
	if file == nil {
		return false
	}
	pkg, mod, _ := names.SplitOwnerAndModule(file.Name)
	return pkg == ownerPkg && sameModPath(mod, ownerMod)
}

func (c *checker) canAccess(file *source.File, ownerPkg string, ownerMod []string, pub bool) bool {
	if c.isSameModule(file, ownerPkg, ownerMod) {
		return true
	}
	return pub
}

func (c *checker) checkPubInterfaces() {
	// Stage0 minimal rule set:
	// - Non-pub items can reference anything in their signatures.
	// - pub fn/struct/enum cannot expose private nominal types from their own module.
	//
	// This prevents "pub API that is unusable outside the module".
	var checkTy func(at source.Span, ownerPkg string, ownerMod []string, t Type, what string)
	checkTy = func(at source.Span, ownerPkg string, ownerMod []string, t Type, what string) {
		switch t.K {
		case TyStruct:
			ss, ok := c.structSigs[t.Name]
			if ok && ss.OwnerPkg == ownerPkg && sameModPath(ss.OwnerMod, ownerMod) && !ss.Pub {
				c.errorAt(at, what+" exposes private type: "+t.Name)
			}
		case TyEnum:
			es, ok := c.enumSigs[t.Name]
			if ok && es.OwnerPkg == ownerPkg && sameModPath(es.OwnerMod, ownerMod) && !es.Pub {
				c.errorAt(at, what+" exposes private type: "+t.Name)
			}
		case TyVec:
			if t.Elem != nil {
				checkTy(at, ownerPkg, ownerMod, *t.Elem, what)
			}
		}
	}

	for _, fn := range c.prog.Funcs {
		if fn == nil || fn.Span.File == nil || !fn.Pub {
			continue
		}
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		sig, ok := c.funcSigs[qname]
		if !ok {
			continue
		}
		ownerPkg, ownerMod, _ := names.SplitOwnerAndModule(fn.Span.File.Name)
		for _, p := range sig.Params {
			checkTy(fn.Span, ownerPkg, ownerMod, p, "public function "+qname)
		}
		checkTy(fn.Span, ownerPkg, ownerMod, sig.Ret, "public function "+qname)
	}

	for _, st := range c.prog.Structs {
		if st == nil || st.Span.File == nil || !st.Pub {
			continue
		}
		qname := names.QualifyFunc(st.Span.File.Name, st.Name)
		if _, ok := c.structSigs[qname]; !ok {
			continue
		}
		ownerPkg, ownerMod, _ := names.SplitOwnerAndModule(st.Span.File.Name)
		for _, f := range c.structSigs[qname].Fields {
			if !f.Pub {
				continue
			}
			checkTy(st.Span, ownerPkg, ownerMod, f.Ty, "public struct "+qname)
		}
	}

	for _, en := range c.prog.Enums {
		if en == nil || en.Span.File == nil || !en.Pub {
			continue
		}
		qname := names.QualifyFunc(en.Span.File.Name, en.Name)
		es, ok := c.enumSigs[qname]
		if !ok {
			continue
		}
		ownerPkg, ownerMod, _ := names.SplitOwnerAndModule(en.Span.File.Name)
		for _, v := range es.Variants {
			for _, f := range v.Fields {
				checkTy(en.Span, ownerPkg, ownerMod, f, "public enum "+qname)
			}
		}
	}
}
