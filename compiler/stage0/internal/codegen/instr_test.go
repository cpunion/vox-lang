package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"voxlang/internal/ir"
)

func TestCodegenInstrCoverage_ConstBinCmpLogicSlotsCallsBranches(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not found")
	}

	cases := []struct {
		name string
		prog *ir.Program
		want string
	}{
		{
			name: "i32_arith",
			prog: progMainI32(func(b *ir.Block) {
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				t2 := &ir.Temp{ID: 2}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 40}},
					&ir.Const{Dst: t1, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 2}},
					&ir.BinOp{Dst: t2, Op: ir.OpAdd, Ty: ir.Type{K: ir.TI32}, A: t0, B: t1},
				)
				b.Term = &ir.Ret{Val: t2}
			}),
			want: "42",
		},
		{
			name: "i32_bitwise_shift",
			prog: progMainI32(func(b *ir.Block) {
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				t2 := &ir.Temp{ID: 2}
				t3 := &ir.Temp{ID: 3}
				t4 := &ir.Temp{ID: 4}
				t5 := &ir.Temp{ID: 5}
				t6 := &ir.Temp{ID: 6}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 6}},
					&ir.Const{Dst: t1, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 3}},
					&ir.Const{Dst: t2, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 1}},
					&ir.Const{Dst: t3, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 4}},
					&ir.BinOp{Dst: t4, Op: ir.OpBitAnd, Ty: ir.Type{K: ir.TI32}, A: t0, B: t1}, // 6 & 3 = 2
					&ir.BinOp{Dst: t5, Op: ir.OpShl, Ty: ir.Type{K: ir.TI32}, A: t2, B: t3},    // 1 << 4 = 16
					&ir.BinOp{Dst: t6, Op: ir.OpBitOr, Ty: ir.Type{K: ir.TI32}, A: t4, B: t5},  // 2 | 16 = 18
				)
				b.Term = &ir.Ret{Val: t6}
			}),
			want: "18",
		},
		{
			name: "slots_load_store",
			prog: progMainI32(func(b *ir.Block) {
				s0 := &ir.Slot{ID: 0}
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				b.Instr = append(b.Instr,
					&ir.SlotDecl{Slot: s0, Ty: ir.Type{K: ir.TI64}},
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TI64}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI64}, Bits: 7}},
					&ir.Store{Slot: s0, Val: t0},
					&ir.Load{Dst: t1, Ty: ir.Type{K: ir.TI64}, Slot: s0},
				)
				// return i32; cast isn't in IR, so compare then choose with branches:
				// if (t1 == 7) return 1 else return 0
				cond := &ir.Temp{ID: 2}
				b.Instr = append(b.Instr, &ir.Cmp{Dst: cond, Op: ir.CmpEq, Ty: ir.Type{K: ir.TI64}, A: t1, B: &ir.ConstInt{Ty: ir.Type{K: ir.TI64}, Bits: 7}})
				b.Term = &ir.CondBr{Cond: cond, Then: "then", Else: "else"}
			}, func(fn *ir.Func) {
				thenBlk := &ir.Block{Name: "then"}
				elseBlk := &ir.Block{Name: "else"}
				thenBlk.Instr = append(thenBlk.Instr, &ir.Const{Dst: &ir.Temp{ID: 3}, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 1}})
				thenBlk.Term = &ir.Ret{Val: &ir.Temp{ID: 3}}
				elseBlk.Instr = append(elseBlk.Instr, &ir.Const{Dst: &ir.Temp{ID: 4}, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 0}})
				elseBlk.Term = &ir.Ret{Val: &ir.Temp{ID: 4}}
				fn.Blocks = append(fn.Blocks, thenBlk, elseBlk)
			}),
			want: "1",
		},
		{
			name: "condbr_and_br",
			prog: progMainBool(func(fn *ir.Func) {
				entry := &ir.Block{Name: "entry"}
				thenBlk := &ir.Block{Name: "then"}
				elseBlk := &ir.Block{Name: "else"}
				endBlk := &ir.Block{Name: "end"}

				t0 := &ir.Temp{ID: 0}
				entry.Instr = append(entry.Instr, &ir.Const{Dst: t0, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: true}})
				entry.Term = &ir.CondBr{Cond: t0, Then: thenBlk.Name, Else: elseBlk.Name}

				thenBlk.Term = &ir.Br{Target: endBlk.Name}
				elseBlk.Term = &ir.Br{Target: endBlk.Name}

				// end returns true
				t1 := &ir.Temp{ID: 1}
				endBlk.Instr = append(endBlk.Instr, &ir.Const{Dst: t1, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: true}})
				endBlk.Term = &ir.Ret{Val: t1}

				fn.Blocks = []*ir.Block{entry, thenBlk, elseBlk, endBlk}
			}),
			want: "true",
		},
		{
			name: "logical_ops",
			prog: progMainBool(func(fn *ir.Func) {
				b := &ir.Block{Name: "entry"}
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				t2 := &ir.Temp{ID: 2}
				t3 := &ir.Temp{ID: 3}
				t4 := &ir.Temp{ID: 4}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: true}},
					&ir.Const{Dst: t1, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: false}},
					&ir.And{Dst: t2, A: t0, B: t1},
					&ir.Or{Dst: t3, A: t0, B: t1},
					&ir.Not{Dst: t4, A: t2},
				)
				// !(true && false) == true
				b.Term = &ir.Ret{Val: t4}
				fn.Blocks = []*ir.Block{b}
			}),
			want: "true",
		},
		{
			name: "casts_i32_i64",
			prog: progMainI32(func(b *ir.Block) {
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				t2 := &ir.Temp{ID: 2}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 7}},
					&ir.IntCastChecked{Dst: t1, From: ir.Type{K: ir.TI32}, To: ir.Type{K: ir.TI64}, V: t0},
					&ir.IntCastChecked{Dst: t2, From: ir.Type{K: ir.TI64}, To: ir.Type{K: ir.TI32}, V: t1},
				)
				b.Term = &ir.Ret{Val: t2}
			}),
			want: "7",
		},
		{
			name: "range_check",
			prog: progMainI32(func(b *ir.Block) {
				t0 := &ir.Temp{ID: 0}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 2}},
					&ir.RangeCheckInt{Ty: ir.Type{K: ir.TI32}, V: t0, Lo: 0, Hi: 3},
				)
				b.Term = &ir.Ret{Val: t0}
			}),
			want: "2",
		},
		{
			name: "call_and_print",
			prog: &ir.Program{Funcs: map[string]*ir.Func{
				"main": {
					Name: "main",
					Ret:  ir.Type{K: ir.TI32},
					Blocks: []*ir.Block{
						{
							Name: "entry",
							Instr: []ir.Instr{
								&ir.Const{Dst: &ir.Temp{ID: 0}, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: "x"}},
								&ir.Call{Ret: ir.Type{K: ir.TUnit}, Name: "print", Args: []ir.Value{&ir.Temp{ID: 0}}},
								&ir.Call{Dst: &ir.Temp{ID: 1}, Ret: ir.Type{K: ir.TI32}, Name: "callee", Args: nil},
							},
							Term: &ir.Ret{Val: &ir.Temp{ID: 1}},
						},
					},
				},
				"callee": {
					Name: "callee",
					Ret:  ir.Type{K: ir.TI32},
					Blocks: []*ir.Block{
						{
							Name: "entry",
							Instr: []ir.Instr{
								&ir.Const{Dst: &ir.Temp{ID: 0}, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 7}},
							},
							Term: &ir.Ret{Val: &ir.Temp{ID: 0}},
						},
					},
				},
			}},
			want: "x7",
		},
		{
			name: "panic_not_taken",
			prog: &ir.Program{Funcs: map[string]*ir.Func{
				"main": {
					Name: "main",
					Ret:  ir.Type{K: ir.TI32},
					Blocks: []*ir.Block{
						{
							Name: "entry",
							Instr: []ir.Instr{
								&ir.Const{Dst: &ir.Temp{ID: 0}, Ty: ir.Type{K: ir.TBool}, Val: &ir.ConstBool{V: false}},
							},
							Term: &ir.CondBr{Cond: &ir.Temp{ID: 0}, Then: "panic", Else: "cont"},
						},
						{
							Name: "cont",
							Instr: []ir.Instr{
								&ir.Call{Dst: &ir.Temp{ID: 1}, Ret: ir.Type{K: ir.TI32}, Name: "callee", Args: nil},
							},
							Term: &ir.Ret{Val: &ir.Temp{ID: 1}},
						},
						{
							Name: "panic",
							Instr: []ir.Instr{
								&ir.Const{Dst: &ir.Temp{ID: 2}, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: "boom"}},
								&ir.Call{Ret: ir.Type{K: ir.TUnit}, Name: "panic", Args: []ir.Value{&ir.Temp{ID: 2}}},
								&ir.Const{Dst: &ir.Temp{ID: 3}, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 0}},
							},
							Term: &ir.Ret{Val: &ir.Temp{ID: 3}},
						},
					},
				},
				"callee": {
					Name: "callee",
					Ret:  ir.Type{K: ir.TI32},
					Blocks: []*ir.Block{
						{
							Name: "entry",
							Instr: []ir.Instr{
								&ir.Const{Dst: &ir.Temp{ID: 0}, Ty: ir.Type{K: ir.TI32}, Val: &ir.ConstInt{Ty: ir.Type{K: ir.TI32}, Bits: 9}},
							},
							Term: &ir.Ret{Val: &ir.Temp{ID: 0}},
						},
					},
				},
			}},
			want: "9",
		},
		{
			name: "string_return",
			prog: progMainStr(func(fn *ir.Func) {
				b := &ir.Block{Name: "entry"}
				t0 := &ir.Temp{ID: 0}
				b.Instr = append(b.Instr, &ir.Const{Dst: t0, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: "hello"}})
				b.Term = &ir.Ret{Val: t0}
				fn.Blocks = []*ir.Block{b}
			}),
			want: "hello",
		},
		{
			name: "string_cmp_eq",
			prog: progMainBool(func(fn *ir.Func) {
				b := &ir.Block{Name: "entry"}
				t0 := &ir.Temp{ID: 0}
				t1 := &ir.Temp{ID: 1}
				t2 := &ir.Temp{ID: 2}
				b.Instr = append(b.Instr,
					&ir.Const{Dst: t0, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: "a"}},
					&ir.Const{Dst: t1, Ty: ir.Type{K: ir.TString}, Val: &ir.ConstStr{S: "a"}},
					&ir.Cmp{Dst: t2, Op: ir.CmpEq, Ty: ir.Type{K: ir.TString}, A: t0, B: t1},
				)
				b.Term = &ir.Ret{Val: t2}
				fn.Blocks = []*ir.Block{b}
			}),
			want: "true",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csrc, err := EmitC(tc.prog, EmitOptions{EmitDriverMain: true})
			if err != nil {
				t.Fatal(err)
			}
			out := compileAndRunC(t, cc, csrc)
			if got := strings.TrimSpace(out); got != tc.want {
				t.Fatalf("expected %q, got %q\nC:\n%s", tc.want, got, csrc)
			}
		})
	}
}

