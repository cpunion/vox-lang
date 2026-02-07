package typecheck

import (
	"voxlang/internal/names"
)

func (c *checker) collectImports() {
	for _, imp := range c.prog.Imports {
		if imp == nil || imp.Span.File == nil {
			continue
		}

		path := imp.Path
		alias := imp.Alias

		var tgt importTarget
		if c.opts.AllowedPkgs != nil && c.opts.AllowedPkgs[path] {
			// dependency root import
			tgt = importTarget{Pkg: path, Mod: nil}
			if alias == "" {
				alias = path
			}
		} else {
			// local module import
			if !c.isKnownLocalModule(imp.Span.File.Name, path) {
				c.errorAt(imp.Span, "unknown local module: "+path)
				continue
			}
			tgt = importTarget{Pkg: "", Mod: splitModPath(path)}
			if alias == "" {
				alias = defaultImportAlias(path)
			}
		}

		// Named imports are resolved after function signatures are known.
		if len(imp.Names) > 0 {
			c.pending = append(c.pending, pendingNamedImport{File: imp.Span.File, Names: imp.Names, Tgt: tgt, Span: imp.Span})
			continue
		}

		m := c.imports[imp.Span.File]
		if m == nil {
			m = map[string]importTarget{}
			c.imports[imp.Span.File] = m
		}
		if _, exists := m[alias]; exists {
			c.errorAt(imp.Span, "duplicate import alias: "+alias)
			continue
		}
		m[alias] = tgt
	}
}

func (c *checker) resolveNamedImports() {
	for _, pi := range c.pending {
		if pi.File == nil {
			continue
		}
		ownerPkg, ownerMod, _ := names.SplitOwnerAndModule(pi.File.Name)

		m := c.namedFuncs[pi.File]
		if m == nil {
			m = map[string]string{}
			c.namedFuncs[pi.File] = m
		}
		tm := c.namedTypes[pi.File]
		if tm == nil {
			tm = map[string]Type{}
			c.namedTypes[pi.File] = tm
		}
		for _, nm := range pi.Names {
			local := nm.Alias
			if local == "" {
				local = nm.Name
			}
			if local == "" {
				continue
			}

			// Reject collisions with local module definitions.
			qLocal := names.QualifyParts(ownerPkg, ownerMod, local)
			if _, ok := c.funcSigs[qLocal]; ok {
				c.errorAt(pi.Span, "import name conflicts with local definition: "+local)
				continue
			}
			if _, ok := c.structSigs[qLocal]; ok {
				c.errorAt(pi.Span, "import name conflicts with local definition: "+local)
				continue
			}
			if _, ok := c.enumSigs[qLocal]; ok {
				c.errorAt(pi.Span, "import name conflicts with local definition: "+local)
				continue
			}

			if _, exists := m[local]; exists || tm[local].K != TyBad {
				c.errorAt(pi.Span, "duplicate imported name: "+local)
				continue
			}

			target := names.QualifyParts(pi.Tgt.Pkg, pi.Tgt.Mod, nm.Name)

			var found int
			// function
			if sig, ok := c.funcSigs[target]; ok {
				found++
				if !c.canAccess(pi.File, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
					c.errorAt(pi.Span, "function is private: "+target)
					continue
				}
				m[local] = target
			}
			// struct
			if ss, ok := c.structSigs[target]; ok {
				found++
				if !c.canAccess(pi.File, ss.OwnerPkg, ss.OwnerMod, ss.Pub) {
					c.errorAt(pi.Span, "type is private: "+target)
					continue
				}
				tm[local] = Type{K: TyStruct, Name: target}
			}
			// enum
			if es, ok := c.enumSigs[target]; ok {
				found++
				if !c.canAccess(pi.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
					c.errorAt(pi.Span, "type is private: "+target)
					continue
				}
				tm[local] = Type{K: TyEnum, Name: target}
			}

			if found == 0 {
				c.errorAt(pi.Span, "unknown imported name: "+target)
				continue
			}
			if found > 1 {
				// Keep it simple: avoid mixing namespaces in stage0.
				c.errorAt(pi.Span, "ambiguous imported name: "+target)
				// best-effort: drop any partial bindings
				delete(m, local)
				delete(tm, local)
				continue
			}
		}
	}
}

func (c *checker) isKnownLocalModule(fileName string, importPath string) bool {
	// If no validation info is provided, accept.
	if c.opts.LocalModulesByPkg == nil && c.opts.LocalModules == nil {
		return true
	}
	pkg, _, _ := names.SplitOwnerAndModule(fileName)
	if c.opts.LocalModulesByPkg != nil {
		m := c.opts.LocalModulesByPkg[pkg]
		if m == nil {
			// No data for this package; accept to avoid spurious failures.
			return true
		}
		return m[importPath]
	}
	return c.opts.LocalModules[importPath]
}
