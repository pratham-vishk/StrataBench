package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/engine"
)

func sbkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sbk",
		Short: "SBK application-layer driver utilities",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "tools",
		Short: "Probe native SBK drivers on PATH (pgbench, db_bench, kafka-producer-perf-test)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rep := engine.ProbeSBKDrivers()
			for _, d := range rep.Drivers {
				status := "missing"
				if d.Available {
					status = d.Path
				}
				fmt.Printf("%-12s %-28s %s\n", d.Driver, d.Tool, status)
			}
			if rep.AllAvailable {
				fmt.Println("sbk-tools: all native drivers available")
				return nil
			}
			return fmt.Errorf("sbk-tools: one or more native drivers missing")
		},
	})
	return cmd
}
