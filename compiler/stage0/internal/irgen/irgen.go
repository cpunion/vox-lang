package irgen

import (
	"fmt"
	"sort"
	"strconv"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

// Generate lowers a typechecked program into IR v0.
func Generate(p *typecheck.CheckedProgram) (*ir.Program, error) {
	g := &gen{
		p:        p,
		out:      &ir.Program{Structs: map[string]*ir.Struct{}, Funcs: map[string]*ir.Func{}},
		tmpID:    0,
		slotID:   0,
		funcSigs: p.FuncSigs,
	}
	if err := g.genStructDefs(); err != nil {
		return nil, err
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

	loopStack []loopCtx
}

type loopCtx struct {
	breakTarget    string
	continueTarget string
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

func (g *gen) genStructDefs() error {
	if g.p == nil {
		return nil
	}

	// Stable ordering for reproducible output.
	var names []string
	for name := range g.p.StructSigs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ss := g.p.StructSigs[name]
		st := &ir.Struct{Name: ss.Name}
		for _, f := range ss.Fields {
			ty, err := g.irTypeFromChecked(f.Ty)
			if err != nil {
				return err
			}
			st.Fields = append(st.Fields, ir.StructField{Name: f.Name, Ty: ty})
		}
		g.out.Structs[st.Name] = st
	}

	// Enums are lowered to a synthetic struct { tag: i32, payload?: T } in IR v0.
	var enames []string
	for name := range g.p.EnumSigs {
		enames = append(enames, name)
	}
	sort.Strings(enames)
	for _, name := range enames {
		es := g.p.EnumSigs[name]
		// Determine payload type (must be consistent; typechecker enforces).
		payload := typecheck.Type{K: typecheck.TyUnit}
		for _, v := range es.Variants {
			if len(v.Fields) == 1 {
				payload = v.Fields[0]
				break
			}
		}
		st := &ir.Struct{Name: es.Name}
		st.Fields = append(st.Fields, ir.StructField{Name: "tag", Ty: ir.Type{K: ir.TI32}})
		if payload.K != typecheck.TyUnit {
			pty, err := g.irTypeFromChecked(payload)
			if err != nil {
				return err
			}
			st.Fields = append(st.Fields, ir.StructField{Name: "payload", Ty: pty})
		}
		g.out.Structs[st.Name] = st
	}
	return nil
}

func (g *gen) genFunc(fn *ast.FuncDecl) (*ir.Func, error) {
	g.curFn = fn
	g.tmpID = 0
	g.slotID = 0
	g.slotTypes = map[int]ir.Type{}
	g.blocks = nil
	g.curBlock = nil
	g.loopStack = nil

	qname := names.QualifyFunc(fn.Span.File.Name, fn.Name)
	ret, err := g.irTypeFromChecked(g.funcSigs[qname].Ret)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn.Name, err)
	}
	f := &ir.Func{Name: qname, Ret: ret}
	for i, p := range fn.Params {
		pty, err := g.irTypeFromChecked(g.funcSigs[qname].Params[i])
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
			zero, err := g.zeroValue(ret)
			if err != nil {
				return nil, err
			}
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
		return ir.Type{K: ir.TString}, nil
	case typecheck.TyStruct:
		return ir.Type{K: ir.TStruct, Name: t.Name}, nil
	case typecheck.TyEnum:
		// Enums are represented as synthetic structs in IR v0.
		return ir.Type{K: ir.TStruct, Name: t.Name}, nil
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
	if s.Init != nil {
		return g.p.ExprTypes[s.Init]
	}
	if s.AnnType != nil {
		return g.typeFromAstLimited(s.AnnType)
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
		// Enum constructors are lowered as struct_init { tag, payload? }.
		if ctor, ok := g.p.EnumCtors[e]; ok {
			ety, err := g.irTypeFromChecked(ctor.Enum)
			if err != nil {
				return nil, err
			}
			tagTmp := g.newTemp()
			g.emit(&ir.Const{Dst: tagTmp, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: int64(ctor.Tag)}})
			fields := []ir.StructInitField{{Name: "tag", Val: tagTmp}}
			if ctor.Payload.K != typecheck.TyUnit {
				if len(e.Args) != 1 {
					return nil, fmt.Errorf("enum constructor expects 1 arg")
				}
				pv, err := g.genExpr(e.Args[0])
				if err != nil {
					return nil, err
				}
				fields = append(fields, ir.StructInitField{Name: "payload", Val: pv})
			} else {
				// Enum may still have a payload field (if other variants carry one); set it to zero.
				if st, ok := g.out.Structs[ety.Name]; ok {
					for _, f := range st.Fields {
						if f.Name == "payload" {
							z, err := g.zeroValue(f.Ty)
							if err != nil {
								return nil, err
							}
							fields = append(fields, ir.StructInitField{Name: "payload", Val: z})
						}
					}
				}
			}
			tmp := g.newTemp()
			g.emit(&ir.StructInit{Dst: tmp, Ty: ety, Fields: fields})
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
			tag, ok := es.VariantIndex[e.Name]
			if !ok {
				return nil, fmt.Errorf("unknown variant: %s", e.Name)
			}
			ety, err := g.irTypeFromChecked(ty)
			if err != nil {
				return nil, err
			}
			tagTmp := g.newTemp()
			g.emit(&ir.Const{Dst: tagTmp, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: int64(tag)}})
			fields := []ir.StructInitField{{Name: "tag", Val: tagTmp}}
			if st, ok := g.out.Structs[ety.Name]; ok {
				for _, f := range st.Fields {
					if f.Name == "payload" {
						z, err := g.zeroValue(f.Ty)
						if err != nil {
							return nil, err
						}
						fields = append(fields, ir.StructInitField{Name: "payload", Val: z})
					}
				}
			}
			tmp := g.newTemp()
			g.emit(&ir.StructInit{Dst: tmp, Ty: ety, Fields: fields})
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

func (g *gen) genMatchExpr(m *ast.MatchExpr) (ir.Value, error) {
	// Evaluate scrutinee once.
	scrut, err := g.genExpr(m.Scrutinee)
	if err != nil {
		return nil, err
	}
	// Extract tag.
	tagTmp := g.newTemp()
	g.emit(&ir.FieldGet{Dst: tagTmp, Ty: ir.Type{K: ir.TI32}, Recv: scrut, Field: "tag"})

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
		arm  ast.MatchArm
		blk  *ir.Block
		tag  *int
		bind string
	}
	var arms []armInfo
	var wildBlk *ir.Block

	scrutTy := g.p.ExprTypes[m.Scrutinee]
	if scrutTy.K != typecheck.TyEnum {
		return nil, fmt.Errorf("match scrutinee must be enum")
	}
	es := g.p.EnumSigs[scrutTy.Name]

	for _, a := range m.Arms {
		info := armInfo{arm: a, blk: g.newBlock(fmt.Sprintf("match_arm_%d", len(g.blocks)))}
		switch p := a.Pat.(type) {
		case *ast.WildPat:
			wildBlk = info.blk
		case *ast.VariantPat:
			t := es.VariantIndex[p.Variant]
			info.tag = &t
			if len(p.Binds) == 1 {
				info.bind = p.Binds[0]
			}
		default:
			return nil, fmt.Errorf("unsupported pattern in IR gen")
		}
		arms = append(arms, info)
	}

	// Decision chain starting from current block.
	for i := 0; i < len(arms); i++ {
		info := arms[i]
		if info.tag == nil {
			continue
		}
		nextDecide := g.newBlock(fmt.Sprintf("match_decide_%d", len(g.blocks)))
		// Compare tag.
		cmpTmp := g.newTemp()
		g.emit(&ir.Cmp{Dst: cmpTmp, Op: ir.CmpEq, Ty: ir.Type{K: ir.TI32}, A: tagTmp, B: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: int64(*info.tag)}})
		g.term(&ir.CondBr{Cond: cmpTmp, Then: info.blk.Name, Else: nextDecide.Name})
		// Start emitting in next decision block.
		g.setBlock(nextDecide)
	}
	// Default branch.
	if wildBlk != nil {
		g.term(&ir.Br{Target: wildBlk.Name})
	} else {
		// Typechecker should guarantee exhaustiveness; keep a safe default.
		g.term(&ir.Br{Target: endBlk.Name})
	}

	// Arm blocks.
	for _, info := range arms {
		g.setBlock(info.blk)
		g.pushScope()
		// If binder exists, it must be payload type of enum.
		if info.bind != "" {
			st, ok := g.out.Structs[scrutTy.Name]
			if !ok {
				return nil, fmt.Errorf("missing enum layout for %s", scrutTy.Name)
			}
			var payloadTy ir.Type
			for _, f := range st.Fields {
				if f.Name == "payload" {
					payloadTy = f.Ty
				}
			}
			if payloadTy.K == ir.TBad {
				return nil, fmt.Errorf("enum %s has no payload field", scrutTy.Name)
			}
			tmp := g.newTemp()
			g.emit(&ir.FieldGet{Dst: tmp, Ty: payloadTy, Recv: scrut, Field: "payload"})
			slot := g.newSlot()
			g.slotTypes[slot.ID] = payloadTy
			g.emit(&ir.SlotDecl{Slot: slot, Ty: payloadTy})
			g.emit(&ir.Store{Slot: slot, Val: tmp})
			g.declare(info.bind, slot)
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

func (g *gen) zeroValue(t ir.Type) (ir.Value, error) {
	switch t.K {
	case ir.TUnit:
		return nil, nil
	case ir.TBool:
		return &ir.ConstBool{V: false}, nil
	case ir.TI32, ir.TI64:
		return &ir.ConstInt{Ty: t, V: 0}, nil
	case ir.TString:
		return &ir.ConstStr{S: ""}, nil
	case ir.TStruct:
		if ss, ok := g.p.StructSigs[t.Name]; ok {
			tmp := g.newTemp()
			fields := make([]ir.StructInitField, 0, len(ss.Fields))
			for _, f := range ss.Fields {
				fty, err := g.irTypeFromChecked(f.Ty)
				if err != nil {
					return nil, err
				}
				fv, err := g.zeroValue(fty)
				if err != nil {
					return nil, err
				}
				fields = append(fields, ir.StructInitField{Name: f.Name, Val: fv})
			}
			g.emit(&ir.StructInit{Dst: tmp, Ty: t, Fields: fields})
			return tmp, nil
		}
		// Enums are represented as synthetic structs; synthesize a zero value as tag=0 (+ payload=0 if present).
		if _, ok := g.p.EnumSigs[t.Name]; ok {
			tmp := g.newTemp()
			tagTmp := g.newTemp()
			g.emit(&ir.Const{Dst: tagTmp, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, V: 0}})
			fields := []ir.StructInitField{{Name: "tag", Val: tagTmp}}
			if st, ok := g.out.Structs[t.Name]; ok {
				for _, f := range st.Fields {
					if f.Name != "payload" {
						continue
					}
					z, err := g.zeroValue(f.Ty)
					if err != nil {
						return nil, err
					}
					fields = append(fields, ir.StructInitField{Name: "payload", Val: z})
				}
			}
			g.emit(&ir.StructInit{Dst: tmp, Ty: t, Fields: fields})
			return tmp, nil
		}
		return nil, fmt.Errorf("unknown struct: %s", t.Name)
	default:
		return &ir.ConstInt{Ty: t, V: 0}, nil
	}
}

func parseInt64(text string) int64 {
	var n int64
	for i := 0; i < len(text); i++ {
		n = n*10 + int64(text[i]-'0')
	}
	return n
}

func (g *gen) typeFromAstLimited(t ast.Type) typecheck.Type {
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
			if g != nil && g.p != nil && g.curFn != nil && g.curFn.Span.File != nil {
				pkg, mod, _ := names.SplitOwnerAndModule(g.curFn.Span.File.Name)
				q1 := names.QualifyParts(pkg, mod, tt.Name)
				if _, ok := g.p.StructSigs[q1]; ok {
					return typecheck.Type{K: typecheck.TyStruct, Name: q1}
				}
				if _, ok := g.p.EnumSigs[q1]; ok {
					return typecheck.Type{K: typecheck.TyEnum, Name: q1}
				}
				q2 := names.QualifyParts(pkg, nil, tt.Name)
				if _, ok := g.p.StructSigs[q2]; ok {
					return typecheck.Type{K: typecheck.TyStruct, Name: q2}
				}
				if _, ok := g.p.EnumSigs[q2]; ok {
					return typecheck.Type{K: typecheck.TyEnum, Name: q2}
				}
			}
			return typecheck.Type{K: typecheck.TyBad}
		}
	default:
		return typecheck.Type{K: typecheck.TyBad}
	}
}

func unquoteUnescape(lit string) (string, error) {
	// Lexer keeps the full token lexeme including quotes; reuse Go-like unquoting.
	// This accepts standard escapes like \n, \t, \\, \", \r.
	return strconv.Unquote(lit)
}
