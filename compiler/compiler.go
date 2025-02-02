package compiler

import (
	"fmt"
	"monkey/ast"
	"monkey/code"
	"monkey/object"
)

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

// generate instructions and constants
type Compiler struct {
	instructions code.Instructions
	constants    []object.Object
	// to only keep the last Instruction on the stack
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

// init compiler reference
func New() *Compiler {
	return &Compiler{
		instructions: code.Instructions{},
		constants:    []object.Object{},
		// track last Instruction that should be kept on stack
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
}

// walk the AST recursively
// find *ast.Literals -> turn into *object.Objects -> add to constants
//
// returns an error if compilation failed
func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	// NOTE: start with all the program statements
	// go through all statements and call Compile
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop) // pop from stack after every expression

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("compiler: unknown prefix operator %s", node.Operator)
		}

	case *ast.BlockStatement:
		for _, s := range node.Statements { // compiling all the statements
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.InfixExpression:
		// +------+----------+-------+
		// | left | Operator | right |
		// +------+----------+-------+

		if node.Operator == "<" {
			// compile right
			err := c.Compile(node.Right)
			if err != nil {
				return nil
			}
			// before left
			err = c.Compile(node.Left)
			if err != nil {
				return nil
			}

			c.emit(code.OpGreaterThan)
			return nil
		}

		// compile left
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		// before right
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		// switch
		// get the Operator from the InfixExpression
		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case ">":
			c.emit(code.OpGreaterThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("compiler: unknown infix operator %s", node.Operator)
		}

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// emit an JumpIfFalse with a bogus value but save the location
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}

		// decide if alternative exists
		if node.Alternative == nil {
			// substitute the goto
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		} else {
			// if alternative exists,
			// emit an `OpJump` with a bogus value
			jumpPos := c.emit(code.OpJump, 9999)

			// place the jumpto location after the jump
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}

			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}

			afterAlternativePos := len(c.instructions)
			c.changeOperand(jumpPos, afterAlternativePos)
		}

	case *ast.IntegerLiteral:
		// NOTE: literals are constant expressions and their value does not change
		integer := &object.Integer{Value: node.Value}
		// we generate the OpConstant instruction with the constant identifier
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	}
	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

// emit
// generate an instruction, add it to the results
// adds the instruction to a collection in memory (in this case)
func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.instructions)        // get next position
	c.instructions = append(c.instructions, ins...) // append instruction
	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.previousInstruction = previous
	c.lastInstruction = last
}

// Compile Helper

// addConstant adds the object of a constant to the "stack" (constants slice)
// returns the index in the constants slice
//
// we can use the index as its identifier to be used as the
// OPERAND for the OpConstant instruction
//
// +---------------------+---------------------+
// | OpCode "OpConstant" | Constant Identifier |
// +---------------------+---------------------+
func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) lastInstructionIsPop() bool {
	return c.lastInstruction.Opcode == code.OpPop
}

func (c *Compiler) removeLastPop() {
	c.instructions = c.instructions[:c.lastInstruction.Position]
	c.lastInstruction = c.previousInstruction
}

// replaceInstruction replaces Instructions from position pos on
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

// changeOperand
func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos]) // get the old opcode
	newInstruction := code.Make(op, operand) // recreate the instruction with the new operand

	c.replaceInstruction(opPos, newInstruction)
}
