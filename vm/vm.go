package vm

import (
	"fmt"
	"zetsu/builtin"
	"zetsu/code"
	"zetsu/compiler"
	"zetsu/global"
	"zetsu/mutil"
	"zetsu/object"
	"zetsu/security"
)

// VM structure defines virtual machine
type VM struct {
	constants    []object.Object
	stack        []object.Object
	stackPointer int // top of stack is stack[stackPointer-1]
	globals      []object.Object
	frames       []*Frame
	frameIndex   int
	inslen       int
}

func New(bc *compiler.ByteCode) *VM {
	mainfn := &object.CompiledFunction{Instructions: bc.Instructions}
	frames := make([]*Frame, global.MaxFrames)

	mainClosure := &object.Closure{Fn: mainfn}
	mainFrame := NewFrame(mainClosure, 0)
	frames[0] = mainFrame

	return &VM{
		constants:    bc.Constants,
		stack:        make([]object.Object, global.StackSize),
		stackPointer: 0,
		globals:      make([]object.Object, global.GlobalSize),
		frames:       frames,
		frameIndex:   1,
		inslen:       len(bc.Instructions),
	}
}

func NewWithGlobalStore(bc *compiler.ByteCode, globals []object.Object) *VM {
	vm := New(bc)
	vm.globals = globals
	return vm
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		ins[ip] = security.XOROne(ins[ip], vm.inslen)
		op = code.Opcode(ins[ip])
		ins[ip] = security.XOROne(ins[ip], vm.inslen)

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint16(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip += 2

			if err := vm.push(vm.constants[constIndex]); err != nil {
				return err
			}
		case code.OpBang:
			if err := vm.executeBangOperation(); err != nil {
				return err
			}
		case code.OpMinus:
			if err := vm.executeMinusOperation(); err != nil {
				return err
			}
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			if err := vm.execBinaryOperation(op); err != nil {
				return err
			}
		case code.OpTrue:
			if err := vm.push(global.True); err != nil {
				return err
			}
		case code.OpFalse:
			if err := vm.push(global.False); err != nil {
				return err
			}
		case code.OpArray:
			numElements := int(code.ReadUint16(ins[ip+1:], vm.inslen))
			vm.currentFrame().ip += 2
			array := vm.buildArray(vm.stackPointer-numElements, vm.stackPointer)
			if err := vm.push(array); err != nil {
				return err
			}
		case code.OpHash:
			numElements := int(code.ReadUint16(ins[ip+1:], vm.inslen))
			vm.currentFrame().ip += 2
			hash, err := vm.buildHash(vm.stackPointer-numElements, vm.stackPointer)
			if err != nil {
				return err
			}
			vm.stackPointer = vm.stackPointer - numElements
			if err := vm.push(hash); err != nil {
				return err
			}
		case code.OpEqual, code.OpUnEqual, code.OpGreater:
			if err := vm.executeComparison(op); err != nil {
				return err
			}
		case code.OpJump:
			pos := int(code.ReadUint16(ins[ip+1:], vm.inslen))
			vm.currentFrame().ip = pos - 1
		case code.OpJumpFalse:
			pos := int(code.ReadUint16(ins[ip+1:], vm.inslen))
			vm.currentFrame().ip += 2
			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip += 2
			vm.globals[globalIndex] = vm.pop()
		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip += 2
			if err := vm.push(vm.globals[globalIndex]); err != nil {
				return err
			}
		case code.OpSetLocal:
			localIndex := code.ReadUint8(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip++
			frame := vm.currentFrame()
			obj := vm.pop()
			encObj, err := mutil.EncryptObject(obj, vm.inslen)
			if err != nil {
				vm.stack[frame.bp+int(localIndex)] = obj
			} else {
				vm.stack[frame.bp+int(localIndex)] = encObj
			}
		case code.OpGetLocal:
			localIndex := code.ReadUint8(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip++
			frame := vm.currentFrame()
			if err := vm.push(vm.stack[frame.bp+int(localIndex)]); err != nil {
				return err
			}
		case code.OpGetBuiltin:
			builtinIndex := code.ReadUint8(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip++
			definition := builtin.Builtins[builtinIndex]
			if err := vm.push(definition.Builtin); err != nil {
				return err
			}
		case code.OpGetFree:
			freeIndex := code.ReadUint8(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip++
			currentClosure := vm.currentFrame().cl
			if err := vm.push(currentClosure.Free[freeIndex]); err != nil {
				return err
			}
		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()
			if err := vm.execIndexOperation(left, index); err != nil {
				return err
			}
		case code.OpClosure:
			constIndex := code.ReadUint16(ins[ip+1:], vm.inslen)
			numFree := code.ReadUint8(ins[ip+3:], vm.inslen)
			vm.currentFrame().ip += 3
			if err := vm.pushClosure(int(constIndex), int(numFree)); err != nil {
				return err
			}
		case code.OpCurrentClosure:
			currentClosure := vm.currentFrame().cl
			if err := vm.push(currentClosure); err != nil {
				return err
			}
		case code.OpCall:
			numArgs := code.ReadUint8(ins[ip+1:], vm.inslen)
			vm.currentFrame().ip++
			if err := vm.executeCall(int(numArgs)); err != nil {
				return err
			}
		case code.OpReturnValue:
			returnValue := vm.pop()
			frame := vm.popFrame()
			vm.stackPointer = frame.bp - 1
			if err := vm.push(returnValue); err != nil {
				return err
			}
		case code.OpReturn:
			frame := vm.popFrame()
			vm.stackPointer = frame.bp - 1
			if err := vm.push(global.Null); err != nil {
				return err
			}
		case code.OpNull:
			if err := vm.push(global.Null); err != nil {
				return err
			}
		case code.OpPop:
			vm.pop()
		}
	}
	return nil
}

func (vm *VM) StackTop() object.Object {
	if vm.stackPointer == 0 {
		return nil
	}

	return vm.stack[vm.stackPointer-1]
}

func (vm *VM) LastPoppedStackElement() object.Object {
	obj := vm.stack[vm.stackPointer]
	if decObj, err := mutil.DecryptObject(obj, vm.inslen); err == nil {
		obj = decObj
	}
	return obj
}

func (vm *VM) pushClosure(constIndex, numFree int) error {
	constant := vm.constants[constIndex]
	fun, ok := constant.(*object.CompiledFunction)
	if !ok {
		return fmt.Errorf("not a function: %+v", constant)
	}

	free := make([]object.Object, numFree)
	for i := 0; i < numFree; i++ {
		free[i] = vm.stack[vm.stackPointer-numFree+i]
	}
	vm.stackPointer = vm.stackPointer - numFree

	closure := &object.Closure{Fn: fun, Free: free}
	return vm.push(closure)
}

func (vm *VM) push(obj object.Object) error {
	if vm.stackPointer >= global.StackSize {
		return fmt.Errorf("stack overflow")
	}

	if encObj, err := mutil.EncryptObject(obj, vm.inslen); err == nil {
		obj = encObj
	}

	vm.stack[vm.stackPointer] = obj
	vm.stackPointer++

	return nil
}

func (vm *VM) pop() object.Object {
	obj := vm.stack[vm.stackPointer-1]
	if newObj, err := mutil.DecryptObject(obj, vm.inslen); err == nil {
		obj = newObj
	}

	vm.stackPointer--
	return obj
}

func (vm *VM) execBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	rtype := right.Type()
	ltype := left.Type()

	if rtype == object.INTEGER_OBJ && ltype == object.INTEGER_OBJ {
		return vm.execBinaryIntegerOperation(op, left, right)
	}

	switch {
	case rtype == object.INTEGER_OBJ && ltype == object.INTEGER_OBJ:
		return vm.execBinaryIntegerOperation(op, left, right)
	case rtype == object.STRING_OBJ && ltype == object.STRING_OBJ:
		return vm.execBinaryStringOperation(op, left, right)
	}

	return fmt.Errorf("Unsupported types for binary operation: %s, %s", ltype, rtype)
}

func (vm *VM) execBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	rval := right.(*object.Integer).Value
	lval := left.(*object.Integer).Value
	var result int64

	switch op {
	case code.OpAdd:
		result = lval + rval
	case code.OpSub:
		result = lval - rval
	case code.OpMul:
		result = lval * rval
	case code.OpDiv:
		result = lval / rval
	default:
		return fmt.Errorf("Unknown integer operator: %d", op)
	}

	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) execBinaryStringOperation(op code.Opcode, left, right object.Object) error {
	rval := right.(*object.String).Value
	lval := left.(*object.String).Value

	if op != code.OpAdd {
		return fmt.Errorf("Unknown string operator: %d", op)
	}

	return vm.push(&object.String{Value: lval + rval})
}

func (vm *VM) execIndexOperation(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.execArrayIndex(left, index)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.execStringIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.execHashIndex(left, index)
	default:
		return fmt.Errorf("index operator not supported: %s", left.Type())
	}
}

