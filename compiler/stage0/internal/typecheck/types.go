package typecheck

import (
	"fmt"
	"strconv"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/diag"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

type Kind int

const (
	TyBad Kind = iota
	TyUnit
	TyBool
	TyI32
	TyI64
	TyString
	TyUntypedInt
	TyStruct
	TyEnum
)

type Type struct {
	K Kind
	// Name is set when K == TyStruct, using the qualified name format:
	//   "<pkg>::<mod.path>::<Name>" or "<Name>" for root package/root module.
	Name string
}

func (t Type) String() string {
	switch t.K {
	case TyUnit:
		return "()"
	case TyBool:
		return "bool"
	case TyI32:
		return "i32"
	case TyI64:
		return "i64"
	case TyString:
		return "String"
	case TyUntypedInt:
		return "untyped-int"
	case TyStruct:
		return t.Name
	case TyEnum:
		return t.Name
	default:
		return "<bad>"
	}
}

type CheckedProgram struct {
	Prog       *ast.Program
	FuncSigs   map[string]FuncSig
	StructSigs map[string]StructSig
	EnumSigs   map[string]EnumSig
	ExprTypes  map[ast.Expr]Type
	LetTypes   map[*ast.LetStmt]Type
	// CallTargets stores the resolved function name (possibly qualified, e.g. "dep::foo").
	// Note: Vox surface syntax may use `dep.foo(...)`; it still resolves to `dep::foo` internally.
	// for each call expression.
	CallTargets map[*ast.CallExpr]string
	// EnumCtors records which call expressions are enum constructors instead of function calls.
	EnumCtors map[*ast.CallExpr]EnumCtorTarget
}

type FuncSig struct {
	Pub      bool
	OwnerPkg string
	OwnerMod []string
	Params   []Type
	Ret      Type
}

type StructSig struct {
	Name       string
	Pub        bool
	OwnerPkg   string
	OwnerMod   []string
	Fields     []StructFieldSig
	FieldIndex map[string]int
}

type StructFieldSig struct {
	Pub  bool
	Name string
	Ty   Type
}

type EnumSig struct {
	Name         string
	Pub          bool
	OwnerPkg     string
	OwnerMod     []string
	Variants     []EnumVariantSig
	VariantIndex map[string]int
}

type EnumVariantSig struct {
	Name   string
	Fields []Type // stage0: arity 0/1
}

type EnumCtorTarget struct {
	Enum    Type // TyEnum
	Variant string
	Tag     int
	// Payload is TyUnit for unit variants.
	Payload Type
}

type Options struct {
	// AllowedPkgs is the set of importable package names in the current build.
	// When nil, imports are accepted without validation.
	AllowedPkgs map[string]bool
	// LocalModules is the set of importable local modules (paths like "utils" or "utils/io").
	// When nil, local module imports are accepted without validation.
	LocalModules map[string]bool
	// LocalModulesByPkg validates local module imports per package.
	// Key is dependency package name; root package uses "".
	// When nil, LocalModules is used as a fallback.
	LocalModulesByPkg map[string]map[string]bool
}

func Check(prog *ast.Program, opts Options) (*CheckedProgram, *diag.Bag) {
	c := &checker{
		prog:       prog,
		diags:      &diag.Bag{},
		funcSigs:   map[string]FuncSig{},
		structSigs: map[string]StructSig{},
		enumSigs:   map[string]EnumSig{},
		exprTypes:  map[ast.Expr]Type{},
		letTypes:   map[*ast.LetStmt]Type{},
		callTgts:   map[*ast.CallExpr]string{},
		enumCtors:  map[*ast.CallExpr]EnumCtorTarget{},
		opts:       opts,
		imports:    map[*source.File]map[string]importTarget{},
		namedFuncs: map[*source.File]map[string]string{},
	}
	c.collectImports()
	c.collectStructSigs()
	c.collectEnumSigs()
	c.collectFuncSigs()
	c.resolveNamedImports()
	c.checkPubInterfaces()
	c.checkAll()
	return &CheckedProgram{
		Prog:        prog,
		FuncSigs:    c.funcSigs,
		StructSigs:  c.structSigs,
		EnumSigs:    c.enumSigs,
		ExprTypes:   c.exprTypes,
		LetTypes:    c.letTypes,
		CallTargets: c.callTgts,
		EnumCtors:   c.enumCtors,
	}, c.diags
}

type checker struct {
	prog       *ast.Program
	diags      *diag.Bag
	funcSigs   map[string]FuncSig
	structSigs map[string]StructSig
	enumSigs   map[string]EnumSig
	exprTypes  map[ast.Expr]Type
	letTypes   map[*ast.LetStmt]Type
	callTgts   map[*ast.CallExpr]string
	enumCtors  map[*ast.CallExpr]EnumCtorTarget

	curFn     *ast.FuncDecl
	scope     []map[string]varInfo
	loopDepth int

	opts    Options
	imports map[*source.File]map[string]importTarget // file -> qualifier -> target

	// Named function imports: file -> localName -> qualifiedTarget.
	namedFuncs map[*source.File]map[string]string
	pending    []pendingNamedImport
}

type varInfo struct {
	ty      Type
	mutable bool
}

type importTarget struct {
	Pkg string   // dependency package name; empty for local modules
	Mod []string // base module path segments
}

type pendingNamedImport struct {
	File  *source.File
	Names []ast.ImportName
	Tgt   importTarget
	Span  source.Span
}

func sameModPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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
	checkTy := func(at source.Span, ownerPkg string, ownerMod []string, t Type, what string) {
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
		ss, ok := c.structSigs[qname]
		if !ok {
			continue
		}
		ownerPkg, ownerMod, _ := names.SplitOwnerAndModule(st.Span.File.Name)
		_ = ss
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

func (c *checker) checkAll() {
	for _, fn := range c.prog.Funcs {
		c.curFn = fn
		qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
		sig := c.funcSigs[qname]
		c.pushScope()
		for i, p := range fn.Params {
			c.scopeTop()[p.Name] = varInfo{ty: sig.Params[i], mutable: false}
		}
		c.checkBlock(fn.Body, sig.Ret)
		c.popScope()
	}
}

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
		m := c.namedFuncs[pi.File]
		if m == nil {
			m = map[string]string{}
			c.namedFuncs[pi.File] = m
		}
		for _, nm := range pi.Names {
			local := nm.Alias
			if local == "" {
				local = nm.Name
			}
			if local == "" {
				continue
			}
			if _, exists := m[local]; exists {
				c.errorAt(pi.Span, "duplicate imported name: "+local)
				continue
			}

			target := names.QualifyParts(pi.Tgt.Pkg, pi.Tgt.Mod, nm.Name)
			sig, ok := c.funcSigs[target]
			if !ok {
				c.errorAt(pi.Span, "unknown imported function: "+target)
				continue
			}
			if !c.canAccess(pi.File, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
				c.errorAt(pi.Span, "function is private: "+target)
				continue
			}
			m[local] = target
		}
	}
}

