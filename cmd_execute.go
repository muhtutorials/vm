package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"vm/cpu"
)

type executeCmd struct{}

func (*executeCmd) Name() string { return "execute" }

func (*executeCmd) Synopsis() string { return "Execute a compiled program." }

func (*executeCmd) Usage() string {
	return `execute:
Execute the bytecode contained in the given input file.
`
}

func (*executeCmd) SetFlags(f *flag.FlagSet) {}

func (*executeCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	for _, file := range f.Args() {
		c := cpu.NewCPU()

		if err := c.ReadFile(file); err != nil {
			fmt.Println("error reading file:", err)
		}

		if err := c.Run(); err != nil {
			fmt.Println("error running file:", err)
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}
