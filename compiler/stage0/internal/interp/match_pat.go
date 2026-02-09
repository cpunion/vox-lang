package interp

import (
	"fmt"
	"strconv"

	"voxlang/internal/ast"
	"voxlang/internal/typecheck"
)

// matchPat checks whether v (of expected type) matches pattern p.
// On success, it may bind names into the current stack frame.
func (rt *Runtime) matchPat(p ast.Pattern, v Value, expected typecheck.Type) (bool, error) {
	return rt.matchPatInto(p, v, expected, rt.frame())
}

func (rt *Runtime) matchPatInto(p ast.Pattern, v Value, expected typecheck.Type, bindings map[string]Value) (bool, error) {
	expected = stripRange(expected)

	switch p := p.(type) {
	case *ast.WildPat:
		return true, nil

	case *ast.BindPat:
		// Bind pattern always matches and binds the value to Name.
		// "_" is parsed as WildPat, but keep this defensive check.
		if p.Name != "" && p.Name != "_" && bindings != nil {
			bindings[p.Name] = cloneValue(v)
		}
		return true, nil

	case *ast.IntPat:
		if v.K != VInt || !isIntType(expected) {
			return false, nil
		}
		n, ok := rt.intPatCache[p.Text]
		if !ok {
			parsed, err := strconv.ParseInt(p.Text, 10, 64)
			if err != nil {
				return false, fmt.Errorf("invalid int pattern")
			}
			rt.intPatCache[p.Text] = parsed
			n = parsed
		}
		patBits, err := castIntChecked(uint64(n), typecheck.Type{K: typecheck.TyI64}, expected)
		if err != nil {
			// Out of range for the expected type: pattern can never match.
			return false, nil
		}
		if truncBits(v.I, intBitWidth(expected)) != patBits {
			return false, nil
		}
		return true, nil

	case *ast.BoolPat:
		if expected.K != typecheck.TyBool || v.K != VBool {
			return false, nil
		}
		return v.B == p.Value, nil

	case *ast.StrPat:
		if v.K != VString {
			return false, nil
		}
		s, ok := rt.strLitCache[p.Text]
		if !ok {
			parsed, err := strconv.Unquote(p.Text)
			if err != nil {
				return false, fmt.Errorf("invalid string pattern")
			}
			rt.strLitCache[p.Text] = parsed
			s = parsed
		}
		if v.S != s {
			return false, nil
		}
		return true, nil

	case *ast.VariantPat:
		if v.K != VEnum {
			return false, nil
		}

		ety := expected
		if ety.K != typecheck.TyEnum {
			// match is typechecked, but keep this as a fallback for robustness.
			if v.E == "" {
				return false, fmt.Errorf("internal: missing enum name for value")
			}
			ety = typecheck.Type{K: typecheck.TyEnum, Name: v.E}
		}

		es, ok := rt.prog.EnumSigs[ety.Name]
		if !ok {
			return false, fmt.Errorf("unknown enum: %s", ety.Name)
		}
		tag, ok := es.VariantIndex[p.Variant]
		if !ok {
			return false, fmt.Errorf("unknown variant: %s", p.Variant)
		}
		if v.T != tag {
			return false, nil
		}
		vsig := es.Variants[tag]
		if len(p.Args) != len(vsig.Fields) {
			return false, fmt.Errorf("wrong pattern arity for variant %s.%s", es.Name, p.Variant)
		}
		if len(v.P) != len(vsig.Fields) {
			return false, fmt.Errorf("internal: enum payload arity mismatch for %s.%s", es.Name, p.Variant)
		}
		for i := 0; i < len(p.Args); i++ {
			ok, err := rt.matchPatInto(p.Args[i], v.P[i], vsig.Fields[i], bindings)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil

	default:
		return false, fmt.Errorf("unsupported pattern")
	}
}
