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
