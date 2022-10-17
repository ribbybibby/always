package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ribbybibby/always/internal/registry"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	ListenAddress string
}

var ro = &rootOptions{}

var rootCmd = &cobra.Command{
	Use:   "always",
	Short: "A registry mirror that serves the same image for every tag.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		reg, err := registry.NewRegistry(args[0], registry.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("creating new registry: %w", err)
		}

		http.Handle("/", reg)

		log.Printf("Listening on %s", ro.ListenAddress)
		return http.ListenAndServe(ro.ListenAddress, nil)
	},
}

// Execute the root command
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&ro.ListenAddress, "listen-address", ":8080", "Listen address")
}
