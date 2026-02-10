package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"voxlang/internal/ast"
	"voxlang/internal/source"
)

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func TestParseTestOptionsAndDir_RunAndRerun(t *testing.T) {
	opts, err := parseTestOptionsAndDir([]string{"--engine=interp", "--run", "foo", "--rerun-failed", "pkg"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.eng != engineInterp {
		t.Fatalf("eng = %v, want %v", opts.eng, engineInterp)
	}
	if opts.runPattern != "foo" {
		t.Fatalf("runPattern = %q, want %q", opts.runPattern, "foo")
	}
	if !opts.rerunFailed {
		t.Fatalf("rerunFailed = false, want true")
	}
	if opts.dir != "pkg" {
		t.Fatalf("dir = %q, want %q", opts.dir, "pkg")
	}
	if opts.listOnly {
		t.Fatalf("listOnly = true, want false")
	}
}

func TestParseTestOptionsAndDir_List(t *testing.T) {
	opts, err := parseTestOptionsAndDir([]string{"--list", "."})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.listOnly {
		t.Fatalf("listOnly = false, want true")
	}
}

func TestParseTestOptionsAndDir_JSON(t *testing.T) {
	opts, err := parseTestOptionsAndDir([]string{"--json", "."})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.jsonOutput {
		t.Fatalf("jsonOutput = false, want true")
	}
}

func TestParseTestOptionsAndDir_InvalidRegexHandledLater(t *testing.T) {
	opts, err := parseTestOptionsAndDir([]string{"--run=([", "."})
	if err != nil {
		t.Fatal(err)
	}
	if opts.runPattern != "([" {
		t.Fatalf("runPattern = %q, want invalid regex to pass through", opts.runPattern)
	}
}

func TestFilterTestsByPattern(t *testing.T) {
	in := []string{
		"tests::test_alpha",
		"a.b::test_beta",
		"a.b::bench_gamma",
	}
	got, err := filterTestsByPattern(in, "test_(alpha|beta)")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2, got=%v", len(got), got)
	}
	if got[0] != "tests::test_alpha" || got[1] != "a.b::test_beta" {
		t.Fatalf("unexpected filter result: %v", got)
	}
}

func TestFailedTestsReadWrite(t *testing.T) {
	root := t.TempDir()
	want := []string{"tests::test_a", "tests::test_b"}
	if err := writeFailedTests(root, want); err != nil {
		t.Fatal(err)
	}
	got, err := readFailedTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := 0; i < len(want); i++ {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
	if err := writeFailedTests(root, nil); err != nil {
		t.Fatal(err)
	}
	got2, err := readFailedTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got2) != 0 {
		t.Fatalf("expected empty failed cache, got %v", got2)
	}
}

func TestInterpTestRerunFailed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "vox.toml"), []byte("[package]\nname = \"t\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.vox"), []byte("fn main() -> i32 { return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First run: one test fails, should populate failed cache.
	failSrc := "fn test_ok() -> () { }\nfn test_fail() -> () { panic(\"boom\"); }\n"
	if err := os.WriteFile(filepath.Join(root, "tests", "basic.vox"), []byte(failSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	err := testWithOptions(testOptions{eng: engineInterp, dir: root})
	if err == nil {
		t.Fatalf("expected failing test run")
	}
	failed, err := readFailedTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(failed, "tests::test_fail") {
		t.Fatalf("expected failed cache to include tests::test_fail, got %v", failed)
	}

	// Second run: fix failed test, rerun only failed set, should pass and clear cache.
	passSrc := "fn test_ok() -> () { }\nfn test_fail() -> () { }\n"
	if err := os.WriteFile(filepath.Join(root, "tests", "basic.vox"), []byte(passSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := testWithOptions(testOptions{eng: engineInterp, dir: root, rerunFailed: true}); err != nil {
		t.Fatalf("rerun failed tests should pass, got: %v", err)
	}
	failed2, err := readFailedTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(failed2) != 0 {
		t.Fatalf("expected failed cache cleared, got %v", failed2)
	}
}

func TestSummarizeTestResults_ModuleAndSlowest(t *testing.T) {
	testNames := []string{
		"mod.a::test_fast",
		"mod.a::test_slow",
		"mod.b::test_fail",
	}
	results := map[string]testExecResult{
		"mod.a::test_fast": {dur: 1 * time.Millisecond, err: nil},
		"mod.a::test_slow": {dur: 4 * time.Millisecond, err: nil},
		"mod.b::test_fail": {dur: 2 * time.Millisecond, err: os.ErrInvalid},
	}
	mods, slowest := summarizeTestResults(testNames, results)
	if len(mods) != 2 {
		t.Fatalf("module count = %d, want 2", len(mods))
	}
	if mods[0].module != "mod.a" || mods[0].passed != 2 || mods[0].failed != 0 {
		t.Fatalf("unexpected module summary[0]: %+v", mods[0])
	}
	if mods[1].module != "mod.b" || mods[1].passed != 0 || mods[1].failed != 1 {
		t.Fatalf("unexpected module summary[1]: %+v", mods[1])
	}
	if len(slowest) != 3 {
		t.Fatalf("slowest count = %d, want 3", len(slowest))
	}
	if slowest[0].name != "mod.a::test_slow" {
		t.Fatalf("slowest[0] = %q, want mod.a::test_slow", slowest[0].name)
	}
	if slowest[1].name != "mod.b::test_fail" {
		t.Fatalf("slowest[1] = %q, want mod.b::test_fail", slowest[1].name)
	}
	if slowest[2].name != "mod.a::test_fast" {
		t.Fatalf("slowest[2] = %q, want mod.a::test_fast", slowest[2].name)
	}
}

func TestTestRerunCommand(t *testing.T) {
	gotC := testRerunCommand(engineC, "/tmp/p")
	if gotC != "vox test --engine=c --rerun-failed /tmp/p" {
		t.Fatalf("unexpected c rerun command: %q", gotC)
	}
	gotInterp := testRerunCommand(engineInterp, "/tmp/p")
	if gotInterp != "vox test --engine=interp --rerun-failed /tmp/p" {
		t.Fatalf("unexpected interp rerun command: %q", gotInterp)
	}
}

func TestPrintSelectedTests(t *testing.T) {
	var b bytes.Buffer
	printSelectedTests(&b, []string{
		"mod.a::test_one",
		"mod.a::test_two",
		"mod.b::test_ok",
	})
	out := b.String()
	if !strings.Contains(out, "[list] mod.a (2)") {
		t.Fatalf("missing mod.a summary: %q", out)
	}
	if !strings.Contains(out, "[test] mod.a::test_one") {
		t.Fatalf("missing test line: %q", out)
	}
	if !strings.Contains(out, "[list] total: 3") {
		t.Fatalf("missing total summary: %q", out)
	}
}

func TestPrintSelectionSummary(t *testing.T) {
	var b bytes.Buffer
	printSelectionSummary(&b, 7, 2, testOptions{runPattern: "foo", rerunFailed: true}, 3)
	out := b.String()
	if !strings.Contains(out, "[select] discovered: 7, selected: 2") {
		t.Fatalf("missing discover/selected summary: %q", out)
	}
	if !strings.Contains(out, "[select] --run: \"foo\"") {
		t.Fatalf("missing run summary: %q", out)
	}
	if !strings.Contains(out, "[select] --rerun-failed: 3 cached") {
		t.Fatalf("missing rerun summary: %q", out)
	}
}

func TestParseDirArg(t *testing.T) {
	dir, err := parseDirArg(nil)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "." {
		t.Fatalf("dir = %q, want %q", dir, ".")
	}
	dir, err = parseDirArg([]string{"pkg"})
	if err != nil {
		t.Fatal(err)
	}
	if dir != "pkg" {
		t.Fatalf("dir = %q, want %q", dir, "pkg")
	}
	if _, err := parseDirArg([]string{"a", "b"}); err == nil {
		t.Fatalf("expected parseDirArg to reject extra args")
	}
}

func TestFormatVoxSource(t *testing.T) {
	in := "fn main() -> i32 {  \r\n  return 0; \t\r\n}\r\n\r\n"
	got := formatVoxSource(in)
	want := "fn main() -> i32 {\n  return 0;\n}\n"
	if got != want {
		t.Fatalf("formatted text mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRunFmt(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "src", "main.vox")
	if err := os.WriteFile(p, []byte("fn main() -> i32 {  \n  return 0; \t\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runFmt(root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "fn main() -> i32 {\n  return 0;\n}\n" {
		t.Fatalf("unexpected formatted file:\n%s", string(b))
	}
}

func TestRenderAPIDoc(t *testing.T) {
	fRoot := source.NewFile("src/main.vox", "")
	fMod := source.NewFile("src/math/add.vox", "")
	prog := &ast.Program{
		Funcs: []*ast.FuncDecl{
			{Pub: true, Name: "main", Ret: &ast.NamedType{Parts: []string{"i32"}}, Span: source.Span{File: fRoot}},
			{Pub: true, Name: "add", Params: []ast.Param{{Name: "a", Type: &ast.NamedType{Parts: []string{"i32"}}}, {Name: "b", Type: &ast.NamedType{Parts: []string{"i32"}}}}, Ret: &ast.NamedType{Parts: []string{"i32"}}, Span: source.Span{File: fMod}},
		},
		Structs: []*ast.StructDecl{
			{Pub: true, Name: "Point", TypeParams: []string{"T"}, Span: source.Span{File: fMod}},
		},
	}
	md := renderAPIDoc("demo", prog)
	if !strings.Contains(md, "# API: demo") {
		t.Fatalf("missing doc title: %q", md)
	}
	if !strings.Contains(md, "## Module (root)") {
		t.Fatalf("missing root module section: %q", md)
	}
	if !strings.Contains(md, "## Module math") {
		t.Fatalf("missing module section: %q", md)
	}
	if !strings.Contains(md, "`fn add(a: i32, b: i32) -> i32`") {
		t.Fatalf("missing function signature: %q", md)
	}
	if !strings.Contains(md, "`struct Point[T]`") {
		t.Fatalf("missing struct line: %q", md)
	}
}

func TestBuildJSONTestReport(t *testing.T) {
	testNames := []string{"mod.a::test_ok", "mod.a::test_fail"}
	results := map[string]testExecResult{
		"mod.a::test_ok":   {dur: 2 * time.Millisecond, err: nil},
		"mod.a::test_fail": {dur: 3 * time.Millisecond, err: os.ErrInvalid},
	}
	mods, slowest := summarizeTestResults(testNames, results)
	rep := buildJSONTestReport(testOptions{eng: engineC, dir: "."}, "/tmp/p", 2, 2, 0, testNames, results, mods, "rerun", "")
	rep.Slowest = jsonSlowestFromNamed(slowest)
	rep.TotalDurationMicros = 5000

	var b bytes.Buffer
	if err := emitJSONReport(&b, rep); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["engine"] != "c" {
		t.Fatalf("engine = %v, want c", got["engine"])
	}
	if got["passed"] != float64(1) || got["failed"] != float64(1) {
		t.Fatalf("unexpected pass/fail: %+v", got)
	}
	failedTests, ok := got["failed_tests"].([]any)
	if !ok {
		t.Fatalf("missing failed_tests: %+v", got)
	}
	if len(failedTests) != 1 || failedTests[0] != "mod.a::test_fail" {
		t.Fatalf("failed_tests = %+v, want [mod.a::test_fail]", failedTests)
	}
	moduleDetails, ok := got["module_details"].([]any)
	if !ok || len(moduleDetails) != 1 {
		t.Fatalf("module_details = %+v, want len 1", got["module_details"])
	}
	md0, ok := moduleDetails[0].(map[string]any)
	if !ok {
		t.Fatalf("module_details[0] not object: %+v", moduleDetails[0])
	}
	if md0["module"] != "mod.a" {
		t.Fatalf("module_details[0].module = %v, want mod.a", md0["module"])
	}
	testsAny, ok := md0["tests"].([]any)
	if !ok || len(testsAny) != 2 {
		t.Fatalf("module_details[0].tests = %+v, want len 2", md0["tests"])
	}
	failedAny, ok := md0["failed_tests"].([]any)
	if !ok || len(failedAny) != 1 || failedAny[0] != "mod.a::test_fail" {
		t.Fatalf("module_details[0].failed_tests = %+v, want [mod.a::test_fail]", md0["failed_tests"])
	}
}

func TestBuildJSONTestReport_ListOnlyIncludesModuleDetails(t *testing.T) {
	testNames := []string{"mod.a::test_a", "mod.b::test_b", "mod.b::test_c"}
	rep := buildJSONTestReport(
		testOptions{eng: engineInterp, dir: ".", listOnly: true},
		"/tmp/p",
		3,
		3,
		0,
		testNames,
		nil,
		nil,
		"",
		"",
	)
	if len(rep.ModuleDetails) != 2 {
		t.Fatalf("module_details len = %d, want 2", len(rep.ModuleDetails))
	}
	if rep.ModuleDetails[0].Module != "mod.a" || len(rep.ModuleDetails[0].Tests) != 1 {
		t.Fatalf("unexpected module_details[0]: %+v", rep.ModuleDetails[0])
	}
	if rep.ModuleDetails[1].Module != "mod.b" || len(rep.ModuleDetails[1].Tests) != 2 {
		t.Fatalf("unexpected module_details[1]: %+v", rep.ModuleDetails[1])
	}
	if len(rep.FailedTests) != 0 {
		t.Fatalf("failed_tests len = %d, want 0", len(rep.FailedTests))
	}
}
