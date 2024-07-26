//
// This file contains the callbacks that the virtual machine
// can implement via the "int" operation.
//

package cpu

// TrapFunction is a function that is available as a trap
type TrapFunction func(c *CPU, num int) error

// TRAPS is an array of trap function
var TRAPS [maxMemSize]TrapFunction
