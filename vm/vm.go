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

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("vm: stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
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
		}
	}
	return nil
}
