package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "timber-stage-qualifier:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	configuration, err := parseConfig(args)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if configuration.SelfCheck {
		return runSelfCheck(ctx, configuration)
	}
	return runServer(ctx, configuration)
}
