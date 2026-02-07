package interp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (rt *Runtime) callBuiltin(name string, args []Value) (Value, bool, error) {
	switch name {
	case "panic":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("panic expects (String)")
		}
		return unit(), true, fmt.Errorf("%s", args[0].S)
	case "print":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("print expects (String)")
		}
		// Stage0 interpreter doesn't capture stdout. Print is still useful for debugging.
		fmt.Print(args[0].S)
		return unit(), true, nil
	case "__args":
		if len(args) != 0 {
			return unit(), true, fmt.Errorf("__args expects ()")
		}
		var out []Value
		for _, a := range rt.args {
			out = append(out, Value{K: VString, S: a})
		}
		return Value{K: VVec, A: out}, true, nil
	case "__read_file":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__read_file expects (String)")
		}
		b, err := os.ReadFile(args[0].S)
		if err != nil {
			return unit(), true, err
		}
		return Value{K: VString, S: string(b)}, true, nil
	case "__write_file":
		if len(args) != 2 || args[0].K != VString || args[1].K != VString {
			return unit(), true, fmt.Errorf("__write_file expects (String, String)")
		}
		if err := os.WriteFile(args[0].S, []byte(args[1].S), 0o644); err != nil {
			return unit(), true, err
		}
		return unit(), true, nil
	case "__exec":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__exec expects (String)")
		}
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", args[0].S)
		} else {
			cmd = exec.Command("sh", "-c", args[0].S)
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err == nil {
			return Value{K: VInt, I: 0}, true, nil
		}
		if ee, ok := err.(*exec.ExitError); ok {
			return Value{K: VInt, I: int64(ee.ExitCode())}, true, nil
		}
		return unit(), true, err
	case "__walk_vox_files":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__walk_vox_files expects (String)")
		}
		root := args[0].S
		if root == "" {
			root = "."
		}
		var paths []string
		walk := func(dir string) error {
			return filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
				if err != nil {
					// Keep it simple: surface the first error.
					return err
				}
				if d.IsDir() {
					return nil
				}
				if !strings.HasSuffix(path, ".vox") {
					return nil
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				paths = append(paths, filepath.ToSlash(rel))
				return nil
			})
		}
		// Missing src/tests is ok.
		_ = walk("src")
		_ = walk("tests")
		var out []Value
		for _, p := range paths {
			out = append(out, Value{K: VString, S: p})
		}
		return Value{K: VVec, A: out}, true, nil
	default:
		return unit(), false, nil
	}
}
