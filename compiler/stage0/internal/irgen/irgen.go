package irgen

import (
	"fmt"

	"voxlang/internal/ast"
	"voxlang/internal/ir"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

// Generate lowers a typechecked program into IR v0.
func Generate(p *typecheck.CheckedProgram) (*ir.Program, error) {
	g := &gen{
		p:        p,
		out:      &ir.Program{Structs: map[string]*ir.Struct{}, Enums: map[string]*ir.Enum{}, Funcs: map[string]*ir.Func{}},
		tmpID:    0,
		slotID:   0,
		funcSigs: p.FuncSigs,
	}
	if err := g.genNominalDefs(); err != nil {
		return nil, err
	}
	for _, fn := range p.Prog.Funcs {
		if fn != nil && len(fn.TypeParams) != 0 {
			// Stage0 generic functions are monomorphized on demand; only concrete instantiations are lowered.
			continue
		}
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
	case typecheck.TyI8:
		return ir.Type{K: ir.TI8}, nil
	case typecheck.TyU8:
		return ir.Type{K: ir.TU8}, nil
	case typecheck.TyI16:
		return ir.Type{K: ir.TI16}, nil
	case typecheck.TyU16:
		return ir.Type{K: ir.TU16}, nil
	case typecheck.TyI32:
		return ir.Type{K: ir.TI32}, nil
	case typecheck.TyU32:
		return ir.Type{K: ir.TU32}, nil
	case typecheck.TyI64:
		return ir.Type{K: ir.TI64}, nil
	case typecheck.TyU64:
		return ir.Type{K: ir.TU64}, nil
	case typecheck.TyISize:
		return ir.Type{K: ir.TISize}, nil
	case typecheck.TyUSize:
		return ir.Type{K: ir.TUSize}, nil
	case typecheck.TyString:
		return ir.Type{K: ir.TString}, nil
	case typecheck.TyRange:
		// Range types are represented as their base integer type in IR v0.
		if t.Base == nil {
			return ir.Type{}, fmt.Errorf("range missing base")
		}
		return g.irTypeFromChecked(*t.Base)
	case typecheck.TyStruct:
		return ir.Type{K: ir.TStruct, Name: t.Name}, nil
	case typecheck.TyEnum:
		return ir.Type{K: ir.TEnum, Name: t.Name}, nil
	case typecheck.TyVec:
		if t.Elem == nil {
			return ir.Type{}, fmt.Errorf("vec missing element type")
		}
		elem, err := g.irTypeFromChecked(*t.Elem)
		if err != nil {
			return ir.Type{}, err
		}
		return ir.Type{K: ir.TVec, Elem: &elem}, nil
	default:
		return ir.Type{}, fmt.Errorf("unsupported type")
	}
}

func (g *gen) zeroValue(t ir.Type) (ir.Value, error) {
	switch t.K {
	case ir.TUnit:
		return nil, nil
	case ir.TBool:
		return &ir.ConstBool{V: false}, nil
	case ir.TI8, ir.TU8, ir.TI16, ir.TU16, ir.TI32, ir.TU32, ir.TI64, ir.TU64, ir.TISize, ir.TUSize:
		return &ir.ConstInt{Ty: t, Bits: 0}, nil
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
		return nil, fmt.Errorf("unknown struct: %s", t.Name)
	case ir.TEnum:
		en, ok := g.out.Enums[t.Name]
		if !ok || en == nil || len(en.Variants) == 0 {
			return nil, fmt.Errorf("unknown enum: %s", t.Name)
		}
		// Deterministic zero value: first variant, with zero payload if needed.
		v0 := en.Variants[0]
		payload := make([]ir.Value, 0, len(v0.Fields))
		for _, ft := range v0.Fields {
			z, err := g.zeroValue(ft)
			if err != nil {
				return nil, err
			}
			payload = append(payload, z)
		}
		tmp := g.newTemp()
		g.emit(&ir.EnumInit{Dst: tmp, Ty: t, Variant: v0.Name, Payload: payload})
		return tmp, nil
	case ir.TVec:
		if t.Elem == nil {
			return nil, fmt.Errorf("vec missing element type")
		}
		tmp := g.newTemp()
		g.emit(&ir.VecNew{Dst: tmp, Ty: t, Elem: *t.Elem})
		return tmp, nil
	default:
		return &ir.ConstInt{Ty: t, Bits: 0}, nil
	}
}

func (g *gen) typeFromAstLimited(t ast.Type) typecheck.Type {
	switch tt := t.(type) {
	case *ast.UnitType:
		return typecheck.Type{K: typecheck.TyUnit}
	case *ast.NamedType:
		if len(tt.Parts) != 1 {
			return typecheck.Type{K: typecheck.TyBad}
		}
		switch tt.Parts[0] {
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
				q1 := names.QualifyParts(pkg, mod, tt.Parts[0])
				if _, ok := g.p.StructSigs[q1]; ok {
					return typecheck.Type{K: typecheck.TyStruct, Name: q1}
				}
				if _, ok := g.p.EnumSigs[q1]; ok {
					return typecheck.Type{K: typecheck.TyEnum, Name: q1}
				}
				q2 := names.QualifyParts(pkg, nil, tt.Parts[0])
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
