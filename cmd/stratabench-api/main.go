package main

import (
	"log"
	"net/http"
	"os"

	"github.com/pratham-vishk/stratabench/internal/api"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
)

func main() {
	listen := envOr("STRATABENCH_API_LISTEN", ":8080")
	svc, err := orchestrator.NewService(paths.DataDir())
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	srv := &api.Server{Svc: svc}
	log.Printf("stratabench-api listening on %s (metrics at /metrics)", listen)
	log.Fatal(http.ListenAndServe(listen, srv.Handler()))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
