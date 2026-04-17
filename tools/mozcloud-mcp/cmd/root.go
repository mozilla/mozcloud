// Package cmd implements the mozcloud-mcp command-line interface.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/server"
	"github.com/spf13/cobra"
)

// Version is populated at build time via -ldflags "-X .../cmd.Version=...".
var Version string

var (
	transport         string
	allowedWriteRoots []string
)

var rootCmd = &cobra.Command{
	Use:   "mozcloud-mcp",
	Short: "MCP server exposing Helm and mozcloud tooling to AI assistants",
	Long: `mozcloud-mcp is a Model Context Protocol (MCP) server that exposes
Helm chart operations, OCI registry tooling, render-diff diffing, and
mozcloud migration utilities to AI coding assistants such as Claude Code.

Transport modes:
  stdio   — communicate over stdin/stdout (default, for Claude Desktop / Claude Code)
  sse     — HTTP server-sent events (for browser-based clients)`,
	Version: getVersion(),
	RunE:    run,
}

func run(cmd *cobra.Command, args []string) error {
	// All diagnostic output goes to stderr so stdout stays clean for MCP protocol.
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	s := server.New(getVersion(), allowedWriteRoots)

	switch transport {
	case "stdio":
		log.Printf("[mozcloud-mcp] starting stdio transport (version %s)", getVersion())
		srv := mcpserver.NewStdioServer(s)
		listenErr := make(chan error, 1)
		go func() {
			listenErr <- srv.Listen(ctx, os.Stdin, os.Stdout)
		}()
		select {
		case err := <-listenErr:
			if err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("stdio server error: %w", err)
			}
		case <-ctx.Done():
			log.Printf("[mozcloud-mcp] shutting down stdio server")
			// Close stdin to unblock the ReadString goroutine inside Listen.
			if err := os.Stdin.Close(); err != nil {
				log.Printf("[mozcloud-mcp] error closing stdin: %v", err)
			}
			if err := <-listenErr; err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("stdio server error: %w", err)
			}
		}
	case "sse":
		addr := ":8080"
		log.Printf("[mozcloud-mcp] starting SSE transport on %s (version %s)", addr, getVersion())
		srv := mcpserver.NewSSEServer(s)
		serveErr := make(chan error, 1)
		go func() {
			if err := srv.Start(addr); err != nil {
				serveErr <- err
			}
		}()
		select {
		case err := <-serveErr:
			return fmt.Errorf("SSE server error: %w", err)
		case <-ctx.Done():
			log.Printf("[mozcloud-mcp] shutting down SSE server")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("SSE server shutdown error: %w", err)
			}
		}
	default:
		return fmt.Errorf("unknown transport %q: must be stdio or sse", transport)
	}

	return nil
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&transport, "transport", "stdio",
		"Transport mode: stdio or sse")
	rootCmd.Flags().StringSliceVar(&allowedWriteRoots, "allowed-write-roots", nil,
		"Comma-separated list of directory paths that side-effect tools may write into.\n"+
			"Defaults to each tool's own chart_path argument.")
}

// getVersion returns a version string. It prefers the ldflags-injected Version
// (used by goreleaser builds), then falls back to Go module build info (for
// `go install …@vX.Y.Z`), and finally returns "dev".
func getVersion() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}
