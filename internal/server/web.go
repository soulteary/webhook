package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/mux"
	"github.com/soulteary/webhook/internal/flags"
	"github.com/soulteary/webhook/internal/link"
	"github.com/soulteary/webhook/internal/middleware"
)

func Launch(appFlags flags.AppFlags, addr string, ln net.Listener) {
	r := mux.NewRouter()

	r.Use(middleware.RequestID(
		middleware.UseXRequestIDHeaderOption(appFlags.UseXRequestID),
		middleware.XRequestIDLimitOption(appFlags.XRequestIDLimit),
	))
	r.Use(middleware.NewLogger())
	r.Use(chimiddleware.Recoverer)

	if appFlags.Debug {
		r.Use(middleware.Dumper(log.Writer()))
	}

	// Clean up input
	appFlags.HttpMethods = strings.ToUpper(strings.ReplaceAll(appFlags.HttpMethods, " ", ""))

	hooksURL := link.MakeRoutePattern(&appFlags.HooksURLPrefix)

	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		setResponseHeaders(w, appFlags.ResponseHeaders)

		fmt.Fprint(w, "OK")
	})

	hookHandler := createHookHandler(appFlags)
	r.HandleFunc(hooksURL, hookHandler)

	// Create common HTTP server settings
	svr := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
	}

	// Serve HTTP
	log.Printf("serving hooks on http://%s%s", addr, link.MakeHumanPattern(&appFlags.HooksURLPrefix))
	log.Print(svr.Serve(ln))
}
