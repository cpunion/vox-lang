package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"voxlang/internal/codegen"
	"voxlang/internal/diag"
	"voxlang/internal/irgen"
	"voxlang/internal/loader"
)

func usage() {
	fmt.Fprintln(os.Stderr, "vox - stage0 prototype")
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  vox init [dir]")
	fmt.Fprintln(os.Stderr, "  vox ir [dir]")
	fmt.Fprintln(os.Stderr, "  vox build [dir]")
	fmt.Fprintln(os.Stderr, "  vox run [dir]")
	fmt.Fprintln(os.Stderr, "  vox test [dir]")
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
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := dumpIR(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "build":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := build(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "run":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := run(dir); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "test":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := test(dir); err != nil {
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

func build(dir string) error {
	_, err := compile(dir)
	return err
}

func run(dir string) error {
	bin, err := compile(dir)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func test(dir string) error {
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
	if res != nil && res.TestLog != "" {
		fmt.Fprint(os.Stdout, res.TestLog)
	}
	return nil
}

func compile(dir string) (string, error) {
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
	csrc, err := codegen.EmitC(irp, codegen.EmitOptions{EmitDriverMain: true})
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
