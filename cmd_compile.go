package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"os"
	"path/filepath"
	"strings"
	"vm/compiler"
	"vm/lexer"
)

type compileCmd struct{}

func (*compileCmd) Name() string { return "compile" }

func (*compileCmd) Synopsis() string { return "Compile a simple VM program." }

func (*compileCmd) Usage() string {
	return `compile:
compile the given input file into bytecode.
`
}

func (*compileCmd) SetFlags(f *flag.FlagSet) {}

func (*compileCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	for _, file := range f.Args() {
		input, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("error reading %s: %s", file, err.Error())
			return subcommands.ExitFailure
		}

		l := lexer.New(string(input))

		c := compiler.New(l)
		c.Compile()

		// remove original extension
		name := strings.TrimSuffix(file, filepath.Ext(file))

		// add new extension and write
		c.WriteFile(name + ".raw")
	}
	return subcommands.ExitSuccess
}
