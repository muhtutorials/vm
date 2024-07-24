package cpu

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"
	"vm/opcode"
)

// maxMemSize maximum available RAM
const maxMemSize = 0xffff

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
	mem [maxMemSize]byte

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

	if len(data) >= maxMemSize {
		return fmt.Errorf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes",
			maxMemSize, len(data))
	}

	c.LoadBytes(data)
	return nil
}

// LoadBytes populates the given program into RAM.
// NOTE: The CPU state is reset prior to the load.
func (c *CPU) LoadBytes(data []byte) {
	c.Reset()

	if len(data) >= maxMemSize {
		fmt.Printf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes",
			maxMemSize, len(data))
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
	if strLen >= maxMemSize {
		return "", fmt.Errorf(
			"string is too large for memory: RAM size => %d bytes, string size => %d bytes",
			maxMemSize, strLen)
	}

	// build the string
	ip := c.ip
	str := ""
	for i := 0; i < strLen; i++ {
		tmpIP := ip + i
		// wrap around
		// todo: Why don't we break here?
		if tmpIP == maxMemSize {
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
		if c.ip >= maxMemSize {
			return fmt.Errorf("reading beyond RAM")
		}

		op := opcode.NewOpcode(c.mem[c.ip])

		debugPrintf("%04x %02x [%s]\n", c.ip, op.Value(), op.String())

		// Test context at every iteration.
		// This is a little slow and inefficient, but allows the execution to be time limited.
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("timeout during execution\n")
		default:
			// nop
		}

		switch int(op.Value()) {
		case opcode.EXIT:
			run = false

		case opcode.INT_STORE:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++
			val := c.readInt()
			c.regs[reg].SetInt(val)

		case opcode.INT_PRINT:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			val, err := c.regs[reg].GetInt()
			if err != nil {
				return err
			}
			if val < 256 {
				_, err = c.STDOUT.WriteString(fmt.Sprintf("%02x", val))
				if err != nil {
					return err
				}
			} else {
				_, err = c.STDOUT.WriteString(fmt.Sprintf("%04x", val))
				if err != nil {
					return err
				}
			}

			err = c.STDOUT.Flush()
			if err != nil {
				return err
			}

			// next instruction
			c.ip++

		case opcode.INT_TO_STR:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			i, err := c.regs[reg].GetInt()
			if err != nil {
				return err
			}

			// change from int to string
			c.regs[reg].SetStr(fmt.Sprintf("%d", i))

			// next instruction
			c.ip++

		case opcode.INT_RAND:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			c.regs[reg].SetInt(r.Intn(maxMemSize))
			c.ip++

		case opcode.JMP:
			c.ip++
			addr := c.readInt()
			c.ip = addr

		case opcode.JMP_Z:
			c.ip++
			addr := c.readInt()
			if c.flags.z {
				c.ip = addr
			}

		case opcode.JMP_NZ:
			c.ip++
			addr := c.readInt()
			if !c.flags.z {
				c.ip = addr
			}

		case opcode.ADD:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal + bVal)

		case opcode.SUB:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal - bVal)

			// set the zero flag if the result was zero or less
			resVal, err := c.regs[res].GetInt()
			if err != nil {
				return err
			}
			if resVal <= 0 {
				c.flags.z = true
			}

		case opcode.MUL:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal * bVal)

		case opcode.DIV:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}

			if bVal == 0 {
				return fmt.Errorf("devision by zero")
			}

			c.regs[res].SetInt(aVal / bVal)

		case opcode.INC:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			i, err := c.regs[reg].GetInt()
			if err != nil {
				return err
			}

			// if the value is the max it will wrap around
			if i == maxMemSize {
				i = 0
			} else {
				i++
			}

			c.flags.z = i == 0

			c.regs[reg].SetInt(i)

			c.ip++

		case opcode.DEC:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			i, err := c.regs[reg].GetInt()
			if err != nil {
				return err
			}

			// if the value is the max it will wrap around
			if i == 0 {
				i = maxMemSize
			} else {
				i--
			}

			c.flags.z = i == 0

			c.regs[reg].SetInt(i)

			c.ip++

		case opcode.AND:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal & bVal)

		case opcode.OR:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal | bVal)

		case opcode.XOR:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetInt()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetInt()
			if err != nil {
				return err
			}
			c.regs[res].SetInt(aVal ^ bVal)

		case opcode.STR_STORE:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++
			str, err := c.readStr()
			if err != nil {
				return err
			}

			c.regs[reg].SetStr(str)

		case opcode.STR_PRINT:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			str, err := c.regs[reg].GetStr()
			if err != nil {
				return err
			}

			_, err = c.STDOUT.WriteString(str)
			if err != nil {
				return err
			}

			err = c.STDOUT.Flush()
			if err != nil {
				return err
			}

			// next instruction
			c.ip++

		case opcode.CONCAT:
			c.ip++
			// result
			res := c.mem[c.ip]
			if int(res) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			a := c.mem[c.ip]
			if int(a) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", res)
			}

			c.ip++

			aVal, err := c.regs[a].GetStr()
			if err != nil {
				return err
			}
			bVal, err := c.regs[b].GetStr()
			if err != nil {
				return err
			}
			c.regs[res].SetStr(aVal + bVal)

		case opcode.SYSTEM:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			str, err := c.regs[reg].GetStr()
			if err != nil {
				return err
			}

		}
	}

	return nil
}
