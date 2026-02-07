package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

func (g *gen) genBlockStmt(b *ast.BlockStmt, retTy ir.Type) error {
	g.pushScope()
	for _, st := range b.Stmts {
		if err := g.genStmt(st, retTy); err != nil {
			return err
		}
		if g.curBlock.Term != nil {
			break
		}
	}
	g.popScope()
	return nil
}

func (g *gen) genStmt(st ast.Stmt, retTy ir.Type) error {
	switch s := st.(type) {
	case *ast.BlockStmt:
		return g.genBlockStmt(s, retTy)
	case *ast.LetStmt:
		ty := g.typeOfLet(s)
		irty, err := g.irTypeFromChecked(ty)
		if err != nil {
			return err
		}
		slot := g.newSlot()
		g.slotTypes[slot.ID] = irty
		g.emit(&ir.SlotDecl{Slot: slot, Ty: irty})
		if s.Init != nil {
			v, err := g.genExpr(s.Init)
			if err != nil {
				return err
			}
			g.emit(&ir.Store{Slot: slot, Val: v})
		} else {
			// Ensure deterministic value for stage0 codegen.
			z, err := g.zeroValue(irty)
			if err != nil {
				return err
			}
			if z != nil {
				g.emit(&ir.Store{Slot: slot, Val: z})
			}
		}
		g.declare(s.Name, slot)
		return nil
	case *ast.AssignStmt:
		slot, ok := g.lookup(s.Name)
		if !ok {
			return fmt.Errorf("unknown variable: %s", s.Name)
		}
		v, err := g.genExpr(s.Expr)
		if err != nil {
			return err
		}
		g.emit(&ir.Store{Slot: slot, Val: v})
		return nil
	case *ast.FieldAssignStmt:
		slot, ok := g.lookup(s.Recv)
		if !ok {
			return fmt.Errorf("unknown variable: %s", s.Recv)
		}
		v, err := g.genExpr(s.Expr)
		if err != nil {
			return err
		}
		g.emit(&ir.StoreField{Slot: slot, Field: s.Field, Val: v})
		return nil
	case *ast.ReturnStmt:
		if retTy.K == ir.TUnit {
			g.term(&ir.Ret{})
			return nil
		}
		if s.Expr == nil {
			z, err := g.zeroValue(retTy)
			if err != nil {
				return err
			}
			g.term(&ir.Ret{Val: z})
			return nil
		}
		v, err := g.genExpr(s.Expr)
		if err != nil {
			return err
		}
		g.term(&ir.Ret{Val: v})
		return nil
	case *ast.IfStmt:
		cond, err := g.genExpr(s.Cond)
		if err != nil {
			return err
		}
		thenBlk := g.newBlock(fmt.Sprintf("then_%d", len(g.blocks)))
		elseBlk := g.newBlock(fmt.Sprintf("else_%d", len(g.blocks)))
		endBlk := g.newBlock(fmt.Sprintf("end_%d", len(g.blocks)))
		g.term(&ir.CondBr{Cond: cond, Then: thenBlk.Name, Else: elseBlk.Name})

		// then
		g.setBlock(thenBlk)
		if err := g.genBlockStmt(s.Then, retTy); err != nil {
			return err
		}
		if g.curBlock.Term == nil {
			g.term(&ir.Br{Target: endBlk.Name})
		}

		// else
		g.setBlock(elseBlk)
		if s.Else != nil {
			if err := g.genStmt(s.Else, retTy); err != nil {
				return err
			}
		}
		if g.curBlock.Term == nil {
			g.term(&ir.Br{Target: endBlk.Name})
		}

		// end
		g.setBlock(endBlk)
		return nil
	case *ast.WhileStmt:
		condBlk := g.newBlock(fmt.Sprintf("while_cond_%d", len(g.blocks)))
		bodyBlk := g.newBlock(fmt.Sprintf("while_body_%d", len(g.blocks)))
		endBlk := g.newBlock(fmt.Sprintf("while_end_%d", len(g.blocks)))

		// Jump to condition.
		g.term(&ir.Br{Target: condBlk.Name})

		// cond
		g.setBlock(condBlk)
		cond, err := g.genExpr(s.Cond)
		if err != nil {
			return err
		}
		g.term(&ir.CondBr{Cond: cond, Then: bodyBlk.Name, Else: endBlk.Name})

		// body
		g.setBlock(bodyBlk)
		g.loopStack = append(g.loopStack, loopCtx{breakTarget: endBlk.Name, continueTarget: condBlk.Name})
		if err := g.genBlockStmt(s.Body, retTy); err != nil {
			return err
		}
		g.loopStack = g.loopStack[:len(g.loopStack)-1]
		if g.curBlock.Term == nil {
			g.term(&ir.Br{Target: condBlk.Name})
		}

		// end
		g.setBlock(endBlk)
		return nil
	case *ast.BreakStmt:
		if len(g.loopStack) == 0 {
			return fmt.Errorf("break outside of loop")
		}
		g.term(&ir.Br{Target: g.loopStack[len(g.loopStack)-1].breakTarget})
		return nil
	case *ast.ContinueStmt:
		if len(g.loopStack) == 0 {
			return fmt.Errorf("continue outside of loop")
		}
		g.term(&ir.Br{Target: g.loopStack[len(g.loopStack)-1].continueTarget})
		return nil
	case *ast.ExprStmt:
		_, err := g.genExpr(s.Expr)
		return err
	default:
		return fmt.Errorf("unsupported stmt in IR gen")
	}
}

func (g *gen) typeOfLet(s *ast.LetStmt) typecheck.Type {
	if g.p != nil {
		if ty, ok := g.p.LetTypes[s]; ok {
			return ty
		}
	}
	if s.Init != nil {
		return g.p.ExprTypes[s.Init]
	}
	if s.AnnType != nil {
		return g.typeFromAstLimited(s.AnnType)
	}
	return typecheck.Type{K: typecheck.TyBad}
}
