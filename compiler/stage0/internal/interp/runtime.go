package interp

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"voxlang/internal/ast"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

type Runtime struct {
	prog  *typecheck.CheckedProgram
	funcs map[string]*ast.FuncDecl
	stack []map[string]Value
	// Reuse stack frames to reduce map allocations in hot paths.
	framePool []map[string]Value
	args      []string
	// Cache decoded string literals (token text -> runtime string).
	strLitCache map[string]string
	// Cache parsed integer literals (token text -> uint64 bits before truncation).
	intLitCache map[string]uint64
	// Cache parsed integer patterns (token text -> int64).
	intPatCache map[string]int64
	// Cache whether a pattern subtree introduces bindings.
	patBindsCache map[ast.Pattern]bool
	// Pre-classified call kinds keyed by call expr node.
	callKinds map[*ast.CallExpr]callKind
}

type callKind uint8

const (
	callKindPlain callKind = iota
	callKindVec
	callKindStr
	callKindToStr
	callKindEnumCtor
)

func RunMain(p *typecheck.CheckedProgram) (string, error) {
	return RunMainWithArgs(p, nil)
}

func RunMainWithArgs(p *typecheck.CheckedProgram, args []string) (string, error) {
	rt := newRuntime(p)
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
	rt := newRuntime(p)
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
	log, _, err := RunTestsNamed(p, testNames)
	return log, err
}

func RunTestsNamed(p *typecheck.CheckedProgram, testNames []string) (string, []string, error) {
	if len(testNames) == 0 {
		return "[test] no tests found\n", nil, nil
	}

	results := runInterpTestsByModule(p, names.GroupQualifiedTestsByModule(testNames))
	var log strings.Builder
	failed := 0
	failedNames := make([]string, 0)
	for _, name := range testNames {
		r, ok := results[name]
		if !ok {
			failed++
			failedNames = append(failedNames, name)
			fmt.Fprintf(&log, "[FAIL] %s (%s): internal error: missing test result\n", name, formatTestDuration(0))
			continue
		}
		if r.err != nil {
			failed++
			failedNames = append(failedNames, name)
			fmt.Fprintf(&log, "[FAIL] %s (%s): %v\n", name, formatTestDuration(r.dur), r.err)
			continue
		}
		fmt.Fprintf(&log, "[OK] %s (%s)\n", name, formatTestDuration(r.dur))
	}
	fmt.Fprintf(&log, "[test] %d passed, %d failed\n", len(testNames)-failed, failed)
	if failed != 0 {
		return log.String(), failedNames, fmt.Errorf("%d test(s) failed", failed)
	}
	return log.String(), nil, nil
}

type interpTestResult struct {
	dur time.Duration
	err error
}

func runtimeForTests(p *typecheck.CheckedProgram) *Runtime {
	rt := newRuntime(p)
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	return rt
}

func runInterpTestsByModule(p *typecheck.CheckedProgram, groups []names.TestModuleGroup) map[string]interpTestResult {
	total := 0
	for _, g := range groups {
		total += len(g.Tests)
	}
	out := make(map[string]interpTestResult, total)
	if total == 0 {
		return out
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > len(groups) {
		workers = len(groups)
	}

	jobs := make(chan names.TestModuleGroup, len(groups))
	type namedResult struct {
		name string
		res  interpTestResult
	}
	results := make(chan namedResult, total)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for g := range jobs {
				rt := runtimeForTests(p)
				for _, name := range g.Tests {
					start := time.Now()
					sig, ok := p.FuncSigs[name]
					if !ok || len(sig.Params) != 0 || sig.Ret.K != typecheck.TyUnit {
						results <- namedResult{name: name, res: interpTestResult{
							dur: time.Since(start),
							err: fmt.Errorf("invalid test signature (expected fn %s() -> ())", name),
						}}
						continue
					}
					_, err := rt.call(name, nil)
					results <- namedResult{name: name, res: interpTestResult{dur: time.Since(start), err: err}}
				}
			}
		}()
	}
	for _, g := range groups {
		jobs <- g
	}
	close(jobs)
	go func() {
		wg.Wait()
		close(results)
	}()
	for r := range results {
		out[r.name] = r.res
	}
	return out
}

func formatTestDuration(d time.Duration) string {
	us := d.Microseconds()
	if us < 1000 {
		return fmt.Sprintf("%dus", us)
	}
	return fmt.Sprintf("%.2fms", float64(us)/1000.0)
}

func newRuntime(p *typecheck.CheckedProgram) *Runtime {
	rt := &Runtime{
		prog:          p,
		funcs:         map[string]*ast.FuncDecl{},
		strLitCache:   map[string]string{},
		intLitCache:   map[string]uint64{},
		intPatCache:   map[string]int64{},
		patBindsCache: map[ast.Pattern]bool{},
		callKinds:     map[*ast.CallExpr]callKind{},
	}
	for call := range p.VecCalls {
		rt.callKinds[call] = callKindVec
	}
	for call := range p.StrCalls {
		rt.callKinds[call] = callKindStr
	}
	for call := range p.ToStrCalls {
		rt.callKinds[call] = callKindToStr
	}
	for call := range p.EnumCtors {
		rt.callKinds[call] = callKindEnumCtor
	}
	return rt
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
	fn, ok := rt.funcs[name]
	if ok {
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
	if v, ok, err := rt.callBuiltin(name, args); ok {
		return v, err
	}
	return unit(), fmt.Errorf("unknown function: %s", name)
}

func (rt *Runtime) pushFrame() {
	n := len(rt.framePool)
	if n == 0 {
		rt.stack = append(rt.stack, map[string]Value{})
		return
	}
	fr := rt.framePool[n-1]
	rt.framePool = rt.framePool[:n-1]
	rt.stack = append(rt.stack, fr)
}

func (rt *Runtime) popFrame() {
	n := len(rt.stack)
	fr := rt.stack[n-1]
	rt.stack = rt.stack[:n-1]
	for k := range fr {
		delete(fr, k)
	}
	// Keep pool bounded.
	if len(rt.framePool) < 1024 {
		rt.framePool = append(rt.framePool, fr)
	}
}

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
	for i := len(rt.stack) - 1; i >= 0; i-- {
		if v, ok := rt.stack[i][name]; ok {
			return v, true
		}
	}
	return unit(), false
}

func (rt *Runtime) patHasBindings(p ast.Pattern) bool {
	if b, ok := rt.patBindsCache[p]; ok {
		return b
	}
	bind := false
	switch pt := p.(type) {
	case *ast.BindPat:
		bind = pt.Name != "" && pt.Name != "_"
	case *ast.VariantPat:
		for i := 0; i < len(pt.Args); i++ {
			if rt.patHasBindings(pt.Args[i]) {
				bind = true
				break
			}
		}
	}
	rt.patBindsCache[p] = bind
	return bind
}
