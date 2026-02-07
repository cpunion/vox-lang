package typecheck

import (
	"voxlang/internal/ast"
	"voxlang/internal/diag"
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
	TyVec
)

type Type struct {
	K Kind
	// Name is set when K == TyStruct, using the qualified name format:
	//   "<pkg>::<mod.path>::<Name>" or "<Name>" for root package/root module.
	Name string
	// Elem is set when K == TyVec.
	Elem *Type
}

type CheckedProgram struct {
	Prog       *ast.Program
	FuncSigs   map[string]FuncSig
	StructSigs map[string]StructSig
	EnumSigs   map[string]EnumSig
	ExprTypes  map[ast.Expr]Type
	LetTypes   map[*ast.LetStmt]Type
	VecCalls   map[*ast.CallExpr]VecCallTarget
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

type VecCallKind int

const (
	VecCallBad VecCallKind = iota
	VecCallNew
	VecCallPush
	VecCallLen
	VecCallGet
)

type VecCallTarget struct {
	Kind     VecCallKind
	RecvName string // for methods on local variables
	Elem     Type
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
		vecCalls:   map[*ast.CallExpr]VecCallTarget{},
		callTgts:   map[*ast.CallExpr]string{},
		enumCtors:  map[*ast.CallExpr]EnumCtorTarget{},
		opts:       opts,
		imports:    map[*source.File]map[string]importTarget{},
		namedFuncs: map[*source.File]map[string]string{},
		namedTypes: map[*source.File]map[string]Type{},
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
		VecCalls:    c.vecCalls,
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
	vecCalls   map[*ast.CallExpr]VecCallTarget
	callTgts   map[*ast.CallExpr]string
	enumCtors  map[*ast.CallExpr]EnumCtorTarget

	curFn     *ast.FuncDecl
	scope     []map[string]varInfo
	loopDepth int

	opts    Options
	imports map[*source.File]map[string]importTarget // file -> qualifier -> target

	// Named function imports: file -> localName -> qualifiedTarget.
	namedFuncs map[*source.File]map[string]string
	// Named type imports: file -> localName -> qualified type.
	namedTypes map[*source.File]map[string]Type
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

// String and all checker implementation details live in other files (split for maintainability).
