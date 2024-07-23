package cpu

import "fmt"

// Object is the interface for something we store in a register
type Object interface {
	Type() string
}

// IntObject is an object holding an integer
type IntObject struct {
	Value int
}

func (IntObject) Type() string {
	return "int"
}

// StrObject is an object holding an integer
type StrObject struct {
	Value string
}

func (StrObject) Type() string {
	return "str"
}

// Register holds the contents of a single register as an object.
// This means it can hold either an IntObject or a StrObject.
type Register struct {
	obj Object
}

func NewRegister() *Register {
	r := &Register{}
	// todo: Why are only integers allowed?
	r.SetInt(0)
	return r
}

// SetInt stores the given integer in the register.
// Note that a register may only contain integers in the range 0x0000-0xffff.
func (r *Register) SetInt(v int) {
	if v <= 0 {
		r.obj = &IntObject{Value: 0}
	} else if v >= 0xffff {
		r.obj = &IntObject{Value: 0xffff}
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
	return 0, fmt.Errorf("attempting to call GetInt on a register holding a non-integer value: %v", r.obj)
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
	return "", fmt.Errorf("attempting to call GetStr on a register holding a non-string value: %v", r.obj)
}

// Type returns the type of register's contents (integer or string)
func (r *Register) Type() string {
	return r.obj.Type()
}
