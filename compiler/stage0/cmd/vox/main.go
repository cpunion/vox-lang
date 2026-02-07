package main

import (
	"fmt"
	"os"
	"path/filepath"

	"voxlang/internal/diag"
	"voxlang/internal/loader"
)

func usage() {
	fmt.Fprintln(os.Stderr, "vox - stage0 prototype")
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  vox init [dir]")
	fmt.Fprintln(os.Stderr, "  vox build [dir]")
	fmt.Fprintln(os.Stderr, "  vox run [dir]")
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
	case "build":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := build(dir, false); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case "run":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := build(dir, true); err != nil {
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

func build(dir string, run bool) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	res, diags, err := loader.BuildPackage(abs, run)
	if err != nil {
		return err
	}
	if diags != nil && len(diags.Items) > 0 {
		diag.Print(os.Stderr, diags)
		return fmt.Errorf("build failed")
	}
	if run && res != nil && res.RunResult != "" {
		fmt.Fprintln(os.Stdout, res.RunResult)
	}
	return nil
}
