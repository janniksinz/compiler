package compiler

import (
	"fmt"
	"monkey/ast"
	"monkey/code"
	"monkey/object"
)

// generate instructions and constants
type Compiler struct {
	instructions code.Instructions
	constants    []object.Object
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
	case *ast.InfixExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		// switch
		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		default:
			return fmt.Errorf("compiler: unknown operator %s", node.Operator)
		}
	case *ast.IntegerLiteral:
		// NOTE: literals are constant expressions and their value does not change
		integer := &object.Integer{Value: node.Value}
		// we generate the OpConstant instruction with the constant identifier
		c.emit(code.OpConstant, c.addConstant(integer))
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
	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.instructions)        // get next position
	c.instructions = append(c.instructions, ins...) // append instruction
	return posNewInstruction
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
