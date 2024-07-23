// Package compiler contains the compiler for the virtual machine.
//
// It reads the string of tokens from the lexer and outputs the bytecode
// which is equivalent.
//
// The approach to labels:
// Every time we come across a label we output a pair of temporary bytes in our bytecode.
// Later, once we've read the whole program and assume we've found all existing
// labels, we go back up and fix the generated addresses.
//
// This mechanism is the reason for the "fixups" and "labels" maps in the
// Compiler struct. The former keeps track of offsets in our generated
// bytecodes that need to be patched with the address/offset of a given
// label, and the latter lets us record the offset at which labels were seen.
//
//	name  addr
//
// labels map[string]int
//
//	addr name
//
// fixups map[int]string - used by callOp, jumpOp, storeOp
//
// len(bytecode) = 5
// label = ":print"
// labels["print"] = 5
//
// len(bytecode) = 12
// label = "print"
// fixups[12] = "print"
// c.bytecode = append(c.bytecode, byte(0)) // index 12
// c.bytecode = append(c.bytecode, byte(0)) // index 13
//
//		     12  "print"
//		for addr, name := range c.fixups {
//		      5
//			value := c.labels["print"]
//
//			p1 := 5 % 256
//			p2 := 5 / 256
//
//	                 12
//			c.bytecode[addr] = byte(5)
//	                 13
//			c.bytecode[addr+1] = byte(0)
//		}
package compiler

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"vm/lexer"
	"vm/opcode"
	"vm/token"
)

type Compiler struct {
	lexer     *lexer.Lexer
	token     token.Token // current token
	peekToken token.Token // next token
	bytecode  []byte
	labels    map[string]int
	fixups    map[int]string
}

func New(l *lexer.Lexer) *Compiler {
	c := &Compiler{lexer: l}
	c.labels = make(map[string]int)
	c.fixups = make(map[int]string)

	// prime the pump
	c.nextToken()
	c.nextToken()

	return c
}

// nextToken gets the next token from the lexer stream
func (c *Compiler) nextToken() {
	c.token = c.peekToken
	c.peekToken = c.lexer.NextToken()
}

// isRegister returns true if the given string is a register ID (e.g. "#1")
func (c *Compiler) isRegister(input string) bool {
	return strings.HasPrefix(input, "#")
}

// getRegister converts a register string to an integer (e.g. "#2" to 2)
func (c *Compiler) getRegister(input string) byte {
	num := strings.TrimPrefix(input, "#")
	i, err := strconv.Atoi(num)
	if err != nil {
		panic(err)
	}

	if 0 <= i && i <= 15 {
		return byte(i)
	}

	fmt.Printf("register is out of bounds: #%s\n", input)
	os.Exit(1)
	return 0
}

