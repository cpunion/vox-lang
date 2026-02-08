package typecheck

import "voxlang/internal/ast"

func cloneType(t ast.Type) ast.Type {
	switch tt := t.(type) {
	case nil:
		return nil
	case *ast.UnitType:
		cp := *tt
		return &cp
	case *ast.NamedType:
		cp := &ast.NamedType{Parts: append([]string{}, tt.Parts...), S: tt.S}
		for _, a := range tt.Args {
			cp.Args = append(cp.Args, cloneType(a))
		}
		return cp
	default:
		// stage0: keep it conservative; unknown types will be rejected by parser/typechecker anyway.
		return tt
	}
}

func cloneExpr(e ast.Expr) ast.Expr {
	switch x := e.(type) {
	case nil:
		return nil
	case *ast.IdentExpr:
		cp := *x
		return &cp
	case *ast.DotExpr:
		cp := *x
		return &cp
	case *ast.MemberExpr:
		return &ast.MemberExpr{Recv: cloneExpr(x.Recv), Name: x.Name, S: x.S}
	case *ast.IntLit:
		cp := *x
		return &cp
	case *ast.StringLit:
		cp := *x
		return &cp
	case *ast.BoolLit:
		cp := *x
		return &cp
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: x.Op, Expr: cloneExpr(x.Expr), S: x.S}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{Op: x.Op, Left: cloneExpr(x.Left), Right: cloneExpr(x.Right), S: x.S}
	case *ast.CallExpr:
		cp := &ast.CallExpr{Callee: cloneExpr(x.Callee), S: x.S}
		for _, ta := range x.TypeArgs {
			cp.TypeArgs = append(cp.TypeArgs, cloneType(ta))
		}
		for _, a := range x.Args {
			cp.Args = append(cp.Args, cloneExpr(a))
		}
		return cp
	case *ast.StructLitExpr:
		cp := &ast.StructLitExpr{TypeParts: append([]string{}, x.TypeParts...), S: x.S}
		for _, it := range x.Inits {
			cp.Inits = append(cp.Inits, ast.FieldInit{Name: it.Name, Expr: cloneExpr(it.Expr), Span: it.Span})
		}
		return cp
	case *ast.MatchExpr:
		cp := &ast.MatchExpr{Scrutinee: cloneExpr(x.Scrutinee), S: x.S}
		for _, arm := range x.Arms {
			cp.Arms = append(cp.Arms, ast.MatchArm{Pat: clonePat(arm.Pat), Expr: cloneExpr(arm.Expr), S: arm.S})
		}
		return cp
	case *ast.IfExpr:
		return &ast.IfExpr{Cond: cloneExpr(x.Cond), Then: cloneExpr(x.Then), Else: cloneExpr(x.Else), S: x.S}
	case *ast.BlockExpr:
		cp := &ast.BlockExpr{S: x.S}
		for _, st := range x.Stmts {
			cp.Stmts = append(cp.Stmts, cloneStmt(st))
		}
		cp.Tail = cloneExpr(x.Tail)
		return cp
	default:
		return x
	}
}

func clonePat(p ast.Pattern) ast.Pattern {
	switch x := p.(type) {
	case nil:
		return nil
	case *ast.WildPat:
		cp := *x
		return &cp
	case *ast.BindPat:
		cp := *x
		return &cp
	case *ast.IntPat:
		cp := *x
		return &cp
	case *ast.StrPat:
		cp := *x
		return &cp
	case *ast.VariantPat:
		cp := &ast.VariantPat{
			TypeParts: append([]string{}, x.TypeParts...),
			Variant:   x.Variant,
			Binds:     append([]string{}, x.Binds...),
			S:         x.S,
		}
		return cp
	default:
		return x
	}
}

func cloneBlock(b *ast.BlockStmt) *ast.BlockStmt {
	if b == nil {
		return nil
	}
	cp := &ast.BlockStmt{S: b.S}
	for _, st := range b.Stmts {
		cp.Stmts = append(cp.Stmts, cloneStmt(st))
	}
	return cp
}

func cloneStmt(s ast.Stmt) ast.Stmt {
	switch x := s.(type) {
	case nil:
		return nil
	case *ast.BlockStmt:
		return cloneBlock(x)
	case *ast.LetStmt:
		return &ast.LetStmt{
			Mutable: x.Mutable,
			Name:    x.Name,
			AnnType: cloneType(x.AnnType),
			Init:    cloneExpr(x.Init),
			S:       x.S,
		}
	case *ast.AssignStmt:
		return &ast.AssignStmt{Name: x.Name, Expr: cloneExpr(x.Expr), S: x.S}
	case *ast.FieldAssignStmt:
		return &ast.FieldAssignStmt{Recv: x.Recv, Field: x.Field, Expr: cloneExpr(x.Expr), S: x.S}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{Expr: cloneExpr(x.Expr), S: x.S}
	case *ast.IfStmt:
		return &ast.IfStmt{Cond: cloneExpr(x.Cond), Then: cloneBlock(x.Then), Else: cloneStmt(x.Else), S: x.S}
	case *ast.WhileStmt:
		return &ast.WhileStmt{Cond: cloneExpr(x.Cond), Body: cloneBlock(x.Body), S: x.S}
	case *ast.BreakStmt:
		cp := *x
		return &cp
	case *ast.ContinueStmt:
		cp := *x
		return &cp
	case *ast.ExprStmt:
		return &ast.ExprStmt{Expr: cloneExpr(x.Expr), S: x.S}
	default:
		return x
	}
}

func cloneFuncDecl(fn *ast.FuncDecl) *ast.FuncDecl {
	if fn == nil {
		return nil
	}
	cp := &ast.FuncDecl{
		Pub:        fn.Pub,
		Name:       fn.Name,
		TypeParams: append([]string{}, fn.TypeParams...),
		Span:       fn.Span,
		Ret:        cloneType(fn.Ret),
		Body:       cloneBlock(fn.Body),
	}
	for _, p := range fn.Params {
		cp.Params = append(cp.Params, ast.Param{Name: p.Name, Type: cloneType(p.Type), Span: p.Span})
	}
	return cp
}