func compileAndRunC(t *testing.T, cc string, csrc string) string {
	t.Helper()
	dir := t.TempDir()
	cPath := filepath.Join(dir, "a.c")
	binPath := filepath.Join(dir, "a.out")
	if err := os.WriteFile(cPath, []byte(csrc), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, string(out))
	}
	run := exec.Command(binPath)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, string(out))
	}
	return string(out)
}

func progMainI32(entryFn func(b *ir.Block), extra ...func(fn *ir.Func)) *ir.Program {
	fn := &ir.Func{Name: "main", Ret: ir.Type{K: ir.TI32}}
	b := &ir.Block{Name: "entry"}
	entryFn(b)
	fn.Blocks = []*ir.Block{b}
	for _, f := range extra {
		f(fn)
	}
	return &ir.Program{Funcs: map[string]*ir.Func{"main": fn}}
}

func progMainBool(fill func(fn *ir.Func)) *ir.Program {
	fn := &ir.Func{Name: "main", Ret: ir.Type{K: ir.TBool}}
	fill(fn)
	return &ir.Program{Funcs: map[string]*ir.Func{"main": fn}}
}

func progMainStr(fill func(fn *ir.Func)) *ir.Program {
	fn := &ir.Func{Name: "main", Ret: ir.Type{K: ir.TString}}
	fill(fn)
	return &ir.Program{Funcs: map[string]*ir.Func{"main": fn}}
}
