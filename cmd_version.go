package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"io"
	"os"
)

var (
	out     io.Writer = os.Stdout
	version           = "1.0.0"
)

type versionCmd struct{}

func (*versionCmd) Name() string { return "version" }

func (*versionCmd) Synopsis() string { return "Show version." }

func (*versionCmd) Usage() string {
	return `version:
Report version an exit.
`
}

func (*versionCmd) SetFlags(f *flag.FlagSet) {}

func (*versionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	fmt.Fprintf(out, "%s\n", version)
	return subcommands.ExitSuccess
}
