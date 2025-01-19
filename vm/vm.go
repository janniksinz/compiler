package vm

import (
	"fmt"
	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const StackSize = 2048

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}
var Null = &object.Null{}

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
			pos := int(code.ReadUint16(vm.instructions[ip+1:])) // decode the operand after the opcode
			ip = pos - 1                                        // set instruction pointer to jump target
			// ip increases with the start of the next iteration
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(vm.instructions[ip+1:])) // decode operand after opcode
			ip += 2                                             // skip 2 bype operand

			// check if condition is true
			condition := vm.pop()
			if !isTruthy(condition) {
				// if not true, we jump to the alternative
				ip = pos - 1
			}
			// if true, we do nothing and run the consequence

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		default:
			panic("VM: run(): Encountered unknown OpCode")

		}

	}
	return nil
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {

	case *object.Boolean:
		return obj.Value

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
