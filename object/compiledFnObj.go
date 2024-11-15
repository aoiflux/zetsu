package object

import (
	"fmt"
	"zetsu/code"
)

type CompiledFunction struct {
	Instructions code.Instructions
	NumLocals    int
	NumParams    int
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FN_OBJ }
func (cf *CompiledFunction) Inspect() string  { return fmt.Sprintf("Compiled Function[%p]", cf) }
