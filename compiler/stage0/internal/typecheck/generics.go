package typecheck

import (
	"fmt"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/source"
)

type pendingInstantiation struct {
	BaseQname string
	InstName  string // unqualified within the same module (suffix added)
	Subst     map[string]Type
	At        source.Span
}

func (c *checker) maybeInstantiateCall(call *ast.CallExpr, baseQname string, sig FuncSig, expected Type) (string, FuncSig, bool) {
	if len(sig.TypeParams) == 0 {
		return "", FuncSig{}, false
	}

	subs := map[string]Type{}
	// First, use expected return type as a constraint when available.
	if expected.K != TyBad {
		_ = unifyType(sig.Ret, expected, subs)
	}

	// Explicit type args win.
	if len(call.TypeArgs) > 0 {
		if len(call.TypeArgs) != len(sig.TypeParams) {
			c.errorAt(call.S, fmt.Sprintf("wrong number of type arguments: expected %d, got %d", len(sig.TypeParams), len(call.TypeArgs)))
			return baseQname, sig, true
		}
		for i, tp := range sig.TypeParams {
			ty := c.typeFromAstInFile(call.TypeArgs[i], c.curFn.Span.File)
			subs[tp] = normalizeInferredType(ty)
		}
	}

	// Infer from arguments if needed.
	for i := 0; i < len(call.Args) && i < len(sig.Params); i++ {
		pt := substType(sig.Params[i], subs)
		at := c.checkExpr(call.Args[i], pt)
		if !unifyType(sig.Params[i], at, subs) {
			// diagnostics are emitted by the regular argument check below
		}
	}

	// Ensure all type params are bound.
	for _, tp := range sig.TypeParams {
		if _, ok := subs[tp]; !ok {
			c.errorAt(call.S, "cannot infer type parameter: "+tp)
			return baseQname, sig, true
		}
	}
	for k, v := range subs {
		subs[k] = normalizeInferredType(v)
	}

	instSig := sig
	instSig.TypeParams = nil
	instSig.Params = nil
	for _, p := range sig.Params {
		instSig.Params = append(instSig.Params, substType(p, subs))
	}
	instSig.Ret = substType(sig.Ret, subs)

	decl := c.funcDecls[baseQname]
	if decl == nil || decl.Span.File == nil {
		c.errorAt(call.S, "internal error: missing decl for generic instantiation")
		return baseQname, sig, true
	}

	instName := decl.Name + "$" + instSuffix(sig.TypeParams, subs)
	instQname := names.QualifyFunc(decl.Span.File.Name, instName)

	if _, ok := c.funcSigs[instQname]; !ok {
		c.pendingInsts = append(c.pendingInsts, pendingInstantiation{
			BaseQname: baseQname,
			InstName:  instName,
			Subst:     subs,
			At:        call.S,
		})
		c.funcSigs[instQname] = instSig
		c.instantiated[instQname] = false
	}
	return instQname, instSig, true
}

func normalizeInferredType(t Type) Type {
	if t.K == TyUntypedInt {
		return Type{K: TyI64}
	}
	return t
}

func substType(t Type, subs map[string]Type) Type {
	switch t.K {
	case TyParam:
		if subs == nil {
			return t
		}
		if r, ok := subs[t.Name]; ok {
			return r
		}
		return t
	case TyVec:
		if t.Elem == nil {
			return t
		}
		e := substType(*t.Elem, subs)
		return Type{K: TyVec, Elem: &e}
	case TyRange:
		if t.Base == nil {
			return t
		}
		b := substType(*t.Base, subs)
		return Type{K: TyRange, Base: &b, Lo: t.Lo, Hi: t.Hi}
	default:
		return t
	}
}

func unifyType(pattern Type, got Type, subs map[string]Type) bool {
	switch pattern.K {
	case TyParam:
		if subs == nil {
			return false
		}
		if cur, ok := subs[pattern.Name]; ok {
			return sameType(cur, got)
		}
		subs[pattern.Name] = got
		return true
	case TyVec:
		if got.K != TyVec || pattern.Elem == nil || got.Elem == nil {
			return false
		}
		return unifyType(*pattern.Elem, *got.Elem, subs)
	case TyRange:
		if got.K != TyRange || pattern.Base == nil || got.Base == nil {
			return false
		}
		if pattern.Lo != got.Lo || pattern.Hi != got.Hi {
			return false
		}
		return unifyType(*pattern.Base, *got.Base, subs)
	default:
		return sameType(pattern, got)
	}
}

