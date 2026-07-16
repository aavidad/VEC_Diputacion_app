package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// OrdenCustodiarDecisionBaremacion aporta referencias operativas opacas; la
// huella de solicitud y el seudonimo se generan siempre en servidor.
type OrdenCustodiarDecisionBaremacion struct {
	Actor             ActorBaremacion
	Decision          DecisionBaremacionCodificada
	OperacionRef      string
	ClaveIdempotencia string
	CargaRef          string
}

// DecisionBaremacionCustodiada conserva el recibo tecnico exacto del almacen
// cifrado que recibio el documento firmable.
type DecisionBaremacionCustodiada struct {
	decision           DecisionBaremacionCodificada
	solicitud          puertosbolsa.SolicitudCustodiarDocumentoFirmable
	documento          puertosbolsa.DocumentoFirmableCustodiado
	autorizacionesRefs []string
}

func (DecisionBaremacionCustodiada) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (*DecisionBaremacionCustodiada) UnmarshalJSON([]byte) error { return ErrOrdenBaremacionInvalida }
func (DecisionBaremacionCustodiada) MarshalText() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (DecisionBaremacionCustodiada) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (DecisionBaremacionCustodiada) String() string     { return "[DECISION-BAREMACION-CUSTODIADA-OPACA]" }
func (d DecisionBaremacionCustodiada) GoString() string { return d.String() }
func (d DecisionBaremacionCustodiada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DecisionBaremacionCustodiada) LogValue() slog.Value { return slog.StringValue(d.String()) }

