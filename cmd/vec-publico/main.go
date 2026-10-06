package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/publica"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
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

	// Registro de acceso técnico (una línea JSON por petición en stderr).
	umbrales, _ := telemetria.UmbralesDeEntorno(os.Getenv)
	_, cerrarAcceso := telemetria.MontarEnServidor(servidor, telemetria.Opciones{
		Destino:    os.Stderr,
		Servicio:   "vec-publico",
		Superficie: "publica",
		Entorno:    telemetria.EntornoDe(os.Getenv("VEC_ENTORNO"), cfg.PerfilEjecucion),
		Version:    telemetria.RevisionBinario(),
		Umbrales:   umbrales,
	}, os.Stderr)
	defer cerrarAcceso()

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
