package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"unicode/utf8"

	"sort"
	"strings"
	"sync"
	"time"
	"voxlang/internal/ast"

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
	fmt.Fprintln(os.Stderr, "  vox fmt [dir]")
	fmt.Fprintln(os.Stderr, "  vox lint [dir]")
	fmt.Fprintln(os.Stderr, "  vox doc [dir]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "flags:")
	fmt.Fprintln(os.Stderr, "  --engine=c|interp   execution engine (default: c)")
	fmt.Fprintln(os.Stderr, "  --compile           alias for --engine=c")
	fmt.Fprintln(os.Stderr, "  --interp            alias for --engine=interp")
	fmt.Fprintln(os.Stderr, "test flags:")
	fmt.Fprintln(os.Stderr, "  --run=<regex>       run tests whose qualified name (or short name) matches regex")
	fmt.Fprintln(os.Stderr, "  --filter=<text>     run tests whose qualified name (or short name) contains text")
	fmt.Fprintln(os.Stderr, "  --jobs=N, -j N      module-level test parallelism (default: GOMAXPROCS)")
	fmt.Fprintln(os.Stderr, "  --rerun-failed      rerun only tests failed in previous run")
	fmt.Fprintln(os.Stderr, "  --list              list selected tests and exit without running")
	fmt.Fprintln(os.Stderr, "  --json              emit machine-readable JSON report")
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
	filterText  string
	jobs        int
	rerunFailed bool
	listOnly    bool
	jsonOutput  bool
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
		if a == "--filter" {
			if i+1 >= len(args) {
				return testOptions{}, fmt.Errorf("missing value for --filter")
			}
			i++
			opts.filterText = args[i]
			continue
		}
		if strings.HasPrefix(a, "--filter=") {
			opts.filterText = strings.TrimPrefix(a, "--filter=")
			continue
		}
		if a == "-j" || a == "--jobs" {
			if i+1 >= len(args) {
				return testOptions{}, fmt.Errorf("missing value for --jobs")
			}
			i++
			n, err := parsePositiveInt(args[i])
			if err != nil {
				return testOptions{}, fmt.Errorf("invalid --jobs value: %q", args[i])
			}
			opts.jobs = n
			continue
		}
		if strings.HasPrefix(a, "--jobs=") {
			v := strings.TrimPrefix(a, "--jobs=")
			n, err := parsePositiveInt(v)
			if err != nil {
				return testOptions{}, fmt.Errorf("invalid --jobs value: %q", v)
			}
			opts.jobs = n
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
		if a == "--json" {
			opts.jsonOutput = true
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

func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < '0' || b > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int(b-'0')
		if n <= 0 {
			return 0, fmt.Errorf("overflow")
		}
	}
	if n <= 0 {
		return 0, fmt.Errorf("non-positive")
	}
	return n, nil
}

func parseDirArg(args []string) (string, error) {
	switch len(args) {
	case 0:
		return ".", nil
	case 1:
		if strings.HasPrefix(args[0], "-") {
			return "", fmt.Errorf("unknown flag: %s", args[0])
		}
		return args[0], nil
	default:
		return "", fmt.Errorf("unexpected extra arg: %s", args[1])
	}
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
	case "fmt":
		dir, err := parseDirArg(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := runFmt(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "lint":
		dir, err := parseDirArg(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := runLint(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "doc":
		dir, err := parseDirArg(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := runDoc(dir); err != nil {
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
			if opts.jsonOutput {
				rep := buildJSONTestReport(opts, abs, discoveredTests, 0, prevFailedCount, nil, nil, nil, "", "no previous failed tests")
				if err := emitJSONReport(os.Stdout, rep); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(os.Stdout, "[test] no previous failed tests (%s)\n", failedTestsPath(abs))
			}
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
	if opts.filterText != "" {
		testNames = filterTestsBySubstring(testNames, opts.filterText)
	}
	if !opts.jsonOutput {
		printSelectionSummary(os.Stdout, discoveredTests, len(testNames), opts, prevFailedCount)
	}
	if len(testNames) == 0 {
		if opts.jsonOutput {
			rep := buildJSONTestReport(opts, abs, discoveredTests, 0, prevFailedCount, testNames, nil, nil, "", "no tests found")
			rep.TotalDurationMicros = time.Since(runStart).Microseconds()
			if err := emitJSONReport(os.Stdout, rep); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(os.Stdout, "[test] no tests found")
			fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		}
		if err := writeFailedTests(abs, nil); err != nil {
			return err
		}
		return nil
	}
	if opts.listOnly {
		if opts.jsonOutput {
			rep := buildJSONTestReport(opts, abs, discoveredTests, len(testNames), prevFailedCount, testNames, nil, nil, "", "")
			rep.TotalDurationMicros = time.Since(runStart).Microseconds()
			if err := emitJSONReport(os.Stdout, rep); err != nil {
				return err
			}
		} else {
			printSelectedTests(os.Stdout, testNames)
			fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		}
		return nil
	}

	if opts.eng == engineInterp {
		repI, err := interp.RunTestsNamedDetailedWithJobs(res.Program, testNames, opts.jobs)
		if !opts.jsonOutput {
			if repI.Log != "" {
				fmt.Fprint(os.Stdout, repI.Log)
			}
			fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
		}
		if werr := writeFailedTests(abs, repI.FailedNames); werr != nil {
			return werr
		}
		hint := ""
		if len(repI.FailedNames) != 0 {
			hint = testRerunCommand(opts.eng, abs)
		}
		if opts.jsonOutput {
			results := interpResultsToExecResults(repI.Results)
			moduleSummaries := interpModuleSummariesToTestSummaries(repI.ModuleSummaries)
			slowest := interpSlowestToNamedDurations(repI.Slowest)
			rep := buildJSONTestReport(opts, abs, discoveredTests, len(testNames), prevFailedCount, testNames, results, moduleSummaries, hint, "")
			rep.Slowest = jsonSlowestFromNamed(slowest)
			rep.TotalDurationMicros = time.Since(runStart).Microseconds()
			if err := emitJSONReport(os.Stdout, rep); err != nil {
				return err
			}
		} else if len(repI.FailedNames) != 0 {
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
	results := runCompiledTestsByModule(bin, groups, opts.jobs)

	passed := 0
	failed := 0
	failedNames := make([]string, 0)
	for _, name := range testNames {
		r, ok := results[name]
		if !ok {
			failed++
			failedNames = append(failedNames, name)
			if !opts.jsonOutput {
				fmt.Fprintf(os.Stdout, "[FAIL] %s (%s)\n", name, formatTestDuration(0))
			}
			continue
		}
		if r.err != nil {
			failed++
			failedNames = append(failedNames, name)
			if !opts.jsonOutput {
				fmt.Fprintf(os.Stdout, "[FAIL] %s (%s)\n", name, formatTestDuration(r.dur))
			}
			continue
		}
		passed++
		if !opts.jsonOutput {
			fmt.Fprintf(os.Stdout, "[OK] %s (%s)\n", name, formatTestDuration(r.dur))
		}
	}
	moduleSummaries, slowest := summarizeTestResults(testNames, results)
	if !opts.jsonOutput {
		for _, ms := range moduleSummaries {
			fmt.Fprintf(os.Stdout, "[module] %s: %d passed, %d failed (%s)\n", displayModuleName(ms.module), ms.passed, ms.failed, formatTestDuration(ms.dur))
		}
		for _, s := range slowest {
			fmt.Fprintf(os.Stdout, "[slowest] %s (%s)\n", s.name, formatTestDuration(s.dur))
		}
		fmt.Fprintf(os.Stdout, "[test] %d passed, %d failed\n", passed, failed)
		fmt.Fprintf(os.Stdout, "[time] total: %s\n", formatTestDuration(time.Since(runStart)))
	}
	if err := writeFailedTests(abs, failedNames); err != nil {
		return err
	}
	hint := ""
	if failed != 0 {
		hint = testRerunCommand(opts.eng, abs)
		if !opts.jsonOutput {
			fmt.Fprintf(os.Stdout, "[hint] rerun failed: %s\n", hint)
		}
	}
	if opts.jsonOutput {
		rep := buildJSONTestReport(opts, abs, discoveredTests, len(testNames), prevFailedCount, testNames, results, moduleSummaries, hint, "")
		rep.Slowest = jsonSlowestFromNamed(slowest)
		rep.TotalDurationMicros = time.Since(runStart).Microseconds()
		if err := emitJSONReport(os.Stdout, rep); err != nil {
			return err
		}
	}
	if failed != 0 {
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
	if opts.filterText != "" {
		fmt.Fprintf(out, "[select] --filter: %q\n", opts.filterText)
	}
	if opts.jobs > 0 {
		fmt.Fprintf(out, "[select] --jobs: %d\n", opts.jobs)
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

type jsonSelection struct {
	Discovered      int    `json:"discovered"`
	Selected        int    `json:"selected"`
	RunPattern      string `json:"run_pattern,omitempty"`
	FilterText      string `json:"filter,omitempty"`
	Jobs            int    `json:"jobs,omitempty"`
	RerunFailed     bool   `json:"rerun_failed,omitempty"`
	PrevFailedCount int    `json:"prev_failed_count,omitempty"`
}

type jsonTestResult struct {
	Name           string `json:"name"`
	Module         string `json:"module"`
	Status         string `json:"status"`
	DurationMicros int64  `json:"duration_us"`
	Error          string `json:"error,omitempty"`
}

type jsonModuleSummary struct {
	Module         string `json:"module"`
	Passed         int    `json:"passed"`
	Failed         int    `json:"failed"`
	DurationMicros int64  `json:"duration_us"`
}

type jsonModuleDetail struct {
	Module      string   `json:"module"`
	Tests       []string `json:"tests"`
	FailedTests []string `json:"failed_tests,omitempty"`
	Passed      int      `json:"passed"`
	Failed      int      `json:"failed"`
}

type jsonSlowest struct {
	Name           string `json:"name"`
	DurationMicros int64  `json:"duration_us"`
}

type jsonTestReport struct {
	Engine              string              `json:"engine"`
	Dir                 string              `json:"dir"`
	Selection           jsonSelection       `json:"selection"`
	ListOnly            bool                `json:"list_only,omitempty"`
	SelectedTests       []string            `json:"selected_tests,omitempty"`
	Results             []jsonTestResult    `json:"results,omitempty"`
	Modules             []jsonModuleSummary `json:"modules,omitempty"`
	ModuleDetails       []jsonModuleDetail  `json:"module_details,omitempty"`
	FailedTests         []string            `json:"failed_tests,omitempty"`
	Slowest             []jsonSlowest       `json:"slowest,omitempty"`
	Passed              int                 `json:"passed"`
	Failed              int                 `json:"failed"`
	TotalDurationMicros int64               `json:"total_duration_us"`
	Hint                string              `json:"hint,omitempty"`
	Message             string              `json:"message,omitempty"`
}

func emitJSONReport(out io.Writer, rep jsonTestReport) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(rep)
}

func interpResultsToExecResults(results map[string]interp.TestResult) map[string]testExecResult {
	out := make(map[string]testExecResult, len(results))
	for name, r := range results {
		out[name] = testExecResult{dur: r.Dur, err: r.Err}
	}
	return out
}

func interpModuleSummariesToTestSummaries(summaries []interp.ModuleSummary) []moduleTestSummary {
	out := make([]moduleTestSummary, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, moduleTestSummary{
			module: s.Module,
			passed: s.Passed,
			failed: s.Failed,
			dur:    s.Dur,
		})
	}
	return out
}

func interpSlowestToNamedDurations(slowest []interp.NamedDuration) []namedTestDuration {
	out := make([]namedTestDuration, 0, len(slowest))
	for _, s := range slowest {
		out = append(out, namedTestDuration{name: s.Name, dur: s.Dur})
	}
	return out
}

func jsonSlowestFromNamed(slowest []namedTestDuration) []jsonSlowest {
	out := make([]jsonSlowest, 0, len(slowest))
	for _, s := range slowest {
		out = append(out, jsonSlowest{Name: s.name, DurationMicros: s.dur.Microseconds()})
	}
	return out
}

func buildJSONModuleDetails(testNames []string, results map[string]testExecResult) []jsonModuleDetail {
	if len(testNames) == 0 {
		return nil
	}
	detailMap := make(map[string]*jsonModuleDetail, len(testNames))
	for _, name := range testNames {
		mod := moduleFromQualifiedTest(name)
		d, ok := detailMap[mod]
		if !ok {
			d = &jsonModuleDetail{
				Module: displayModuleName(mod),
				Tests:  make([]string, 0, 1),
			}
			detailMap[mod] = d
		}
		d.Tests = append(d.Tests, name)
		if results == nil {
			continue
		}
		r, rok := results[name]
		if !rok || r.err != nil {
			d.Failed++
			d.FailedTests = append(d.FailedTests, name)
		} else {
			d.Passed++
		}
	}
	keys := make([]string, 0, len(detailMap))
	for k := range detailMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]jsonModuleDetail, 0, len(keys))
	for _, k := range keys {
		out = append(out, *detailMap[k])
	}
	return out
}

func buildJSONTestReport(
	opts testOptions,
	abs string,
	discovered int,
	selected int,
	prevFailedCount int,
	testNames []string,
	results map[string]testExecResult,
	moduleSummaries []moduleTestSummary,
	hint string,
	message string,
) jsonTestReport {
	engineName := "c"
	if opts.eng == engineInterp {
		engineName = "interp"
	}
	rep := jsonTestReport{
		Engine: engineName,
		Dir:    abs,
		Selection: jsonSelection{
			Discovered: discovered,
			Selected:   selected,
		},
		ListOnly:      opts.listOnly,
		SelectedTests: append([]string{}, testNames...),
		Hint:          hint,
		Message:       message,
	}
	rep.ModuleDetails = buildJSONModuleDetails(testNames, results)
	if opts.runPattern != "" {
		rep.Selection.RunPattern = opts.runPattern
	}
	if opts.filterText != "" {
		rep.Selection.FilterText = opts.filterText
	}
	if opts.jobs > 0 {
		rep.Selection.Jobs = opts.jobs
	}
	if opts.rerunFailed {
		rep.Selection.RerunFailed = true
		rep.Selection.PrevFailedCount = prevFailedCount
	}
	if results != nil {
		rep.Results = make([]jsonTestResult, 0, len(testNames))
		for _, n := range testNames {
			r, ok := results[n]
			j := jsonTestResult{Name: n, Module: displayModuleName(moduleFromQualifiedTest(n)), DurationMicros: 0}
			if !ok {
				j.Status = "fail"
				j.Error = "missing result"
				rep.FailedTests = append(rep.FailedTests, n)
				rep.Failed++
			} else if r.err != nil {
				j.Status = "fail"
				j.DurationMicros = r.dur.Microseconds()
				j.Error = r.err.Error()
				rep.FailedTests = append(rep.FailedTests, n)
				rep.Failed++
			} else {
				j.Status = "pass"
				j.DurationMicros = r.dur.Microseconds()
				rep.Passed++
			}
			rep.Results = append(rep.Results, j)
		}
	}
	if moduleSummaries != nil {
		rep.Modules = make([]jsonModuleSummary, 0, len(moduleSummaries))
		for _, ms := range moduleSummaries {
			rep.Modules = append(rep.Modules, jsonModuleSummary{
				Module:         displayModuleName(ms.module),
				Passed:         ms.passed,
				Failed:         ms.failed,
				DurationMicros: ms.dur.Microseconds(),
			})
		}
	}
	return rep
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

func runCompiledTestsByModule(bin string, groups []names.TestModuleGroup, jobs int) map[string]testExecResult {
	total := 0
	for _, g := range groups {
		total += len(g.Tests)
	}
	out := make(map[string]testExecResult, total)
	if total == 0 {
		return out
	}

	workers := moduleTestWorkers(len(groups), jobs)

	workq := make(chan names.TestModuleGroup, len(groups))
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
			for g := range workq {
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

func moduleTestWorkers(moduleCount int, jobs int) int {
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
	if us < 1000*1000 {
		return fmt.Sprintf("%.2fms", float64(us)/1000.0)
	}
	return fmt.Sprintf("%.2fs", float64(us)/1000.0/1000.0)
}

func collectPackageVoxFiles(root string, includeTests bool) ([]string, error) {
	srcDir := filepath.Join(root, "src")
	if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("missing src/ in %s", root)
	}
	dirs := []string{srcDir}
	if includeTests {
		testsDir := filepath.Join(root, "tests")
		if st, err := os.Stat(testsDir); err == nil && st.IsDir() {
			dirs = append(dirs, testsDir)
		}
	}
	var out []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".vox" {
				return nil
			}
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func formatVoxSource(src string) string {
	s := strings.ReplaceAll(src, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	out := strings.Join(lines, "\n")
	out = strings.TrimRight(out, "\n")
	return out + "\n"
}

func runFmt(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	files, err := collectPackageVoxFiles(abs, true)
	if err != nil {
		return err
	}
	changed := 0
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted := formatVoxSource(string(b))
		if string(b) == formatted {
			continue
		}
		if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
			return err
		}
		changed++
	}
	fmt.Fprintf(os.Stdout, "[fmt] files: %d, changed: %d\n", len(files), changed)
	return nil
}

func runLint(dir string) error {
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
		return fmt.Errorf("lint failed")
	}

	files, err := collectPackageVoxFiles(abs, true)
	if err != nil {
		return err
	}
	warnCount := 0
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		line := 1
		for _, ln := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			if utf8.RuneCountInString(ln) > 160 {
				fmt.Fprintf(os.Stdout, "%s:%d: warning: line too long (>160 runes)\n", path, line)
				warnCount++
			}
			line++
		}
	}
	fmt.Fprintf(os.Stdout, "[lint] ok (%d warning(s))\n", warnCount)
	return nil
}

func docModuleForFile(fileName string) (string, bool) {
	if names.PackageFromFileName(fileName) != "" {
		return "", false
	}
	_, mod, _ := names.SplitOwnerAndModule(fileName)
	if len(mod) == 0 {
		return "(root)", true
	}
	if len(mod) > 0 && mod[0] == "tests" {
		return "", false
	}
	return strings.Join(mod, "."), true
}

func docTypeString(t ast.Type) string {
	switch x := t.(type) {
	case *ast.NamedType:
		base := strings.Join(x.Parts, ".")
		if len(x.Args) == 0 {
			return base
		}
		args := make([]string, 0, len(x.Args))
		for _, a := range x.Args {
			args = append(args, docTypeString(a))
		}
		return base + "[" + strings.Join(args, ", ") + "]"
	case *ast.UnitType:
		return "()"
	case *ast.RangeType:
		return fmt.Sprintf("@range(%d..=%d) %s", x.Lo, x.Hi, docTypeString(x.Base))
	default:
		return "<type>"
	}
}

func docTypeParams(tps []string) string {
	if len(tps) == 0 {
		return ""
	}
	return "[" + strings.Join(tps, ", ") + "]"
}

func docFuncSig(fn *ast.FuncDecl) string {
	parts := make([]string, 0, len(fn.Params))
	for _, p := range fn.Params {
		parts = append(parts, p.Name+": "+docTypeString(p.Type))
	}
	return "fn " + fn.Name + docTypeParams(fn.TypeParams) + "(" + strings.Join(parts, ", ") + ") -> " + docTypeString(fn.Ret)
}

func renderAPIDoc(pkg string, p *ast.Program) string {
	if pkg == "" {
		pkg = "app"
	}
	var b strings.Builder
	b.WriteString("# API: " + pkg + "\n\n")
	if p == nil {
		b.WriteString("_No API symbols found._\n")
		return b.String()
	}

	byModule := map[string][]string{}
	add := func(fileName string, line string) {
		mod, ok := docModuleForFile(fileName)
		if !ok {
			return
		}
		byModule[mod] = append(byModule[mod], "- `"+line+"`")
	}

	for _, td := range p.Types {
		if td != nil && td.Pub {
			add(td.Span.File.Name, "type "+td.Name+" = "+docTypeString(td.Type))
		}
	}
	for _, cd := range p.Consts {
		if cd != nil && cd.Pub {
			add(cd.Span.File.Name, "const "+cd.Name+": "+docTypeString(cd.Type))
		}
	}
	for _, sd := range p.Structs {
		if sd != nil && sd.Pub {
			add(sd.Span.File.Name, "struct "+sd.Name+docTypeParams(sd.TypeParams))
		}
	}
	for _, ed := range p.Enums {
		if ed != nil && ed.Pub {
			add(ed.Span.File.Name, "enum "+ed.Name+docTypeParams(ed.TypeParams))
		}
	}
	for _, td := range p.Traits {
		if td != nil && td.Pub {
			add(td.Span.File.Name, "trait "+td.Name)
		}
	}
	for _, fn := range p.Funcs {
		if fn != nil && fn.Pub {
			add(fn.Span.File.Name, docFuncSig(fn))
		}
	}

	if len(byModule) == 0 {
		b.WriteString("_No public symbols found._\n")
		return b.String()
	}

	mods := make([]string, 0, len(byModule))
	for mod := range byModule {
		mods = append(mods, mod)
	}
	sort.Strings(mods)
	for _, mod := range mods {
		b.WriteString("## Module " + mod + "\n\n")
		lines := byModule[mod]
		sort.Strings(lines)
		for _, ln := range lines {
			b.WriteString(ln + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func runDoc(dir string) error {
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
		return fmt.Errorf("doc failed")
	}
	md := renderAPIDoc(res.Manifest.Package.Name, res.Program.Prog)
	outDir := filepath.Join(res.Root, "target", "doc")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, "API.md")
	if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "[doc] wrote %s\n", outPath)
	return nil
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
	ccArgs := []string{"-std=c11", "-O0", "-g", cPath, "-o", binPath}
	if runtime.GOOS == "windows" {
		ccArgs = append(ccArgs, "-lws2_32")
	}
	cmd := exec.Command(cc, ccArgs...)
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

func filterTestsBySubstring(testNames []string, needle string) []string {
	if needle == "" {
		return append([]string{}, testNames...)
	}
	out := make([]string, 0, len(testNames))
	for _, name := range testNames {
		short := shortTestName(name)
		if strings.Contains(name, needle) || strings.Contains(short, needle) {
			out = append(out, name)
		}
	}
	return out
}

func failedTestsPath(root string) string {
	return filepath.Join(root, "target", "debug", ".vox_failed_tests")
}

type failedTestsCache struct {
	Version         int      `json:"version"`
	UpdatedUnixUsec int64    `json:"updated_unix_us"`
	Tests           []string `json:"tests"`
}

func normalizeFailedTests(names []string) []string {
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
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
	text := strings.TrimSpace(string(b))
	if text == "" {
		return nil, nil
	}
	// v1 json cache; fallback to legacy plaintext list for compatibility.
	var cache failedTestsCache
	if err := json.Unmarshal(b, &cache); err == nil {
		return normalizeFailedTests(cache.Tests), nil
	}
	return normalizeFailedTests(strings.Split(text, "\n")), nil
}

func writeFailedTests(root string, failed []string) error {
	failed = normalizeFailedTests(failed)
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
	cache := failedTestsCache{
		Version:         1,
		UpdatedUnixUsec: time.Now().UnixMicro(),
		Tests:           failed,
	}
	buf, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
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
	ccArgs := []string{"-std=c11", "-O0", "-g", cPath, "-o", binPath}
	if runtime.GOOS == "windows" {
		ccArgs = append(ccArgs, "-lws2_32")
	}
	cmd := exec.Command(cc, ccArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return binPath, nil
}
