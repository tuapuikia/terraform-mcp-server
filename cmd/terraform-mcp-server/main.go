// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/terraform-mcp-server/version"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const KEEP_ALIVE_INTERVAL = 25 * time.Second

type sessionManager struct {
	sessions map[string]context.CancelFunc
	logger   *log.Logger
}

func newSessionManager(logger *log.Logger) *sessionManager {
	return &sessionManager{
		sessions: make(map[string]context.CancelFunc),
		logger:   logger,
	}
}

func (sm *sessionManager) startKeepAlive(sessionId string, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	sm.sessions[sessionId] = cancel

	go func() {
		ticker := time.NewTicker(KEEP_ALIVE_INTERVAL)
		defer ticker.Stop()
		defer func() {
			sm.logger.Infof("[Keep-Alive] Stopping for session: %s", sessionId)
			delete(sm.sessions, sessionId)
		}()

		sm.logger.Infof("[Keep-Alive] Started for session: %s", sessionId)

		for {
			select {
			case <-ticker.C:
				// Check if the connection is still alive before writing
				if flusher, ok := w.(http.Flusher); ok {
					if r.Method == http.MethodGet {
						// SSE keep-alive
				sm.logger.Infof("[Keep-Alive] Sending SSE ping for session: %s", sessionId)
                // Generate a unique ID for the SSE ping event
                id := fmt.Sprintf("_GET_stream_%d_%s", time.Now().UnixNano()/int64(time.Millisecond), randomString(10))
                pingMessage := fmt.Sprintf("event: message\nid: %s\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"ping\"}\n\n", id)
                sm.logger.Infof("[Keep-Alive] Sending SSE JSON-RPC ping for session: %s, ID: %s", sessionId, id)
                _, err := w.Write([]byte(pingMessage))
				if err != nil {
							sm.logger.WithError(err).Warnf("[Keep-Alive] Failed to write SSE keep-alive for session %s, stopping.", sessionId)
							return
						}
						flusher.Flush()
					} else if r.Method == http.MethodPost {
						// JSON-RPC ping for POST requests
						// This assumes the underlying StreamableHTTPServer can handle raw writes
						// or that a JSON-RPC ping is acceptable.
						// A more robust solution would involve the mcp-go/server library exposing a ping method.
                        pingMessage := []byte(`{"jsonrpc":"2.0","method":"ping"}` + "\n")
                        sm.logger.Infof("[Keep-Alive] Sending JSON-RPC ping for session: %s", sessionId)
                        _, err := w.Write(pingMessage)
                        if err != nil {
							sm.logger.WithError(err).Warnf("[Keep-Alive] Failed to write JSON-RPC ping for session %s, stopping.", sessionId)
							return
						}
						flusher.Flush()
					}
				} else {
					sm.logger.Warnf("[Keep-Alive] http.ResponseWriter does not implement http.Flusher for session %s, stopping keep-alive.", sessionId)
					return
				}
			case <-ctx.Done():
				sm.logger.Infof("[Keep-Alive] Context done for session: %s", sessionId)
				return
			}
		}
	}()
}

func (sm *sessionManager) stopKeepAlive(sessionId string) {
	if cancel, ok := sm.sessions[sessionId]; ok {
		cancel()
	}
}

