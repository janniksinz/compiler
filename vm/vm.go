package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048
const GlobalSize = 65536
const MaxFrames = 1024

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}
var Null = &object.Null{}

// VM
// a struct with 4 fields
type VM struct {
	constants []object.Object

	stack []object.Object // objects in the stack
	sp    int             // Always points to the next value. Top of stack is stack[sp-1]

	globals []object.Object

	frames      []*Frame // the instruction pointer "ip" is now part of the frame
	framesIndex int
}

// takes the bytecode from the compiler
// returns a vm from that bytecode
func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn) // add main function to main frame

	frames := make([]*Frame, MaxFrames) // create frames array
	frames[0] = mainFrame               // push mainFrame to index 0

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globals: make([]object.Object, GlobalSize),

		frames:      frames, // set out frames
		framesIndex: 1,      // and init the index for our next frame (current is 0)
	}
}

func NewWithGlobalStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = s
	return vm
}

// returns the object on top of the stack
func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

// push an object onto the stack
func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("vm: stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

// pop the top object from the stack
func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

// check the last object that's on the stack before we pop it
func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
	// we don't delete from the stack, we only decrease the vm.sp
	// so vm.sp is the last popped object
}

// FETCH-DECODE-EXECUTE cycle
// iterate through vm.instructions by incrementing the instruction pointer
func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	// execute OpCodes, while the instruction pointer is not at the end of the instruction stack
	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++ // increment the instruction pointer in the current frame

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		// fetch the opcode
		op = code.Opcode(ins[ip]) // fetch the next opcode from instructions at the current instruction pointer

		// execute OpCode
		switch op {
		case code.OpConstant:
			// decoding the operands of the instruction in the bytecode
			constIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2 // increment the instruction pointer ip to point to the next Opcode instead of an operand

			// Execute
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpTrue:
			err := vm.push(True) // push global true
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False) // push global false
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		// Prefix
		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}
		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		// end expression
		case code.OpPop:
			vm.pop()

		// conditionals
		case code.OpJump:
			pos := int(code.ReadUint16(ins[ip+1:])) // decode the operand after the opcode
			vm.currentFrame().ip = pos - 1          // set instruction pointer to jump target
			// ip increases with the start of the next iteration
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(ins[ip+1:])) // decode operand after opcode
			vm.currentFrame().ip += 2               // skip 2 bype operand

			// check if condition is true
			condition := vm.pop()
			if !isTruthy(condition) {
				// if not true, we jump to the alternative
				vm.currentFrame().ip = pos - 1
			}
			// if true, we do nothing and run the consequence

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2 // skip 2 byte instructions

			vm.globals[globalIndex] = vm.pop()

		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2 // skip 2 byte operands

			err := vm.push(vm.globals[globalIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(code.ReadUint16(ins[ip+1:])) // read the number of elements from the OpArray operand
			vm.currentFrame().ip += 2

			array := vm.buildArray(vm.sp-numElements, vm.sp)
			vm.sp = vm.sp - numElements

			err := vm.push(array) // push array on stack
			if err != nil {
				return fmt.Errorf("vm: Run(OpArray): failed to push array to stack. %s", err)
			}

		case code.OpHash:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-numElements, vm.sp) // build hash from current stack pointer to sp - elements of the hash
			if err != nil {
				return fmt.Errorf("vm: Run(): unable to build Hash. %s", err)
			}
			vm.sp -= numElements // update new stack pointer

			err = vm.push(hash)
			if err != nil {
				return fmt.Errorf("vm: Run(): failed to push hash to stack. %s", err)
			}

		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}

		default:
			op_code, _ := code.Lookup(byte(op))
			errString := fmt.Sprintf("VM: run(): Encountered unknown OpCode: %v", op_code)
			panic(errString)

		}

	}
	return nil
}

//
// FRAMES
//

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1] // the current frame is Index-1 because we initialize our first mainFrame as 0 and initialize the *VM framesIndex as 1
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

// END FRAMES

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

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		// return a possible error when pushing the result
		return vm.executeBinaryIntegerOperation(op, left, right)
	}
	if leftType == object.STRING_OBJ && rightType == object.STRING_OBJ {
		return vm.executeBinaryStringOperation(op, left, right)
	}

	return fmt.Errorf("unsupported types for binary operation: %s %s",
		leftType, rightType)

}

func (vm *VM) executeBinaryIntegerOperation(
	op code.Opcode,
	left, right object.Object,
) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64

	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return vm.push(&object.Integer{Value: result})
	}
	// return a possible error when pushing the result
	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeBinaryStringOperation(
	op code.Opcode,
	left, right object.Object,
) error {
	if op != code.OpAdd {
		return fmt.Errorf("vm: executeBinaryOperation: unknown string operator: %d", op)
	}

	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	return vm.push(&object.String{Value: leftValue + rightValue})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()
	//fmt.Printf("right=%v, left=%v", right, left)

	// nil checks
	if left == nil || right == nil {
		return fmt.Errorf("vm: executeComparison: cannot compare nil values: left=%v, right=%v", left, right)
	}

	// manage integer comparison
	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}

	// just manage booleans
	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)",
			op, left.Type(), right.Type())
	}
}

func (vm *VM) executeIntegerComparison(
	op code.Opcode,
	left, right object.Object,
) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue == leftValue))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue != leftValue))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

// IndexExpressions
func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)
	default:
		return fmt.Errorf("vm: executeIndexExpression: index operator not supported: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(arrayObject.Elements[i])
	// an OpIndex should always follow a pop operator that takes the element from the stack
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("vm: executeHashIndex: %s is not a Hashable key", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.push(Null) // key doesn't exist
	}

	return vm.push(pair.Value)
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()

	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	case Null:
		return vm.push(True) // Null is false and therefore we push True
	default:
		return vm.push(False)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("vm: unsupported type for negation: %s", operand.Type())
	}

	value := operand.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -value})
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}
	return False
}

// array

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex) // make a slice of objects with the length end-start

	// pop stack into elements array
	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}

	return &object.Array{Elements: elements} // return slice into array object
}

// hash

func (vm *VM) buildHash(startIndex, endIndex int) (object.Object, error) {
	hashedPairs := make(map[object.HashKey]object.HashPair)

	for i := startIndex; i < endIndex; i += 2 {
		// get values from the stack bottom up
		key := vm.stack[i]
		value := vm.stack[i+1]

		// build HashPair
		pair := object.HashPair{Key: key, Value: value}

		// generate a HashKey from the key
		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("vm: buildHash: unusable as hash key: %s", key.Type())
		}

		// build a list of hashpairs attached to our integer HashKey
		hashedPairs[hashKey.HashKey()] = pair
	}
	// return a list of hashpairs (indexed by our int64 HashKey)
	return &object.Hash{Pairs: hashedPairs}, nil
}