// Compile processes the stream of tokens from the lexer and builds
// up the bytecode program
func (c *Compiler) Compile() {
	// Tokens are processed until the end of the stream (EOF).
	// During this process bytecode is generated.
	for c.token.Type != token.EOF {
		switch c.token.Type {
		case token.LABEL:
			// remove the ":" prefix from the label
			label := strings.TrimPrefix(c.token.Literal, ":")
			// the label points to the current point in our bytecode
			c.labels[label] = len(c.bytecode)
		case token.ADD:
			c.mathOp(opcode.ADD)
		case token.SUB:
			c.mathOp(opcode.SUB)
		case token.MUL:
			c.mathOp(opcode.MUL)
		case token.DIV:
			c.mathOp(opcode.DIV)
		case token.AND:
			c.mathOp(opcode.AND)
		case token.OR:
			c.mathOp(opcode.OR)
		case token.XOR:
			c.mathOp(opcode.XOR)
		case token.INC:
			c.incOp()
		case token.DEC:
			c.decOp()
		case token.CALL:
			c.callOp()
		case token.RET:
			c.retOp()
		case token.JMP:
			c.jumpOp(opcode.JMP)
		case token.JMP_Z:
			c.jumpOp(opcode.JMP_Z)
		case token.JMP_NZ:
			c.jumpOp(opcode.JMP_NZ)
		case token.PUSH:
			c.pushOp()
		case token.POP:
			c.popOp()
		case token.IS_INT:
			c.isIntOp()
		case token.IS_STR:
			c.isStrOp()
		case token.INT_TO_STR:
			c.intToStrOp()
		case token.STR_TO_INT:
			c.strToIntOp()
		case token.CMP:
			c.cmpOp()
		case token.STORE:
			c.storeOp()
		case token.PRINT_INT:
			c.printIntOp()
		case token.PRINT_STR:
			c.printStrOp()
		case token.PEEK:
			c.peekOp()
		case token.POKE:
			c.pokeOp()
		case token.CONCAT:
			c.concatOp()
		case token.DATA:
			c.dataOp()
		case token.DB:
			c.dataOp()
		case token.EXIT:
			c.exitOp()
		case token.MEM_CPY:
			c.memCpyOp()
		case token.NOP:
			c.nopOp()
		case token.RAND:
			c.randOp()
		case token.SYSTEM:
			c.systemOp()
		case token.TRAP:
			c.trapOp()
		default:
			fmt.Println("unhandled token:", c.token)
		}
		c.nextToken()
	}

	for addr, name := range c.fixups {
		value := c.labels[name]
		if value == 0 {
			fmt.Printf("Possible use of undefined label '%s'\n", name)
		}

		p1 := value % 256
		p2 := value / 256

		c.bytecode[addr] = byte(p1)
		c.bytecode[addr+1] = byte(p2)
	}
}

// mathOp handles math operations: add, sub, mul, div, and, or and xor
// e.g. xor #0, #1, #2
func (c *Compiler) mathOp(op int) {
	// check if the next token is an identifier
	// token = XOR
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// token = "#0"
	dst := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}

	// token = ","
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// token = "#1"
	reg1 := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}

	// token = ","
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// token = "#2"
	reg2 := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(op))
	c.bytecode = append(c.bytecode, dst)
	c.bytecode = append(c.bytecode, reg1)
	c.bytecode = append(c.bytecode, reg2)
}

// incOp increments the contents of the given register
// e.g. inc #1
func (c *Compiler) incOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.INC))
	c.bytecode = append(c.bytecode, reg)
}

// decOp decrements the contents of the given register
// e.g. dec #2
func (c *Compiler) decOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.DEC))
	c.bytecode = append(c.bytecode, reg)
}

// callOp generates a call instruction
func (c *Compiler) callOp() {
	// add the call instruction
	c.bytecode = append(c.bytecode, byte(opcode.CALL))

	// advance to the target
	c.nextToken()

	// the call might be to an absolute target or a label
	switch c.token.Type {
	case token.INT:
		addr, _ := strconv.ParseInt(c.token.Literal, 0, 64)
		// len1 (remainder) and len2 (quotient) make up a 16-bit number
		// which gets read and reconstructed (remainder + quotient*256) by the interpreter
		len1 := addr % 256
		len2 := addr / 256

		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))
	case token.IDENT:
		// record that a fixup is needed here
		c.fixups[len(c.bytecode)] = c.token.Literal

		// Output two temporary numbers.
		// Later those bytes will be filled with the label address,
		// which is the bytecode slice index (c.labels[label] = len(c.bytecode).
		c.bytecode = append(c.bytecode, byte(0))
		c.bytecode = append(c.bytecode, byte(0))
	}
}

// retOp returns from a call
func (c *Compiler) retOp() {
	c.bytecode = append(c.bytecode, byte(opcode.RET))
}

// todo: what are jumps?
// jumpOp inserts a direct jump
func (c *Compiler) jumpOp(op int) {
	// add the jump
	c.bytecode = append(c.bytecode, byte(op))

	// advance to the target
	c.nextToken()

	// the jump might be an absolute target or a label
	switch c.token.Type {
	case token.INT:
		addr, _ := strconv.ParseInt(c.token.Literal, 0, 64)
		len1 := addr % 256
		len2 := addr / 256

		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))
	case token.IDENT:
		// record that a fixup is needed here
		c.fixups[len(c.bytecode)] = c.token.Literal

		// Output two temporary numbers.
		// Later those bytes will be filled with the label address,
		// which is the bytecode slice index (c.labels[label] = len(c.bytecode).
		c.bytecode = append(c.bytecode, byte(0))
		c.bytecode = append(c.bytecode, byte(0))
	}
}

