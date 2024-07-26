//
// This file contains the callbacks that the virtual machine
// can implement via the "int" operation
//

package cpu

import (
	"fmt"
	"strings"
)

// TrapFunction is a function that is available as a trap
type TrapFunction func(c *CPU, num int) error

// TRAPS is an array of trap functions
var TRAPS [maxMemSize]TrapFunction

// TrapNOP is the default trap function for any trap IDs that haven't
// explicitly been set up
func TrapNOP(c *CPU, num int) error {
	return fmt.Errorf("trap function not defined: 0x%04x", num)
}

// StrLenTrap returns the length of a string.
//
// Input: the string to measure in register #0.
//
// Output: sets register #0 with the length.
func StrLenTrap(c *CPU, num int) error {
	str, err := c.regs[0].GetStr()
	if err != nil {
		return err
	}
	c.regs[0].SetInt(len(str))
	return nil
}

// ReadStringTrap reads a string from the console.
//
// Input: none.
//
// Output: sets register #0 with the user-provided string.
func ReadStringTrap(c *CPU, num int) error {
	str, err := c.STDIN.ReadString('\n')
	if err != nil {
		return err
	}
	c.regs[0].SetStr(str)
	return nil
}

// RemoveNewLineTrap removes any trailing newline from the string in register #0.
//
// Input: the string in register #0.
//
// Output: sets register #0 with the updated string.
func RemoveNewLineTrap(c *CPU, num int) error {
	str, err := c.regs[0].GetStr()
	if err != nil {
		return err
	}
	c.regs[0].SetStr(strings.TrimSpace(str))
	return nil
}

func init() {
	// default to all traps being "empty", i.e. configured to
	// contain a reference to a function that just reports an error
	for i := 0; i < maxMemSize; i++ {
		TRAPS[i] = TrapNOP
	}

	// set up implemented traps
	TRAPS[0] = StrLenTrap
	TRAPS[1] = ReadStringTrap
	TRAPS[2] = RemoveNewLineTrap
}
