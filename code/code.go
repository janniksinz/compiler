package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte

type Opcode byte

// document opcode name and len of operands
type Definition struct {
	Name          string
	OperandWidths []int // number of bytes each operand takes up
}

const (
	OpConstant Opcode = iota
)

// maping opcode definitions
//
//	with name and width
var definitions = map[Opcode]*Definition{
	OpConstant: {"OpConstant", []int{2}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}

	return def, nil
}

// MAKE
func Make(op Opcode, operands ...int) []byte { // (opcode, int offset (location) to constant operands)
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	// find out the resulting instruction length
	instructionLen := 1 // start with 1 as opcode byte
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	// allocate a byte slice of the appropriate length
	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	// iterate over the defined operands_width
	offset := 1
	// run this loop for every operand
	for i, o := range operands {
		// match element from operands
		width := def.OperandWidths[i]
		// put it in the instruction according to its defined width
		switch width {
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		}
		// first offset is 1 (opcode), increase offset by operand width
		offset += width
	}

	return instruction
}

// String decompilation
func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))

		i += 1 + read

	}
	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands),
			operandCount,
		)
	}

	switch operandCount {
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		}

		offset += width
	}

	return operands, offset
}

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins)
}
