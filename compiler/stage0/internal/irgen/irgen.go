package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/typecheck"
)

// Generate lowers a typechecked program into IR v0.
// Stage0 IR only supports i32/i64/bool/unit; String is rejected.
func Generate(p *typecheck.CheckedProgram) (*ir.Program, error) {
	g := &gen{
		p:        p,
		out:      &ir.Program{Funcs: map[string]*ir.Func{}},
		tmpID:    0,
		slotID:   0,
		funcSigs: p.FuncSigs,
	}
	for _, fn := range p.Prog.Funcs {
		f, err := g.genFunc(fn)
		if err != nil {
			return nil, err
		}
		g.out.Funcs[f.Name] = f
	}
	return g.out, nil
}

type gen struct {
	p        *typecheck.CheckedProgram
	out      *ir.Program
	tmpID    int
	slotID   int
	funcSigs map[string]typecheck.FuncSig

	curFn    *ast.FuncDecl
	curIRFn  *ir.Func
	blocks   []*ir.Block
	curBlock *ir.Block

	// scopes: name -> slot
	scopes []map[string]*ir.Slot
	// slot types
	slotTypes map[int]ir.Type
}

func (g *gen) pushScope() { g.scopes = append(g.scopes, map[string]*ir.Slot{}) }
func (g *gen) popScope()  { g.scopes = g.scopes[:len(g.scopes)-1] }

func (g *gen) lookup(name string) (*ir.Slot, bool) {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if s, ok := g.scopes[i][name]; ok {
			return s, true
		}
	}
	return nil, false
}

func (g *gen) declare(name string, slot *ir.Slot) {
	g.scopes[len(g.scopes)-1][name] = slot
}

func (g *gen) newTemp() *ir.Temp {
	t := &ir.Temp{ID: g.tmpID}
	g.tmpID++
	return t
}

func (g *gen) newSlot() *ir.Slot {
	s := &ir.Slot{ID: g.slotID}
	g.slotID++
	return s
}

func (g *gen) newBlock(name string) *ir.Block {
	b := &ir.Block{Name: name}
	g.blocks = append(g.blocks, b)
	return b
}

func (g *gen) setBlock(b *ir.Block) { g.curBlock = b }

func (g *gen) emit(i ir.Instr) { g.curBlock.Instr = append(g.curBlock.Instr, i) }

func (g *gen) term(t ir.Term) { g.curBlock.Term = t }

func (g *gen) genFunc(fn *ast.FuncDecl) (*ir.Func, error) {
	g.curFn = fn
	g.tmpID = 0
	g.slotID = 0
	g.slotTypes = map[int]ir.Type{}
	g.blocks = nil
	g.curBlock = nil

	ret, err := g.irTypeFromChecked(g.funcSigs[fn.Name].Ret)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn.Name, err)
	}
	f := &ir.Func{Name: fn.Name, Ret: ret}
	for i, p := range fn.Params {
		pty, err := g.irTypeFromChecked(g.funcSigs[fn.Name].Params[i])
		if err != nil {
			return nil, err
		}
		f.Params = append(f.Params, ir.Param{Name: p.Name, Ty: pty})
	}
	entry := g.newBlock("entry")
	g.setBlock(entry)
	g.pushScope()
	// params are lowered into slots for simplicity
	for i, p := range fn.Params {
		pty := f.Params[i].Ty
		slot := g.newSlot()
		g.slotTypes[slot.ID] = pty
		g.emit(&ir.SlotDecl{Slot: slot, Ty: pty})
		g.emit(&ir.Store{Slot: slot, Val: &ir.ParamRef{Index: i}})
		g.declare(p.Name, slot)
	}

	if err := g.genBlockStmt(fn.Body, ret); err != nil {
		return nil, err
	}
	// Ensure terminator.
	if g.curBlock.Term == nil {
		if ret.K == ir.TUnit {
			g.term(&ir.Ret{})
		} else {
			// missing return
			zero := g.zeroValue(ret)
			g.term(&ir.Ret{Val: zero})
		}
	}
	g.popScope()

	f.Blocks = g.blocks
	return f, nil
}

func (g *gen) irTypeFromChecked(t typecheck.Type) (ir.Type, error) {
	switch t.K {
	case typecheck.TyUnit:
		return ir.Type{K: ir.TUnit}, nil
	case typecheck.TyBool:
		return ir.Type{K: ir.TBool}, nil
	case typecheck.TyI32:
		return ir.Type{K: ir.TI32}, nil
	case typecheck.TyI64:
		return ir.Type{K: ir.TI64}, nil
	case typecheck.TyString:
		return ir.Type{}, fmt.Errorf("String not supported by stage0 IR backend")
	default:
		return ir.Type{}, fmt.Errorf("unsupported type")
	}
}

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
			z := g.zeroValue(irty)
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
	case *ast.ReturnStmt:
		if retTy.K == ir.TUnit {
			g.term(&ir.Ret{})
			return nil
		}
		if s.Expr == nil {
			g.term(&ir.Ret{Val: g.zeroValue(retTy)})
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
	case *ast.ExprStmt:
		_, err := g.genExpr(s.Expr)
		return err
	default:
		return fmt.Errorf("unsupported stmt in IR gen")
	}
}

func (g *gen) typeOfLet(s *ast.LetStmt) typecheck.Type {
	if s.Init != nil {
		return g.p.ExprTypes[s.Init]
	}
	if s.AnnType != nil {
		return typeFromAstLimited(s.AnnType)
	}
	return typecheck.Type{K: typecheck.TyBad}
}

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
		id, ok := e.Callee.(*ast.IdentExpr)
		if !ok {
			return nil, fmt.Errorf("callee must be ident (stage0)")
		}
		sig, ok := g.funcSigs[id.Name]
		if !ok {
			return nil, fmt.Errorf("unknown function: %s", id.Name)
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
		call := &ir.Call{Ret: ret, Name: id.Name, Args: args}
		if ret.K != ir.TUnit {
			call.Dst = g.newTemp()
			g.emit(call)
			return call.Dst, nil
		}
		g.emit(call)
		return g.zeroValue(ret), nil
	default:
		return nil, fmt.Errorf("unsupported expr in IR gen")
	}
}

func (g *gen) zeroValue(t ir.Type) ir.Value {
	switch t.K {
	case ir.TUnit:
		return nil
	case ir.TBool:
		return &ir.ConstBool{V: false}
	case ir.TI32, ir.TI64:
		return &ir.ConstInt{Ty: t, V: 0}
	default:
		return &ir.ConstInt{Ty: t, V: 0}
	}
}

func parseInt64(text string) int64 {
	var n int64
	for i := 0; i < len(text); i++ {
		n = n*10 + int64(text[i]-'0')
	}
	return n
}

func typeFromAstLimited(t ast.Type) typecheck.Type {
	switch tt := t.(type) {
	case *ast.UnitType:
		return typecheck.Type{K: typecheck.TyUnit}
	case *ast.NamedType:
		switch tt.Name {
		case "i32":
			return typecheck.Type{K: typecheck.TyI32}
		case "i64":
			return typecheck.Type{K: typecheck.TyI64}
		case "bool":
			return typecheck.Type{K: typecheck.TyBool}
		case "String":
			return typecheck.Type{K: typecheck.TyString}
		default:
			return typecheck.Type{K: typecheck.TyBad}
		}
	default:
		return typecheck.Type{K: typecheck.TyBad}
	}
}
