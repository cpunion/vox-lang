package interp

import (
	"fmt"
	"net"
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
	// Intrinsic runtime handles used by std/sync wrappers.
	nextHandle uint64
	mutexI32   map[uint64]int32
	atomicI32  map[uint64]int32
	mutexI64   map[uint64]int64
	atomicI64  map[uint64]int64
	tcpConns   map[uint64]net.Conn
}

type callKind uint8

const (
	callKindPlain callKind = iota
	callKindVec
	callKindStr
	callKindToStr
	callKindEnumCtor
)

// TestResult describes one interpreted test execution.
type TestResult struct {
	Dur time.Duration
	Err error
}

// ModuleSummary aggregates pass/fail and duration for one test module.
type ModuleSummary struct {
	Module string
	Passed int
	Failed int
	Dur    time.Duration
}

// NamedDuration is used for the slowest-tests report.
type NamedDuration struct {
	Name string
	Dur  time.Duration
}

// TestRunReport contains detailed interpreter test results for CLI reporting.
type TestRunReport struct {
	Log             string
	Results         map[string]TestResult
	FailedNames     []string
	ModuleSummaries []ModuleSummary
	Slowest         []NamedDuration
}

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
	return RunTestsNamedWithJobs(p, testNames, 0)
}

func RunTestsNamedWithJobs(p *typecheck.CheckedProgram, testNames []string, jobs int) (string, []string, error) {
	rep, err := RunTestsNamedDetailedWithJobs(p, testNames, jobs)
	return rep.Log, rep.FailedNames, err
}

func RunTestsNamedDetailedWithJobs(p *typecheck.CheckedProgram, testNames []string, jobs int) (TestRunReport, error) {
	if len(testNames) == 0 {
		return TestRunReport{
			Log:             "[test] no tests found\n",
			Results:         map[string]TestResult{},
			FailedNames:     nil,
			ModuleSummaries: nil,
			Slowest:         nil,
		}, nil
	}

	results := runInterpTestsByModule(p, names.GroupQualifiedTestsByModule(testNames), jobs)
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
	moduleSummaries, slowest := summarizeInterpTestResults(testNames, results)
	for _, ms := range moduleSummaries {
		fmt.Fprintf(&log, "[module] %s: %d passed, %d failed (%s)\n", displayModuleNameForInterp(ms.module), ms.passed, ms.failed, formatTestDuration(ms.dur))
	}
	for _, s := range slowest {
		fmt.Fprintf(&log, "[slowest] %s (%s)\n", s.name, formatTestDuration(s.dur))
	}
	fmt.Fprintf(&log, "[test] %d passed, %d failed\n", len(testNames)-failed, failed)
	rep := TestRunReport{
		Log:             log.String(),
		Results:         exportInterpResults(results),
		FailedNames:     failedNames,
		ModuleSummaries: exportInterpModuleSummaries(moduleSummaries),
		Slowest:         exportInterpSlowest(slowest),
	}
	if failed != 0 {
		return rep, fmt.Errorf("%d test(s) failed", failed)
	}
	return rep, nil
}

type interpTestResult struct {
	dur time.Duration
	err error
}

type interpModuleSummary struct {
	module string
	passed int
	failed int
	dur    time.Duration
}

type namedInterpDuration struct {
	name string
	dur  time.Duration
}

func summarizeInterpTestResults(testNames []string, results map[string]interpTestResult) ([]interpModuleSummary, []namedInterpDuration) {
	modMap := map[string]interpModuleSummary{}
	slowest := make([]namedInterpDuration, 0, len(testNames))
	for _, name := range testNames {
		mod := moduleFromQualifiedTestName(name)
		sum, ok := modMap[mod]
		if !ok {
			sum = interpModuleSummary{module: mod}
		}
		r, rok := results[name]
		if !rok || r.err != nil {
			sum.failed++
			if rok {
				sum.dur += r.dur
				slowest = append(slowest, namedInterpDuration{name: name, dur: r.dur})
			}
		} else {
			sum.passed++
			sum.dur += r.dur
			slowest = append(slowest, namedInterpDuration{name: name, dur: r.dur})
		}
		modMap[mod] = sum
	}

	keys := make([]string, 0, len(modMap))
	for k := range modMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	mods := make([]interpModuleSummary, 0, len(keys))
	for _, k := range keys {
		mods = append(mods, modMap[k])
	}

	sort.Slice(slowest, func(i, j int) bool {
		if slowest[i].dur == slowest[j].dur {
			return slowest[i].name < slowest[j].name
		}
		return slowest[i].dur > slowest[j].dur
	})
	if len(slowest) > 5 {
		slowest = slowest[:5]
	}
	return mods, slowest
}

func exportInterpResults(results map[string]interpTestResult) map[string]TestResult {
	out := make(map[string]TestResult, len(results))
	for name, r := range results {
		out[name] = TestResult{Dur: r.dur, Err: r.err}
	}
	return out
}

func exportInterpModuleSummaries(summaries []interpModuleSummary) []ModuleSummary {
	out := make([]ModuleSummary, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, ModuleSummary{
			Module: s.module,
			Passed: s.passed,
			Failed: s.failed,
			Dur:    s.dur,
		})
	}
	return out
}

func exportInterpSlowest(slowest []namedInterpDuration) []NamedDuration {
	out := make([]NamedDuration, 0, len(slowest))
	for _, s := range slowest {
		out = append(out, NamedDuration{Name: s.name, Dur: s.dur})
	}
	return out
}

func runtimeForTests(p *typecheck.CheckedProgram) *Runtime {
	rt := newRuntime(p)
	for _, fn := range p.Prog.Funcs {
		rt.funcs[names.QualifyFunc(fn.Span.File.Name, fn.Name)] = fn
	}
	return rt
}

func runInterpTestsByModule(p *typecheck.CheckedProgram, groups []names.TestModuleGroup, jobs int) map[string]interpTestResult {
	total := 0
	for _, g := range groups {
		total += len(g.Tests)
	}
	out := make(map[string]interpTestResult, total)
	if total == 0 {
		return out
	}

	workers := interpModuleWorkers(len(groups), jobs)

	workq := make(chan names.TestModuleGroup, len(groups))
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
			for g := range workq {
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
		workq <- g
	}
	close(workq)
	go func() {
		wg.Wait()
		close(results)
	}()
	for r := range results {
		out[r.name] = r.res
	}
	return out
}

func interpModuleWorkers(moduleCount int, jobs int) int {
	if moduleCount <= 0 {
		return 1
	}
	if jobs > 0 {
		if jobs > moduleCount {
			return moduleCount
		}
		return jobs
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > moduleCount {
		workers = moduleCount
	}
	return workers
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
		nextHandle:    1,
		mutexI32:      map[uint64]int32{},
		atomicI32:     map[uint64]int32{},
		mutexI64:      map[uint64]int64{},
		atomicI64:     map[uint64]int64{},
		tcpConns:      map[uint64]net.Conn{},
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

func moduleFromQualifiedTestName(name string) string {
	i := strings.LastIndex(name, "::")
	if i < 0 {
		return ""
	}
	return name[:i]
}

func displayModuleNameForInterp(module string) string {
	if module == "" {
		return "(root)"
	}
	return module
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
