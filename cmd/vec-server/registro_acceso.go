package main

import (
	"io"
	"net/http"
	"os"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
)

// superficieServidor nombra el portal que atiende este proceso en el
// registro de acceso: "interno", "externo" o "integrada" si sirve ambos.
func superficieServidor(cfg config.Config) string {
	switch cfg.PortalProceso {
	case config.ValorPortalProcesoInterno, config.ValorPortalProcesoExterno:
		return cfg.PortalProceso
	default:
		return "integrada"
	}
}

// montarRegistroAcceso envuelve el servidor con el registro de acceso
// técnico (una línea JSON por petición en registro). Va antes de la
// supervisión de respuestas, que queda por fuera y aporta la correlación.
func montarRegistroAcceso(srv *http.Server, cfg config.Config, registro io.Writer) func() {
	umbrales, valido := telemetria.UmbralesDeEntorno(os.Getenv)
	if !valido {
		escribirRegistroFijo(registro, "vec-server: umbrales de telemetria no validos; se usan los predeterminados\n")
	}
	_, cerrar := telemetria.MontarEnServidor(srv, telemetria.Opciones{
		Destino:    registro,
		Servicio:   "vec-server",
		Superficie: superficieServidor(cfg),
		Entorno:    entornoSupervision(),
		Version:    revisionCompilada(),
		Umbrales:   umbrales,
	}, registro)
	return cerrar
}
