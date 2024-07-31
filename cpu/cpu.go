package cpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
	"vm/opcode"
)

// maxMemSize maximum available memory (RAM)
const maxMemSize = 0xffff

type Flags struct {
	// zero flag
	z bool
}

// CPU is the virtual machine's state
type CPU struct {
	// registers
	regs [15]*Register

	flags Flags

	// mem is memory (RAM) where the program is loaded.
	// Loaded program size shouldn't exceed maxMemSize-1,
	// so the last memory byte will always be a "0" and the program can terminate
	// since "0" is the EXIT opcode.
	mem [maxMemSize]byte

	// instruction pointer
	ip int

	stack *Stack

	// context is used by callers to implement timeouts
	ctx context.Context

	// STDIN is an input reader used for the input trap
	STDIN *bufio.Reader

	// STDOUT is the writer used for output
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

// ReadFile reads the program (bytecode) from the named file into RAM.
// NOTE: The CPU state is reset prior to the load.
func (c *CPU) ReadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %s - %s", path, err.Error())
	}

	if len(data) >= maxMemSize {
		return fmt.Errorf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes\n",
			maxMemSize, len(data))
	}

	c.LoadBytes(data)
	return nil
}

// LoadBytes loads the given program into RAM.
// NOTE: The CPU state is reset prior to the load.
func (c *CPU) LoadBytes(data []byte) {
	c.Reset()

	if len(data) >= maxMemSize {
		fmt.Printf(
			"program is too large for memory: RAM size => %d bytes, program size => %d bytes\n",
			maxMemSize, len(data))
	}

	// copy contents of file to our memory
	copy(c.mem[:], data)
	//for i := 0; i < len(data); i++ {
	//	fmt.Printf("%x\n", c.mem[i])
	//}
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
		//fmt.Printf("%s: %x\n", op.String(), op.Value())

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

			if err = c.STDOUT.Flush(); err != nil {
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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

			// Set the zero flag if the result was zero or less.
			// Used during iteration (see examples/concat.in).
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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

			// if the value equals maximum memory size it will wrap around
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

			// if the value equals zero it will wrap around
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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

			if err = c.STDOUT.Flush(); err != nil {
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
				return fmt.Errorf("register [%d] is out of range", a)
			}

			c.ip++
			b := c.mem[c.ip]
			if int(b) >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", b)
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

			toExec := splitCommand(str)
			cmd := exec.Command(toExec[0], toExec[1:]...)

			var (
				out *bytes.Buffer
				er  *bytes.Buffer
			)
			cmd.Stdout = out
			cmd.Stderr = er

			if err = cmd.Run(); err != nil {
				return fmt.Errorf("error invoking system (%s): %s", str, err)
			}

			// stdout
			fmt.Printf("%s\n", out.String())

			// stderr, if non-empty
			if len(er.String()) > 0 {
				fmt.Printf("%s\n", er.String())
			}

		case opcode.STR_TO_INT:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			s, err := c.regs[reg].GetStr()
			if err != nil {
				return err
			}

			i, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("failed to convert string (%s) to int: %s", s, err)
			}

			c.regs[reg].SetInt(i)

			// next instruction
			c.ip++

		case opcode.CMP_INT:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++
			val := c.readInt()

			c.flags.z = false

			if c.regs[reg].Type() == "int" {
				regVal, err := c.regs[reg].GetInt()
				if err != nil {
					return err
				}
				if regVal == val {
					c.flags.z = true
				}
			}

		case opcode.CMP_STR:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++
			val, err := c.readStr()
			if err != nil {
				return err
			}

			c.flags.z = false

			if c.regs[reg].Type() == "str" {
				regVal, err := c.regs[reg].GetStr()
				if err != nil {
					return err
				}
				if regVal == val {
					c.flags.z = true
				}
			}

		case opcode.CMP_REG:
			c.ip++
			reg1 := int(c.mem[c.ip])
			if reg1 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg1)
			}

			c.ip++
			reg2 := int(c.mem[c.ip])
			if reg2 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg2)
			}

			c.flags.z = false

			switch c.regs[reg1].Type() {
			case "int":
				a, err := c.regs[reg1].GetInt()
				if err != nil {
					return err
				}
				b, err := c.regs[reg2].GetInt()
				if err != nil {
					return err
				}
				if a == b {
					c.flags.z = true
				}
			case "str":
				a, err := c.regs[reg1].GetStr()
				if err != nil {
					return err
				}
				b, err := c.regs[reg2].GetStr()
				if err != nil {
					return err
				}
				if a == b {
					c.flags.z = true
				}
			}

			// next instruction
			c.ip++

		case opcode.IS_INT:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++

			if c.regs[reg].Type() == "int" {
				c.flags.z = true
			} else {
				c.flags.z = false
			}

		case opcode.IS_STR:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++

			if c.regs[reg].Type() == "str" {
				c.flags.z = true
			} else {
				c.flags.z = false
			}

		case opcode.NOP:
			c.ip++

		case opcode.REG_STORE:
			c.ip++
			dst := int(c.mem[c.ip])
			if dst >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", dst)
			}

			c.ip++
			src := int(c.mem[c.ip])
			if src >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", src)
			}

			if c.regs[src].Type() == "int" {
				val, err := c.regs[src].GetInt()
				if err != nil {
					return err
				}
				c.regs[dst].SetInt(val)
			} else if c.regs[src].Type() == "str" {
				val, err := c.regs[src].GetStr()
				if err != nil {
					return err
				}
				c.regs[dst].SetStr(val)
			} else {
				return fmt.Errorf("invalid register type")
			}

		case opcode.PEEK:
			c.ip++
			reg1 := int(c.mem[c.ip])
			if reg1 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg1)
			}

			c.ip++
			reg2 := int(c.mem[c.ip])
			if reg2 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg2)
			}

			// get the address from the reg2 register
			addr, err := c.regs[reg2].GetInt()
			if err != nil {
				return err
			}
			if addr >= maxMemSize {
				return fmt.Errorf("address [%d] is out of range", addr)
			}

			// store the contents of the given address
			c.regs[reg1].SetInt(int(c.mem[addr]))
			c.ip++

		case opcode.POKE:
			c.ip++
			reg1 := int(c.mem[c.ip])
			if reg1 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg1)
			}

			c.ip++
			reg2 := int(c.mem[c.ip])
			if reg2 >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg2)
			}

			// reg1 contains value which will be stored to memory (RAM)
			val, err := c.regs[reg1].GetInt()
			if err != nil {
				return err
			}
			if val >= maxMemSize {
				return fmt.Errorf("value [%d] is out of range", val)
			}

			// reg2 contains memory address (bytecode index) where value from reg1 will be stored
			addr, err := c.regs[reg2].GetInt()
			if err != nil {
				return err
			}
			if addr >= maxMemSize {
				return fmt.Errorf("address [%d] is out of range", addr)
			}

			c.mem[addr] = byte(val)

		case opcode.MEM_CPY:
			c.ip++
			dst := int(c.mem[c.ip])
			if dst >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", dst)
			}

			c.ip++
			src := int(c.mem[c.ip])
			if src >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", src)
			}

			c.ip++
			lng := int(c.mem[c.ip])
			if lng >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", lng)
			}

			dstAddr, err := c.regs[dst].GetInt()
			if err != nil {
				return err
			}

			srcAddr, err := c.regs[src].GetInt()
			if err != nil {
				return err
			}

			length, err := c.regs[lng].GetInt()
			if err != nil {
				return err
			}

			i := 0
			for i < length {
				if dstAddr >= maxMemSize {
					dstAddr = 0
				}
				if srcAddr >= maxMemSize {
					srcAddr = 0
				}
				c.mem[dstAddr] = c.mem[srcAddr]
				dstAddr++
				srcAddr++
				i++
			}

			// next instruction
			c.ip++

		case opcode.PUSH:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++

			val, err := c.regs[reg].GetInt()
			if err != nil {
				return err
			}

			c.stack.Push(val)

		case opcode.POP:
			// register
			c.ip++
			reg := int(c.mem[c.ip])
			if reg >= len(c.regs) {
				return fmt.Errorf("register [%d] is out of range", reg)
			}

			c.ip++

			// ensure that the stack isn't empty
			if c.stack.Empty() {
				return fmt.Errorf("stackunderflow")
			}

			// store the value from the stack in the register
			val, _ := c.stack.Pop()
			c.regs[reg].SetInt(val)

		case opcode.CALL:
			c.ip++

			addr := c.readInt()

			// push current IP to the stack
			c.stack.Push(c.ip)

			// jump to the call address
			c.ip = addr
		case opcode.RET:
			// ensure that the stack isn't empty
			if c.stack.Empty() {
				return fmt.Errorf("stackunderflow")
			}

			addr, _ := c.stack.Pop()

			// jump
			c.ip = addr

		case opcode.TRAP:
			c.ip++

			num := c.readInt()

			if num < 0 || num >= maxMemSize {
				return fmt.Errorf("invalid trap number: %d", num)
			}

			fn := TRAPS[num]
			if fn != nil {
				if err := fn(c, num); err != nil {
					return err
				}
			}

		default:
			return fmt.Errorf("unknown opcode %02x at IP %04x", op.Value(), c.ip)
		}

		// ensure that instruction pointer wraps around
		if c.ip > maxMemSize {
			c.ip = 0
		}
	}

	return nil
}