// pushOp pushes to the stack
func (c *Compiler) pushOp() {
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// save the register we're storing to
	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.PUSH))
	c.bytecode = append(c.bytecode, reg)
}

// popOp pops from the stack
func (c *Compiler) popOp() {
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// save the register we're storing to
	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.POP))
	c.bytecode = append(c.bytecode, reg)
}

// isIntOp tests if a register contains an integer
func (c *Compiler) isIntOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.IS_INT))
	c.bytecode = append(c.bytecode, reg)
}

// isStrOp tests if a register contains a string
func (c *Compiler) isStrOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.IS_STR))
	c.bytecode = append(c.bytecode, reg)
}

// intToStrOp converts the given int register to a string
func (c *Compiler) intToStrOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.INT_TO_STR))
	c.bytecode = append(c.bytecode, reg)
}

// strToIntOp converts the given string register to an integer
func (c *Compiler) strToIntOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.STR_TO_INT))
	c.bytecode = append(c.bytecode, reg)
}

// cmpOp handles comparing a register with a string, integer, register,
// or label address
// e.g. cmp #1, 44
func (c *Compiler) cmpOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}
	reg := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}
	c.nextToken()

	// now that we know what source register we're comparing we need to see
	// if that comparison is with an integer, string, register value, or a
	// label address
	switch c.token.Type {
	case token.INT:
		c.bytecode = append(c.bytecode, byte(opcode.CMP_INT))
		c.bytecode = append(c.bytecode, reg)

		i, _ := strconv.ParseInt(c.token.Literal, 0, 64)
		len1 := i % 256
		len2 := i / 256

		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))
	case token.STR:
		c.bytecode = append(c.bytecode, byte(opcode.CMP_STR))
		c.bytecode = append(c.bytecode, reg)

		strLen := len(c.token.Literal)
		len1 := strLen % 256
		len2 := strLen / 256

		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))

		// append the string
		for i := 0; i < strLen; i++ {
			c.bytecode = append(c.bytecode, c.token.Literal[i])
		}
	case token.IDENT:
		if c.isRegister(c.token.Literal) {
			c.bytecode = append(c.bytecode, byte(opcode.CMP_REG))
			c.bytecode = append(c.bytecode, reg)
			c.bytecode = append(c.bytecode, c.getRegister(c.token.Literal))
		} else {
			// store the address of a label
			//
			// INT_STORE $REG $NUM1 $NUM2
			c.bytecode = append(c.bytecode, byte(opcode.CMP_INT))
			c.bytecode = append(c.bytecode, reg)

			// record that a fixup is needed here
			c.fixups[len(c.bytecode)] = c.token.Literal

			// Output two temporary numbers.
			// Later those bytes will be filled with the label address,
			// which is the bytecode slice index (c.labels[label] = len(c.bytecode).
			c.bytecode = append(c.bytecode, byte(0))
			c.bytecode = append(c.bytecode, byte(0))
		}
	default:
		fmt.Printf("ERROR: invalid value to compare: %v\n", c.token)
		os.Exit(1)
	}
}

