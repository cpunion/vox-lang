package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
)

func (g *gen) genIfExpr(e *ast.IfExpr) (ir.Value, error) {
	cond, err := g.genExpr(e.Cond)
	if err != nil {
		return nil, err
	}

	resTyChecked := g.p.ExprTypes[e]
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

	thenBlk := g.newBlock(fmt.Sprintf("if_then_%d", len(g.blocks)))
	elseBlk := g.newBlock(fmt.Sprintf("if_else_%d", len(g.blocks)))
	endBlk := g.newBlock(fmt.Sprintf("if_end_%d", len(g.blocks)))

	g.term(&ir.CondBr{Cond: cond, Then: thenBlk.Name, Else: elseBlk.Name})

	// Then.
	g.setBlock(thenBlk)
	vThen, err := g.genExpr(e.Then)
	if err != nil {
		return nil, err
	}
	if resSlot != nil && vThen != nil {
		g.emit(&ir.Store{Slot: resSlot, Val: vThen})
	}
	g.term(&ir.Br{Target: endBlk.Name})

	// Else.
	g.setBlock(elseBlk)
	vElse, err := g.genExpr(e.Else)
	if err != nil {
		return nil, err
	}
	if resSlot != nil && vElse != nil {
		g.emit(&ir.Store{Slot: resSlot, Val: vElse})
	}
	g.term(&ir.Br{Target: endBlk.Name})

	// End.
	g.setBlock(endBlk)
	if resTy.K == ir.TUnit {
		return nil, nil
	}
	tmp := g.newTemp()
	g.emit(&ir.Load{Dst: tmp, Ty: resTy, Slot: resSlot})
	return tmp, nil
}
