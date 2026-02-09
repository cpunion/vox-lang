package irgen

import (
	"fmt"
	"strconv"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

func truncBits(u uint64, w int) uint64 {
	if w >= 64 || w <= 0 {
		return u
	}
	return u & ((uint64(1) << uint64(w)) - 1)
}

func isIntType(t typecheck.Type) bool {
	switch t.K {
	case typecheck.TyI8, typecheck.TyU8, typecheck.TyI32, typecheck.TyU32, typecheck.TyI64, typecheck.TyU64, typecheck.TyUSize:
		return true
	default:
		return false
	}
}

func isIntLikeType(t typecheck.Type) bool {
	if t.K == typecheck.TyUntypedInt {
		return true
	}
	if t.K == typecheck.TyRange && t.Base != nil {
		return isIntType(*t.Base)
	}
	return isIntType(t)
}

func isUnsignedIntType(t typecheck.Type) bool {
	switch t.K {
	case typecheck.TyU8, typecheck.TyU32, typecheck.TyU64, typecheck.TyUSize:
		return true
	default:
		return false
	}
}

func intBitWidth(t typecheck.Type) int {
	switch t.K {
	case typecheck.TyI8, typecheck.TyU8:
		return 8
	case typecheck.TyI32, typecheck.TyU32:
		return 32
	case typecheck.TyI64, typecheck.TyU64, typecheck.TyUSize, typecheck.TyUntypedInt:
		return 64
	default:
		return 0
	}
}

func intConstBitsForType(text string, ty typecheck.Type) (uint64, error) {
	base := ty
	if base.K == typecheck.TyRange && base.Base != nil {
		base = *base.Base
	}
	if !isIntLikeType(base) || base.K == typecheck.TyUntypedInt {
		return 0, fmt.Errorf("expected int type")
	}
	i64v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int literal")
	}
	// v0 limitation: literals/pattern literals live in int64 syntax.
	if isUnsignedIntType(base) && i64v < 0 {
		return 0, fmt.Errorf("negative int literal for unsigned type")
	}
	w := intBitWidth(base)
	return truncBits(uint64(i64v), w), nil
}

func enumSigForTy(p *typecheck.CheckedProgram, ty typecheck.Type) (typecheck.EnumSig, bool) {
	base := ty
	if base.K == typecheck.TyRange && base.Base != nil {
		base = *base.Base
	}
	if base.K != typecheck.TyEnum {
		return typecheck.EnumSig{}, false
	}
	es, ok := p.EnumSigs[base.Name]
	return es, ok
}