// CustodiarDecision conserva el PDF canónico todavía no firmado con cifrado,
// integridad, versionado y referencias opacas. También comprueba que el mismo
// conector dispone de retención y bloqueo legal para la copia firmada final;
// no afirma que este artefacto temporal esté ya retenido.
func (s *ServicioBaremacion) CustodiarDecision(
	ctx context.Context,
	orden OrdenCustodiarDecisionBaremacion,
) (DecisionBaremacionCustodiada, error) {
	if validarDecisionCodificada(orden.Decision) != nil {
		return DecisionBaremacionCustodiada{}, ErrOrdenBaremacionInvalida
	}
	if err := s.validarRevisionVigente(orden.Decision.revision.revision); err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	if validarActorRevision(orden.Actor, orden.Decision.revision.revision) != nil ||
		!referenciaAplicacionBaremacionValida(orden.OperacionRef) ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotencia) ||
		!referenciaAplicacionBaremacionValida(orden.CargaRef) {
		return DecisionBaremacionCustodiada{}, ErrOrdenBaremacionInvalida
	}
	if err := s.validarSesionRevision(ctx, orden.Actor, orden.Decision.revision.revision); err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	contenido := orden.Decision.revision.contenido
	solicitudSeudonimo, err := puertosvec.NuevaSolicitudSeudonimizarSujetoAlmacen(
		contenido.SujetoRef, "bolsa_baremacion_decision",
	)
	if err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	seudonimo, err := s.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitudSeudonimo)
	if err != nil {
		return DecisionBaremacionCustodiada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	efectoRef, err := s.generador.NuevaReferenciaEfectoAlmacen()
	if err != nil || !referenciaAplicacionBaremacionValida(efectoRef) {
		return DecisionBaremacionCustodiada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	huellaPlan, err := s.sellarPartesBaremacion(ctx, []string{
		"plan_custodia_decision_baremacion_v2", orden.OperacionRef, orden.ClaveIdempotencia,
		orden.CargaRef, seudonimo, efectoRef, contenido.ProcesoRef, contenido.SolicitudRef,
		contenido.BaremacionMeritoRef, contenido.ID, s.clasificacion,
		orden.Decision.codificacion.HuellaDocumentoSHA256, contenido.CorrelacionRef,
	})
	if err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	vinculosAlmacen := puertosvec.VinculosOperacionAlmacen{
		OperacionRef: orden.OperacionRef, CargaRef: orden.CargaRef,
		Clasificacion: s.clasificacion, SujetoSeudonimoHMAC: seudonimo,
		HuellaSolicitudHMAC: huellaPlan, EfectoRef: efectoRef,
	}
	recursoAlmacen := recursoAlmacenBaremacion(
		contenido.ID, puertosbolsa.ClaseRecursoDecision, contenido.SujetoRef, vinculosAlmacen,
	)
	autorizacion, err := s.autorizarAlmacenRevision(
		ctx, orden.Actor, orden.Decision.revision.revision, puertosbolsa.AccionCustodiarDecisionBaremacion,
		puertosbolsa.ClaseRecursoDecision, contenido.ID, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef, recursoAlmacen,
	)
	if err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	autorizaciones, err := incorporarAutorizacionesBaremacion(orden.Decision.autorizacionesRefs, autorizacion)
	if err != nil || autorizacionesRepetidas(orden.Decision.revision.contextoAdopcion, autorizacion) ||
		autorizacion.Proyeccion().AutorizacionRef == orden.Decision.codificacion.AutorizacionCodificacionRef {
		return DecisionBaremacionCustodiada{}, ErrResultadoBaremacionNoConfiable
	}
	solicitud := puertosbolsa.SolicitudCustodiarDocumentoFirmable{
		Contexto: autorizacion, OperacionRef: orden.OperacionRef,
		ClaveIdempotencia: orden.ClaveIdempotencia, CargaRef: orden.CargaRef,
		SujetoSeudonimoHMAC: seudonimo, HuellaAlmacenHMAC: huellaPlan, EfectoRef: efectoRef,
		ProcesoRef: contenido.ProcesoRef, SolicitudRef: contenido.SolicitudRef,
		BaremacionMeritoRef: contenido.BaremacionMeritoRef, DecisionRef: contenido.ID,
		ClasificacionClave: s.clasificacion, Codificacion: orden.Decision.codificacion,
	}
	if err := solicitud.Validar(); err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	capacidades, err := s.almacen.Capacidades(ctx)
	if err != nil || capacidades.ConectorID != s.conectorAlmacen ||
		puertosvec.VerificarCapacidadesAlmacen(capacidades, puertosvec.RequisitosAlmacenObjetos{
			EscrituraEnFlujo: true, ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
			Retencion: true, BloqueoLegal: true, CifradoEnTransito: true, CifradoEnReposo: true,
			CifradoPorObjeto: true, PreservaObjetoOriginal: true,
		}) != nil {
		return DecisionBaremacionCustodiada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	escritura, err := solicitud.PrepararEscritura()
	if err != nil {
		return DecisionBaremacionCustodiada{}, err
	}
	resultadoAlmacen, err := s.almacen.Escribir(ctx, escritura)
	if err != nil {
		if resultadoAlmacen.Validar() == nil {
			return DecisionBaremacionCustodiada{}, &ErrorCustodiaBaremacionIncompleta{
				DecisionRef:  contenido.ID,
				DocumentoRef: contenido.ID,
				Escritura:    resultadoAlmacen,
				Causa:        err,
			}
		}
		return DecisionBaremacionCustodiada{}, err
	}
	documento, err := puertosbolsa.NuevoDocumentoFirmableCustodiado(solicitud, resultadoAlmacen)
	if err != nil || documento.ValidarPara(solicitud) != nil {
		return DecisionBaremacionCustodiada{}, &ErrorCustodiaBaremacionIncompleta{
			DecisionRef:  contenido.ID,
			DocumentoRef: contenido.ID,
			Escritura:    resultadoAlmacen,
			Causa:        errors.Join(ErrResultadoBaremacionNoConfiable, err),
		}
	}
	return DecisionBaremacionCustodiada{
		decision: orden.Decision, solicitud: solicitud, documento: documento,
		autorizacionesRefs: autorizaciones,
	}, nil
}
