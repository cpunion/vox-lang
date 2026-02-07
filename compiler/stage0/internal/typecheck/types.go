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
	TyParam
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
	StrCalls   map[*ast.CallExpr]StrCallTarget
	ToStrCalls map[*ast.CallExpr]ToStrTarget
	// CallTargets stores the resolved function name (possibly qualified, e.g. "dep::foo").
	// Note: Vox surface syntax may use `dep.foo(...)`; it still resolves to `dep::foo` internally.
	// for each call expression.
	CallTargets map[*ast.CallExpr]string
	// EnumCtors records which call expressions are enum constructors instead of function calls.
	EnumCtors map[*ast.CallExpr]EnumCtorTarget
	// EnumUnitVariants records which member expressions are unit enum variant values (Enum.Variant).
	// This disambiguates enum literals from struct-field access that returns an enum-typed value.
	EnumUnitVariants map[ast.Expr]EnumCtorTarget
}

type FuncSig struct {
	Pub        bool
	OwnerPkg   string
	OwnerMod   []string
	TypeParams []string
	Params     []Type
	Ret        Type
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
	Fields []Type // tuple-like payload, arity 0..N
}

type EnumCtorTarget struct {
	Enum    Type // TyEnum
	Variant string
	Tag     int
	// Fields is empty for unit variants.
	Fields []Type
}

type VecCallKind int

const (
	VecCallBad VecCallKind = iota
	VecCallNew
	VecCallPush
	VecCallLen
	VecCallGet
	VecCallJoin // Vec[String].join(sep)
)

type VecCallTarget struct {
	Kind VecCallKind
	// RecvName is used when the receiver is a local variable (addressable slot).
	// This is the most direct lowering path for operations that mutate the receiver (e.g. Vec.push).
	RecvName string
	// Recv is used when the receiver is not a simple local variable.
	// - For non-mutating operations, it can be any value expression (e.g. s.items.len()).
	// - For Vec.push, it must be a supported "place" expression (stage0: ident.field).
	Recv ast.Expr
	Elem Type
}

type StrCallKind int

const (
	StrCallBad StrCallKind = iota
	StrCallLen
	StrCallByteAt
	StrCallSlice
	StrCallConcat  // String.concat(String)
	StrCallEscapeC // String.escape_c()
)

type StrCallTarget struct {
	Kind     StrCallKind
	RecvName string
	Recv     ast.Expr
}

type ToStrKind int

const (
	ToStrBad ToStrKind = iota
	ToStrI32
	ToStrI64
	ToStrBool
)

type ToStrTarget struct {
	Kind     ToStrKind
	RecvName string
	Recv     ast.Expr
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
		prog:          prog,
		diags:         &diag.Bag{},
		funcSigs:      map[string]FuncSig{},
		structSigs:    map[string]StructSig{},
		enumSigs:      map[string]EnumSig{},
		exprTypes:     map[ast.Expr]Type{},
		letTypes:      map[*ast.LetStmt]Type{},
		vecCalls:      map[*ast.CallExpr]VecCallTarget{},
		strCalls:      map[*ast.CallExpr]StrCallTarget{},
		toStrCalls:    map[*ast.CallExpr]ToStrTarget{},
		callTgts:      map[*ast.CallExpr]string{},
		enumCtors:     map[*ast.CallExpr]EnumCtorTarget{},
		enumUnits:     map[ast.Expr]EnumCtorTarget{},
		opts:          opts,
		imports:       map[*source.File]map[string]importTarget{},
		namedFuncs:    map[*source.File]map[string]string{},
		namedTypes:    map[*source.File]map[string]Type{},
		funcDecls:     map[string]*ast.FuncDecl{},
		instantiated:  map[string]bool{},
		instantiating: map[string]bool{},
	}
	c.collectImports()
	c.collectNominalSigs()
	c.fillStructSigs()
	c.fillEnumSigs()
	c.collectFuncSigs()
	c.resolveNamedImports()
	c.checkPubInterfaces()
	c.checkAll()
	return &CheckedProgram{
		Prog:             prog,
		FuncSigs:         c.funcSigs,
		StructSigs:       c.structSigs,
		EnumSigs:         c.enumSigs,
		ExprTypes:        c.exprTypes,
		LetTypes:         c.letTypes,
		VecCalls:         c.vecCalls,
		StrCalls:         c.strCalls,
		ToStrCalls:       c.toStrCalls,
		CallTargets:      c.callTgts,
		EnumCtors:        c.enumCtors,
		EnumUnitVariants: c.enumUnits,
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
	strCalls   map[*ast.CallExpr]StrCallTarget
	toStrCalls map[*ast.CallExpr]ToStrTarget
	callTgts   map[*ast.CallExpr]string
	enumCtors  map[*ast.CallExpr]EnumCtorTarget
	enumUnits  map[ast.Expr]EnumCtorTarget

	curFn     *ast.FuncDecl
	curTyVars map[string]bool
	scope     []map[string]varInfo
	loopDepth int

	opts    Options
	imports map[*source.File]map[string]importTarget // file -> qualifier -> target

	// Named function imports: file -> localName -> qualifiedTarget.
	namedFuncs map[*source.File]map[string]string
	// Named type imports: file -> localName -> qualified type.
	namedTypes map[*source.File]map[string]Type
	pending    []pendingNamedImport

	// Generic monomorphization (stage0 minimal).
	funcDecls     map[string]*ast.FuncDecl // qualified name -> decl (includes generic defs)
	instantiated  map[string]bool          // qualified name of instantiated concrete function
	pendingInsts  []pendingInstantiation
	instantiating map[string]bool // recursion guard
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