// storeOp stores a string, integer, register, or label address to a register
// e.g. store #2, 16
func (c *Compiler) storeOp() {
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}
	c.nextToken()

	switch c.token.Type {
	case token.INT:
		c.bytecode = append(c.bytecode, byte(opcode.INT_STORE))
		c.bytecode = append(c.bytecode, reg)

		i, _ := strconv.ParseInt(c.token.Literal, 0, 64)
		len1 := i % 256
		len2 := i / 256

		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))
	case token.STR:
		c.bytecode = append(c.bytecode, byte(opcode.STR_STORE))
		c.bytecode = append(c.bytecode, reg)

		strLen := len(c.token.Literal)
		len1 := strLen % 256
		len2 := strLen / 256
		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))

		// append the string
		for i := 0; i < strLen; i++ {
			c.bytecode = append(c.bytecode, c.token.Literal[i])
		}
	case token.IDENT:
		if c.isRegister(c.token.Literal) {
			c.bytecode = append(c.bytecode, byte(opcode.REG_STORE))
			c.bytecode = append(c.bytecode, reg)
			c.bytecode = append(c.bytecode, c.getRegister(c.token.Literal))
		} else {
			// store the address of a label
			//
			// INT_STORE $REG $NUM1 $NUM2
			c.bytecode = append(c.bytecode, byte(opcode.INT_STORE))
			c.bytecode = append(c.bytecode, reg)

			// record that a fixup is needed here
			c.fixups[len(c.bytecode)] = c.token.Literal

			// Output two temporary numbers.
			// Later those bytes will be filled with the label address,
			// which is the bytecode slice index (c.labels[label] = len(c.bytecode).
			c.bytecode = append(c.bytecode, byte(0))
			c.bytecode = append(c.bytecode, byte(0))
		}
	default:
		fmt.Printf("ERROR: invalid value to store: %v\n", c.token)
		os.Exit(1)
	}
}

// printIntOp handles printing the contents of a register as an integer
func (c *Compiler) printIntOp() {
	if !c.checkNextToken(token.IDENT) {
		return
	}

	c.bytecode = append(c.bytecode, byte(opcode.INT_PRINT))
	c.bytecode = append(c.bytecode, c.getRegister(c.token.Literal))
}

// printStrOp handles printing the contents of a register as a string
func (c *Compiler) printStrOp() {
	if !c.checkNextToken(token.IDENT) {
		return
	}

	c.bytecode = append(c.bytecode, byte(opcode.STR_PRINT))
	c.bytecode = append(c.bytecode, c.getRegister(c.token.Literal))
}

// peekOp reads the contents of a memory address and stores in a register
// e.g. peek #0, #1
func (c *Compiler) peekOp() {
	// token = PEEK
	if !c.checkNextToken(token.IDENT) {
		return
	}
	// token = "#0"
	reg := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}

	// token = ","
	if !c.checkNextToken(token.IDENT) {
		return
	}
	// token = "#1"
	addr := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.PEEK))
	c.bytecode = append(c.bytecode, reg)
	c.bytecode = append(c.bytecode, addr)
}

// pokeOp writes to memory
// e.g. poke #1, #2
func (c *Compiler) pokeOp() {
	// token = POKE
	if !c.checkNextToken(token.IDENT) {
		return
	}
	// token = "#1"
	reg := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}

	// token = ","
	if !c.checkNextToken(token.IDENT) {
		return
	}
	// token = "#2"
	addr := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.POKE))
	c.bytecode = append(c.bytecode, reg)
	c.bytecode = append(c.bytecode, addr)
}

// concatOp concatenates two strings
// e.g. concat #1, #3, #4
func (c *Compiler) concatOp() {
	// token = CONCAT
	if !c.checkNextToken(token.IDENT) {
		return
	}

	// token = "#1"
	reg := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}
	// token = ","
	c.nextToken()

	// token = "#3"
	reg1 := c.getRegister(c.token.Literal)

	if !c.checkNextToken(token.COMMA) {
		return
	}
	// token = ","
	c.nextToken()

	// token = "#4"
	reg2 := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.CONCAT))
	c.bytecode = append(c.bytecode, reg)
	c.bytecode = append(c.bytecode, reg1)
	c.bytecode = append(c.bytecode, reg2)
}

