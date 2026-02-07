package ir

import (
	"strconv"
	"testing"
)

func TestFmtStringValues(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		want string
	}{
		{name: "param", v: &ParamRef{Index: 2}, want: "%p2"},
		{name: "temp", v: &Temp{ID: 3}, want: "%t3"},
		{name: "slot", v: &Slot{ID: 4}, want: "$v4"},
		{name: "int", v: &ConstInt{Ty: Type{K: TI32}, V: 7}, want: "7"},
		{name: "bool_true", v: &ConstBool{V: true}, want: "true"},
		{name: "bool_false", v: &ConstBool{V: false}, want: "false"},
		{name: "str", v: &ConstStr{S: "a\nb"}, want: strconv.Quote("a\nb")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.fmtString(); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFmtStringInstrAndTerm(t *testing.T) {
	t0 := &Temp{ID: 0}
	p0 := &ParamRef{Index: 0}
	s0 := &Slot{ID: 0}

	insCases := []struct {
		name string
		ins  Instr
		want string
	}{
		{name: "const", ins: &Const{Dst: t0, Ty: Type{K: TI32}, Val: &ConstInt{Ty: Type{K: TI32}, V: 1}}, want: "%t0 = const i32 1"},
		{name: "binop", ins: &BinOp{Dst: &Temp{ID: 1}, Op: OpAdd, Ty: Type{K: TI32}, A: t0, B: p0}, want: "%t1 = add i32 %t0 %p0"},
		{name: "cmp", ins: &Cmp{Dst: &Temp{ID: 2}, Op: CmpLt, Ty: Type{K: TI64}, A: t0, B: &ConstInt{Ty: Type{K: TI64}, V: 5}}, want: "%t2 = cmp_lt i64 %t0 5"},
		{name: "and", ins: &And{Dst: &Temp{ID: 3}, A: &ConstBool{V: true}, B: &ConstBool{V: false}}, want: "%t3 = and true false"},
		{name: "or", ins: &Or{Dst: &Temp{ID: 4}, A: &ConstBool{V: true}, B: &ConstBool{V: false}}, want: "%t4 = or true false"},
		{name: "not", ins: &Not{Dst: &Temp{ID: 5}, A: &ConstBool{V: true}}, want: "%t5 = not true"},
		{name: "slot", ins: &SlotDecl{Slot: s0, Ty: Type{K: TI32}}, want: "$v0 = slot i32"},
		{name: "store", ins: &Store{Slot: s0, Val: t0}, want: "store $v0 %t0"},
		{name: "load", ins: &Load{Dst: &Temp{ID: 6}, Ty: Type{K: TI32}, Slot: s0}, want: "%t6 = load i32 $v0"},
		{name: "call_unit", ins: &Call{Ret: Type{K: TUnit}, Name: "foo", Args: []Value{t0, p0}}, want: "call unit foo(%t0, %p0)"},
		{name: "call_ret", ins: &Call{Dst: &Temp{ID: 7}, Ret: Type{K: TI32}, Name: "bar", Args: nil}, want: "%t7 = call i32 bar()"},
		{name: "enum_init_payload", ins: &EnumInit{Dst: &Temp{ID: 8}, Ty: Type{K: TEnum, Name: "E"}, Variant: "A", Payload: []Value{&ConstInt{Ty: Type{K: TI32}, V: 1}}}, want: "%t8 = enum_init enum(E) A(1)"},
		{name: "enum_init_unit", ins: &EnumInit{Dst: &Temp{ID: 9}, Ty: Type{K: TEnum, Name: "E"}, Variant: "None"}, want: "%t9 = enum_init enum(E) None"},
		{name: "enum_tag", ins: &EnumTag{Dst: &Temp{ID: 10}, Recv: t0}, want: "%t10 = enum_tag %t0"},
		{name: "enum_payload", ins: &EnumPayload{Dst: &Temp{ID: 11}, Ty: Type{K: TI32}, Recv: t0, Variant: "A", Index: 0}, want: "%t11 = enum_payload i32 %t0 A 0"},
		{name: "vec_new", ins: &VecNew{Dst: &Temp{ID: 12}, Ty: Type{K: TVec, Elem: &Type{K: TI32}}, Elem: Type{K: TI32}}, want: "%t12 = vec_new vec(i32)"},
		{name: "vec_push", ins: &VecPush{Recv: s0, Elem: Type{K: TI32}, Val: t0}, want: "vec_push $v0 %t0"},
		{name: "vec_len", ins: &VecLen{Dst: &Temp{ID: 13}, Recv: s0}, want: "%t13 = vec_len $v0"},
		{name: "vec_get", ins: &VecGet{Dst: &Temp{ID: 14}, Ty: Type{K: TI32}, Recv: s0, Idx: &ConstInt{Ty: Type{K: TI32}, V: 0}}, want: "%t14 = vec_get i32 $v0 0"},
	}
	for _, tc := range insCases {
		t.Run("ins_"+tc.name, func(t *testing.T) {
			if got := tc.ins.fmtString(); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}

	termCases := []struct {
		name string
		term Term
		want string
	}{
		{name: "ret_unit", term: &Ret{}, want: "ret"},
		{name: "ret_val", term: &Ret{Val: t0}, want: "ret %t0"},
		{name: "br", term: &Br{Target: "next"}, want: "br next"},
		{name: "condbr", term: &CondBr{Cond: &ConstBool{V: true}, Then: "t", Else: "e"}, want: "condbr true t e"},
	}
	for _, tc := range termCases {
		t.Run("term_"+tc.name, func(t *testing.T) {
			if got := tc.term.fmtString(); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
