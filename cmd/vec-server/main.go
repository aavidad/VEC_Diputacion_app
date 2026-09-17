package main

import _ "time/tzdata"

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "importar-convoca" {
		a, err := leerArgumentosImportarConvoca(os.Args[2:], os.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		cfg := config.Load()
		r, err := bootstrap.EjecutarImportacionConvoca(context.Background(), cfg, bootstrap.SolicitudImportacionConvoca{Fichero: a.fichero, Categoria: a.categoria, BolsaRef: a.bolsaRef})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("acta=%s huella=%s leidas=%d aceptadas=%d rechazadas=%d", r.Acta.ActaRef, r.Acta.HuellaFicheroSHA256, r.Acta.FilasLeidas, r.Acta.FilasAceptadas, r.Acta.FilasRechazadas)
		if r.Acta.FilasRechazadas > 0 && !a.admitirRechazos {
			os.Exit(2)
		}
		return
	}
	cfg := config.Load()
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
