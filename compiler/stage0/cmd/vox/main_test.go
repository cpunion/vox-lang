package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
