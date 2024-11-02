package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048

// VM
// a struct with 4 fields
type VM struct {
	constants    []object.Object
	instructions code.Instructions

	stack []object.Object // objects in the stack
	sp    int             // Always points to the next value. Top of stack is stack[sp-1]
}

// takes the bytecode from the compiler
// returns a vm from that bytecode
func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,
	}
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
	// instruction pointer
	for ip := 0; ip < len(vm.instructions); ip++ {
		// fetch the opcode
		op := code.Opcode(vm.instructions[ip])

		switch op {
		case code.OpConstant:
			// decoding the operands of the instruction in the bytecode
			constIndex := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2 // increment the instruction pointer ip to point to the next Opcode instead of an operand

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

		case code.OpPop:
			vm.pop()
		}
	}
	return nil
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