func (vm *VM) execStringIndex(str, index object.Object) error {
	strVal := str.(*object.String).Value
	i := index.(*object.Integer).Value
	max := int64(len(strVal) - 1)
	if i > max {
		return vm.push(global.Null)
	} else if i < 0 {
		strObj := &object.String{Value: string(strVal[max+i+1])}
		return vm.push(strObj)
	}
	strObj := &object.String{Value: string(strVal[i])}
	return vm.push(strObj)
}

func (vm *VM) execArrayIndex(array, index object.Object) error {
	arrayObj := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObj.Elements) - 1)
	if i > max {
		return vm.push(global.Null)
	} else if i < 0 {
		return vm.push(arrayObj.Elements[max+i+1])
	}
	return vm.push(arrayObj.Elements[i])
}

func (vm *VM) execHashIndex(hash, index object.Object) error {
	hashObj := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObj.Pairs[key.HashKey()]
	if !ok {
		return vm.push(global.Null)
	}

	return vm.push(pair.Value)
}

func (vm *VM) executeBangOperation() error {
	operand := vm.pop()

	switch operand {
	case global.True:
		return vm.push(global.False)
	case global.False:
		return vm.push(global.True)
	case global.Null:
		return vm.push(global.True)
	default:
		return vm.push(global.False)
	}
}

