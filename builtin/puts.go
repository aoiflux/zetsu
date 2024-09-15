package builtin

import (
	"fmt"
	"zetsu/object"
)

func Puts(args ...object.Object) object.Object {
	for _, arg := range args {
		fmt.Print(arg.Inspect())
	}
	return nil
}
