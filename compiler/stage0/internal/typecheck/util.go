package typecheck

import (
	"strings"

	"voxlang/internal/ast"
)

func (t Type) String() string {
	switch t.K {
	case TyUnit:
		return "()"
	case TyBool:
		return "bool"
	case TyI32:
		return "i32"
	case TyI64:
		return "i64"
	case TyString:
		return "String"
	case TyUntypedInt:
		return "untyped-int"
	case TyStruct:
		return t.Name
	case TyEnum:
		return t.Name
	case TyVec:
		if t.Elem == nil {
			return "Vec[<bad>]"
		}
		return "Vec[" + t.Elem.String() + "]"
	case TyParam:
		return t.Name
	default:
		return "<bad>"
	}
}

func sameModPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameType(a, b Type) bool {
	if a.K == TyBad || b.K == TyBad {
		return false
	}
	// Resolve untyped int to concrete only via constraints before comparing.
	if a.K == TyUntypedInt || b.K == TyUntypedInt {
		return false
	}
	if a.K != b.K {
		return false
	}
	if a.K == TyStruct || a.K == TyEnum {
		return a.Name == b.Name
	}
	if a.K == TyVec {
		if a.Elem == nil || b.Elem == nil {
			return false
		}
		return sameType(*a.Elem, *b.Elem)
	}
	if a.K == TyParam {
		return a.Name == b.Name
	}
	return true
}

func chooseType(ann, init Type) Type {
	if ann.K != TyBad {
		return ann
	}
	return init
}

func defaultImportAlias(path string) string {
	// For stage0, dependency package import paths are simple names like "mathlib".
	// If we later support nested module paths, use the last segment.
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func splitModPath(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return out
}

func calleeParts(ex ast.Expr) ([]string, bool) {
	switch e := ex.(type) {
	case *ast.IdentExpr:
		return []string{e.Name}, true
	case *ast.MemberExpr:
		p, ok := calleeParts(e.Recv)
		if !ok {
			return nil, false
		}
		return append(p, e.Name), true
	default:
		return nil, false
	}
}