func instSuffix(typeParams []string, subs map[string]Type) string {
	parts := make([]string, 0, len(typeParams))
	for _, tp := range typeParams {
		t := subs[tp]
		parts = append(parts, tp+"="+t.String())
	}
	raw := strings.Join(parts, ",")
	return mangleIdent(raw)
}

func mangleIdent(s string) string {
	// Collision-free-ish for our usage: keep [A-Za-z0-9], hex-escape everything else.
	// Always prefix with 'g' so it can't start with a digit.
	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteByte('g')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if ok {
			b.WriteByte(ch)
			continue
		}
		fmt.Fprintf(&b, "_%02x", ch)
	}
	return b.String()
}

func (c *checker) materializePendingInstantiations() {
	for len(c.pendingInsts) > 0 {
		pi := c.pendingInsts[0]
		c.pendingInsts = c.pendingInsts[1:]

		decl := c.funcDecls[pi.BaseQname]
		if decl == nil {
			continue
		}
		// Create a fresh AST clone so ExprTypes/CallTargets maps don't collide across instantiations.
		instDecl := cloneFuncDecl(decl)
		instDecl.Name = pi.InstName
		instDecl.TypeParams = nil

		// Substitute any explicit type args in calls and any let type annotations.
		substAstTypesInFunc(instDecl, pi.Subst, decl.Span.File)

		instQname := names.QualifyFunc(decl.Span.File.Name, instDecl.Name)
		if c.instantiating[instQname] {
			continue
		}
		c.instantiating[instQname] = true
		c.funcDecls[instQname] = instDecl
		c.prog.Funcs = append(c.prog.Funcs, instDecl)
		c.instantiated[instQname] = true
		c.instantiating[instQname] = false
	}
}

func substAstTypesInFunc(fn *ast.FuncDecl, subs map[string]Type, file *source.File) {
	if fn == nil {
		return
	}
	// signature (best-effort)
	for i := range fn.Params {
		fn.Params[i].Type = substAstType(fn.Params[i].Type, subs)
	}
	fn.Ret = substAstType(fn.Ret, subs)
	// body
	substAstTypesInBlock(fn.Body, subs)
	_ = file
}

func substAstTypesInBlock(b *ast.BlockStmt, subs map[string]Type) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		substAstTypesInStmt(st, subs)
	}
}

func substAstTypesInStmt(st ast.Stmt, subs map[string]Type) {
	switch s := st.(type) {
	case *ast.BlockStmt:
		substAstTypesInBlock(s, subs)
	case *ast.LetStmt:
		if s.AnnType != nil {
			s.AnnType = substAstType(s.AnnType, subs)
		}
		substAstTypesInExpr(s.Init, subs)
	case *ast.AssignStmt:
		substAstTypesInExpr(s.Expr, subs)
	case *ast.FieldAssignStmt:
		substAstTypesInExpr(s.Expr, subs)
	case *ast.ReturnStmt:
		substAstTypesInExpr(s.Expr, subs)
	case *ast.IfStmt:
		substAstTypesInExpr(s.Cond, subs)
		substAstTypesInBlock(s.Then, subs)
		substAstTypesInStmt(s.Else, subs)
	case *ast.WhileStmt:
		substAstTypesInExpr(s.Cond, subs)
		substAstTypesInBlock(s.Body, subs)
	case *ast.ExprStmt:
		substAstTypesInExpr(s.Expr, subs)
	}
}

func substAstTypesInExpr(ex ast.Expr, subs map[string]Type) {
	switch e := ex.(type) {
	case nil:
		return
	case *ast.UnaryExpr:
		substAstTypesInExpr(e.Expr, subs)
	case *ast.AsExpr:
		substAstTypesInExpr(e.Expr, subs)
		e.Ty = substAstType(e.Ty, subs)
	case *ast.BinaryExpr:
		substAstTypesInExpr(e.Left, subs)
		substAstTypesInExpr(e.Right, subs)
	case *ast.MemberExpr:
		substAstTypesInExpr(e.Recv, subs)
	case *ast.TypeAppExpr:
		substAstTypesInExpr(e.Expr, subs)
		for i := range e.TypeArgs {
			e.TypeArgs[i] = substAstType(e.TypeArgs[i], subs)
		}
	case *ast.CallExpr:
		for i := range e.TypeArgs {
			e.TypeArgs[i] = substAstType(e.TypeArgs[i], subs)
		}
		substAstTypesInExpr(e.Callee, subs)
		for _, a := range e.Args {
			substAstTypesInExpr(a, subs)
		}
	case *ast.StructLitExpr:
		for i := range e.TypeArgs {
			e.TypeArgs[i] = substAstType(e.TypeArgs[i], subs)
		}
		for i := range e.Inits {
			substAstTypesInExpr(e.Inits[i].Expr, subs)
		}
	case *ast.MatchExpr:
		substAstTypesInExpr(e.Scrutinee, subs)
		for _, arm := range e.Arms {
			substAstTypesInExpr(arm.Expr, subs)
		}
	case *ast.IfExpr:
		substAstTypesInExpr(e.Cond, subs)
		substAstTypesInExpr(e.Then, subs)
		substAstTypesInExpr(e.Else, subs)
	case *ast.BlockExpr:
		for _, st := range e.Stmts {
			substAstTypesInStmt(st, subs)
		}
		substAstTypesInExpr(e.Tail, subs)
	}
}

