package interp

import "fmt"

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
	default:
		return unit(), false, nil
	}
}
