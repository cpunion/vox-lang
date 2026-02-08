package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

func (g *gen) genMatchExpr(m *ast.MatchExpr) (ir.Value, error) {
	// Evaluate scrutinee once.
	scrut, err := g.genExpr(m.Scrutinee)
	if err != nil {
		return nil, err
	}

	scrutTy := g.p.ExprTypes[m.Scrutinee]
	isEnum := scrutTy.K == typecheck.TyEnum
	isI32 := scrutTy.K == typecheck.TyI32
	isI64 := scrutTy.K == typecheck.TyI64
	isStr := scrutTy.K == typecheck.TyString
	if !isEnum && !isI32 && !isI64 && !isStr {
		return nil, fmt.Errorf("match scrutinee must be enum/i32/i64/String")
	}
	var es typecheck.EnumSig
	var tagTmp *ir.Temp
	if isEnum {
		es = g.p.EnumSigs[scrutTy.Name]
		// Extract tag.
		tagTmp = g.newTemp()
		g.emit(&ir.EnumTag{Dst: tagTmp, Recv: scrut})
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

	// Prepare arm blocks.
	type armInfo struct {
		arm       ast.MatchArm
		blk       *ir.Block
		variant   string
		tag       *int
		bindScrut string
		intPat    *int64
		strPat    *string
		binds     []string
		bindTys   []typecheck.Type
	}
	var arms []armInfo
	var wildBlk *ir.Block

		for _, a := range m.Arms {
			info := armInfo{arm: a, blk: g.newBlock(fmt.Sprintf("match_arm_%d", len(g.blocks)))}
			switch p := a.Pat.(type) {
		case *ast.WildPat:
			wildBlk = info.blk
		case *ast.BindPat:
			wildBlk = info.blk
			info.bindScrut = p.Name
			case *ast.IntPat:
				if !isI32 && !isI64 {
					return nil, fmt.Errorf("int pattern requires i32/i64 scrutinee")
				}
				v := parseInt64(p.Text)
				info.intPat = &v
		case *ast.StrPat:
			if !isStr {
				return nil, fmt.Errorf("string pattern requires String scrutinee")
			}
			s, err := unquoteUnescape(p.Text)
			if err != nil {
				return nil, err
			}
			info.strPat = &s
		case *ast.VariantPat:
			if !isEnum {
				return nil, fmt.Errorf("enum variant pattern requires enum scrutinee")
			}
			t := es.VariantIndex[p.Variant]
			info.variant = p.Variant
			info.tag = &t
			if len(p.Binds) != 0 {
				fields := es.Variants[t].Fields
				n := len(p.Binds)
				if n > len(fields) {
					n = len(fields)
				}
				info.binds = append([]string{}, p.Binds[:n]...)
				for i := 0; i < n; i++ {
					info.bindTys = append(info.bindTys, fields[i])
				}
			}
		default:
			return nil, fmt.Errorf("unsupported pattern in IR gen")
		}
		arms = append(arms, info)
	}

	// Decision chain starting from current block.
	for i := 0; i < len(arms); i++ {
		info := arms[i]
		if info.tag == nil && info.intPat == nil && info.strPat == nil {
			continue
		}
		nextDecide := g.newBlock(fmt.Sprintf("match_decide_%d", len(g.blocks)))
		cmpTmp := g.newTemp()
		if info.tag != nil {
			g.emit(&ir.Cmp{
				Dst: cmpTmp,
				Op:  ir.CmpEq,
				Ty:  ir.Type{K: ir.TI32},
				A:   tagTmp,
				B:   &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: int64(*info.tag)},
			})
			g.term(&ir.CondBr{Cond: cmpTmp, Then: info.blk.Name, Else: nextDecide.Name})
			} else if info.intPat != nil {
				cmpTy := ir.Type{K: ir.TI32}
				if isI64 {
					cmpTy = ir.Type{K: ir.TI64}
				}
				g.emit(&ir.Cmp{
					Dst: cmpTmp,
					Op:  ir.CmpEq,
					Ty:  cmpTy,
					A:   scrut,
					B:   &ir.ConstInt{Ty: cmpTy, V: *info.intPat},
				})
				g.term(&ir.CondBr{Cond: cmpTmp, Then: info.blk.Name, Else: nextDecide.Name})
			} else {
			g.emit(&ir.Cmp{
				Dst: cmpTmp,
				Op:  ir.CmpEq,
				Ty:  ir.Type{K: ir.TString},
				A:   scrut,
				B:   &ir.ConstStr{S: *info.strPat},
			})
			g.term(&ir.CondBr{Cond: cmpTmp, Then: info.blk.Name, Else: nextDecide.Name})
		}
		g.setBlock(nextDecide)
	}

	// Default branch.
	if wildBlk != nil {
		g.term(&ir.Br{Target: wildBlk.Name})
	} else {
		g.term(&ir.Br{Target: endBlk.Name})
	}

	// Arm blocks.
	scrutIRTy, err := g.irTypeFromChecked(scrutTy)
	if err != nil {
		return nil, err
	}
	for _, info := range arms {
		g.setBlock(info.blk)
		g.pushScope()

		if info.bindScrut != "" {
			slot := g.newSlot()
			g.slotTypes[slot.ID] = scrutIRTy
			g.emit(&ir.SlotDecl{Slot: slot, Ty: scrutIRTy})
			g.emit(&ir.Store{Slot: slot, Val: scrut})
			g.declare(info.bindScrut, slot)
		}

		for i := 0; i < len(info.binds) && i < len(info.bindTys); i++ {
			pty, err := g.irTypeFromChecked(info.bindTys[i])
			if err != nil {
				return nil, err
			}
			tmp := g.newTemp()
			if !isEnum {
				return nil, fmt.Errorf("enum payload only valid for enum scrutinee")
			}
			g.emit(&ir.EnumPayload{Dst: tmp, Ty: pty, Recv: scrut, Variant: info.variant, Index: i})
			slot := g.newSlot()
			g.slotTypes[slot.ID] = pty
			g.emit(&ir.SlotDecl{Slot: slot, Ty: pty})
			g.emit(&ir.Store{Slot: slot, Val: tmp})
			g.declare(info.binds[i], slot)
		}

		v, err := g.genExpr(info.arm.Expr)
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
