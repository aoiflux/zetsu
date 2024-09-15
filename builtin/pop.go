package builtin

import "zetsu/object"

func Pop(args ...object.Object) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments. got=%d, want=1", len(args))
	}
	if args[0].Type() != object.ARRAY_OBJ {
		return newError("argument to `push` must be ARRAY, got=%s", args[0].Type())
	}
	arr := args[0].(*object.Array)
	length := len(arr.Elements)
	if length == 1 {
		return nil
	}

	newElements := make([]object.Object, length-1)
	copy(newElements, arr.Elements[:len(arr.Elements)-1])

	return &object.Array{Elements: newElements}
}
