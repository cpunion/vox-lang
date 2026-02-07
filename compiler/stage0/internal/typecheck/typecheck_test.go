package typecheck

import (
	"testing"

	"voxlang/internal/parser"
	"voxlang/internal/source"
)

func TestUntypedIntConstraint(t *testing.T) {
	f := source.NewFile("test.vox", `fn main() -> i32 { let x: i32 = 1; return x; }`)
	prog, pdiags := parser.Parse(f)
	if pdiags != nil && len(pdiags.Items) > 0 {
		t.Fatalf("parse diags: %+v", pdiags.Items)
	}
	_, tdiags := Check(prog)
	if tdiags != nil && len(tdiags.Items) > 0 {
		t.Fatalf("type diags: %+v", tdiags.Items)
	}
}
