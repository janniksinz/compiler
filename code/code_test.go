package code

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		op       Opcode
		operands []int
		expected []byte
	}{
		{OpConstant, []int{65534}, []byte{byte(OpConstant), 255, 254}},
		// we expect a byte array []byte holding 3 bytes
		// 1 - opcode (OpConstant); 2&3 - big endian encoding of 65534 (most significant comes first)
		{OpAdd, []int{}, []byte{byte(OpAdd)}},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)

		if len(instruction) != len(tt.expected) {
			t.Errorf("instruction has wrong length. want=%d, got=%d",
				len(tt.expected),
				len(instruction),
			)
		}

		for i, b := range tt.expected {
			if instruction[i] != tt.expected[i] {
				t.Errorf("wrong byte at pos %d. want=%d, got=%d",
					i,              // expected.byte(OpConstant)
					b,              // expected.255
					instruction[i], // expected.254
				)
			}
		}
	}
}

func TestInstructionsString(t *testing.T) {
	instructions := []Instructions{
		Make(OpAdd),
		Make(OpConstant, 2),
		Make(OpConstant, 65535),
	}

	expected := `0000 OpAdd
0001 OpConstant 2
0004 OpConstant 65535
`

	concatted := Instructions{}
	for _, ins := range instructions {
		concatted = append(concatted, ins...)
	}

	if concatted.String() != expected {
		t.Errorf("code: instructions wrongly formated.\nwant=%q\ngot=%q",
			expected,
			concatted.String(),
		)
	}
}

func TestReadOperands(t *testing.T) {
	tests := []struct {
		op        Opcode
		operands  []int
		bytesRead int
	}{
		{OpConstant, []int{65535}, 2},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)

		def, err := Lookup(byte(tt.op))
		if err != nil {
			t.Fatalf("definition not found: %q\n", err)
		}

		operandsRead, n := ReadOperands(def, instruction[1:])
		if n != tt.bytesRead {
			t.Fatalf("n wrong. want=%d, got=%d",
				tt.bytesRead,
				n,
			)
		}

		for i, want := range tt.operands {
			if operandsRead[i] != want {
				t.Errorf("operand wrong. want=%d, got=%d",
					want,
					operandsRead[i],
				)
			}
		}
	}
}

func TestArrayLiterals(t *testing.T) {

}
