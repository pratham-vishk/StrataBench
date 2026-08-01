package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pratham-vishk/stratabench/internal/mcp"
	"github.com/pratham-vishk/stratabench/internal/paths"
)

func main() {
	log.SetOutput(os.Stderr)
	dataDir := os.Getenv("STRATABENCH_DATA")
	if dataDir == "" {
		dataDir = paths.DataDir()
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := mcp.ServeStdio(ctx, &mcp.Tools{DataDir: dataDir}); err != nil {
		log.Fatal(err)
	}
}