// dataOp embeds literal/binary data into the output
func (c *Compiler) dataOp() {
	c.nextToken()

	// data can be a string or a series of integers
	//
	// if it's a string handle it first
	if c.token.Type == token.STR {
		for i := 0; i < len(c.token.Literal); i++ {
			c.bytecode = append(c.bytecode, c.token.Literal[i])
		}
		return
	}

	// otherwise a single integer is expected
	i, _ := strconv.ParseInt(c.token.Literal, 0, 64)
	c.bytecode = append(c.bytecode, byte(i))

	// loop for more data if there's any
	for c.isNextToken(token.COMMA) {
		// skip the comma
		// peekToken = ","
		c.nextToken()
		// token = ","

		// read the next integer
		// peekToken = INT
		if c.checkNextToken(token.INT) {
			// token = INT
			i, _ = strconv.ParseInt(c.token.Literal, 0, 64)
			c.bytecode = append(c.bytecode, byte(i))
		}
	}
}

// exitOp terminates the interpreter
func (c *Compiler) exitOp() {
	c.bytecode = append(c.bytecode, byte(opcode.EXIT))
}

// memCpyOp inserts a memory copy
// e.g. memCpy #1, #2, #3
func (c *Compiler) memCpyOp() {
	c.nextToken()
	reg1 := c.getRegister(c.token.Literal)
	if !c.checkNextToken(token.COMMA) {
		return
	}

	c.nextToken()
	reg2 := c.getRegister(c.token.Literal)
	if !c.checkNextToken(token.COMMA) {
		return
	}

	c.nextToken()
	reg3 := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.MEM_CPY))
	c.bytecode = append(c.bytecode, reg1)
	c.bytecode = append(c.bytecode, reg2)
	c.bytecode = append(c.bytecode, reg3)
}

// nopOp does nothing
func (c *Compiler) nopOp() {
	c.bytecode = append(c.bytecode, byte(opcode.NOP))
}

// randOp returns a random value
func (c *Compiler) randOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.INT_RAND))
	c.bytecode = append(c.bytecode, reg)
}

// systemOp runs the string command in the given register
func (c *Compiler) systemOp() {
	// check if the next token is an identifier
	if !c.checkNextToken(token.IDENT) {
		return
	}

	reg := c.getRegister(c.token.Literal)

	c.bytecode = append(c.bytecode, byte(opcode.SYSTEM))
	c.bytecode = append(c.bytecode, reg)
}

// todo: what exactly does it do?
// trapOp inserts an interrupt call/trap
func (c *Compiler) trapOp() {
	// advance to the target
	c.nextToken()

	// the jump might be an absolute target or a label
	switch c.token.Type {
	case token.INT:
		addr, _ := strconv.ParseInt(c.token.Literal, 0, 64)

		len1 := addr % 256
		len2 := addr / 256

		c.bytecode = append(c.bytecode, byte(opcode.TRAP))
		c.bytecode = append(c.bytecode, byte(len1))
		c.bytecode = append(c.bytecode, byte(len2))
	default:
		fmt.Println("Fail!")
	}
}

// check next token is t
// success: return true and forward token
// failure: return false and print error
func (c *Compiler) checkNextToken(t token.Type) bool {
	if c.isNextToken(t) {
		c.nextToken()
		return true
	}
	c.nextError(t)
	return false
}

// isNextToken checks if the next token is t
func (c *Compiler) isNextToken(t token.Type) bool {
	return c.peekToken.Type == t
}

func (c *Compiler) nextError(t token.Type) {
	fmt.Printf("expected next token to be %s, got %s instead", t, c.peekToken.Type)
	os.Exit(1)
}

// Dump processes the stream of tokens from the lexer and shows the structure
// of the program
func (c *Compiler) Dump() {
	for c.token.Type != token.EOF {
		fmt.Printf("token: type -> %s, literal -> %s\n", c.token.Type, c.token.Literal)
		c.nextToken()
	}
}

// Output returns the bytecode of the compiled program
func (c *Compiler) Output() []byte {
	return c.bytecode
}

// WriteFile outputs our generated bytecode to the named file
func (c *Compiler) WriteFile(path string) {
	fmt.Printf("Generated bytecode is %d bytes long\n", len(c.bytecode))
	if err := os.WriteFile(path, c.bytecode, 0644); err != nil {
		fmt.Printf("Error writing output file: %s\n", err.Error())
		os.Exit(1)
	}
}