func (g *gen) genPatTest(pat ast.Pattern, v ir.Value, vTy typecheck.Type, succ, fail *ir.Block) error {
	switch p := pat.(type) {
	case *ast.WildPat, *ast.BindPat:
		g.term(&ir.Br{Target: succ.Name})
		return nil
	case *ast.IntPat:
		base := vTy
		if base.K == typecheck.TyRange && base.Base != nil {
			base = *base.Base
		}
		if base.K == typecheck.TyUntypedInt {
			base = typecheck.Type{K: typecheck.TyI64}
		}
		if !isIntLikeType(base) {
			return fmt.Errorf("int pattern requires int scrutinee")
		}
		cmpTy, err := g.irTypeFromChecked(base)
		if err != nil {
			return err
		}
		bits, err := intConstBitsForType(p.Text, base)
		if err != nil {
			return err
		}
		cmpTmp := g.newTemp()
		g.emit(&ir.Cmp{Dst: cmpTmp, Op: ir.CmpEq, Ty: cmpTy, A: v, B: &ir.ConstInt{Ty: cmpTy, Bits: bits}})
		g.term(&ir.CondBr{Cond: cmpTmp, Then: succ.Name, Else: fail.Name})
		return nil
	case *ast.StrPat:
		if vTy.K != typecheck.TyString {
			return fmt.Errorf("string pattern requires String scrutinee")
		}
		s, err := unquoteUnescape(p.Text)
		if err != nil {
			return err
		}
		cmpTmp := g.newTemp()
		g.emit(&ir.Cmp{Dst: cmpTmp, Op: ir.CmpEq, Ty: ir.Type{K: ir.TString}, A: v, B: &ir.ConstStr{S: s}})
		g.term(&ir.CondBr{Cond: cmpTmp, Then: succ.Name, Else: fail.Name})
		return nil
	case *ast.VariantPat:
		es, ok := enumSigForTy(g.p, vTy)
		if !ok {
			return fmt.Errorf("enum variant pattern requires enum scrutinee")
		}
		tag, ok := es.VariantIndex[p.Variant]
		if !ok {
			return fmt.Errorf("unknown variant: %s", p.Variant)
		}
		// Check tag first.
		tagTmp := g.newTemp()
		g.emit(&ir.EnumTag{Dst: tagTmp, Recv: v})
		cmpTmp := g.newTemp()
		g.emit(&ir.Cmp{
			Dst: cmpTmp,
			Op:  ir.CmpEq,
			Ty:  ir.Type{K: ir.TI32},
			A:   tagTmp,
			B:   &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: uint64(tag)},
		})

		// Payload checks (if any) run in a separate block.
		if len(p.Args) == 0 {
			g.term(&ir.CondBr{Cond: cmpTmp, Then: succ.Name, Else: fail.Name})
			return nil
		}

		payloadStart := g.newBlock(fmt.Sprintf("match_pat_payload_%d", len(g.blocks)))
		g.term(&ir.CondBr{Cond: cmpTmp, Then: payloadStart.Name, Else: fail.Name})
		g.setBlock(payloadStart)

		vdecl := es.Variants[tag]
		// Arity mismatch should have been rejected by typecheck; keep lowering defensive.
		n := len(p.Args)
		if n > len(vdecl.Fields) {
			n = len(vdecl.Fields)
		}
		cur := payloadStart
		for i := 0; i < n; i++ {
			// Extract payload field.
			fty, err := g.irTypeFromChecked(vdecl.Fields[i])
			if err != nil {
				return err
			}
			ftmp := g.newTemp()
			g.emit(&ir.EnumPayload{Dst: ftmp, Ty: fty, Recv: v, Variant: p.Variant, Index: i})

			nextOK := succ
			if i != n-1 {
				nextOK = g.newBlock(fmt.Sprintf("match_pat_ok_%d", len(g.blocks)))
			}
			// genPatTest terminates the current block.
			if err := g.genPatTest(p.Args[i], ftmp, vdecl.Fields[i], nextOK, fail); err != nil {
				return err
			}
			if i != n-1 {
				g.setBlock(nextOK)
				cur = nextOK
				_ = cur
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported pattern in IR gen")
	}
}

func (g *gen) genPatBinds(pat ast.Pattern, v ir.Value, vTy typecheck.Type) error {
	switch p := pat.(type) {
	case *ast.WildPat:
		return nil
	case *ast.IntPat, *ast.StrPat:
		return nil
	case *ast.BindPat:
		if p.Name == "" || p.Name == "_" {
			return nil
		}
		irty, err := g.irTypeFromChecked(vTy)
		if err != nil {
			return err
		}
		slot := g.newSlot()
		g.slotTypes[slot.ID] = irty
		g.emit(&ir.SlotDecl{Slot: slot, Ty: irty})
		g.emit(&ir.Store{Slot: slot, Val: v})
		g.declare(p.Name, slot)
		return nil
	case *ast.VariantPat:
		es, ok := enumSigForTy(g.p, vTy)
		if !ok {
			return fmt.Errorf("enum variant bind requires enum value")
		}
		tag, ok := es.VariantIndex[p.Variant]
		if !ok {
			return fmt.Errorf("unknown variant: %s", p.Variant)
		}
		vdecl := es.Variants[tag]
		n := len(p.Args)
		if n > len(vdecl.Fields) {
			n = len(vdecl.Fields)
		}
		for i := 0; i < n; i++ {
			fty := vdecl.Fields[i]
			irty, err := g.irTypeFromChecked(fty)
			if err != nil {
				return err
			}
			ftmp := g.newTemp()
			g.emit(&ir.EnumPayload{Dst: ftmp, Ty: irty, Recv: v, Variant: p.Variant, Index: i})
			if err := g.genPatBinds(p.Args[i], ftmp, fty); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported pattern bind in IR gen")
	}
}

func (g *gen) genMatchExpr(m *ast.MatchExpr) (ir.Value, error) {
	// Evaluate scrutinee once.
	scrut, err := g.genExpr(m.Scrutinee)
	if err != nil {
		return nil, err
	}

	scrutTy := g.p.ExprTypes[m.Scrutinee]
	scrutBase := scrutTy
	if scrutBase.K == typecheck.TyRange && scrutBase.Base != nil {
		scrutBase = *scrutBase.Base
	}
	isEnum := scrutBase.K == typecheck.TyEnum
	isInt := scrutBase.K == typecheck.TyI8 ||
		scrutBase.K == typecheck.TyU8 ||
		scrutBase.K == typecheck.TyI32 ||
		scrutBase.K == typecheck.TyU32 ||
		scrutBase.K == typecheck.TyI64 ||
		scrutBase.K == typecheck.TyU64 ||
		scrutBase.K == typecheck.TyUSize ||
		scrutBase.K == typecheck.TyUntypedInt
	isStr := scrutTy.K == typecheck.TyString
	if !isEnum && !isInt && !isStr {
		return nil, fmt.Errorf("match scrutinee must be enum/int/String")
	}

	resTyChecked := g.p.ExprTypes[m]
	resTy, err := g.irTypeFromChecked(resTyChecked)
	if err != nil {
		return nil, err
	}

	var resSlot *ir.Slot
	if resTy.K != ir.TUnit {
		resSlot = g.newSlot()
		g.slotTypes[resSlot.ID] = resTy
		g.emit(&ir.SlotDecl{Slot: resSlot, Ty: resTy})
		z, err := g.zeroValue(resTy)
		if err != nil {
			return nil, err
		}
		if z != nil {
			g.emit(&ir.Store{Slot: resSlot, Val: z})
		}
	}

	endBlk := g.newBlock(fmt.Sprintf("match_end_%d", len(g.blocks)))

	// Generate arm decision chain in source order.
	curDecide := g.curBlock
	armBlks := make([]*ir.Block, 0, len(m.Arms))

	for _, arm := range m.Arms {
		armBlk := g.newBlock(fmt.Sprintf("match_arm_%d", len(g.blocks)))
		armBlks = append(armBlks, armBlk)
		next := g.newBlock(fmt.Sprintf("match_decide_%d", len(g.blocks)))

		g.setBlock(curDecide)
		if err := g.genPatTest(arm.Pat, scrut, scrutTy, armBlk, next); err != nil {
			return nil, err
		}
		curDecide = next
	}

	// Fallthrough default: should be unreachable for well-typed exhaustive matches.
	g.setBlock(curDecide)
	g.term(&ir.Br{Target: endBlk.Name})

	// Arm blocks.
	for i, arm := range m.Arms {
		g.setBlock(armBlks[i])
		g.pushScope()

		// Bindings for this arm (after successful pattern tests).
		if err := g.genPatBinds(arm.Pat, scrut, scrutTy); err != nil {
			return nil, err
		}

		v, err := g.genExpr(arm.Expr)
		if err != nil {
			return nil, err
		}
		if resSlot != nil && v != nil {
			g.emit(&ir.Store{Slot: resSlot, Val: v})
		}

		g.popScope()
		g.term(&ir.Br{Target: endBlk.Name})
	}

	// End.
	g.setBlock(endBlk)
	if resTy.K == ir.TUnit {
		return nil, nil
	}
	tmp := g.newTemp()
	g.emit(&ir.Load{Dst: tmp, Ty: resTy, Slot: resSlot})
	return tmp, nil
}
