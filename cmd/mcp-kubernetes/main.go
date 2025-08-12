package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/mcp-kubernetes/pkg/config"
	"github.com/Azure/mcp-kubernetes/pkg/server"
	"github.com/Azure/mcp-kubernetes/pkg/version"
)

func main() {
	// Create configuration instance and parse command line arguments
	cfg := config.NewConfig()
	if err := cfg.ParseFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Create validator and run validation checks
	v := config.NewValidator(cfg)
	if !v.Validate() {
		fmt.Fprintln(os.Stderr, "Validation failed:")
		v.PrintErrors()
		os.Exit(1)
	}

	// Initialize telemetry
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize telemetry service in config
	cfg.InitializeTelemetry(ctx, "mcp-kubernetes", version.GetVersion())

	// Ensure telemetry shutdown on exit
	defer func() {
		if cfg.TelemetryService != nil {
			if err := cfg.TelemetryService.Shutdown(context.Background()); err != nil {
				log.Printf("Failed to shutdown telemetry: %v", err)
			}
		}
	}()

	// Create and initialize the service
	service := server.NewService(cfg)
	if err := service.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}

	// Start service in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- service.Run()
	}()

	// Wait for shutdown signal or service error
	select {
	case <-sigChan:
		cancel()
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Service error: %v\n", err)
		}
	}
}