func (c *checker) checkBlock(b *ast.BlockStmt, expectedRet Type) {
	c.pushScope()
	for _, st := range b.Stmts {
		c.checkStmt(st, expectedRet)
	}
	c.popScope()
}

func (c *checker) checkStmt(st ast.Stmt, expectedRet Type) {
	switch s := st.(type) {
	case *ast.BlockStmt:
		c.checkBlock(s, expectedRet)
	case *ast.LetStmt:
		var ann Type
		if s.AnnType != nil {
			ann = c.typeFromAstInFile(s.AnnType, c.curFn.Span.File)
		}
		var initTy Type
		if s.Init != nil {
			initTy = c.checkExpr(s.Init, ann)
			if initTy.K == TyUntypedInt && ann.K == TyBad {
				initTy = Type{K: TyI64}
			}
			if ann.K != TyBad && !sameType(ann, initTy) {
				c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", ann.String(), initTy.String()))
			}
		} else {
			if ann.K == TyBad {
				c.errorAt(s.S, "let binding requires a type annotation or initializer")
				ann = Type{K: TyBad}
			}
			initTy = ann
		}
		final := chooseType(ann, initTy)
		c.scopeTop()[s.Name] = varInfo{ty: final, mutable: s.Mutable}
		c.letTypes[s] = final
	case *ast.AssignStmt:
		vi, ok := c.lookupVar(s.Name)
		if !ok {
			c.errorAt(s.S, "unknown variable: "+s.Name)
			return
		}
		if !vi.mutable {
			c.errorAt(s.S, "cannot assign to immutable variable: "+s.Name)
			return
		}
		rhs := c.checkExpr(s.Expr, vi.ty)
		if !sameType(vi.ty, rhs) {
			c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", vi.ty.String(), rhs.String()))
		}
	case *ast.FieldAssignStmt:
		vi, ok := c.lookupVar(s.Recv)
		if !ok {
			c.errorAt(s.S, "unknown variable: "+s.Recv)
			return
		}
		if !vi.mutable {
			c.errorAt(s.S, "cannot assign to immutable variable: "+s.Recv)
			return
		}
		if vi.ty.K != TyStruct {
			c.errorAt(s.S, "field assignment requires a struct receiver")
			return
		}
		ss, ok := c.structSigs[vi.ty.Name]
		if !ok {
			c.errorAt(s.S, "unknown struct type: "+vi.ty.Name)
			return
		}
		idx, ok := ss.FieldIndex[s.Field]
		if !ok {
			c.errorAt(s.S, "unknown field: "+s.Field)
			return
		}
		want := ss.Fields[idx].Ty
		got := c.checkExpr(s.Expr, want)
		if !sameType(want, got) {
			c.errorAt(s.S, fmt.Sprintf("type mismatch: expected %s, got %s", want.String(), got.String()))
		}
	case *ast.ReturnStmt:
		var ty Type
		if s.Expr == nil {
			ty = Type{K: TyUnit}
		} else {
			ty = c.checkExpr(s.Expr, expectedRet)
		}
		if !sameType(expectedRet, ty) {
			c.errorAt(s.S, fmt.Sprintf("return type mismatch: expected %s, got %s", expectedRet.String(), ty.String()))
		}
	case *ast.IfStmt:
		condTy := c.checkExpr(s.Cond, Type{K: TyBool})
		if condTy.K != TyBool {
			c.errorAt(s.Cond.Span(), "if condition must be bool")
		}
		c.checkBlock(s.Then, expectedRet)
		if s.Else != nil {
			c.checkStmt(s.Else, expectedRet)
		}
	case *ast.WhileStmt:
		condTy := c.checkExpr(s.Cond, Type{K: TyBool})
		if condTy.K != TyBool {
			c.errorAt(s.Cond.Span(), "while condition must be bool")
		}
		c.loopDepth++
		c.checkBlock(s.Body, expectedRet)
		c.loopDepth--
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			c.errorAt(s.S, "`break` outside of loop")
		}
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			c.errorAt(s.S, "`continue` outside of loop")
		}
	case *ast.ExprStmt:
		_ = c.checkExpr(s.Expr, Type{K: TyBad})
	default:
		c.errorAt(st.Span(), "unsupported statement")
	}
}

