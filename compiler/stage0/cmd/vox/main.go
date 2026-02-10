package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"voxlang/internal/codegen"
	"voxlang/internal/diag"
	"voxlang/internal/interp"
	"voxlang/internal/irgen"
	"voxlang/internal/loader"
	"voxlang/internal/names"
	"voxlang/internal/typecheck"
)

func usage() {
	fmt.Fprintln(os.Stderr, "vox - stage0 prototype")
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  vox init [dir]")
	fmt.Fprintln(os.Stderr, "  vox ir [--engine=c|interp] [dir]")
	fmt.Fprintln(os.Stderr, "  vox c [dir]")
	fmt.Fprintln(os.Stderr, "  vox build [--engine=c|interp] [dir]")
	fmt.Fprintln(os.Stderr, "  vox run [--engine=c|interp] [dir]")
	fmt.Fprintln(os.Stderr, "  vox test [--engine=c|interp] [dir]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "flags:")
	fmt.Fprintln(os.Stderr, "  --engine=c|interp   execution engine (default: c)")
	fmt.Fprintln(os.Stderr, "  --compile           alias for --engine=c")
	fmt.Fprintln(os.Stderr, "  --interp            alias for --engine=interp")
	fmt.Fprintln(os.Stderr, "test flags:")
	fmt.Fprintln(os.Stderr, "  --run=<regex>       run tests whose qualified name (or short name) matches regex")
	fmt.Fprintln(os.Stderr, "  --rerun-failed      rerun only tests failed in previous run")
	fmt.Fprintln(os.Stderr, "  --list              list selected tests and exit without running")
}

type engine int

const (
	engineC engine = iota
	engineInterp
)

type testOptions struct {
	eng         engine
	dir         string
	runPattern  string
	rerunFailed bool
	listOnly    bool
}

func parseEngineAndDir(args []string) (eng engine, dir string, err error) {
	eng = engineC
	dir = "."
	setDir := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--compile" {
			eng = engineC
			continue
		}
		if a == "--interp" {
			eng = engineInterp
			continue
		}
		if a == "--engine" {
			if i+1 >= len(args) {
				return engineC, ".", fmt.Errorf("missing value for --engine")
			}
			i++
			a = "--engine=" + args[i]
		}
		if strings.HasPrefix(a, "--engine=") {
			v := strings.TrimPrefix(a, "--engine=")
			switch v {
			case "c":
				eng = engineC
			case "interp":
				eng = engineInterp
			default:
				return engineC, ".", fmt.Errorf("unknown engine: %q", v)
			}
			continue
		}
		if strings.HasPrefix(a, "-") {
			return engineC, ".", fmt.Errorf("unknown flag: %s", a)
		}
		if setDir {
			return engineC, ".", fmt.Errorf("unexpected extra arg: %s", a)
		}
		dir = a
		setDir = true
	}
	return eng, dir, nil
}

