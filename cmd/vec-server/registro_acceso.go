package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria/diagnostico"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria/medidorpg"
	"vec-diputacion-granada/internal/vec/ports"
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
func montarRegistroAcceso(srv *http.Server, cfg config.Config, registro io.Writer) (func(), *telemetria.Registro) {
	umbrales, valido := telemetria.UmbralesDeEntorno(os.Getenv)
	if !valido {
		escribirRegistroFijo(registro, "vec-server: umbrales de telemetria no validos; se usan los predeterminados\n")
	}
	reg, cerrar := telemetria.MontarEnServidor(srv, telemetria.Opciones{
		Destino:    registro,
		Servicio:   "vec-server",
		Superficie: superficieServidor(cfg),
		Entorno:    entornoSupervision(),
		Version:    revisionCompilada(),
		Umbrales:   umbrales,
	}, registro)
	return cerrar, reg
}

// montarDiagnostico abre, solo si Sistemas lo configura, la superficie de
// métricas y perfiles en bucle local con token (paquete diagnostico).
func montarDiagnostico(reg *telemetria.Registro, emisor ports.ConsultaMetricasEmisionIncidencias, registro io.Writer) func() {
	fuentes := []diagnostico.Fuente{
		reg.EscribirMetricas,
		func(w io.Writer) { medidorpg.EscribirMetricas(w, "vec-server") },
		func(w io.Writer) { escribirMetricasIncidencias(w, emisor) },
	}
	return diagnostico.MontarDesdeEntorno(os.Getenv, "vec-server", fuentes, registro)
}

func escribirMetricasIncidencias(w io.Writer, emisor ports.ConsultaMetricasEmisionIncidencias) {
	if emisor == nil {
		return
	}
	m := emisor.MetricasEmision()
	fmt.Fprintf(w, "# HELP vec_incidencias_total Incidencias técnicas por destino.\n# TYPE vec_incidencias_total counter\n"+
		"vec_incidencias_total{servicio=\"vec-server\",estado=\"escritas\"} %d\n"+
		"vec_incidencias_total{servicio=\"vec-server\",estado=\"descartadas\"} %d\n"+
		"vec_incidencias_total{servicio=\"vec-server\",estado=\"fallos_escritura\"} %d\n",
		m.Escritas, m.Descartadas, m.FallosEscritura)
}
