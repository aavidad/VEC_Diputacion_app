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

// OrdenPrepararFirmaBaremacion solicita una sesion interactiva acotada por la
// politica de firma vigente; la reserva OCC se adquiere tras custodiar la firma.
type OrdenPrepararFirmaBaremacion struct {
	Actor             ActorBaremacion
	Decision          DecisionBaremacionCustodiada
	OperacionRef      string
	ClaveIdempotencia string
}

// FirmaBaremacionPreparada contiene la sesion opaca del firmador y todos los
// enlaces probatorios previos necesarios para comprobar su resultado.
type FirmaBaremacionPreparada struct {
	decision           DecisionBaremacionCustodiada
	solicitud          puertosbolsa.SolicitudPrepararFirmaInteractiva
	sesion             puertosbolsa.SesionFirmaInteractiva
	seudonimoFirmado   string
	efectoCustodiaRef  string
	autorizacionesRefs []string
}

func (FirmaBaremacionPreparada) MarshalJSON() ([]byte, error) { return nil, ErrOrdenBaremacionInvalida }
func (*FirmaBaremacionPreparada) UnmarshalJSON([]byte) error  { return ErrOrdenBaremacionInvalida }
func (FirmaBaremacionPreparada) MarshalText() ([]byte, error) { return nil, ErrOrdenBaremacionInvalida }
func (FirmaBaremacionPreparada) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (FirmaBaremacionPreparada) String() string     { return "[FIRMA-BAREMACION-PREPARADA-OPACA]" }
func (f FirmaBaremacionPreparada) GoString() string { return f.String() }
func (f FirmaBaremacionPreparada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, f.String())
}
func (f FirmaBaremacionPreparada) LogValue() slog.Value { return slog.StringValue(f.String()) }

// PrepararFirma crea una sesion vinculada al documento, firmante, perfil,
// politica y autorizacion exactos.
func (s *ServicioBaremacion) PrepararFirma(
	ctx context.Context,
	orden OrdenPrepararFirmaBaremacion,
) (FirmaBaremacionPreparada, error) {
	if validarDecisionCustodiada(orden.Decision) != nil {
		return FirmaBaremacionPreparada{}, ErrOrdenBaremacionInvalida
	}
	if err := s.validarRevisionVigente(orden.Decision.decision.revision.revision); err != nil {
		return FirmaBaremacionPreparada{}, err
	}
	if validarActorRevision(orden.Actor, orden.Decision.decision.revision.revision) != nil ||
		!referenciaAplicacionBaremacionValida(orden.OperacionRef) ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotencia) {
		return FirmaBaremacionPreparada{}, ErrOrdenBaremacionInvalida
	}
	contenido := orden.Decision.decision.revision.contenido
	autorizacion, err := s.autorizarRevision(ctx, orden.Actor, orden.Decision.decision.revision.revision, puertosbolsa.AccionPrepararFirmaDecisionBaremacion,
		puertosbolsa.ClaseRecursoDecision, contenido.ID, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef)
	if err != nil {
		return FirmaBaremacionPreparada{}, err
	}
	autorizaciones, err := incorporarAutorizacionesBaremacion(orden.Decision.autorizacionesRefs, autorizacion)
	if err != nil || autorizacionYaUsadaEnCustodia(autorizacion, orden.Decision) {
		return FirmaBaremacionPreparada{}, ErrResultadoBaremacionNoConfiable
	}
	solicitudSeudonimo, err := puertosvec.NuevaSolicitudSeudonimizarSujetoAlmacen(
		contenido.SujetoRef, "bolsa_baremacion_firmado:"+contenido.ID,
	)
	if err != nil {
		return FirmaBaremacionPreparada{}, err
	}
	seudonimoFirmado, err := s.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitudSeudonimo)
	if err != nil {
		return FirmaBaremacionPreparada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	efectoCustodiaRef, err := s.generador.NuevaReferenciaEfectoAlmacen()
	if err != nil || !referenciaAplicacionBaremacionValida(efectoCustodiaRef) {
		return FirmaBaremacionPreparada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	ahora, err := s.ahora()
	if err != nil {
		return FirmaBaremacionPreparada{}, err
	}
	expiraEn := ahora.Add(s.duracionFirma)
	if !expiraEn.After(ahora) {
		return FirmaBaremacionPreparada{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	solicitud := puertosbolsa.SolicitudPrepararFirmaInteractiva{
		Contexto:          puertosbolsa.ContextoOperacionFirma{ContextoOperacionBaremacion: autorizacion, OperacionRef: orden.OperacionRef},
		ClaveIdempotencia: orden.ClaveIdempotencia, ProcesoRef: contenido.ProcesoRef,
		SolicitudRef: contenido.SolicitudRef, BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		DecisionRef: contenido.ID, Documento: orden.Decision.documento,
		FirmanteRef: contenido.DecisorRef, PerfilFirmanteClave: contenido.PerfilDecisorClave,
		Politica: orden.Decision.decision.politica, SolicitadaEn: ahora, ExpiraEn: expiraEn,
	}
	if err := solicitud.Validar(); err != nil {
		return FirmaBaremacionPreparada{}, err
	}
	sesion, err := s.firmador.PrepararFirmaInteractiva(ctx, solicitud)
	if err != nil || sesion.ValidarPara(solicitud) != nil ||
		(sesion.Estado != puertosbolsa.EstadoSesionFirmaPreparada && sesion.Estado != puertosbolsa.EstadoSesionFirmaPendiente) {
		return FirmaBaremacionPreparada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	return FirmaBaremacionPreparada{
		decision: orden.Decision, solicitud: solicitud, sesion: sesion,
		seudonimoFirmado: seudonimoFirmado, efectoCustodiaRef: efectoCustodiaRef,
		autorizacionesRefs: autorizaciones,
	}, nil
}