func (vm *VM) executeMinusOperation() error {
	operand := vm.pop()
	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported object type for negation: %s", operand.Type())
	}
	value := operand.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -value})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if left.Type() == object.INTEGER_OBJ || right.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right.Inspect() == left.Inspect()))
	case code.OpUnEqual:
		return vm.push(nativeBoolToBooleanObject(right.Inspect() != left.Inspect()))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeIntegerComparison(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value
	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue == leftValue))
	case code.OpUnEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue != leftValue))
	case code.OpGreater:
		return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)
	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
		element, err := mutil.DecryptObject(elements[i-startIndex], vm.inslen)
		if err == nil {
			elements[i-startIndex] = element
		}
	}
	return &object.Array{Elements: elements}
}

func (vm *VM) buildHash(startIndex, endIndex int) (object.Object, error) {
	hashedPairs := make(map[object.HashKey]object.HashPair)
	for i := startIndex; i < endIndex; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]

		hkey, err := mutil.DecryptObject(key, vm.inslen)
		if err == nil {
			key = hkey
		}

		hvalue, err := mutil.DecryptObject(value, vm.inslen)
		if err == nil {
			value = hvalue
		}

		pair := object.HashPair{Key: key, Value: value}
		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable as a hashkey: %s", key.Type())
		}
		hashedPairs[hashKey.HashKey()] = pair
	}
	return &object.Hash{Pairs: hashedPairs}, nil
}

func (vm *VM) currentFrame() *Frame { return vm.frames[vm.frameIndex-1] }
func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.frameIndex] = f
	vm.frameIndex++
}
func (vm *VM) popFrame() *Frame {
	vm.frameIndex--
	return vm.frames[vm.frameIndex]
}

func (vm *VM) executeCall(numArgs int) error {
	var callee object.Object
	if vm.stack[vm.stackPointer-1-numArgs].Type() == object.CLOSURE_OBJ || vm.stack[vm.stackPointer-1-numArgs].Type() == object.BUILTIN_OBJ {
		callee = vm.stack[vm.stackPointer-1-numArgs]
	} else {
		callee = vm.stack[0]
	}

	switch calleeType := callee.(type) {
	case *object.Closure:
		return vm.callClosure(calleeType, numArgs)
	case *builtin.BuiltIn:
		return vm.callBuiltin(calleeType, numArgs)

	default:
		return fmt.Errorf("calling non-function and non-built-in")
	}
}

func (vm *VM) callClosure(cl *object.Closure, numArgs int) error {
	if numArgs != cl.Fn.NumParams {
		return fmt.Errorf("wrong number of arguments. want=%d, got=%d", cl.Fn.NumParams, numArgs)
	}

	frame := NewFrame(cl, vm.stackPointer-numArgs)
	vm.pushFrame(frame)
	vm.stackPointer = frame.bp + cl.Fn.NumLocals
	return nil
}

func (vm *VM) callBuiltin(builtin *builtin.BuiltIn, numArgs int) error {
	args := vm.stack[vm.stackPointer-numArgs : vm.stackPointer]
	for i := range args {
		dec, err := mutil.DecryptObject(args[i], vm.inslen)
		if err == nil {
			args[i] = dec
		}
	}
	result := builtin.Fn(args...)

	vm.stackPointer = vm.stackPointer - numArgs - 1

	if result != nil {
		vm.push(result)
	} else {
		vm.push(global.Null)
	}

	return nil
}

func nativeBoolToBooleanObject(native bool) *object.Boolean {
	if native {
		return global.True
	}
	return global.False
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	default:
		return true
	}
}
