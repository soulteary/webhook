// Package main provides the webhook config generator WebUI (standalone binary).
// It reuses the configui package so config and static assets have a single source (configui/).
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/soulteary/webhook/internal/configui"
)

const defaultPort = "9080"

func main() {
	port := defaultPort
	if p := os.Getenv("PORT"); p != "" {
		port = strings.TrimSpace(p)
	}
	flagSet := flag.NewFlagSet("config-ui", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	portFlag := flagSet.String("port", port, "HTTP port for the config UI (default "+defaultPort+")")
	_ = flagSet.Parse(os.Args[1:])
	if *portFlag != "" {
		port = strings.TrimSpace(*portFlag)
	}
	if port == "" {
		port = defaultPort
	}

	webhookBaseURL := "http://localhost:" + port
	handler, err := configui.Handler("/", webhookBaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config-ui init: %v\n", err)
		os.Exit(1)
	}

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		}
	}()
	fmt.Printf("Webhook Config UI: http://localhost%s\n", addr)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		os.Exit(1)
	}
}
