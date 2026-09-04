package bootstrap

import (
	"context"
	"time"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
)

const (
	catalogoMotivosDecisionCoberturaDesarrollo = "motivos_cobertura"
	moduloMotivosDecisionCoberturaDesarrollo   = "contratacion_temporal"
)

type dependenciasCoberturaContratacionTemporalDesarrollo struct {
	presentador *application.ServicioPresentacionPropuestaCobertura
	decisor     *application.ServicioConfirmacionDecisionCobertura
	consultor   *application.ServicioConsultaResultadoCobertura
	cerrar      func()
}

func nuevasDependenciasCoberturaContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (dependenciasCoberturaContratacionTemporalDesarrollo, error) {
	vacias := dependenciasCoberturaContratacionTemporalDesarrollo{}
	if derivador == nil || !derivador.valido() || alta == nil ||
		alta.soporte == nil || alta.autorizador == nil ||
		alta.postgresql.ejecucion == nil ||
		alta.postgresql.confirmador == nil ||
		alta.postgresql.lectorResultado == nil {
		return vacias, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}

	lectorAnalisis, err :=
		postgrescontratacion.NuevoLectorExpedienteAnalisisDurableO3PostgreSQL(
			alta.postgresql.ejecucion,
		)
	if err != nil {
		return vacias, err
	}
	gobierno, err :=
		postgrescontratacion.NuevoResolutorGobiernoCoberturaO404BPostgreSQL(
			alta.postgresql.ejecucion,
		)
	if err != nil {
		return vacias, err
	}
	idempotencia, err :=
		postgrescontratacion.NuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(
			alta.postgresql.ejecucion,
		)
	if err != nil {
		return vacias, err
	}
	consultaMotivos, err :=
		postgrescontratacion.NuevaConsultaMotivoDecisionCoberturaPostgreSQL(
			alta.postgresql.ejecucion,
			catalogoMotivosDecisionCoberturaDesarrollo,
			moduloMotivosDecisionCoberturaDesarrollo,
		)
	if err != nil {
		return vacias, err
	}
	motivos, err := cobertura.NuevoResolutorMotivoDecisionCoberturaAcotado(
		consultaMotivos,
		catalogoMotivosDecisionCoberturaDesarrollo,
		moduloMotivosDecisionCoberturaDesarrollo,
	)
	if err != nil {
		return vacias, err
	}

	ejecutorDecision, err :=
		postgrescontratacion.NuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(
			alta.postgresql.confirmador,
		)
	if err != nil {
		return vacias, err
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
			ejecutorDecision,
		)
	if err != nil {
		return vacias, err
	}
	ejecutorReconciliacion, err :=
		postgrescontratacion.NuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
			alta.postgresql.confirmador,
		)
	if err != nil {
		return vacias, err
	}
	reconciliador, err :=
		cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
			ejecutorReconciliacion,
		)
	if err != nil {
		return vacias, err
	}

	ctxArranque, cancelar := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancelar()
	ejecutorHistorico, err :=
		postgrescontratacion.NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			ctxArranque,
			alta.postgresql.lectorResultado,
		)
	if err != nil {
		return vacias, err
	}
	lectorHistorico, err :=
		cobertura.NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(
			ejecutorHistorico,
		)
	if err != nil {
		return vacias, err
	}

	fuentes, err := nuevasDependenciasFuentesCoberturaDesarrollo(
		derivador,
		reloj,
		gobierno,
	)
	if err != nil {
		return vacias, err
	}
	cerrarFuentes := true
	defer func() {
		if cerrarFuentes {
			fuentes.cerrar()
		}
	}()
	consultas, err := application.NuevoPreparadorConsultaCobertura(
		fuentes.fuente,
		fuentes.verificador,
		fuentes.publicador,
		fuentes.autenticador,
		reloj,
		application.TiempoMaximoFuenteCobertura,
	)
	if err != nil {
		return vacias, err
	}
	preparador, err := application.NuevoPreparadorGlobalCobertura(
		consultas,
		fuentes.referencias,
		reloj,
		1,
		application.TiempoMaximoPreparacionGlobalCobertura,
	)
	if err != nil {
		return vacias, err
	}
	sellador, err := nuevoSelladorHMACCoberturaDesarrollo(derivador)
	if err != nil {
		return vacias, err
	}
	autorizador, err := nuevoAutorizadorConsultasCoberturaDesarrollo(
		alta.soporte,
		alta.autorizador,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if err != nil {
		return vacias, err
	}
	presentador, err := application.NuevoServicioPresentacionPropuestaCobertura(
		alta.soporte,
		autorizador,
		lectorAnalisis,
		reloj,
		gobierno,
		preparador,
	)
	if err != nil {
		return vacias, err
	}
	decisor, err := application.NuevoServicioConfirmacionDecisionCobertura(
		alta.soporte,
		motivos,
		sellador,
		idempotencia,
		lectorAnalisis,
		reloj,
		gobierno,
		preparador,
		alta.autorizador,
		transaccion,
		reconciliador,
	)
	if err != nil {
		return vacias, err
	}
	consultor, err := application.NuevoServicioConsultaResultadoCobertura(
		alta.soporte,
		autorizador,
		sellador,
		reloj,
		lectorHistorico,
	)
	if err != nil {
		return vacias, err
	}
	cerrarFuentes = false
	return dependenciasCoberturaContratacionTemporalDesarrollo{
		presentador: presentador,
		decisor:     decisor,
		consultor:   consultor,
		cerrar:      fuentes.cerrar,
	}, nil
}
