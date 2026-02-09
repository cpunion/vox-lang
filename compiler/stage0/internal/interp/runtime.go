package interp

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

type Runtime struct {
	prog  *typecheck.CheckedProgram
	funcs map[string]*ast.FuncDecl
	stack []map[string]Value
	args  []string
}

func RunMain(p *typecheck.CheckedProgram) (string, error) {
	return RunMainWithArgs(p, nil)
}

func RunMainWithArgs(p *typecheck.CheckedProgram, args []string) (string, error) {
	rt := &Runtime{prog: p, funcs: map[string]*ast.FuncDecl{}}
	rt.args = args
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	mainFn, ok := rt.funcs["main"]
	if !ok {
		return "", fmt.Errorf("missing main")
	}
	if len(mainFn.Params) != 0 {
		return "", fmt.Errorf("main must have no parameters (stage0)")
	}
	v, err := rt.call("main", nil)
	if err != nil {
		return "", err
	}
	mainSig, ok := p.FuncSigs["main"]
	if !ok {
		return "", fmt.Errorf("missing main signature")
	}
	switch v.K {
	case VUnit:
		return "", nil
	case VInt:
		return formatInt(v.I, mainSig.Ret), nil
	case VBool:
		if v.B {
			return "true", nil
		}
		return "false", nil
	case VString:
		return v.S, nil
	default:
		return "", nil
	}
}

func formatInt(bits uint64, ty typecheck.Type) string {
	base := ty
	if base.K == typecheck.TyRange && base.Base != nil {
		base = *base.Base
	}
	switch base.K {
	case typecheck.TyU8, typecheck.TyU16, typecheck.TyU32, typecheck.TyU64, typecheck.TyUSize:
		return fmt.Sprintf("%d", bits)
	case typecheck.TyI8:
		return fmt.Sprintf("%d", int64(int8(bits)))
	case typecheck.TyI16:
		return fmt.Sprintf("%d", int64(int16(bits)))
	case typecheck.TyI32:
		return fmt.Sprintf("%d", int64(int32(bits)))
	case typecheck.TyI64, typecheck.TyISize, typecheck.TyUntypedInt:
		return fmt.Sprintf("%d", int64(bits))
	default:
		return fmt.Sprintf("%d", int64(bits))
	}
}

func RunTests(p *typecheck.CheckedProgram) (string, error) {
	rt := &Runtime{prog: p, funcs: map[string]*ast.FuncDecl{}}
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	// Discover tests by naming convention.
	var testNames []string
	for name := range rt.funcs {
		fn := rt.funcs[name]
		if fn == nil || fn.Span.File == nil {
			continue
		}
		if !isTestFile(fn.Span.File.Name) {
			continue
		}
		if strings.HasPrefix(fn.Name, "test_") {
			testNames = append(testNames, name)
		}
	}
	sort.Strings(testNames)
	var log strings.Builder
	failed := 0
	for _, name := range testNames {
		sig, ok := p.FuncSigs[name]
		if !ok || len(sig.Params) != 0 || sig.Ret.K != typecheck.TyUnit {
			failed++
			fmt.Fprintf(&log, "[FAIL] %s: invalid test signature (expected fn %s() -> ())\n", name, name)
			continue
		}
		_, err := rt.call(name, nil)
		if err != nil {
			failed++
			fmt.Fprintf(&log, "[FAIL] %s: %v\n", name, err)
		} else {
			fmt.Fprintf(&log, "[OK] %s\n", name)
		}
	}
	if len(testNames) == 0 {
		log.WriteString("[test] no tests found\n")
	} else {
		fmt.Fprintf(&log, "[test] %d passed, %d failed\n", len(testNames)-failed, failed)
	}
	if failed != 0 {
		return log.String(), fmt.Errorf("%d test(s) failed", failed)
	}
	return log.String(), nil
}

func isTestFile(name string) bool {
	// Stage0 convention:
	// - tests/**/*.vox
	// - src/**/*_test.vox
	name = filepath.ToSlash(name)
	if strings.HasPrefix(name, "tests/") {
		return true
	}
	return strings.HasSuffix(name, "_test.vox")
}

func (rt *Runtime) call(name string, args []Value) (Value, error) {
	if v, ok, err := rt.callBuiltin(name, args); ok {
		return v, err
	}
	fn, ok := rt.funcs[name]
	if !ok {
		return unit(), fmt.Errorf("unknown function: %s", name)
	}
	rt.pushFrame()
	for i, p := range fn.Params {
		rt.frame()[p.Name] = args[i]
	}
	v, err := rt.evalBlock(fn.Body)
	rt.popFrame()
	if err != nil {
		if r, ok := err.(returnSignal); ok {
			return r.V, nil
		}
		return unit(), err
	}
	return v, nil
}

func (rt *Runtime) pushFrame() { rt.stack = append(rt.stack, map[string]Value{}) }
func (rt *Runtime) popFrame()  { rt.stack = rt.stack[:len(rt.stack)-1] }

func (rt *Runtime) frame() map[string]Value {
	return rt.stack[len(rt.stack)-1]
}

func (rt *Runtime) lookup(name string) (map[string]Value, bool) {
	for i := len(rt.stack) - 1; i >= 0; i-- {
		if _, ok := rt.stack[i][name]; ok {
			return rt.stack[i], true
		}
	}
	return nil, false
}

func (rt *Runtime) lookupValue(name string) (Value, bool) {
	if fr, ok := rt.lookup(name); ok {
		return fr[name], true
	}
	return unit(), false
}
