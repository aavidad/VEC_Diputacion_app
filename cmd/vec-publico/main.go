package main

import (
	"errors"
	"log"
	"net/http"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	cfg := config.Load()
	servidor, err := bootstrap.NewHTTPServerPublicoWithConfig(cfg)
	if err != nil {
		log.Fatalf("bootstrap servidor publico: %v", err)
	}

	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			log.Fatal("servir TLS: VEC_TLS_CERT_FILE y VEC_TLS_KEY_FILE deben configurarse juntos")
		}
		log.Printf("servidor publico VEC escuchando con TLS en %s", servidor.Addr)
		err = servidor.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		log.Printf("servidor publico VEC escuchando en %s", servidor.Addr)
		err = servidor.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("servir: %v", err)
	}
}
