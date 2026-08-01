package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pratham-vishk/stratabench/internal/operator"
)

func main() {
	ns := envOr("STRATABENCH_NAMESPACE", "stratabench")
	interval := 30 * time.Second
	if v := os.Getenv("STRATABENCH_OPERATOR_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	rec, err := operator.New(operator.Config{Namespace: ns, ResyncEvery: interval})
	if err != nil {
		log.Fatal(err)
	}
	defer rec.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("stratabench-operator starting (namespace=%s)", ns)
	if err := rec.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
