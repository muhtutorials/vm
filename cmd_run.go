package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"os"
	"vm/compiler"
	"vm/cpu"
	"vm/lexer"
)

type runCmd struct{}

func (*runCmd) Name() string { return "run" }

func (*runCmd) Synopsis() string { return "Run the given source program." }

func (*runCmd) Usage() string {
	return `run:
Run subcommand compiles the given source program and then executes it immediately.
`
}

func (*runCmd) SetFlags(f *flag.FlagSet) {}

func (*runCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	for _, file := range f.Args() {
		input, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("error reading %s: %s", file, err.Error())
			return subcommands.ExitFailure
		}

		l := lexer.New(string(input))

		comp := compiler.New(l)
		comp.Compile()

		c := cpu.NewCPU()
		c.LoadBytes(comp.Output())

		if err = c.Run(); err != nil {
			fmt.Println("error running file:", err)
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}
