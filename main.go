package main

import (
	"fmt"
	"vm/compiler"
	"vm/lexer"
)

func main() {
	l := lexer.New("inc #1")
	c := compiler.New(l)
	// c.Dump()
	c.Compile()
	fmt.Println(c.Output())
}
