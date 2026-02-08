package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
}

type engine int

const (
	engineC engine = iota
	engineInterp
)

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
		eng, dir, err := parseEngineAndDir(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := test(dir, eng); err != nil {
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
	abs, err := filepath.Abs(dir)
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

	if eng == engineInterp {
		log, err := interp.RunTests(res.Program)
		if log != "" {
			fmt.Fprint(os.Stdout, log)
		}
		return err
	}

	// Discover tests (Go-like): tests/**.vox and src/**/*_test.vox, functions named `test_*`.
	testNames := discoverTests(res.Program)
	if len(testNames) == 0 {
		fmt.Fprintln(os.Stdout, "[test] no tests found")
		return nil
	}

	// Build a single test binary and run each test in a separate process so panics
	// can be attributed to the current test.
	bin, err := compileTests(abs, res, testNames)
	if err != nil {
		return err
	}

	passed := 0
	failed := 0
	for _, name := range testNames {
		cmd := exec.Command(bin, name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			failed++
			fmt.Fprintf(os.Stdout, "[FAIL] %s\n", name)
			continue
		}
		passed++
		fmt.Fprintf(os.Stdout, "[OK] %s\n", name)
	}
	fmt.Fprintf(os.Stdout, "[test] %d passed, %d failed\n", passed, failed)
	if failed != 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
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
