package bootstrap

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: subsanacion de reparos de desarrollo no disponible",
)

// fuenteSubsanacionReparosDesarrollo es el límite de composición de la fuente
// gobernada. El bootstrap no fabrica una política: exige que la fuente aporte
// acción, finalidad, motivo, versión y huella verificables por el puerto.
type fuenteSubsanacionReparosDesarrollo interface {
	ports.ResolutorPoliticaSubsanacionReparo
	httpinterno.AutoridadContextoCanalSubsanacionReparos
}

type dependenciasSubsanacionReparosContratacionTemporalDesarrollo struct {
	autoridad httpinterno.AutoridadContextoCanalSubsanacionReparos
	servicio  *application.ServicioSubsanacionReparos
}

type proveedorMaterialSubsanacionReparosDesarrollo struct {
	soporte  *soporteAltaContratacionTemporalDesarrollo
	delegado interface {
		ProveerMaterialConfirmacionSubsanacionReparo(context.Context, ports.OrdenConfirmarSubsanacionReparo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	}
}

func (p proveedorMaterialSubsanacionReparosDesarrollo) ProveerMaterialConfirmacionSubsanacionReparo(ctx context.Context, orden ports.OrdenConfirmarSubsanacionReparo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.soporte == nil || p.delegado == nil || !p.soporte.capacidadSubsanacionVigente(ctx) {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrAutorizacionDenegada
	}
	return p.delegado.ProveerMaterialConfirmacionSubsanacionReparo(ctx, orden)
}

// nuevasDependenciasSubsanacionReparosContratacionTemporalDesarrollo monta la
// operación sólo con una fuente de política explícita y el consumidor atestado
// propio. Una fuente ausente o inválida no produce un manejador utilizable.
func nuevasDependenciasSubsanacionReparosContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	fuente fuenteSubsanacionReparosDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (dependenciasSubsanacionReparosContratacionTemporalDesarrollo, error) {
	vacias := dependenciasSubsanacionReparosContratacionTemporalDesarrollo{}
	if derivador == nil || !derivador.valido() || alta == nil || alta.soporte == nil ||
		alta.autorizador == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.proveedorMaterial == nil || fuente == nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	ambitoActivo, ambitosRetenidos, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, ports.DominioAmbitoIdempotenciaSubsanacionReparo, true,
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	huellaActiva, huellasRetenidas, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, ports.DominioHuellaPeticionSubsanacionReparo, false,
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	sellos, err := seguridadcontratacion.NuevaAutoridadSellosSubsanacionReparoHMAC(
		ambitoActivo, ambitosRetenidos, huellaActiva, huellasRetenidas,
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	preparador, err := postgrescontratacion.NuevoPreparadorSubsanacionReparosPostgreSQL(
		alta.postgresql.ejecucion,
		seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico(),
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	confirmador, err := postgrescontratacion.NuevaTransaccionSubsanacionReparosPostgreSQL(
		alta.postgresql.ejecucion, proveedorMaterialSubsanacionReparosDesarrollo{soporte: alta.soporte, delegado: alta.postgresql.proveedorMaterial},
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := application.NuevoServicioSubsanacionReparos(
		alta.soporte, sellos, sellos, preparador, confirmador, fuente,
		seguridadvec.GeneradorReferenciasCriptograficas{}, alta.autorizador, reloj,
	)
	if err != nil {
		return vacias, errSubsanacionReparosContratacionTemporalDesarrolloNoDisponible
	}
	return dependenciasSubsanacionReparosContratacionTemporalDesarrollo{
		autoridad: fuente, servicio: servicio,
	}, nil
}
