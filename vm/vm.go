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
		case code.OpAdd:
			// pop 2 constants from the stack
			right := vm.pop()
			left := vm.pop()
			leftValue := left.(*object.Integer).Value
			rightValue := right.(*object.Integer).Value

			result := leftValue + rightValue
			vm.push(&object.Integer{Value: result})
		case code.OpPop:
			vm.pop()
		}
	}
	return nil
}
