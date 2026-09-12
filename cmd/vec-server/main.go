package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	cfg := config.Load()
	if cfg.Normalize().ExecutionProfile == config.ExecutionProfileDevelopment {
		ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer detener()
		if err := bootstrap.ServeDesarrollo(ctx, cfg, log.Writer()); err != nil {
			log.Fatalf("serve desarrollo: %v", err)
		}
		return
	}
	srv, err := bootstrap.NewHTTPServerWithConfig(cfg)
	if err != nil {
		log.Fatalf("bootstrap server: %v", err)
	}

	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			log.Fatal("serve TLS: VEC_TLS_CERT_FILE and VEC_TLS_KEY_FILE must be configured together")
		}
		log.Printf("vec server listening with TLS on %s", srv.Addr)
		err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		log.Printf("vec server listening on %s", srv.Addr)
		err = srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}
