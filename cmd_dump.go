package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"os"
	"vm/compiler"
	"vm/lexer"
)

type dumpCmd struct{}

func (*dumpCmd) Name() string { return "dump" }

func (*dumpCmd) Synopsis() string { return "Show the lexed output of the given program." }

func (*dumpCmd) Usage() string {
	return `dump:
Show how the lexer performed by dumping the given input file as a stream of tokens.
`
}

func (*dumpCmd) SetFlags(f *flag.FlagSet) {}

func (*dumpCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	for _, file := range f.Args() {
		input, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("error reading %s: %s", file, err.Error())
			return subcommands.ExitFailure
		}

		l := lexer.New(string(input))

		c := compiler.New(l)
		c.Dump()
	}
	return subcommands.ExitSuccess
}
