package main

import (
	"errors"
	"log"
	"net/http"

	"vec-diputacion-granada/internal/app/composicion/publica"
)

func main() {
	cfg := publica.CargarConfiguracion()
	servidor, err := publica.NuevoServidor(cfg)
	if err != nil {
		log.Fatalf("componer servidor publico: %v", err)
	}

	if cfg.CertificadoTLS != "" || cfg.ClaveTLS != "" {
		if cfg.CertificadoTLS == "" || cfg.ClaveTLS == "" {
			log.Fatal("servir TLS: VEC_TLS_CERT_FILE y VEC_TLS_KEY_FILE deben configurarse juntos")
		}
		log.Printf("servidor publico VEC escuchando con TLS en %s", servidor.Addr)
		err = servidor.ListenAndServeTLS(cfg.CertificadoTLS, cfg.ClaveTLS)
	} else {
		log.Printf("servidor publico VEC escuchando en %s", servidor.Addr)
		err = servidor.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("servir: %v", err)
	}
}
