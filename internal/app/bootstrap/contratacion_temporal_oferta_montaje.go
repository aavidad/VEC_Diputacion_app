package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

// La configuración y la autoridad de la oferta son optativas. Un fallo deja
// esta función cerrada y conserva el arranque y la selección singular.
func configurarSeleccionOfertaOpcionalCT(cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	seleccion *ejecutorSeleccionLlamamientoDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo, fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) func() {
	if seleccion == nil || seleccion.puente == nil || alta == nil ||
		!alta.postgresql.seleccionOferta {
		return func() {}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelar()
	autoridad, cerrar, disponible := nuevaAutoridadSeleccionOfertaCTDesarrollo(ctx, cfg,
		alta, base, fronteras, derivador, reloj)
	if cerrar == nil {
		cerrar = func() {}
	}
	if !disponible {
		cerrar()
		return func() {}
	}
	registro, err := postgresct.NuevoRegistroSeleccionOfertaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		cerrar()
		slog.Warn("seleccion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	resolucion, err := postgresbolsa.NuevoRepositorioResolucionIntencionOfertaCTPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		cerrar()
		slog.Warn("recuperacion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	publicador := publicadorOfertaBolsaCTDesarrollo{puente: seleccion.puente,
		resolucion: resolucion, autoridadCierre: autoridad}
	descarte, err := postgresct.NuevoRegistroDescarteSeleccionOfertaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		cerrar()
		slog.Warn("descarte de oferta de CT no compuesto", "error", err)
		return func() {}
	}
	servicio, err := application.NuevoServicioSeleccionOfertaCTRecuperable(
		preparadorSeleccionOfertaCTDesarrollo{puente: seleccion.puente, autoridad: autoridad},
		autoridad, registro, publicador, publicador, autoridad, descarte, reloj)
	if err != nil {
		cerrar()
		slog.Warn("seleccion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	seleccion.ofertas = servicio
	return cerrar
}
