package main

import (
	"os"
	"path/filepath"
	"testing"
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
