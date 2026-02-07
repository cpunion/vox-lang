package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

func (g *gen) genExpr(ex ast.Expr) (ir.Value, error) {
	switch e := ex.(type) {
	case *ast.IntLit:
		ty := g.p.ExprTypes[ex]
		irty, err := g.irTypeFromChecked(ty)
		if err != nil {
			return nil, err
		}
		tmp := g.newTemp()
		g.emit(&ir.Const{Dst: tmp, Ty: irty, Val: &ir.ConstInt{Ty: irty, V: parseInt64(e.Text)}})
		return tmp, nil
	case *ast.BoolLit:
		tmp := g.newTemp()
		g.emit(&ir.Const{Dst: tmp, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: e.Value}})
		return tmp, nil
	case *ast.StringLit:
		tmp := g.newTemp()
		s, err := unquoteUnescape(e.Text)
		if err != nil {
			return nil, err
		}
		g.emit(&ir.Const{Dst: tmp, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: s}})
		return tmp, nil
	case *ast.IdentExpr:
		slot, ok := g.lookup(e.Name)
		if !ok {
			return nil, fmt.Errorf("unknown variable: %s", e.Name)
		}
		ty := g.slotTypes[slot.ID]
		tmp := g.newTemp()
		g.emit(&ir.Load{Dst: tmp, Ty: ty, Slot: slot})
		return tmp, nil
	case *ast.UnaryExpr:
		if e.Op == "!" {
			a, err := g.genExpr(e.Expr)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			g.emit(&ir.Not{Dst: tmp, A: a})
			return tmp, nil
		}
		if e.Op == "-" {
			// 0 - x
			a, err := g.genExpr(e.Expr)
			if err != nil {
				return nil, err
			}
			ty := g.p.ExprTypes[e.Expr]
			irty, err := g.irTypeFromChecked(ty)
			if err != nil {
				return nil, err
			}
			z := g.newTemp()
			g.emit(&ir.Const{Dst: z, Ty: irty, Val: &ir.ConstInt{Ty: irty, V: 0}})
			tmp := g.newTemp()
			g.emit(&ir.BinOp{Dst: tmp, Op: ir.OpSub, Ty: irty, A: z, B: a})
			return tmp, nil
		}
		return nil, fmt.Errorf("unsupported unary op: %s", e.Op)
	case *ast.BinaryExpr:
		l, err := g.genExpr(e.Left)
		if err != nil {
			return nil, err
		}
		r, err := g.genExpr(e.Right)
		if err != nil {
			return nil, err
		}
		switch e.Op {
		case "+", "-", "*", "/", "%":
			ty := g.p.ExprTypes[e.Left]
			irty, err := g.irTypeFromChecked(ty)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			op := map[string]ir.BinOpKind{"+": ir.OpAdd, "-": ir.OpSub, "*": ir.OpMul, "/": ir.OpDiv, "%": ir.OpMod}[e.Op]
			g.emit(&ir.BinOp{Dst: tmp, Op: op, Ty: irty, A: l, B: r})
			return tmp, nil
		case "<", "<=", ">", ">=", "==", "!=":
			ty := g.p.ExprTypes[e.Left]
			irty, err := g.irTypeFromChecked(ty)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			var op ir.CmpKind
			switch e.Op {
			case "<":
				op = ir.CmpLt
			case "<=":
				op = ir.CmpLe
			case ">":
				op = ir.CmpGt
			case ">=":
				op = ir.CmpGe
			case "==":
				op = ir.CmpEq
			case "!=":
				op = ir.CmpNe
			}
			g.emit(&ir.Cmp{Dst: tmp, Op: op, Ty: irty, A: l, B: r})
			return tmp, nil
		case "&&":
			tmp := g.newTemp()
			g.emit(&ir.And{Dst: tmp, A: l, B: r})
			return tmp, nil
		case "||":
			tmp := g.newTemp()
			g.emit(&ir.Or{Dst: tmp, A: l, B: r})
			return tmp, nil
		default:
			return nil, fmt.Errorf("unsupported binary op: %s", e.Op)
		}
	case *ast.CallExpr:
		// Vec ops are special-cased in stage0 and lowered to dedicated IR.
		if vc, ok := g.p.VecCalls[e]; ok {
			switch vc.Kind {
			case typecheck.VecCallNew:
				ty := g.p.ExprTypes[ex]
				irty, err := g.irTypeFromChecked(ty)
				if err != nil {
					return nil, err
				}
				elem, err := g.irTypeFromChecked(vc.Elem)
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.VecNew{Dst: tmp, Ty: irty, Elem: elem})
				return tmp, nil
			case typecheck.VecCallPush:
				slot, ok := g.lookup(vc.RecvName)
				if !ok {
					return nil, fmt.Errorf("unknown vec receiver: %s", vc.RecvName)
				}
				val, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				elem, err := g.irTypeFromChecked(vc.Elem)
				if err != nil {
					return nil, err
				}
				g.emit(&ir.VecPush{Recv: slot, Elem: elem, Val: val})
				return nil, nil
			case typecheck.VecCallLen:
				slot, ok := g.lookup(vc.RecvName)
				if !ok {
					return nil, fmt.Errorf("unknown vec receiver: %s", vc.RecvName)
				}
				tmp := g.newTemp()
				g.emit(&ir.VecLen{Dst: tmp, Recv: slot})
				return tmp, nil
			case typecheck.VecCallGet:
				slot, ok := g.lookup(vc.RecvName)
				if !ok {
					return nil, fmt.Errorf("unknown vec receiver: %s", vc.RecvName)
				}
				idx, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				elem, err := g.irTypeFromChecked(vc.Elem)
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.VecGet{Dst: tmp, Ty: elem, Recv: slot, Idx: idx})
				return tmp, nil
			default:
				return nil, fmt.Errorf("unsupported vec call kind")
			}
		}

		// Enum constructors are lowered as enum_init.
		if ctor, ok := g.p.EnumCtors[e]; ok {
			ety, err := g.irTypeFromChecked(ctor.Enum)
			if err != nil {
				return nil, err
			}
			var payload ir.Value
			if ctor.Payload.K != typecheck.TyUnit {
				if len(e.Args) != 1 {
					return nil, fmt.Errorf("enum constructor expects 1 arg")
				}
				pv, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				payload = pv
			}
			tmp := g.newTemp()
			g.emit(&ir.EnumInit{Dst: tmp, Ty: ety, Variant: ctor.Variant, Payload: payload})
			return tmp, nil
		}

		target := g.p.CallTargets[e]
		if target == "" {
			return nil, fmt.Errorf("unresolved call target (stage0)")
		}
		sig, ok := g.funcSigs[target]
		if !ok {
			return nil, fmt.Errorf("unknown function: %s", target)
		}
		ret, err := g.irTypeFromChecked(sig.Ret)
		if err != nil {
			return nil, err
		}
		args := make([]ir.Value, 0, len(e.Args))
		for _, a := range e.Args {
			v, err := g.genExpr(a)
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
		call := &ir.Call{Ret: ret, Name: target, Args: args}
		if ret.K != ir.TUnit {
			call.Dst = g.newTemp()
			g.emit(call)
			return call.Dst, nil
		}
		g.emit(call)
		return g.zeroValue(ret)
	case *ast.MemberExpr:
		// Unit enum variant value: `Enum.Variant`.
		if ty := g.p.ExprTypes[ex]; ty.K == typecheck.TyEnum {
			es, ok := g.p.EnumSigs[ty.Name]
			if !ok {
				return nil, fmt.Errorf("unknown enum: %s", ty.Name)
			}
			_, ok = es.VariantIndex[e.Name]
			if !ok {
				return nil, fmt.Errorf("unknown variant: %s", e.Name)
			}
			ety, err := g.irTypeFromChecked(ty)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			g.emit(&ir.EnumInit{Dst: tmp, Ty: ety, Variant: e.Name})
			return tmp, nil
		}

		recv, err := g.genExpr(e.Recv)
		if err != nil {
			return nil, err
		}
		ty := g.p.ExprTypes[ex]
		irty, err := g.irTypeFromChecked(ty)
		if err != nil {
			return nil, err
		}
		tmp := g.newTemp()
		g.emit(&ir.FieldGet{Dst: tmp, Ty: irty, Recv: recv, Field: e.Name})
		return tmp, nil
	case *ast.StructLitExpr:
		ty := g.p.ExprTypes[ex]
		irty, err := g.irTypeFromChecked(ty)
		if err != nil {
			return nil, err
		}
		tmp := g.newTemp()
		fields := make([]ir.StructInitField, 0, len(e.Inits))
		for _, init := range e.Inits {
			v, err := g.genExpr(init.Expr)
			if err != nil {
				return nil, err
			}
			fields = append(fields, ir.StructInitField{Name: init.Name, Val: v})
		}
		g.emit(&ir.StructInit{Dst: tmp, Ty: irty, Fields: fields})
		return tmp, nil
	case *ast.MatchExpr:
		return g.genMatchExpr(e)
	default:
		return nil, fmt.Errorf("unsupported expr in IR gen")
	}
}