var (
	rootCmd = &cobra.Command{
		Use:     "terraform-mcp-server",
		Short:   "Terraform MCP Server",
		Long:    `A Terraform MCP server that handles various tools and resources.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version.GetHumanVersion(), version.GitCommit, version.BuildDate),
		Run:     runDefaultCommand,
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		Run: func(_ *cobra.Command, _ []string) {
			logFile, err := rootCmd.PersistentFlags().GetString("log-file")
			if err != nil {
				stdlog.Fatal("Failed to get log file:", err)
			}
			logger, err := initLogger(logFile)
			if err != nil {
				stdlog.Fatal("Failed to initialize logger:", err)
			}

			if err := runStdioServer(logger); err != nil {
				stdlog.Fatal("failed to run stdio server:", err)
			}
		},
	}

	streamableHTTPCmd = &cobra.Command{
		Use:   "streamable-http",
		Short: "Start StreamableHTTP server",
		Long:  `Start a server that communicates via StreamableHTTP transport on port 8080 at /mcp endpoint.`,
		Run: func(cmd *cobra.Command, _ []string) {
			logFile, err := rootCmd.PersistentFlags().GetString("log-file")
			if err != nil {
				stdlog.Fatal("Failed to get log file:", err)
			}
			logger, err := initLogger(logFile)
			if err != nil {
				stdlog.Fatal("Failed to initialize logger:", err)
			}

			port, err := cmd.Flags().GetString("transport-port")
			if err != nil {
				stdlog.Fatal("Failed to get streamableHTTP port:", err)
			}
			host, err := cmd.Flags().GetString("transport-host")
			if err != nil {
				stdlog.Fatal("Failed to get streamableHTTP host:", err)
			}

			if err := runHTTPServer(logger, host, port); err != nil {
				stdlog.Fatal("failed to run streamableHTTP server:", err)
			}
		},
	}
	
	// Create an alias for backward compatibility
	httpCmdAlias = &cobra.Command{
		Use:        "http",
		Short:      "Start StreamableHTTP server (deprecated, use 'streamable-http' instead)",
		Long:       `This command is deprecated. Please use 'streamable-http' instead.`,
		Deprecated: "Use 'streamable-http' instead",
		Run: func(cmd *cobra.Command, args []string) {
			// Forward to the new command
			streamableHTTPCmd.Run(cmd, args)
		},
	}
)

func randomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func runHTTPServer(logger *log.Logger, host string, port string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hcServer := NewServer(version.Version)
	registryInit(hcServer, logger)

	return streamableHTTPServerInit(ctx, hcServer, logger, host, port)
}

func streamableHTTPServerInit(ctx context.Context, hcServer *server.MCPServer, logger *log.Logger, host string, port string) error {
	// Check if stateless mode is enabled
	isStateless := shouldUseStatelessMode()
	
	// Create StreamableHTTP server which implements the new streamable-http transport
	// This is the modern MCP transport that supports both direct HTTP responses and SSE streams
	opts := []server.StreamableHTTPOption{
		server.WithEndpointPath("/mcp"), // Default MCP endpoint path
		server.WithLogger(logger),
		server.WithHeartbeatInterval(30*time.Second), // Add heartbeat interval
	}

	// Only add the WithStateLess option if stateless mode is enabled
	// TODO: fix this in mcp-go ver 0.33.0 or higher
	if isStateless {
		opts = append(opts, server.WithStateLess(true))
		logger.Infof("Running in stateless mode")
	} else {
		logger.Infof("Running in stateful mode (default)")
	}

	baseStreamableServer := server.NewStreamableHTTPServer(hcServer, opts...)

	// Load CORS configuration
	corsConfig := LoadCORSConfigFromEnv()
	
	// Log CORS configuration
	logger.Infof("CORS Mode: %s", corsConfig.Mode)
	if len(corsConfig.AllowedOrigins) > 0 {
		logger.Infof("Allowed Origins: %s", strings.Join(corsConfig.AllowedOrigins, ", "))
	} else if corsConfig.Mode == "strict" {
		logger.Warnf("No allowed origins configured in strict mode. All cross-origin requests will be rejected.")
	} else if corsConfig.Mode == "development" {
		logger.Infof("Development mode: localhost origins are automatically allowed")
	} else if corsConfig.Mode == "disabled" {
		logger.Warnf("CORS validation is disabled. This is not recommended for production.")
	}
	
	// Create a security wrapper around the streamable server
	streamableServer := NewSecurityHandler(baseStreamableServer, corsConfig.AllowedOrigins, corsConfig.Mode, logger)

	mux := http.NewServeMux()

	// Initialize session manager for keep-alive pings
	sm := newSessionManager(logger)

		// Wrap the streamableServer with keep-alive logic
	keepAliveHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE if it's a GET request
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			// Send an initial keep-alive to establish the connection
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				logger.WithError(err).Warn("Failed to write initial SSE keep-alive.")
				return // Stop processing if initial write fails
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		} else if r.Method == http.MethodPost {
			// Set headers for POST requests to ensure keep-alive
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Cache-Control", "no-cache")
		}

		// Extract session ID from header for existing sessions
		sessionId := r.Header.Get("mcp-session-id")
		if sessionId == "" {
			// If no session ID is provided, generate one.
			// This is crucial for initial POST requests to enable keep-alive.
			sessionId = randomString(32) // Generate a unique session ID
			w.Header().Set("mcp-session-id", sessionId)
			logger.Infof("Generated new session ID: %s for %s request", sessionId, r.Method)
		}

		// Start keep-alive for new sessions or existing SSE connections
		if _, exists := sm.sessions[sessionId]; !exists {
			sm.startKeepAlive(sessionId, w, r)
		}

		// Call the original handler
		streamableServer.ServeHTTP(w, r)

		// Stop keep-alive when the request is done (for non-streaming requests)
		// For streaming requests (SSE), the goroutine will handle its own shutdown
		// based on context cancellation or write errors.
		if sessionId != "" && r.Method == http.MethodPost { // Only stop for non-streaming POST requests
			sm.stopKeepAlive(sessionId)
		}
	})

	// Handle the /mcp endpoint with the keep-alive wrapper
	mux.Handle("/mcp", keepAliveHandler)
	mux.Handle("/mcp/", keepAliveHandler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"terraform-mcp-server","transport":"streamable-http"}`))
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       6 * time.Hour,
		ReadHeaderTimeout: 6 * time.Hour,
		WriteTimeout:      6 * time.Hour,
		IdleTimeout:       6 * time.Hour,
	}

	// Start server in goroutine
	errC := make(chan error, 1)
	go func() {
		logger.Infof("Starting StreamableHTTP server on %s/mcp", addr)
		errC <- httpServer.ListenAndServe()
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		logger.Infof("Shutting down StreamableHTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Stop all active keep-alive goroutines on server shutdown
		for sessionId := range sm.sessions {
			sm.stopKeepAlive(sessionId)
		}
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errC:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("StreamableHTTP server error: %w", err)
		}
	}

	return nil
}

