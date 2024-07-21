// Package token contains the list of token-types that are accepted/recognized
package token

type Type string

// Token struct represent the lexer token
type Token struct {
	Type    Type
	Literal string
}

// pre-defined types
const (
	COMMA   = "COMMA"
	STR     = "STR"
	LABEL   = "LABEL"
	EOF     = "EOF"
	INT     = "INT"
	ILLEGAL = "ILLEGAL"
	IDENT   = "IDENT"

	// math
	ADD = "ADD"
	SUB = "SUB"
	MUL = "MUL"
	DIV = "DIV"
	INC = "INC"
	DEC = "DEC"
	AND = "AND"
	OR  = "OR"
	XOR = "XOR"

	// control flow
	CALL   = "CALL"
	RET    = "RET"
	JMP    = "JMP"
	JMP_Z  = "JMP_Z"
	JMP_NZ = "JMP_NZ"

	// stack
	PUSH = "PUSH"
	POP  = "POP"

	// types
	IS_INT     = "IS_INT"
	IS_STR     = "IS_STR"
	INT_TO_STR = "INT_TO_STR"
	STR_TO_INT = "STR_TO_INT"

	// compare
	CMP = "CMP"

	// store
	STORE = "STORE"

	PRINT_INT = "PRINT_INT"
	PRINT_STR = "PRINT_STR"

	// memory
	PEEK = "PEEK"
	POKE = "POKE"

	// misc
	CONCAT  = "CONCAT"
	DATA    = "DATA"
	DB      = "DB"
	EXIT    = "EXIT"
	MEM_CPY = "MEM_CPY"
	NOP     = "NOP"
	RAND    = "RAND"
	SYSTEM  = "SYSTEM"
	TRAP    = "TRAP"
)

// reserved keywords
var keywords = map[string]Type{
	// math
	"add": ADD,
	"sub": SUB,
	"mul": MUL,
	"div": DIV,
	"inc": INC,
	"dec": DEC,
	"and": AND,
	"or":  OR,
	"xor": XOR,

	// control flow
	"call":   CALL,
	"ret":    RET,
	"jmp":    JMP,
	"jmp_z":  JMP_Z,
	"jmp_nz": JMP_NZ,

	// stack
	"push": PUSH,
	"pop":  POP,

	// types
	"is_int":     IS_INT,
	"is_str":     IS_STR,
	"int_to_str": INT_TO_STR,
	"str_to_int": STR_TO_INT,

	// compare
	"cmp": CMP,

	// store
	"store": STORE,

	"print_int": PRINT_INT,
	"print_str": PRINT_STR,

	// memory
	"peek": PEEK,
	"poke": POKE,

	// misc
	"concat": CONCAT,
	"data":   DATA,
	"db":     DB,
	"exit":   EXIT,
	"memCpy": MEM_CPY,
	"nop":    NOP,
	"rand":   RAND,
	"system": SYSTEM,
	"trap":   TRAP,
}

// LookupIdentifier determines whether identifier is a keyword nor not
func LookupIdentifier(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