func substAstType(t ast.Type, subs map[string]Type) ast.Type {
	switch tt := t.(type) {
	case nil:
		return nil
	case *ast.UnitType:
		return tt
	case *ast.NamedType:
		if len(tt.Parts) == 1 {
			if ty, ok := subs[tt.Parts[0]]; ok {
				return astTypeFromType(ty, tt.S)
			}
		}
		cp := &ast.NamedType{Parts: append([]string{}, tt.Parts...), S: tt.S}
		for _, a := range tt.Args {
			cp.Args = append(cp.Args, substAstType(a, subs))
		}
		return cp
	default:
		return tt
	}
}

func astTypeFromType(t Type, at source.Span) ast.Type {
	switch t.K {
	case TyUnit:
		return &ast.UnitType{S: at}
	case TyBool:
		return &ast.NamedType{Parts: []string{"bool"}, S: at}
	case TyI8:
		return &ast.NamedType{Parts: []string{"i8"}, S: at}
	case TyU8:
		return &ast.NamedType{Parts: []string{"u8"}, S: at}
	case TyI16:
		return &ast.NamedType{Parts: []string{"i16"}, S: at}
	case TyU16:
		return &ast.NamedType{Parts: []string{"u16"}, S: at}
	case TyI32:
		return &ast.NamedType{Parts: []string{"i32"}, S: at}
	case TyU32:
		return &ast.NamedType{Parts: []string{"u32"}, S: at}
	case TyI64:
		return &ast.NamedType{Parts: []string{"i64"}, S: at}
	case TyU64:
		return &ast.NamedType{Parts: []string{"u64"}, S: at}
	case TyISize:
		return &ast.NamedType{Parts: []string{"isize"}, S: at}
	case TyUSize:
		return &ast.NamedType{Parts: []string{"usize"}, S: at}
	case TyString:
		return &ast.NamedType{Parts: []string{"String"}, S: at}
	case TyParam:
		return &ast.NamedType{Parts: []string{t.Name}, S: at}
	case TyRange:
		if t.Base == nil {
			return &ast.RangeType{Lo: 0, Hi: 0, Base: &ast.NamedType{Parts: []string{"i64"}, S: at}, S: at}
		}
		return &ast.RangeType{Lo: t.Lo, Hi: t.Hi, Base: astTypeFromType(*t.Base, at), S: at}
	case TyStruct, TyEnum:
		// Qualified name format: "<pkg>::<mod.path>::<Name>" or "<Name>".
		// Convert mod.path to segments for type path syntax.
		parts := []string{}
		q := t.Name
		if strings.Contains(q, "::") {
			// "<pkg>::<mod.path>::<Name>"
			chunks := strings.Split(q, "::")
			if len(chunks) >= 2 {
				if chunks[0] != "" {
					parts = append(parts, chunks[0])
				}
				if len(chunks) == 2 {
					parts = append(parts, chunks[1])
				} else {
					modParts := strings.Split(chunks[1], ".")
					parts = append(parts, modParts...)
					parts = append(parts, chunks[2])
				}
			} else {
				parts = append(parts, q)
			}
		} else {
			parts = append(parts, q)
		}
		return &ast.NamedType{Parts: parts, S: at}
	case TyVec:
		if t.Elem == nil {
			return &ast.NamedType{Parts: []string{"Vec"}, S: at}
		}
		return &ast.NamedType{Parts: []string{"Vec"}, Args: []ast.Type{astTypeFromType(*t.Elem, at)}, S: at}
	default:
		return &ast.NamedType{Parts: []string{"<bad>"}, S: at}
	}
}
