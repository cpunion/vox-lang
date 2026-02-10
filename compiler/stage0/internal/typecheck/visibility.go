package typecheck

import (
	"voxlang/internal/ast"
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

func moduleParent(mod []string) []string {
	if len(mod) == 0 {
		return nil
	}
	return mod[:len(mod)-1]
}

func moduleInScope(mod []string, scope []string) bool {
	if len(scope) == 0 {
		return true
	}
	if len(mod) < len(scope) {
		return false
	}
	for i := 0; i < len(scope); i++ {
		if mod[i] != scope[i] {
			return false
		}
	}
	return true
}

func (c *checker) canAccess(file *source.File, ownerPkg string, ownerMod []string, vis ast.Visibility) bool {
	if file == nil {
		return false
	}
	userPkg, userMod, _ := names.SplitOwnerAndModule(file.Name)
	if userPkg == ownerPkg && sameModPath(userMod, ownerMod) {
		return true
	}

	switch vis {
	case ast.VisPrivate:
		return false
	case ast.VisPub:
		return true
	case ast.VisCrate:
		return userPkg == ownerPkg
	case ast.VisSuper:
		return userPkg == ownerPkg && moduleInScope(userMod, moduleParent(ownerMod))
	default:
		return false
	}
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
