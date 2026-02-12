package interp

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Tooling/runtime intrinsics reserved for stdlib implementation (names start with "__").
func (rt *Runtime) callToolIntrinsic(name string, args []Value) (Value, bool, error) {
	switch name {
	case "__args":
		if len(args) != 0 {
			return unit(), true, fmt.Errorf("__args expects ()")
		}
		var out []Value
		for _, a := range rt.args {
			out = append(out, Value{K: VString, S: a})
		}
		return newVecValue(out), true, nil
	case "__exe_path":
		if len(args) != 0 {
			return unit(), true, fmt.Errorf("__exe_path expects ()")
		}
		p, err := os.Executable()
		if err != nil {
			return unit(), true, err
		}
		return Value{K: VString, S: p}, true, nil
	case "__getenv":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__getenv expects (String)")
		}
		return Value{K: VString, S: os.Getenv(args[0].S)}, true, nil
	case "__now_ns":
		if len(args) != 0 {
			return unit(), true, fmt.Errorf("__now_ns expects ()")
		}
		return Value{K: VInt, I: uint64(time.Now().UnixNano())}, true, nil
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
	case "__path_exists":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__path_exists expects (String)")
		}
		_, err := os.Stat(args[0].S)
		if err == nil {
			return Value{K: VBool, B: true}, true, nil
		}
		if os.IsNotExist(err) {
			return Value{K: VBool, B: false}, true, nil
		}
		return unit(), true, err
	case "__mkdir_p":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("__mkdir_p expects (String)")
		}
		if err := os.MkdirAll(args[0].S, 0o755); err != nil {
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
			return Value{K: VInt, I: uint64(ee.ExitCode())}, true, nil
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
		return newVecValue(out), true, nil
	case "__mutex_i32_new":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i32_new expects (i32)")
		}
		h := rt.newHandle()
		rt.mutexI32[h] = int32(args[0].I)
		return Value{K: VInt, I: h}, true, nil
	case "__mutex_i32_load":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i32_load expects (i64)")
		}
		h := args[0].I
		v, ok := rt.mutexI32[h]
		if !ok {
			return unit(), true, fmt.Errorf("__mutex_i32_load: invalid handle")
		}
		return Value{K: VInt, I: uint64(uint32(v))}, true, nil
	case "__mutex_i32_store":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i32_store expects (i64, i32)")
		}
		h := args[0].I
		if _, ok := rt.mutexI32[h]; !ok {
			return unit(), true, fmt.Errorf("__mutex_i32_store: invalid handle")
		}
		rt.mutexI32[h] = int32(args[1].I)
		return unit(), true, nil
	case "__atomic_i32_new":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i32_new expects (i32)")
		}
		h := rt.newHandle()
		rt.atomicI32[h] = int32(args[0].I)
		return Value{K: VInt, I: h}, true, nil
	case "__atomic_i32_load":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i32_load expects (i64)")
		}
		h := args[0].I
		v, ok := rt.atomicI32[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i32_load: invalid handle")
		}
		return Value{K: VInt, I: uint64(uint32(v))}, true, nil
	case "__atomic_i32_store":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i32_store expects (i64, i32)")
		}
		h := args[0].I
		if _, ok := rt.atomicI32[h]; !ok {
			return unit(), true, fmt.Errorf("__atomic_i32_store: invalid handle")
		}
		rt.atomicI32[h] = int32(args[1].I)
		return unit(), true, nil
	case "__atomic_i32_fetch_add":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i32_fetch_add expects (i64, i32)")
		}
		h := args[0].I
		old, ok := rt.atomicI32[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i32_fetch_add: invalid handle")
		}
		rt.atomicI32[h] = old + int32(args[1].I)
		return Value{K: VInt, I: uint64(uint32(old))}, true, nil
	case "__atomic_i32_swap":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i32_swap expects (i64, i32)")
		}
		h := args[0].I
		old, ok := rt.atomicI32[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i32_swap: invalid handle")
		}
		rt.atomicI32[h] = int32(args[1].I)
		return Value{K: VInt, I: uint64(uint32(old))}, true, nil
	case "__mutex_i64_new":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i64_new expects (i64)")
		}
		h := rt.newHandle()
		rt.mutexI64[h] = int64(args[0].I)
		return Value{K: VInt, I: h}, true, nil
	case "__mutex_i64_load":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i64_load expects (i64)")
		}
		h := args[0].I
		v, ok := rt.mutexI64[h]
		if !ok {
			return unit(), true, fmt.Errorf("__mutex_i64_load: invalid handle")
		}
		return Value{K: VInt, I: uint64(v)}, true, nil
	case "__mutex_i64_store":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__mutex_i64_store expects (i64, i64)")
		}
		h := args[0].I
		if _, ok := rt.mutexI64[h]; !ok {
			return unit(), true, fmt.Errorf("__mutex_i64_store: invalid handle")
		}
		rt.mutexI64[h] = int64(args[1].I)
		return unit(), true, nil
	case "__atomic_i64_new":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i64_new expects (i64)")
		}
		h := rt.newHandle()
		rt.atomicI64[h] = int64(args[0].I)
		return Value{K: VInt, I: h}, true, nil
	case "__atomic_i64_load":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i64_load expects (i64)")
		}
		h := args[0].I
		v, ok := rt.atomicI64[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i64_load: invalid handle")
		}
		return Value{K: VInt, I: uint64(v)}, true, nil
	case "__atomic_i64_store":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i64_store expects (i64, i64)")
		}
		h := args[0].I
		if _, ok := rt.atomicI64[h]; !ok {
			return unit(), true, fmt.Errorf("__atomic_i64_store: invalid handle")
		}
		rt.atomicI64[h] = int64(args[1].I)
		return unit(), true, nil
	case "__atomic_i64_fetch_add":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i64_fetch_add expects (i64, i64)")
		}
		h := args[0].I
		old, ok := rt.atomicI64[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i64_fetch_add: invalid handle")
		}
		rt.atomicI64[h] = old + int64(args[1].I)
		return Value{K: VInt, I: uint64(old)}, true, nil
	case "__atomic_i64_swap":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__atomic_i64_swap expects (i64, i64)")
		}
		h := args[0].I
		old, ok := rt.atomicI64[h]
		if !ok {
			return unit(), true, fmt.Errorf("__atomic_i64_swap: invalid handle")
		}
		rt.atomicI64[h] = int64(args[1].I)
		return Value{K: VInt, I: uint64(old)}, true, nil
	case "__tcp_connect":
		if len(args) != 2 || args[0].K != VString || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__tcp_connect expects (String, i32)")
		}
		host := args[0].S
		port := int32(args[1].I)
		if port <= 0 {
			return unit(), true, fmt.Errorf("__tcp_connect: invalid port")
		}
		addr := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			return unit(), true, err
		}
		h := rt.newHandle()
		rt.tcpConns[h] = conn
		return Value{K: VInt, I: h}, true, nil
	case "__tcp_send":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VString {
			return unit(), true, fmt.Errorf("__tcp_send expects (i64, String)")
		}
		conn, ok := rt.tcpConns[args[0].I]
		if !ok || conn == nil {
			return unit(), true, fmt.Errorf("__tcp_send: invalid handle")
		}
		n, err := conn.Write([]byte(args[1].S))
		if err != nil {
			return unit(), true, err
		}
		return Value{K: VInt, I: uint64(uint32(n))}, true, nil
	case "__tcp_recv":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("__tcp_recv expects (i64, i32)")
		}
		conn, ok := rt.tcpConns[args[0].I]
		if !ok || conn == nil {
			return unit(), true, fmt.Errorf("__tcp_recv: invalid handle")
		}
		nmax := int(int32(args[1].I))
		if nmax <= 0 {
			return Value{K: VString, S: ""}, true, nil
		}
		buf := make([]byte, nmax)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return Value{K: VString, S: ""}, true, nil
			}
			return unit(), true, err
		}
		return Value{K: VString, S: string(buf[:n])}, true, nil
	case "__tcp_close":
		if len(args) != 1 || args[0].K != VInt {
			return unit(), true, fmt.Errorf("__tcp_close expects (i64)")
		}
		h := args[0].I
		conn, ok := rt.tcpConns[h]
		if !ok || conn == nil {
			return unit(), true, fmt.Errorf("__tcp_close: invalid handle")
		}
		delete(rt.tcpConns, h)
		return unit(), true, conn.Close()
	default:
		return unit(), false, nil
	}
}

func (rt *Runtime) newHandle() uint64 {
	h := rt.nextHandle
	rt.nextHandle++
	if rt.nextHandle == 0 {
		rt.nextHandle = 1
	}
	if h == 0 {
		h = rt.nextHandle
		rt.nextHandle++
	}
	return h
}