func runStdioServer(logger *log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hcServer := NewServer(version.Version)
	registryInit(hcServer, logger)

	return serverInit(ctx, hcServer, logger)
}

func NewServer(version string, opts ...server.ServerOption) *server.MCPServer {
	// Add default options
	defaultOpts := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	}
	opts = append(defaultOpts, opts...)

	// Create a new MCP server
	s := server.NewMCPServer(
		"terraform-mcp-server",
		version,
		opts...,
	)
	return s
}

// runDefaultCommand handles the default behavior when no subcommand is provided
func runDefaultCommand(cmd *cobra.Command, _ []string) {
	// Default to stdio mode when no subcommand is provided
	logFile, err := cmd.PersistentFlags().GetString("log-file")
	if err != nil {
		stdlog.Fatal("Failed to get log file:", err)
	}
	logger, err := initLogger(logFile)
	if err != nil {
		stdlog.Fatal("Failed to initialize logger:", err)
	}

	if err := runStdioServer(logger); err != nil {
		stdlog.Fatal("failed to run stdio server:", err)
	}
}

func main() {
	// Check environment variables first - they override command line args
	if shouldUseStreamableHTTPMode() {
		port := getHTTPPort()
		host := getHTTPHost()

		logFile, _ := rootCmd.PersistentFlags().GetString("log-file")
		logger, err := initLogger(logFile)
		if err != nil {
			stdlog.Fatal("Failed to initialize logger:", err)
		}

		if err := runHTTPServer(logger, host, port); err != nil {
			stdlog.Fatal("failed to run StreamableHTTP server:", err)
		}
		return
	}

	// Fall back to normal CLI behavior
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// shouldUseStreamableHTTPMode checks if environment variables indicate HTTP mode
func shouldUseStreamableHTTPMode() bool {
	transportMode := os.Getenv("TRANSPORT_MODE")
	return transportMode == "http" || transportMode == "streamable-http" || 
	       os.Getenv("TRANSPORT_PORT") != "" || 
	       os.Getenv("TRANSPORT_HOST") != ""
}

// shouldUseStatelessMode returns true if the MCP_SESSION_MODE environment variable is set to "stateless"
func shouldUseStatelessMode() bool {
	mode := strings.ToLower(os.Getenv("MCP_SESSION_MODE"))
	
	// Explicitly check for "stateless" value
	if mode == "stateless" {
		return true
	}
	
	// All other values (including empty string, "stateful", or any other value) default to stateful mode
	return false
}

// getHTTPPort returns the port from environment variables or default
func getHTTPPort() string {
	if port := os.Getenv("TRANSPORT_PORT"); port != "" {
		return port
	}
	return "8080"
}

// getHTTPHost returns the host from environment variables or default
func getHTTPHost() string {
	if host := os.Getenv("TRANSPORT_HOST"); host != "" {
		return host
	}
	return "127.0.0.1"
}
