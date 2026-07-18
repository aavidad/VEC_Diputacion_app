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
	servidor, err := bootstrap.NewHTTPServerPresentacionWithConfig(cfg)
	if err != nil {
		log.Fatalf("componer presentacion RRHH: %v", err)
	}
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		log.Fatal("servir presentacion: certificado y clave TLS deben configurarse juntos")
	}
	log.Printf("PRESENTACION RRHH NO AUTORITATIVA escuchando en %s", servidor.Addr)
	if cfg.TLSCertFile != "" {
		err = servidor.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		err = servidor.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("servir presentacion RRHH: %v", err)
	}
}
