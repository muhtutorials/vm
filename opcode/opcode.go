// Package opcode defines opcode to integer mapping
package opcode

var (
	// EXIT is the first opcode
	EXIT = 0x00

	// INT_STORE stores an integer in a register
	INT_STORE = 0x01

	// INT_PRINT prints the integer contents of a register
	INT_PRINT = 0x02

	// INT_TO_STR converts an integer register value to a string
	INT_TO_STR = 0x03

	// INT_RAND generates a random number
	INT_RAND = 0x04

	// JMP is an unconditional jump
	JMP = 0x10

	// JMP_Z jumps if the Z-flag is set
	JMP_Z = 0x11

	// JMP_NZ jumps if the Z-flag is NOT set
	JMP_NZ = 0x12

	// ADD performs an addition operation against two registers
	ADD = 0x20

	// SUB performs a subtraction operation against two registers
	SUB = 0x21

	// MUL performs a multiplication operation against two registers
	MUL = 0x22

	// DIV performs a division operation against two registers
	DIV = 0x23

	// INC increments the given register by one
	INC = 0x24

	// DEC decrements the given register by one
	DEC = 0x25

	// AND performs a logical AND operation against two registers
	AND = 0x26

	// OR performs a logical OR operation against two registers
	OR = 0x27

	// XOR performs an XOR operation against two registers
	XOR = 0x28

	// STR_STORE stores a string in a register
	STR_STORE = 0x30

	// STR_PRINT prints the string contents of a register
	STR_PRINT = 0x31

	// CONCAT joins two strings
	CONCAT = 0x32

	// SYSTEM executes the system binary stored in the given string register
	SYSTEM = 0x33

	// STR_TO_INT converts the given string register contents to an integer
	STR_TO_INT = 0x34

	// CMP_INT compares a register contents with a number
	CMP_INT = 0x40

	// CMP_STR compares a register contents with a string
	CMP_STR = 0x41

	// CMP_REG compares two registers
	CMP_REG = 0x42

	// IS_INT tests if a register contains an integer
	IS_INT = 0x43

	// IS_STR tests if a register contains a string
	IS_STR = 0x44

	// NOP does nothing
	NOP = 0x50

	// REG_STORE stores the contents of one register in another
	REG_STORE = 0x51

	// PEEK reads from memory
	PEEK = 0x60

	// POKE sets an address content
	POKE = 0x61

	// MEM_CPY copies a region of RAM
	MEM_CPY = 0x62

	// PUSH pushes the given register contents onto the stack
	PUSH = 0x70

	// POP retrieves a value from the stack
	POP = 0x71

	// CALL calls a subroutine
	CALL = 0x72

	// RET returns from a CALL
	RET = 0x73

	// TRAP invokes a CPU trap
	TRAP = 0x80
)

// Opcode is a holder for a single instruction.
// Note that this doesn't take any account of the arguments which might
// be necessary.
type Opcode struct {
	instruction byte
}

// NewOpcode creates a new Opcode
func NewOpcode(instruction byte) *Opcode {
	return &Opcode{instruction: instruction}
}

func (o *Opcode) String() string {
	switch int(o.instruction) {
	case EXIT:
		return "EXIT"
	case INT_STORE:
		return "INT_STORE"
	case INT_PRINT:
		return "INT_PRINT"
	case INT_TO_STR:
		return "INT_TO_STR"
	case INT_RAND:
		return "INT_RAND"
	case JMP:
		return "JMP"
	case JMP_Z:
		return "JMP_Z"
	case JMP_NZ:
		return "JMP_NZ"
	case ADD:
		return "ADD"
	case SUB:
		return "SUB"
	case MUL:
		return "MUL"
	case DIV:
		return "DIV"
	case INC:
		return "INC"
	case DEC:
		return "DEC"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case XOR:
		return "XOR"
	case STR_STORE:
		return "STR_STORE"
	case STR_PRINT:
		return "STR_PRINT"
	case CONCAT:
		return "CONCAT"
	case SYSTEM:
		return "SYSTEM"
	case STR_TO_INT:
		return "STR_TO_INT"
	case CMP_REG:
		return "CMP_REG"
	case CMP_INT:
		return "CMP_INT"
	case CMP_STR:
		return "CMP_STR"
	case IS_INT:
		return "IS_INT"
	case IS_STR:
		return "IS_STR"
	case NOP:
		return "NOP"
	case REG_STORE:
		return "REG_STORE"
	case PEEK:
		return "PEEK"
	case POKE:
		return "POKE"
	case MEM_CPY:
		return "MEM_CPY"
	case PUSH:
		return "PUSH"
	case POP:
		return "POP"
	case CALL:
		return "CALL"
	case RET:
		return "RET"
	case TRAP:
		return "TRAP"
	default:
		return "unknown opcode"
	}
}

// Value returns the byte-value of the opcode
func (o *Opcode) Value() byte {
	return o.instruction
}
