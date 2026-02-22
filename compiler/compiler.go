package compiler

import (
	"fmt"
	"monkey/ast"
	"monkey/code"
	"monkey/object"
	"sort"
)

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

// generate instructions and constants
type Compiler struct {
	// instructions code.Instructions | remove for CompilationScope
	constants []object.Object
	// to only keep the last Instruction on the stack
	// lastInstruction     EmittedInstruction | remove for CompilationScope
	// previousInstruction EmittedInstruction | remove for CompilationScope
	symbolTable *SymbolTable

	// stack of compilation scopes
	scopes     []CompilationScope
	scopeIndex int
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

// before compiling a new scope e.g. a function body, we push a new CompilationScope on to the scopes stack
// while compiling inside this scope, the emit() method will only modify fields of the current CompilationScope
type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

// init compiler reference
func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	return &Compiler{
		constants: []object.Object{},
		// track last Instruction that should be kept on stack
		symbolTable: NewSymbolTable(),

		scopes:     []CompilationScope{mainScope},
		scopeIndex: 0,
	}
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
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
		// compile the condition
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}
		// emit a jump to the alternative
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)
		// compile the consequence
		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}
		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}
		// emit a jump to after the alternative
		jumpPos := c.emit(code.OpJump, 9999)
		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos) // update the jump to the alternative
		// compile the alternative
		if node.Alternative == nil {
			// place a Null Alternative
			c.emit(code.OpNull)
		} else {
			// compile the real alterative
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}
		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos) // update the jump to after the alternative

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

	case *ast.LetStatement:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		symbol := c.symbolTable.Define(node.Name.Value) // retuns (Name, Scope, Index)
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("Compile(): undefined variable %s", node.Value) // "compile time error" !!
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpGetGlobal, symbol.Index)
		} else {
			c.emit(code.OpGetLocal, symbol.Index)
		}

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.ArrayLiteral:
		for i, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return fmt.Errorf("Compile(): array element %d coudn't compile %v", i, err)
			}
		}

		c.emit(code.OpArray, len(node.Elements))

	case *ast.HashLiteral:
		// define keys array as array of Expression objects - and append keys to the array
		keys := []ast.Expression{}
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		// sort all keys by their string representation
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		// go through each pair and compile keys and values
		for i, k := range keys {
			// compile key
			err := c.Compile(k)
			if err != nil {
				return fmt.Errorf("comp: Compile(): (Hash) compilation of key %d failed. %s", i, err)
			}
			// compile the value
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return fmt.Errorf("comp: Compile(): (Hash) compilation of key %d failed. %s", i, err)
			}
		}

		c.emit(code.OpHash, len(node.Pairs)*2) // emit a Hashset with the length of all the keys and values

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}

		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		c.enterScope()

		err := c.Compile(node.Body)
		if err != nil {
			return fmt.Errorf("comp: Compile(): (FunctionLiteral) compilation failed. %s", err)
		}

		// if the last instruction is a pop, we want to implicitely return
		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}

		// if the last instruction is not a return value, we expect to emit a default return -> we didnt' have any instructions
		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}

		instructions := c.leaveScope()

		compiledFn := &object.CompiledFunction{Instructions: instructions}
		c.emit(code.OpConstant, c.addConstant(compiledFn))

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return fmt.Errorf("comp: Compile(): (ReturnStatement) compilation failed. %s", err)
		}

		c.emit(code.OpReturnValue)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return fmt.Errorf("comp: Compile(): (CallExpression) compilation failed. %s", err)
		}

		c.emit(code.OpCall)

	}
	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
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
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)

	// updating instructions
	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	// instead of setting the last instruction directly, we overwrite the scopes
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
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

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position]

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))

	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

// replaceInstruction replaces Instructions from position pos on
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()

	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

// changeOperand
func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos]) // get the old opcode
	newInstruction := code.Make(op, operand)          // recreate the instruction with the new operand

	c.replaceInstruction(opPos, newInstruction)
}

// Scopes
//
// scope helper functions
func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

// enter a new scope by adding a new scope onto the scope stack and inc the stack pointer
func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex += 1
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

// exit the top scope on the stack
func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex -= 1
	c.symbolTable = c.symbolTable.Outer

	return instructions
}
