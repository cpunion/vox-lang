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
		// Special-case: enum equality against unit variants lowers to tag comparison.
		if (e.Op == "==" || e.Op == "!=") && g.isEnumUnitEq(e) {
			return g.genEnumUnitEq(e)
		}

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
			recvSlotFrom := func() (*ir.Slot, error) {
				if vc.RecvName != "" {
					slot, ok := g.lookup(vc.RecvName)
					if !ok {
						return nil, fmt.Errorf("unknown vec receiver: %s", vc.RecvName)
					}
					return slot, nil
				}
				if vc.Recv == nil {
					return nil, fmt.Errorf("missing vec receiver")
				}
				rv, err := g.genExpr(vc.Recv)
				if err != nil {
					return nil, err
				}
				ty := g.p.ExprTypes[vc.Recv]
				irty, err := g.irTypeFromChecked(ty)
				if err != nil {
					return nil, err
				}
				slot := g.newSlot()
				g.slotTypes[slot.ID] = irty
				g.emit(&ir.SlotDecl{Slot: slot, Ty: irty})
				g.emit(&ir.Store{Slot: slot, Val: rv})
				return slot, nil
			}
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
				// push mutates the receiver; stage0 either lowers directly to a local slot
				// or to a "field place" writeback sequence (ident.field).
				if vc.RecvName == "" {
					if vc.Recv == nil {
						return nil, fmt.Errorf("missing vec receiver")
					}
					mem, ok := vc.Recv.(*ast.MemberExpr)
					if !ok {
						return nil, fmt.Errorf("Vec.push receiver must be a place (got %T)", vc.Recv)
					}
					base, ok := mem.Recv.(*ast.IdentExpr)
					if !ok {
						return nil, fmt.Errorf("Vec.push receiver must be a direct field of a local variable")
					}
					baseSlot, ok := g.lookup(base.Name)
					if !ok {
						return nil, fmt.Errorf("unknown vec receiver root: %s", base.Name)
					}

					val, err := g.genExpr(e.Args[0])
					if err != nil {
						return nil, err
					}
					elem, err := g.irTypeFromChecked(vc.Elem)
					if err != nil {
						return nil, err
					}

					// Load struct, extract vec field into a temp slot, push, then write back to the field.
					baseTy := g.p.ExprTypes[mem.Recv]
					irBaseTy, err := g.irTypeFromChecked(baseTy)
					if err != nil {
						return nil, err
					}
					vecTyChecked := g.p.ExprTypes[vc.Recv]
					irVecTy, err := g.irTypeFromChecked(vecTyChecked)
					if err != nil {
						return nil, err
					}
					tmpStruct := g.newTemp()
					g.emit(&ir.Load{Dst: tmpStruct, Ty: irBaseTy, Slot: baseSlot})
					tmpVec := g.newTemp()
					g.emit(&ir.FieldGet{Dst: tmpVec, Ty: irVecTy, Recv: tmpStruct, Field: mem.Name})
					vecSlot := g.newSlot()
					g.slotTypes[vecSlot.ID] = irVecTy
					g.emit(&ir.SlotDecl{Slot: vecSlot, Ty: irVecTy})
					g.emit(&ir.Store{Slot: vecSlot, Val: tmpVec})

					g.emit(&ir.VecPush{Recv: vecSlot, Elem: elem, Val: val})

					tmpVec2 := g.newTemp()
					g.emit(&ir.Load{Dst: tmpVec2, Ty: irVecTy, Slot: vecSlot})
					g.emit(&ir.StoreField{Slot: baseSlot, Field: mem.Name, Val: tmpVec2})
					return nil, nil
				}

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
				slot, err := recvSlotFrom()
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.VecLen{Dst: tmp, Recv: slot})
				return tmp, nil
			case typecheck.VecCallGet:
				slot, err := recvSlotFrom()
				if err != nil {
					return nil, err
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
			case typecheck.VecCallJoin:
				slot, err := recvSlotFrom()
				if err != nil {
					return nil, err
				}
				sep, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.VecStrJoin{Dst: tmp, Recv: slot, Sep: sep})
				return tmp, nil
			default:
				return nil, fmt.Errorf("unsupported vec call kind")
			}
		}

		// String ops are special-cased in stage0 and lowered to dedicated IR.
		if sc, ok := g.p.StrCalls[e]; ok {
			var recv ir.Value
			var err error
			if sc.Recv != nil {
				recv, err = g.genExpr(sc.Recv)
			} else if sc.RecvName != "" {
				// Back-compat: old checked programs used RecvName.
				slot, ok := g.lookup(sc.RecvName)
				if !ok {
					return nil, fmt.Errorf("unknown string receiver: %s", sc.RecvName)
				}
				tmp := g.newTemp()
				g.emit(&ir.Load{Dst: tmp, Ty: ir.Type{K: ir.TString}, Slot: slot})
				recv = tmp
			} else {
				return nil, fmt.Errorf("missing string receiver")
			}
			if err != nil {
				return nil, err
			}
			switch sc.Kind {
			case typecheck.StrCallLen:
				tmp := g.newTemp()
				g.emit(&ir.StrLen{Dst: tmp, Recv: recv})
				return tmp, nil
			case typecheck.StrCallByteAt:
				idx, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.StrByteAt{Dst: tmp, Recv: recv, Idx: idx})
				return tmp, nil
			case typecheck.StrCallSlice:
				if len(e.Args) != 2 {
					return nil, fmt.Errorf("String.slice expects 2 args")
				}
				sv, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				ev, err := g.genExpr(e.Args[1])
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.StrSlice{Dst: tmp, Recv: recv, Start: sv, End: ev})
				return tmp, nil
			case typecheck.StrCallConcat:
				if len(e.Args) != 1 {
					return nil, fmt.Errorf("String.concat expects 1 arg")
				}
				ov, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				tmp := g.newTemp()
				g.emit(&ir.StrConcat{Dst: tmp, A: recv, B: ov})
				return tmp, nil
			case typecheck.StrCallEscapeC:
				tmp := g.newTemp()
				g.emit(&ir.StrEscapeC{Dst: tmp, Recv: recv})
				return tmp, nil
			default:
				return nil, fmt.Errorf("unsupported string call kind")
			}
		}

		// Primitive to_string: lowered to dedicated IR.
		if ts, ok := g.p.ToStrCalls[e]; ok {
			if ts.Recv == nil && ts.RecvName != "" {
				slot, ok := g.lookup(ts.RecvName)
				if !ok {
					return nil, fmt.Errorf("unknown to_string receiver: %s", ts.RecvName)
				}
				// Load receiver from slot; type comes from checked program.
				recvTy := g.p.ExprTypes[ts.Recv]
				irty, err := g.irTypeFromChecked(recvTy)
				if err != nil {
					return nil, err
				}
				tmpv := g.newTemp()
				g.emit(&ir.Load{Dst: tmpv, Ty: irty, Slot: slot})
				tmp := g.newTemp()
				switch ts.Kind {
				case typecheck.ToStrI32:
					g.emit(&ir.I32ToStr{Dst: tmp, V: tmpv})
				case typecheck.ToStrI64:
					g.emit(&ir.I64ToStr{Dst: tmp, V: tmpv})
				case typecheck.ToStrBool:
					g.emit(&ir.BoolToStr{Dst: tmp, V: tmpv})
				default:
					return nil, fmt.Errorf("unsupported to_string kind")
				}
				return tmp, nil
			}
			if ts.Recv == nil {
				return nil, fmt.Errorf("missing to_string receiver")
			}
			rv, err := g.genExpr(ts.Recv)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			switch ts.Kind {
			case typecheck.ToStrI32:
				g.emit(&ir.I32ToStr{Dst: tmp, V: rv})
			case typecheck.ToStrI64:
				g.emit(&ir.I64ToStr{Dst: tmp, V: rv})
			case typecheck.ToStrBool:
				g.emit(&ir.BoolToStr{Dst: tmp, V: rv})
			default:
				return nil, fmt.Errorf("unsupported to_string kind")
			}
			return tmp, nil
		}

		// Enum constructors are lowered as enum_init.
		if ctor, ok := g.p.EnumCtors[e]; ok {
			ety, err := g.irTypeFromChecked(ctor.Enum)
			if err != nil {
				return nil, err
			}
			if len(e.Args) != len(ctor.Fields) {
				return nil, fmt.Errorf("enum constructor arity mismatch: expected %d args, got %d", len(ctor.Fields), len(e.Args))
			}
			payload := make([]ir.Value, 0, len(e.Args))
			for _, a := range e.Args {
				pv, err := g.genExpr(a)
				if err != nil {
					return nil, err
				}
				payload = append(payload, pv)
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
		if cu, ok := g.p.EnumUnitVariants[e]; ok {
			ety, err := g.irTypeFromChecked(cu.Enum)
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			g.emit(&ir.EnumInit{Dst: tmp, Ty: ety, Variant: cu.Variant})
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
	case *ast.IfExpr:
		return g.genIfExpr(e)
	case *ast.MatchExpr:
		return g.genMatchExpr(e)
	default:
		return nil, fmt.Errorf("unsupported expr in IR gen")
	}
}
