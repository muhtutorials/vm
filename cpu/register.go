package cpu

import "fmt"

// Object is the interface for a value stored in a register
type Object interface {
	Type() string
}

// IntObject is an object containing an integer
type IntObject struct {
	Value int
}

func (IntObject) Type() string {
	return "int"
}

// StrObject is an object containing a string
type StrObject struct {
	Value string
}

func (StrObject) Type() string {
	return "str"
}

// Register contains the value of a single register as an object.
// This means it can contain either an IntObject or a StrObject.
type Register struct {
	obj Object
}

func NewRegister() *Register {
	r := &Register{}
	r.SetInt(0)
	return r
}

// SetInt stores the given integer in the register.
// Note that a register may only contain integers in the range 0x0000-0xffff.
func (r *Register) SetInt(v int) {
	if v <= 0 {
		r.obj = &IntObject{Value: 0}
	} else if v >= maxMemSize {
		r.obj = &IntObject{Value: maxMemSize}
	} else {
		r.obj = &IntObject{Value: v}
	}
}

// GetInt retrieves the integer of the given register.
// If the register does not contain an integer that is a fatal error.
func (r *Register) GetInt() (int, error) {
	v, ok := r.obj.(*IntObject)
	if ok {
		return v.Value, nil
	}
	return 0, fmt.Errorf("attempting to call GetInt on a register containing a non-integer value: %v", r.obj)
}

// SetStr stores the given string in the register
func (r *Register) SetStr(v string) {
	r.obj = &StrObject{Value: v}
}

// GetStr retrieves the string of the given register.
// If the register does not contain a string that is a fatal error.
func (r *Register) GetStr() (string, error) {
	v, ok := r.obj.(*StrObject)
	if ok {
		return v.Value, nil
	}
	return "", fmt.Errorf("attempting to call GetStr on a register containing a non-string value: %v", r.obj)
}

// Type returns the type of the register's value (integer or string)
func (r *Register) Type() string {
	return r.obj.Type()
}
