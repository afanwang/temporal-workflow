package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/automaxprocs/maxprocs"
)

func init() {
	maxprocslog := func(format string, args ...any) {
		slog.Info(fmt.Sprintf(format, args...))
	}
	_, _ = maxprocs.Set(maxprocs.Logger(maxprocslog))
}

type command func(context.Context) error

var commands = map[string]command{
	"worker":   RunWorker,
	"pipeline": RunPipeline,
}

func main() {
	if len(os.Args) < 2 {
		help()
		os.Exit(1)
	}

	cmd := commands[os.Args[1]]
	if cmd == nil {
		slog.Error("unknown command", "command", os.Args[1])
		help()
		os.Exit(1)
	}

	if err := cmd(context.Background()); err != nil {
		slog.Error("terminated", "error", err)
		os.Exit(1)
	}
}

func help() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Available commands:\n")
	for cmd := range commands {
		fmt.Fprintf(os.Stderr, "  - %s\n", cmd)
	}
}
