package cpu

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"vm/opcode"
)

// ramSize max available memory
const ramSize = 0xffff

type Flags struct {
	// zero flag
	z bool
}

// CPU is our virtual machine state
type CPU struct {
	// registers
	regs [15]*Register

	flags Flags

	// RAM (where the program is loaded)
	mem [ramSize]byte

	// instruction pointer
	ip int

	stack *Stack

	// context is used by callers to implement timeouts
	ctx context.Context

	// STDIN is an input reader used for the input trap
	STDIN *bufio.Reader

	// STDOUT is the writer used for outputting
	STDOUT *bufio.Writer
}

func NewCPU() *CPU {
	cpu := &CPU{ctx: context.Background()}
	cpu.Reset()

	// allow reading from STDIN
	cpu.STDIN = bufio.NewReader(os.Stdin)

	// set standard output for STDOUT
	cpu.STDOUT = bufio.NewWriter(os.Stdout)

	return cpu
}

// Reset sets the CPU into its initial state by setting registers, IP
// and stack back to zero values.
func (c *CPU) Reset() {
	// reset registers
	for i := 0; i < len(c.regs); i++ {
		c.regs[i] = NewRegister()
	}

	// reset instruction pointer
	c.ip = 0

	// reset stack
	c.stack = NewStack()
}

// ReadFile reads the program from the named file into RAM.
// NOTE: The CPU state is reset prior to the load.
func (c *CPU) ReadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %s - %s", path, err.Error())
	}

	if len(data) >= ramSize {
		return fmt.Errorf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes",
			ramSize, len(data))
	}

	c.LoadBytes(data)
	return nil
}

// LoadBytes populates the given program into RAM.
// NOTE: The CPU state is reset prior to the load.
func (c *CPU) LoadBytes(data []byte) {
	c.Reset()

	if len(data) >= ramSize {
		fmt.Printf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes",
			ramSize, len(data))
	}

	// copy contents of file to our memory
	copy(c.mem[:], data)
}

// readInt reads a two byte number from the current IP.
// i.e this reads two bytes and returns a 16-bit value to the caller,
// skipping over both bytes in the IP.
func (c *CPU) readInt() int {
	// remainder
	r := int(c.mem[c.ip])
	c.ip++
	// quotient
	q := int(c.mem[c.ip])
	c.ip++
	return r + q*256
}

// readStr reads a string from the IP position.
// String is prefixed by its lengths (16-bit value contained in two bytes).
func (c *CPU) readStr() (string, error) {
	// read the length of the string
	strLen := c.readInt()

	// can't read beyond RAM but wrap-around will be allowed
	if strLen >= ramSize {
		return "", fmt.Errorf(
			"string is too large for memory: RAM size => %d bytes, string size => %d bytes",
			ramSize, strLen)
	}

	// build the string
	ip := c.ip
	str := ""
	for i := 0; i < strLen; i++ {
		tmpIP := ip + i
		// wrap around
		// todo: Why don't we break here?
		if tmpIP == ramSize {
			tmpIP = 0
		}
		str += string(c.mem[tmpIP])
	}

	// move the IP over the length of the string
	c.ip += strLen

	return str, nil
}

// Run launches the interpreter.
// It does not terminate until an EXIT instruction.
func (c *CPU) Run() error {
	run := true
	for run {
		if c.ip >= ramSize {
			return fmt.Errorf("reading beyond RAM")
		}

		op := opcode.NewOpcode(c.mem[c.ip])
		debugPrintf("%04X %02X [%s]\n", c.ip, op.Value(), op.String())
	}
}
