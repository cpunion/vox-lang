package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

func (g *gen) enumUnitTag(ex ast.Expr) (int, bool) {
	switch e := ex.(type) {
	case *ast.MemberExpr:
		t, ok := g.p.EnumUnitVariants[e]
		if !ok {
			return 0, false
		}
		return t.Tag, true
	case *ast.DotExpr:
		t, ok := g.p.EnumUnitVariants[e]
		if !ok {
			return 0, false
		}
		return t.Tag, true
	case *ast.CallExpr:
		t, ok := g.p.EnumCtors[e]
		if !ok || len(t.Fields) != 0 {
			return 0, false
		}
		return t.Tag, true
	default:
		return 0, false
	}
}

func (g *gen) isEnumUnitEq(e *ast.BinaryExpr) bool {
	lt := g.p.ExprTypes[e.Left]
	rt := g.p.ExprTypes[e.Right]
	if lt.K != typecheck.TyEnum || rt.K != typecheck.TyEnum {
		return false
	}
	if lt.Name != rt.Name {
		// should have been rejected by typechecker already.
		return false
	}
	_, lok := g.enumUnitTag(e.Left)
	_, rok := g.enumUnitTag(e.Right)
	return lok || rok
}

func (g *gen) genEnumUnitEq(e *ast.BinaryExpr) (ir.Value, error) {
	// Choose which side is the unit variant.
	var (
		unitTag int
		other   ast.Expr
		ok      bool
	)
	if unitTag, ok = g.enumUnitTag(e.Left); ok {
		other = e.Right
	} else if unitTag, ok = g.enumUnitTag(e.Right); ok {
		other = e.Left
	} else {
		return nil, fmt.Errorf("internal error: not an enum unit equality")
	}

	ov, err := g.genExpr(other)
	if err != nil {
		return nil, err
	}

	// Extract tag and compare.
	tagTmp := g.newTemp()
	g.emit(&ir.EnumTag{Dst: tagTmp, Recv: ov})

	cmpTmp := g.newTemp()
	op := ir.CmpEq
	if e.Op == "!=" {
		op = ir.CmpNe
	}
	g.emit(&ir.Cmp{
		Dst: cmpTmp,
		Op:  op,
		Ty:  ir.Type{K: ir.TI32},
		A:   tagTmp,
		B:   &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: int64(unitTag)},
	})
	return cmpTmp, nil
}
