package interp

import "fmt"

func (rt *Runtime) callBuiltin(name string, args []Value) (Value, bool, error) {
	switch name {
	case "assert":
		if len(args) != 1 || args[0].K != VBool {
			return unit(), true, fmt.Errorf("assert expects (bool)")
		}
		if !args[0].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert":
		if len(args) != 1 || args[0].K != VBool {
			return unit(), true, fmt.Errorf("std.testing::assert expects (bool)")
		}
		if !args[0].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_i32", "std.testing::assert_eq_i64":
		if len(args) != 2 || args[0].K != VInt || args[1].K != VInt {
			return unit(), true, fmt.Errorf("%s expects (int, int)", name)
		}
		if args[0].I != args[1].I {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_bool":
		if len(args) != 2 || args[0].K != VBool || args[1].K != VBool {
			return unit(), true, fmt.Errorf("std.testing::assert_eq_bool expects (bool, bool)")
		}
		if args[0].B != args[1].B {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::assert_eq_str":
		if len(args) != 2 || args[0].K != VString || args[1].K != VString {
			return unit(), true, fmt.Errorf("std.testing::assert_eq_str expects (String, String)")
		}
		if args[0].S != args[1].S {
			return unit(), true, fmt.Errorf("assertion failed")
		}
		return unit(), true, nil
	case "std.testing::fail":
		if len(args) != 1 || args[0].K != VString {
			return unit(), true, fmt.Errorf("std.testing::fail expects (String)")
		}
		return unit(), true, fmt.Errorf("%s", args[0].S)
	default:
		return unit(), false, nil
	}
}
