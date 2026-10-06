package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/publica"
)

func main() {
	if err := ejecutar(); err != nil {
		log.Fatal(err)
	}
}

func ejecutar() error {
	// Esta raíz no compone reglas de ejemplo: declarar cualquier catálogo
	// impide arrancar. Fuera de la doble llave de desarrollo se rechaza como en
	// el resto de raíces; con ella tampoco se ignora, porque aquí no se usaría.
	if err := config.Load().RechazarReglasEjemploSinComposicion(); err != nil {
		return err
	}
	cfg := publica.CargarConfiguracion()
	servidor, err := publica.NuevoServidor(cfg)
	if err != nil {
		return fmt.Errorf("componer servidor publico: %w", err)
	}

	if cfg.CertificadoTLS != "" || cfg.ClaveTLS != "" {
		if cfg.CertificadoTLS == "" || cfg.ClaveTLS == "" {
			return errors.New("servir TLS: VEC_TLS_CERT_FILE y VEC_TLS_KEY_FILE deben configurarse juntos")
		}
		log.Printf("servidor publico VEC escuchando con TLS en %s", servidor.Addr)
		err = servidor.ListenAndServeTLS(cfg.CertificadoTLS, cfg.ClaveTLS)
	} else {
		log.Printf("servidor publico VEC escuchando en %s", servidor.Addr)
		err = servidor.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servir: %w", err)
	}
	return nil
}