func (c *checker) checkExpr(ex ast.Expr, expected Type) Type {
	switch e := ex.(type) {
	case *ast.IntLit:
		v, err := strconv.ParseInt(e.Text, 10, 64)
		if err != nil {
			c.errorAt(e.S, "invalid integer literal")
			return c.setExprType(ex, Type{K: TyBad})
		}
		// Constrain by expected type if it's an int.
		if expected.K == TyI32 {
			if v < -2147483648 || v > 2147483647 {
				c.errorAt(e.S, "integer literal out of range for i32")
				return c.setExprType(ex, Type{K: TyBad})
			}
			return c.setExprType(ex, expected)
		}
		if expected.K == TyI64 {
			return c.setExprType(ex, expected)
		}
		return c.setExprType(ex, Type{K: TyUntypedInt})
	case *ast.StringLit:
		return c.setExprType(ex, Type{K: TyString})
	case *ast.BoolLit:
		return c.setExprType(ex, Type{K: TyBool})
	case *ast.IdentExpr:
		if vi, ok := c.lookupVar(e.Name); ok {
			return c.setExprType(ex, vi.ty)
		}
		// function name is allowed as callee only
		if _, ok := c.funcSigs[e.Name]; ok {
			return c.setExprType(ex, Type{K: TyBad})
		}
		c.errorAt(e.S, "unknown identifier: "+e.Name)
		return c.setExprType(ex, Type{K: TyBad})
	case *ast.UnaryExpr:
		switch e.Op {
		case "-":
			ty := c.checkExpr(e.Expr, Type{K: TyI64})
			ty = c.forceIntType(e.Expr, ty, expected)
			return c.setExprType(ex, ty)
		case "!":
			ty := c.checkExpr(e.Expr, Type{K: TyBool})
			if ty.K != TyBool {
				c.errorAt(e.S, "operator ! expects bool")
			}
			return c.setExprType(ex, Type{K: TyBool})
		default:
			c.errorAt(e.S, "unknown unary operator: "+e.Op)
			return c.setExprType(ex, Type{K: TyBad})
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case "+", "-", "*", "/", "%":
			l := c.checkExpr(e.Left, expected)
			r := c.checkExpr(e.Right, l)
			l = c.forceIntType(e.Left, l, expected)
			r = c.forceIntType(e.Right, r, l)
			if !sameType(l, r) {
				c.errorAt(e.S, "binary integer ops require same type")
				return c.setExprType(ex, Type{K: TyBad})
			}
			return c.setExprType(ex, l)
		case "<", "<=", ">", ">=":
			l := c.checkExpr(e.Left, Type{K: TyI64})
			r := c.checkExpr(e.Right, l)
			l = c.forceIntType(e.Left, l, Type{K: TyI64})
			r = c.forceIntType(e.Right, r, l)
			if !sameType(l, r) {
				c.errorAt(e.S, "comparison requires same integer type")
			}
			return c.setExprType(ex, Type{K: TyBool})
		case "==", "!=":
			l := c.checkExpr(e.Left, Type{K: TyBad})
			r := c.checkExpr(e.Right, l)
			if l.K == TyUntypedInt || r.K == TyUntypedInt {
				// default to i64
				l = c.forceIntType(e.Left, l, Type{K: TyI64})
				r = c.forceIntType(e.Right, r, l)
			}
			if !sameType(l, r) {
				c.errorAt(e.S, "equality requires same type")
			}
			return c.setExprType(ex, Type{K: TyBool})
		case "&&", "||":
			l := c.checkExpr(e.Left, Type{K: TyBool})
			r := c.checkExpr(e.Right, Type{K: TyBool})
			if l.K != TyBool || r.K != TyBool {
				c.errorAt(e.S, "logical operators require bool")
			}
			return c.setExprType(ex, Type{K: TyBool})
		default:
			c.errorAt(e.S, "unknown operator: "+e.Op)
			return c.setExprType(ex, Type{K: TyBad})
		}
	case *ast.CallExpr:
		parts, ok := calleeParts(e.Callee)
		if !ok || len(parts) == 0 {
			c.errorAt(e.S, "callee must be an identifier or member path (stage0)")
			return c.setExprType(ex, Type{K: TyBad})
		}

		// Enum constructor: `Enum.Variant(...)` (including qualified types like `dep.Option.Some(...)`).
		if len(parts) >= 2 {
			alias := parts[0]
			if _, ok := c.lookupVar(alias); ok {
				c.errorAt(e.S, "member calls on values are not supported yet")
				return c.setExprType(ex, Type{K: TyBad})
			}
			if c.curFn == nil || c.curFn.Span.File == nil {
				c.errorAt(e.S, "internal error: missing file for call resolution")
				return c.setExprType(ex, Type{K: TyBad})
			}
			ety, es, found := c.findEnumByParts(c.curFn.Span.File, parts[:len(parts)-1])
			if found {
				if !c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
					c.errorAt(e.S, "type is private: "+ety.Name)
					return c.setExprType(ex, Type{K: TyBad})
				}
				varName := parts[len(parts)-1]
				vidx, vok := es.VariantIndex[varName]
				if !vok {
					c.errorAt(e.S, "unknown variant: "+varName)
					return c.setExprType(ex, Type{K: TyBad})
				}
				vs := es.Variants[vidx]
				if len(vs.Fields) > 1 {
					c.errorAt(e.S, "stage0 enum payload arity > 1 is not supported yet")
					return c.setExprType(ex, Type{K: TyBad})
				}
				if len(e.Args) != len(vs.Fields) {
					c.errorAt(e.S, fmt.Sprintf("wrong number of arguments: expected %d, got %d", len(vs.Fields), len(e.Args)))
					return c.setExprType(ex, Type{K: TyBad})
				}
				for i, a := range e.Args {
					at := c.checkExpr(a, vs.Fields[i])
					if !sameType(vs.Fields[i], at) {
						c.errorAt(a.Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", vs.Fields[i].String(), at.String()))
					}
				}
				payload := Type{K: TyUnit}
				if len(vs.Fields) == 1 {
					payload = vs.Fields[0]
				}
				c.enumCtors[e] = EnumCtorTarget{Enum: ety, Variant: varName, Tag: vidx, Payload: payload}
				return c.setExprType(ex, ety)
			}
		}

		target := ""
		if len(parts) == 1 {
			name := parts[0]
			pkg, mod, _ := names.SplitOwnerAndModule(c.curFn.Span.File.Name)
			// 1) current module
			q := names.QualifyParts(pkg, mod, name)
			if _, ok := c.funcSigs[q]; ok {
				target = q
			} else {
				// 2) named imports for this file
				if c.curFn.Span.File != nil {
					if m := c.namedFuncs[c.curFn.Span.File]; m != nil {
						if tgt, ok := m[name]; ok {
							target = tgt
						}
					}
				}
				// 3) fallback: root module of the same package
				if target == "" {
					q2 := names.QualifyParts(pkg, nil, name)
					if _, ok := c.funcSigs[q2]; ok {
						target = q2
					}
				}
			}
		} else {
			// Qualified call: first segment must be an imported alias.
			qualParts := parts[:len(parts)-1]
			member := parts[len(parts)-1]
			alias := qualParts[0]
			extraMods := qualParts[1:]

			if c.curFn.Span.File == nil {
				c.errorAt(e.S, "internal error: missing file for import resolution")
				return c.setExprType(ex, Type{K: TyBad})
			}
			m := c.imports[c.curFn.Span.File]
			tgt, ok := m[alias]
			if !ok {
				c.errorAt(e.S, "unknown module qualifier: "+alias+" (did you forget `import \""+alias+"\"`?)")
				return c.setExprType(ex, Type{K: TyBad})
			}
			mod := append(append([]string{}, tgt.Mod...), extraMods...)
			target = names.QualifyParts(tgt.Pkg, mod, member)
		}

		if target == "" {
			c.errorAt(e.S, "unknown function")
			return c.setExprType(ex, Type{K: TyBad})
		}
		sig, ok := c.funcSigs[target]
		if !ok {
			c.errorAt(e.S, "unknown function: "+target)
			return c.setExprType(ex, Type{K: TyBad})
		}
		if c.curFn != nil && c.curFn.Span.File != nil && !c.canAccess(c.curFn.Span.File, sig.OwnerPkg, sig.OwnerMod, sig.Pub) {
			c.errorAt(e.S, "function is private: "+target)
			return c.setExprType(ex, Type{K: TyBad})
		}
		c.callTgts[e] = target
		if len(e.Args) != len(sig.Params) {
			c.errorAt(e.S, fmt.Sprintf("wrong number of arguments: expected %d, got %d", len(sig.Params), len(e.Args)))
			return c.setExprType(ex, Type{K: TyBad})
		}
		for i, a := range e.Args {
			at := c.checkExpr(a, sig.Params[i])
			if !sameType(sig.Params[i], at) {
				c.errorAt(a.Span(), fmt.Sprintf("argument type mismatch: expected %s, got %s", sig.Params[i].String(), at.String()))
			}
		}
		return c.setExprType(ex, sig.Ret)
	case *ast.MemberExpr:
		// Unit enum variant: `Enum.Variant` (including qualified enum types).
		if c.curFn != nil && c.curFn.Span.File != nil {
			parts, ok := calleeParts(ex)
			if ok && len(parts) >= 2 {
				alias := parts[0]
				if _, ok := c.lookupVar(alias); !ok {
					ety, es, found := c.findEnumByParts(c.curFn.Span.File, parts[:len(parts)-1])
					if found && !c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
						c.errorAt(e.S, "type is private: "+ety.Name)
						return c.setExprType(ex, Type{K: TyBad})
					}
					if found && c.canAccess(c.curFn.Span.File, es.OwnerPkg, es.OwnerMod, es.Pub) {
						vname := parts[len(parts)-1]
						vidx, vok := es.VariantIndex[vname]
						if vok && len(es.Variants[vidx].Fields) == 0 {
							return c.setExprType(ex, ety)
						}
					}
				}
			}
		}

		recvTy := c.checkExpr(e.Recv, Type{K: TyBad})
		if recvTy.K != TyStruct {
			c.errorAt(e.S, "member access requires a struct receiver")
			return c.setExprType(ex, Type{K: TyBad})
		}
		ss, ok := c.structSigs[recvTy.Name]
		if !ok {
			c.errorAt(e.S, "unknown struct type: "+recvTy.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		idx, ok := ss.FieldIndex[e.Name]
		if !ok {
			c.errorAt(e.S, "unknown field: "+e.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		if c.curFn != nil && c.curFn.Span.File != nil && !c.isSameModule(c.curFn.Span.File, ss.OwnerPkg, ss.OwnerMod) && !ss.Fields[idx].Pub {
			c.errorAt(e.S, "field is private: "+recvTy.Name+"."+e.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}
		return c.setExprType(ex, ss.Fields[idx].Ty)
	case *ast.StructLitExpr:
		if e.S.File == nil {
			c.errorAt(e.S, "internal error: missing file for struct literal")
			return c.setExprType(ex, Type{K: TyBad})
		}
		sty, ss, ok := c.resolveStructByParts(e.S.File, e.TypeParts, e.S)
		if !ok {
			return c.setExprType(ex, Type{K: TyBad})
		}
		if !c.isSameModule(e.S.File, ss.OwnerPkg, ss.OwnerMod) {
			for _, f := range ss.Fields {
				if !f.Pub {
					c.errorAt(e.S, "cannot construct struct "+sty.Name+": field "+f.Name+" is private")
					return c.setExprType(ex, Type{K: TyBad})
				}
			}
		}
		seen := map[string]bool{}
		for _, init := range e.Inits {
			if seen[init.Name] {
				c.errorAt(init.Span, "duplicate field init: "+init.Name)
				continue
			}
			seen[init.Name] = true
			idx, ok := ss.FieldIndex[init.Name]
			if !ok {
				c.errorAt(init.Span, "unknown field: "+init.Name)
				continue
			}
			if !c.isSameModule(e.S.File, ss.OwnerPkg, ss.OwnerMod) && !ss.Fields[idx].Pub {
				c.errorAt(init.Span, "field is private: "+sty.Name+"."+init.Name)
				continue
			}
			want := ss.Fields[idx].Ty
			got := c.checkExpr(init.Expr, want)
			if !sameType(want, got) {
				c.errorAt(init.Expr.Span(), fmt.Sprintf("field %s type mismatch: expected %s, got %s", init.Name, want.String(), got.String()))
			}
		}
		for _, f := range ss.Fields {
			if !seen[f.Name] {
				c.errorAt(e.S, "missing field init: "+f.Name)
			}
		}
		return c.setExprType(ex, sty)
	case *ast.MatchExpr:
		scrutTy := c.checkExpr(e.Scrutinee, Type{K: TyBad})
		if scrutTy.K != TyEnum {
			c.errorAt(e.S, "match scrutinee must be an enum (stage0)")
			return c.setExprType(ex, Type{K: TyBad})
		}
		esig, ok := c.enumSigs[scrutTy.Name]
		if !ok {
			c.errorAt(e.S, "unknown enum type: "+scrutTy.Name)
			return c.setExprType(ex, Type{K: TyBad})
		}

		resultTy := expected
		seenVariants := map[string]bool{}
		hasWild := false

		for _, arm := range e.Arms {
			c.pushScope()
			switch p := arm.Pat.(type) {
			case *ast.WildPat:
				hasWild = true
			case *ast.VariantPat:
				if c.curFn == nil || c.curFn.Span.File == nil {
					c.errorAt(arm.S, "internal error: missing file for match")
					break
				}
				pty, psig, ok := c.resolveEnumByParts(c.curFn.Span.File, p.TypeParts, p.S)
				if ok && !sameType(scrutTy, pty) {
					c.errorAt(p.S, "pattern enum type does not match scrutinee")
				}
				vidx, vok := psig.VariantIndex[p.Variant]
				if !vok {
					c.errorAt(p.S, "unknown variant: "+p.Variant)
					break
				}
				if seenVariants[p.Variant] {
					c.errorAt(p.S, "duplicate match arm for variant: "+p.Variant)
				}
				seenVariants[p.Variant] = true
				v := psig.Variants[vidx]
				if len(v.Fields) > 1 {
					c.errorAt(p.S, "stage0 enum payload arity > 1 is not supported yet")
				}
				if len(p.Binds) != len(v.Fields) {
					c.errorAt(p.S, fmt.Sprintf("wrong number of binders: expected %d, got %d", len(v.Fields), len(p.Binds)))
				}
				for i := 0; i < len(p.Binds) && i < len(v.Fields); i++ {
					c.scopeTop()[p.Binds[i]] = varInfo{ty: v.Fields[i], mutable: false}
				}
			default:
				c.errorAt(arm.S, "unsupported pattern (stage0)")
			}

			armTy := c.checkExpr(arm.Expr, resultTy)
			if resultTy.K == TyBad {
				if armTy.K == TyUntypedInt {
					resultTy = Type{K: TyI64}
				} else {
					resultTy = armTy
				}
			} else if !sameType(resultTy, armTy) {
				c.errorAt(arm.S, fmt.Sprintf("match arm type mismatch: expected %s, got %s", resultTy.String(), armTy.String()))
			}
			c.popScope()
		}

		if !hasWild {
			for _, v := range esig.Variants {
				if !seenVariants[v.Name] {
					c.errorAt(e.S, "non-exhaustive match, missing variant: "+v.Name)
				}
			}
		}
		return c.setExprType(ex, resultTy)
	default:
		c.errorAt(ex.Span(), "unsupported expression")
		return c.setExprType(ex, Type{K: TyBad})
	}
}

func (c *checker) forceIntType(ex ast.Expr, got Type, want Type) Type {
	if got.K == TyUntypedInt {
		// untyped int defaults to "want" if want is concrete int, else i64.
		if want.K == TyI32 || want.K == TyI64 {
			return want
		}
		return Type{K: TyI64}
	}
	if got.K == TyI32 || got.K == TyI64 {
		return got
	}
	return got
}

func (c *checker) typeFromAstInFile(t ast.Type, file *source.File) Type {
	switch tt := t.(type) {
	case *ast.UnitType:
		return Type{K: TyUnit}
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
			c.errorAt(tt.S, "unknown type: "+q)
			return Type{K: TyBad}
		}

		// Single-segment types: builtins or local/root nominal types.
		name := tt.Parts[0]
		switch name {
		case "i32":
			return Type{K: TyI32}
		case "i64":
			return Type{K: TyI64}
		case "bool":
			return Type{K: TyBool}
		case "String":
			return Type{K: TyString}
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

func (c *checker) setExprType(ex ast.Expr, t Type) Type {
	c.exprTypes[ex] = t
	return t
}

func (c *checker) pushScope() { c.scope = append(c.scope, map[string]varInfo{}) }
func (c *checker) popScope()  { c.scope = c.scope[:len(c.scope)-1] }
func (c *checker) scopeTop() map[string]varInfo {
	return c.scope[len(c.scope)-1]
}

func (c *checker) lookupVar(name string) (varInfo, bool) {
	for i := len(c.scope) - 1; i >= 0; i-- {
		if v, ok := c.scope[i][name]; ok {
			return v, true
		}
	}
	return varInfo{}, false
}

func (c *checker) errorAt(s source.Span, msg string) {
	fn, line, col := s.LocStart()
	c.diags.Add(fn, line, col, msg)
}

func sameType(a, b Type) bool {
	if a.K == TyBad || b.K == TyBad {
		return false
	}
	// Resolve untyped int to concrete only via constraints before comparing.
	if a.K == TyUntypedInt || b.K == TyUntypedInt {
		return false
	}
	if a.K != b.K {
		return false
	}
	if a.K == TyStruct || a.K == TyEnum {
		return a.Name == b.Name
	}
	return true
}

func chooseType(ann, init Type) Type {
	if ann.K != TyBad {
		return ann
	}
	return init
}

func defaultImportAlias(path string) string {
	// For stage0, dependency package import paths are simple names like "mathlib".
	// If we later support nested module paths, use the last segment.
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func splitModPath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return out
}

func calleeParts(ex ast.Expr) ([]string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return []string{e.Name}, true
	case *ast.MemberExpr:
		p, ok := calleeParts(e.Recv)
		if !ok {
			return nil, false
		}
		return append(p, e.Name), true
	default:
		return nil, false
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
