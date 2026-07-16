package application

import (
	"context"
	"strings"
	"sync"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type relojBaremacionPrueba struct{ instante time.Time }

func (r relojBaremacionPrueba) Ahora() time.Time { return r.instante }

type sesionesBaremacionPrueba struct {
	sesiones []SesionAutenticadaBaremacion
	err      error
}

func (s *sesionesBaremacionPrueba) BuscarSesionesAutenticadasBaremacion(
	context.Context,
) ([]SesionAutenticadaBaremacion, error) {
	return append([]SesionAutenticadaBaremacion(nil), s.sesiones...), s.err
}

type autorizadorBaremacionPrueba struct {
	mu                   sync.Mutex
	ahora                time.Time
	solicitudes          []dominiovec.SolicitudAutorizacion
	referencias          []string
	obligacionEn         puertosbolsa.AccionOperacionBaremacion
	camposInvalidosEn    puertosbolsa.AccionOperacionBaremacion
	reutilizarReferencia bool
	vinculoDecision      dominiovec.VinculoAutenticacionActorV1
}

func (a *autorizadorBaremacionPrueba) Exigir(
	_ context.Context,
	solicitud dominiovec.SolicitudAutorizacion,
) (dominiovec.DecisionAutorizacion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if solicitud.ValidarVinculoAutenticacionActor() != nil {
		return dominiovec.DecisionAutorizacion{}, dominiovec.ErrAutorizacionDenegada
	}
	a.solicitudes = append(a.solicitudes, solicitud)
	accion := puertosbolsa.AccionOperacionBaremacion(solicitud.Accion)
	campos, existe := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
	if !existe {
		return dominiovec.DecisionAutorizacion{}, dominiovec.ErrAutorizacionDenegada
	}
	referencia := "autorizacion:baremacion:" + strings.Repeat("x", len(a.solicitudes))
	if a.reutilizarReferencia {
		referencia = "autorizacion:baremacion:reutilizada"
	}
	a.referencias = append(a.referencias, referencia)
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: referencia, Concedida: true, Codigo: "concedida", PrincipalID: solicitud.Principal.ID,
		PerfilActivoRef: solicitud.PerfilActivoRef, Accion: solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		Finalidad: solicitud.Finalidad, CorrelacionRef: solicitud.CorrelacionRef,
		VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		AsignacionRef:             "asignacion:tecnico-baremacion:v1", AsignacionHuellaSHA256: huellaBaremacionPrueba("3"),
		VersionRolRef: "rol:tecnico-baremacion:v1", VersionRolHuellaSHA256: huellaBaremacionPrueba("4"),
		GarantiaMinima: dominiovec.AuthAssuranceHigh, CamposPermitidos: campos,
		EmitidaEn: a.ahora.Add(-time.Minute), ValidaHasta: a.ahora.Add(4 * time.Minute),
	}
	if accion == puertosbolsa.AccionCustodiarDecisionBaremacion ||
		accion == puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion ||
		accion == puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion {
		decision.ValidaHasta = a.ahora.Add(puertosbolsa.VentanaMaximaUsoAutorizacionBaremacion)
	}
	if a.vinculoDecision.Validar() == nil {
		decision.VinculoAutenticacionActor = a.vinculoDecision
	}
	if accion == a.obligacionEn {
		decision.Obligaciones = []string{"doble_control_no_implementado"}
	}
	if accion == a.camposInvalidosEn {
		decision.CamposPermitidos = decision.CamposPermitidos[:len(decision.CamposPermitidos)-1]
	}
	return completarDecisionAutorizacionPrueba(solicitud, decision), nil
}

func (a *autorizadorBaremacionPrueba) acciones() []puertosbolsa.AccionOperacionBaremacion {
	a.mu.Lock()
	defer a.mu.Unlock()
	resultado := make([]puertosbolsa.AccionOperacionBaremacion, len(a.solicitudes))
	for indice := range a.solicitudes {
		resultado[indice] = puertosbolsa.AccionOperacionBaremacion(a.solicitudes[indice].Accion)
	}
	return resultado
}

func (a *autorizadorBaremacionPrueba) referenciasRepetidas() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	vistas := make(map[string]struct{}, len(a.referencias))
	for _, referencia := range a.referencias {
		if _, existe := vistas[referencia]; existe {
			return true
		}
		vistas[referencia] = struct{}{}
	}
	return false
}
