package main

import (
	"errors"
	"log"
	"net/http"

	"vec-diputacion-granada/internal/app/composicion/interna"
)

func main() {
	cfg := interna.CargarConfiguracion()
	servidor, err := interna.NuevoServidor(cfg)
	if err != nil {
		log.Fatalf("componer servidor interno: %v", err)
	}
	if servidor == nil || servidor.TLSConfig == nil {
		log.Fatal("componer servidor interno: TLS mutuo no disponible")
	}

	log.Printf("servidor interno VEC escuchando con TLS mutuo en %s", servidor.Addr)
	err = servidor.ListenAndServeTLS("", "")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("servir superficie interna: %v", err)
	}
}
