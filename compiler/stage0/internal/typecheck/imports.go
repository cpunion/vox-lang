package typecheck

import (
	"strings"

	"voxlang/internal/names"
)

func splitImportScheme(path string) (scheme string, rest string) {
	if strings.HasPrefix(path, "pkg:") {
		return "pkg", path[len("pkg:"):]
	}
	if strings.HasPrefix(path, "mod:") {
		return "mod", path[len("mod:"):]
	}
	if strings.HasPrefix(path, "std:") {
		return "std", path[len("std:"):]
	}
	return "", path
}

func internalPkgName(name string) string { return "pkg." + name }

func (c *checker) initPresence() {
	if c.presentPkgs != nil {
		return
	}
	c.presentPkgs = map[string]bool{}
	c.presentModsByPkg = map[string]map[string]bool{}

	addFile := func(fileName string) {
		if fileName == "" {
			return
		}
		pkg, mod, _ := names.SplitOwnerAndModule(fileName)
		c.presentPkgs[pkg] = true
		m := c.presentModsByPkg[pkg]
		if m == nil {
			m = map[string]bool{}
			c.presentModsByPkg[pkg] = m
		}
		m[strings.Join(mod, "/")] = true
	}

	for _, imp := range c.prog.Imports {
		if imp != nil && imp.Span.File != nil {
			addFile(imp.Span.File.Name)
		}
	}
	for _, d := range c.prog.Funcs {
		if d != nil && d.Span.File != nil {
			addFile(d.Span.File.Name)
		}
	}
	for _, d := range c.prog.Structs {
		if d != nil && d.Span.File != nil {
			addFile(d.Span.File.Name)
		}
	}
	for _, d := range c.prog.Enums {
		if d != nil && d.Span.File != nil {
			addFile(d.Span.File.Name)
		}
	}
}

func (c *checker) hasPkgInProgram(pkg string) bool {
	c.initPresence()
	return c.presentPkgs[pkg]
}

func (c *checker) hasModuleInProgram(pkg string, modPath string) bool {
	c.initPresence()
	m := c.presentModsByPkg[pkg]
	if m == nil {
		return false
	}
	return m[modPath]
}

func (c *checker) collectImports() {
	for _, imp := range c.prog.Imports {
		if imp == nil || imp.Span.File == nil {
			continue
		}

		path := imp.Path
		alias := imp.Alias
		scheme, raw := splitImportScheme(path)

		ownerPkg, _, _ := names.SplitOwnerAndModule(imp.Span.File.Name)

		var tgt importTarget
		switch scheme {
		case "std":
			// std:... always resolves to the root std module namespace.
			localPath := "std/" + raw
			if !c.isKnownLocalModule(imp.Span.File.Name, localPath) {
				c.errorAt(imp.Span, "unknown local module: "+localPath)
				continue
			}
			tgt = importTarget{Pkg: "", Mod: splitModPath(localPath)}
			if alias == "" {
				alias = defaultImportAlias(localPath)
			}
		case "mod":
			// mod:... forces local module within the importing package.
			localPath := raw
			// Special-case std: allow importing std/** from any package without requiring a local copy.
			localOwnerPkg := ownerPkg
			if localPath == "std" || strings.HasPrefix(localPath, "std/") {
				localOwnerPkg = ""
			}
			if !c.isKnownLocalModule(imp.Span.File.Name, localPath) {
				c.errorAt(imp.Span, "unknown local module: "+localPath)
				continue
			}
			tgt = importTarget{Pkg: localOwnerPkg, Mod: splitModPath(localPath)}
			if alias == "" {
				alias = defaultImportAlias(localPath)
			}
		case "pkg":
			// pkg:... forces dependency package.
			depName, depMod, _ := strings.Cut(raw, "/")
			if depName == "" {
				c.errorAt(imp.Span, "invalid dependency import: "+raw)
				continue
			}
			if c.opts.AllowedPkgs != nil && !c.opts.AllowedPkgs[depName] {
				c.errorAt(imp.Span, "unknown dependency package: "+depName)
				continue
			}
			tgt = importTarget{Pkg: internalPkgName(depName), Mod: splitModPath(depMod)}
			if alias == "" {
				alias = defaultImportAlias(raw)
			}
		default:
			// Default: local module wins; if both local and dep exist, require explicit pkg:/mod:.
			if raw == "" {
				c.errorAt(imp.Span, "invalid import path")
				continue
			}

			// Treat std/** as a root module, regardless of which package is importing it.
			localOwnerPkg := ownerPkg
			if raw == "std" || strings.HasPrefix(raw, "std/") {
				localOwnerPkg = ""
			}

			depName, depMod, _ := strings.Cut(raw, "/")
			depPkg := internalPkgName(depName)

			hasDep := depName != "" && c.hasPkgInProgram(depPkg)
			hasLocal := c.hasModuleInProgram(localOwnerPkg, raw)

			if hasDep && hasLocal {
				c.errorAt(imp.Span, "ambiguous import: "+raw+" (use pkg: or mod:)")
				continue
			}

			// If neither is present in the parsed program, fall back to manifest-based validation if provided.
			if !hasDep && !hasLocal && c.opts.AllowedPkgs != nil && c.opts.AllowedPkgs[depName] {
				hasDep = true
			}

			if hasDep {
				tgt = importTarget{Pkg: depPkg, Mod: splitModPath(depMod)}
				if alias == "" {
					alias = defaultImportAlias(raw)
				}
				break
			}

			// local module import (in-package, unless it's std/** handled above)
			if !c.isKnownLocalModule(imp.Span.File.Name, raw) {
				c.errorAt(imp.Span, "unknown local module: "+raw)
				continue
			}
			tgt = importTarget{Pkg: localOwnerPkg, Mod: splitModPath(raw)}
			if alias == "" {
				alias = defaultImportAlias(raw)
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