func parseTestOptionsAndDir(args []string) (opts testOptions, err error) {
	opts.eng = engineC
	opts.dir = "."
	setDir := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--compile" {
			opts.eng = engineC
			continue
		}
		if a == "--interp" {
			opts.eng = engineInterp
			continue
		}
		if a == "--engine" {
			if i+1 >= len(args) {
				return testOptions{}, fmt.Errorf("missing value for --engine")
			}
			i++
			a = "--engine=" + args[i]
		}
		if strings.HasPrefix(a, "--engine=") {
			v := strings.TrimPrefix(a, "--engine=")
			switch v {
			case "c":
				opts.eng = engineC
			case "interp":
				opts.eng = engineInterp
			default:
				return testOptions{}, fmt.Errorf("unknown engine: %q", v)
			}
			continue
		}
		if a == "--run" {
			if i+1 >= len(args) {
				return testOptions{}, fmt.Errorf("missing value for --run")
			}
			i++
			opts.runPattern = args[i]
			continue
		}
		if strings.HasPrefix(a, "--run=") {
			opts.runPattern = strings.TrimPrefix(a, "--run=")
			continue
		}
		if a == "--rerun-failed" {
			opts.rerunFailed = true
			continue
		}
		if a == "--list" {
			opts.listOnly = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			return testOptions{}, fmt.Errorf("unknown flag: %s", a)
		}
		if setDir {
			return testOptions{}, fmt.Errorf("unexpected extra arg: %s", a)
		}
		opts.dir = a
		setDir = true
	}
	return opts, nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := loader.InitPackage(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "ir":
		_, dir, err := parseEngineAndDir(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := dumpIR(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "c":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := dumpC(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "build":
		eng, dir, err := parseEngineAndDir(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := build(dir, eng); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "run":
		// Allow forwarding program args after `--`.
		raw := os.Args[2:]
		progArgs := []string{}
		for i := 0; i < len(raw); i++ {
			if raw[i] == "--" {
				progArgs = append(progArgs, raw[i+1:]...)
				raw = raw[:i]
				break
			}
		}
		eng, dir, err := parseEngineAndDir(raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := run(dir, eng, progArgs); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "test":
		opts, err := parseTestOptionsAndDir(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := testWithOptions(opts); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func dumpIR(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	res, diags, err := loader.BuildPackage(abs, false)
	if err != nil {
		return err
	}
	if diags != nil && len(diags.Items) > 0 {
		diag.Print(os.Stderr, diags)
		return fmt.Errorf("build failed")
	}
	irp, err := irgen.Generate(res.Program)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, irp.Format())
	return nil
}

func emitCForDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	res, diags, err := loader.BuildPackage(abs, false)
	if err != nil {
		return "", err
	}
	if diags != nil && len(diags.Items) > 0 {
		diag.Print(os.Stderr, diags)
		return "", fmt.Errorf("build failed")
	}
	irp, err := irgen.Generate(res.Program)
	if err != nil {
		return "", err
	}
	return codegen.EmitC(irp, codegen.EmitOptions{EmitDriverMain: true, DriverMainKind: codegen.DriverMainUser})
}

func dumpC(dir string) error {
	csrc, err := emitCForDir(dir)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, csrc)
	return nil
}

func build(dir string, eng engine) error {
	if eng == engineInterp {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		_, diags, err := loader.BuildPackage(abs, false)
		if err != nil {
			return err
		}
		if diags != nil && len(diags.Items) > 0 {
			diag.Print(os.Stderr, diags)
			return fmt.Errorf("build failed")
		}
		return nil
	}
	_, err := compile(dir)
	return err
}

func run(dir string, eng engine, progArgs []string) error {
	if eng == engineInterp {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		res, diags, err := loader.BuildPackage(abs, false)
		if err != nil {
			return err
		}
		if diags != nil && len(diags.Items) > 0 {
			diag.Print(os.Stderr, diags)
			return fmt.Errorf("build failed")
		}
		out, err := interp.RunMainWithArgs(res.Program, progArgs)
		if err != nil {
			return err
		}
		if out != "" {
			fmt.Fprintln(os.Stdout, out)
		}
		return nil
	}
	bin, err := compile(dir)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, progArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func test(dir string, eng engine) error {
	return testWithOptions(testOptions{eng: eng, dir: dir})
}

func testWithOptions(opts testOptions) error {
	runStart := time.Now()
	abs, err := filepath.Abs(opts.dir)
	if err != nil {
		return err
	}
	res, diags, err := loader.TestPackage(abs)
	if err != nil {
		return err
	}
	if diags != nil && len(diags.Items) > 0 {
		diag.Print(os.Stderr, diags)
		return fmt.Errorf("test failed")
	}
	if res == nil || res.Program == nil {
		return fmt.Errorf("internal error: missing checked program")
	}

	// Discover tests (Go-like): tests/**.vox and src/**/*_test.vox, functions named `test_*`.
	testNames := discoverTests(res.Program)
	discoveredTests := len(testNames)
	prevFailedCount := 0
	if opts.rerunFailed {
		prevFailed, err := readFailedTests(abs)
		if err != nil {
			return err
		}
		prevFailedCount = len(prevFailed)
		if len(prevFailed) == 0 {
			fmt.Fprintf(os.Stdout, "[test] no previous failed tests (%s)\n", failedTestsPath(abs))
			return nil
		}
		testNames = intersectTests(testNames, prevFailed)
	}
	if opts.runPattern != "" {
		testNames, err = filterTestsByPattern(testNames, opts.runPattern)
		if err != nil {
			return err
		}
	}
	printSelectionSummary(os.Stdout, discoveredTests, len(testNames), opts, prevFailedCount)
	if len(testNames) == 0 {
		fmt.Fprintln(os.Stdout, "[test] no tests found")
		fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		if err := writeFailedTests(abs, nil); err != nil {
			return err
		}
		return nil
	}
	if opts.listOnly {
		printSelectedTests(os.Stdout, testNames)
		fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		return nil
	}

	if opts.eng == engineInterp {
		log, failedNames, err := interp.RunTestsNamed(res.Program, testNames)
		if log != "" {
			fmt.Fprint(os.Stdout, log)
		}
		fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		if werr := writeFailedTests(abs, failedNames); werr != nil {
			return werr
		}
		if len(failedNames) != 0 {
			fmt.Fprintf(os.Stdout, "[hint] rerun failed: %s\n", testRerunCommand(opts.eng, abs))
		}
		return err
	}

	// Build a single test binary and run each test in a separate process so panics
	// can be attributed to the current test.
	bin, err := compileTests(abs, res, testNames)
	if err != nil {
		return err
	}

	groups := names.GroupQualifiedTestsByModule(testNames)
	results := runCompiledTestsByModule(bin, groups)

	passed := 0
	failed := 0
	failedNames := make([]string, 0)
	for _, name := range testNames {
		r, ok := results[name]
		if !ok {
			failed++
			failedNames = append(failedNames, name)
			fmt.Fprintf(os.Stdout, "[FAIL] %s (%s)\n", name, formatTestDuration(0))
			continue
		}
		if r.err != nil {
			failed++
			failedNames = append(failedNames, name)
			fmt.Fprintf(os.Stdout, "[FAIL] %s (%s)\n", name, formatTestDuration(r.dur))
			continue
		}
		passed++
		fmt.Fprintf(os.Stdout, "[OK] %s (%s)\n", name, formatTestDuration(r.dur))
	}
	moduleSummaries, slowest := summarizeTestResults(testNames, results)
	for _, ms := range moduleSummaries {
		fmt.Fprintf(os.Stdout, "[module] %s: %d passed, %d failed (%s)\n", displayModuleName(ms.module), ms.passed, ms.failed, formatTestDuration(ms.dur))
	}
	for _, s := range slowest {
		fmt.Fprintf(os.Stdout, "[slowest] %s (%s)\n", s.name, formatTestDuration(s.dur))
	}
	fmt.Fprintf(os.Stdout, "[test] %d passed, %d failed\n", passed, failed)
	fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
	if err := writeFailedTests(abs, failedNames); err != nil {
		return err
	}
	if failed != 0 {
		fmt.Fprintf(os.Stdout, "[hint] rerun failed: %s\n", testRerunCommand(opts.eng, abs))
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}

func testRerunCommand(eng engine, absDir string) string {
	engArg := "--engine=c"
	if eng == engineInterp {
		engArg = "--engine=interp"
	}
	return "vox test " + engArg + " --rerun-failed " + absDir
}

func printSelectedTests(out io.Writer, testNames []string) {
	groups := names.GroupQualifiedTestsByModule(testNames)
	for _, g := range groups {
		fmt.Fprintf(out, "[list] %s (%d)\n", displayModuleName(g.Key), len(g.Tests))
		for _, name := range g.Tests {
			fmt.Fprintf(out, "[test] %s\n", name)
		}
	}
	fmt.Fprintf(out, "[list] total: %d\n", len(testNames))
}

func printSelectionSummary(out io.Writer, discovered int, selected int, opts testOptions, prevFailedCount int) {
	fmt.Fprintf(out, "[select] discovered: %d, selected: %d\n", discovered, selected)
	if opts.runPattern != "" {
		fmt.Fprintf(out, "[select] --run: %q\n", opts.runPattern)
	}
	if opts.rerunFailed {
		fmt.Fprintf(out, "[select] --rerun-failed: %d cached\n", prevFailedCount)
	}
}

type testExecResult struct {
	dur time.Duration
	err error
}

type moduleTestSummary struct {
	module string
	passed int
	failed int
	dur    time.Duration
}

type namedTestDuration struct {
	name string
	dur  time.Duration
}

func moduleFromQualifiedTest(name string) string {
	i := strings.LastIndex(name, "::")
	if i < 0 {
		return ""
	}
	return name[:i]
}

func displayModuleName(module string) string {
	if module == "" {
		return "(root)"
	}
	return module
}

func summarizeTestResults(testNames []string, results map[string]testExecResult) ([]moduleTestSummary, []namedTestDuration) {
	modMap := map[string]moduleTestSummary{}
	slowest := make([]namedTestDuration, 0, len(testNames))
	for _, name := range testNames {
		mod := moduleFromQualifiedTest(name)
		sum, ok := modMap[mod]
		if !ok {
			sum = moduleTestSummary{module: mod}
		}
		r, rok := results[name]
		if !rok || r.err != nil {
			sum.failed++
			if rok {
				sum.dur += r.dur
				slowest = append(slowest, namedTestDuration{name: name, dur: r.dur})
			}
		} else {
			sum.passed++
			sum.dur += r.dur
			slowest = append(slowest, namedTestDuration{name: name, dur: r.dur})
		}
		modMap[mod] = sum
	}

	keys := make([]string, 0, len(modMap))
	for k := range modMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	mods := make([]moduleTestSummary, 0, len(keys))
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

func runCompiledTestsByModule(bin string, groups []names.TestModuleGroup) map[string]testExecResult {
	total := 0
	for _, g := range groups {
		total += len(g.Tests)
	}
	out := make(map[string]testExecResult, total)
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
		res  testExecResult
	}
	results := make(chan namedResult, total)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for g := range jobs {
				for _, name := range g.Tests {
					cmd := exec.Command(bin, name)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Stdin = os.Stdin
					start := time.Now()
					err := cmd.Run()
					results <- namedResult{name: name, res: testExecResult{dur: time.Since(start), err: err}}
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
	if us < 1000*1000 {
		return fmt.Sprintf("%.2fms", float64(us)/1000.0)
	}
	return fmt.Sprintf("%.2fs", float64(us)/1000.0/1000.0)
}

func compile(dir string) (string, error) {
	return compileWithDriver(dir, codegen.DriverMainUser)
}

func compileWithDriver(dir string, driver codegen.DriverMainKind) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	res, diags, err := loader.BuildPackage(abs, false)
	if err != nil {
		return "", err
	}
	if diags != nil && len(diags.Items) > 0 {
		diag.Print(os.Stderr, diags)
		return "", fmt.Errorf("build failed")
	}

	irp, err := irgen.Generate(res.Program)
	if err != nil {
		return "", err
	}
	csrc, err := codegen.EmitC(irp, codegen.EmitOptions{EmitDriverMain: true, DriverMainKind: driver})
	if err != nil {
		return "", err
	}

	outDir := filepath.Join(res.Root, "target", "debug")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	base := res.Manifest.Package.Name
	if base == "" {
		base = filepath.Base(res.Root)
	}
	irPath := filepath.Join(outDir, base+".ir")
	cPath := filepath.Join(outDir, base+".c")
	binPath := filepath.Join(outDir, base)

	if err := os.WriteFile(irPath, []byte(irp.Format()), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(cPath, []byte(csrc), 0o644); err != nil {
		return "", err
	}

	cc, err := exec.LookPath("cc")
	if err != nil {
		return "", fmt.Errorf("cc not found in PATH")
	}
	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return binPath, nil
}

func discoverTests(p *typecheck.CheckedProgram) []string {
	if p == nil || p.Prog == nil {
		return nil
	}
	var out []string
	for _, fn := range p.Prog.Funcs {
		if fn == nil || fn.Span.File == nil {
			continue
		}
		_, _, isTest := names.SplitOwnerAndModule(fn.Span.File.Name)
		if !isTest {
			continue
		}
		if strings.HasPrefix(fn.Name, "test_") {
			out = append(out, names.QualifyFunc(fn.Span.File.Name, fn.Name))
		}
	}
	sort.Strings(out)
	return out
}

func shortTestName(name string) string {
	i := strings.LastIndex(name, "::")
	if i < 0 || i+2 >= len(name) {
		return name
	}
	return name[i+2:]
}

func filterTestsByPattern(testNames []string, pattern string) ([]string, error) {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid --run regex: %w", err)
	}
	out := make([]string, 0, len(testNames))
	for _, name := range testNames {
		if rx.MatchString(name) || rx.MatchString(shortTestName(name)) {
			out = append(out, name)
		}
	}
	return out, nil
}

func failedTestsPath(root string) string {
	return filepath.Join(root, "target", "debug", ".vox_failed_tests")
}

func readFailedTests(root string) ([]string, error) {
	path := failedTestsPath(root)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if _, ok := seen[ln]; ok {
			continue
		}
		seen[ln] = struct{}{}
		out = append(out, ln)
	}
	return out, nil
}

func writeFailedTests(root string, failed []string) error {
	path := failedTestsPath(root)
	if len(failed) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, name := range failed {
		b.WriteString(name)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func intersectTests(all []string, keep []string) []string {
	ks := make(map[string]struct{}, len(keep))
	for _, name := range keep {
		ks[name] = struct{}{}
	}
	out := make([]string, 0, len(all))
	for _, name := range all {
		if _, ok := ks[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

func compileTests(dir string, res *loader.BuildResult, testNames []string) (string, error) {
	irp, err := irgen.Generate(res.Program)
	if err != nil {
		return "", err
	}
	tfs := make([]codegen.TestFunc, 0, len(testNames))
	for _, n := range testNames {
		tfs = append(tfs, codegen.TestFunc{Name: n})
	}
	csrc, err := codegen.EmitC(irp, codegen.EmitOptions{EmitTestMain: true, TestFuncs: tfs})
	if err != nil {
		return "", err
	}

	outDir := filepath.Join(res.Root, "target", "debug")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	base := res.Manifest.Package.Name
	if base == "" {
		base = filepath.Base(res.Root)
	}
	irPath := filepath.Join(outDir, base+".test.ir")
	cPath := filepath.Join(outDir, base+".test.c")
	binPath := filepath.Join(outDir, base+".test")

	if err := os.WriteFile(irPath, []byte(irp.Format()), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(cPath, []byte(csrc), 0o644); err != nil {
		return "", err
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		return "", fmt.Errorf("cc not found in PATH")
	}
	cmd := exec.Command(cc, "-std=c11", "-O0", "-g", cPath, "-o", binPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return binPath, nil
}
