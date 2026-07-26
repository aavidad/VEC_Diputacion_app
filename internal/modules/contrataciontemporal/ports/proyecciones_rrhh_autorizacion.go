package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const DuracionMaximaCapacidadConsultaRRHH = 5 * time.Minute

// CapacidadConsultaRRHH es una concesión breve ligada a actor, sesión,
// organización, ámbito, acción, finalidad y, para detalle, expediente exacto.
type CapacidadConsultaRRHH struct {
	decisionRef     string
	correlacionRef  string
	motivoRef       string
	actorRef        string
	sesionRef       string
	perfilRef       string
	organizacionRef string
	claseAmbito     ClaseAmbitoConsultaRRHH
	ambitoRef       string
	accion          string
	finalidad       string
	expedienteRef   string
	validaDesde     time.Time
	validaHasta     time.Time
}

func NuevaCapacidadConsultaRRHH(
	decisionRef, correlacionRef, motivoRef string,
	contexto ContextoConsultaRRHH,
	claseAmbito ClaseAmbitoConsultaRRHH,
	ambitoRef, accion, finalidad, expedienteRef string,
	validaDesde, validaHasta time.Time,
) (CapacidadConsultaRRHH, error) {
	c := CapacidadConsultaRRHH{
		decisionRef: decisionRef, correlacionRef: correlacionRef,
		motivoRef: motivoRef, actorRef: contexto.actorRef,
		sesionRef: contexto.sesionRef, perfilRef: contexto.perfilRef,
		organizacionRef: contexto.organizacionRef,
		claseAmbito:     claseAmbito, ambitoRef: ambitoRef,
		accion: accion, finalidad: finalidad, expedienteRef: expedienteRef,
		validaDesde: validaDesde, validaHasta: validaHasta,
	}
	if contexto.validarEn(validaDesde) != nil ||
		validaHasta.After(contexto.validoHasta) ||
		c.validarEstructura() != nil {
		return CapacidadConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return c, nil
}

func (c CapacidadConsultaRRHH) validarEstructura() error {
	esCuadro := c.accion == AccionConsultarCuadroRRHH &&
		c.finalidad == FinalidadConsultarCuadroRRHH && c.expedienteRef == ""
	esDetalle := c.accion == AccionConsultarDetalleRRHH &&
		c.finalidad == FinalidadConsultarDetalleRRHH &&
		domain.ReferenciaOpacaValida(c.expedienteRef)
	if !domain.ReferenciaOpacaValida(c.decisionRef) ||
		!domain.ReferenciaOpacaValida(c.correlacionRef) ||
		!domain.ReferenciaOpacaValida(c.motivoRef) ||
		!domain.ReferenciaOpacaValida(c.actorRef) ||
		!domain.ReferenciaOpacaValida(c.sesionRef) ||
		!domain.ReferenciaOpacaValida(c.perfilRef) ||
		!domain.ReferenciaOpacaValida(c.organizacionRef) ||
		!c.claseAmbito.valida() ||
		!domain.ReferenciaOpacaValida(c.ambitoRef) ||
		(c.claseAmbito == AmbitoOrganizacionRRHH &&
			c.ambitoRef != c.organizacionRef) ||
		(!esCuadro && !esDetalle) ||
		!domain.InstanteUTCCanonico(c.validaDesde) ||
		!domain.InstanteUTCCanonico(c.validaHasta) ||
		!c.validaHasta.After(c.validaDesde) ||
		c.validaHasta.Sub(c.validaDesde) > DuracionMaximaCapacidadConsultaRRHH {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}

func (c CapacidadConsultaRRHH) validaPara(
	contexto ContextoConsultaRRHH,
	accion, finalidad, expedienteRef string,
	instante time.Time,
) error {
	if c.validarEstructura() != nil || contexto.validarEn(instante) != nil ||
		c.actorRef != contexto.actorRef || c.sesionRef != contexto.sesionRef ||
		c.perfilRef != contexto.perfilRef ||
		c.organizacionRef != contexto.organizacionRef ||
		c.accion != accion || c.finalidad != finalidad ||
		c.expedienteRef != expedienteRef ||
		instante.Before(c.validaDesde) || !instante.Before(c.validaHasta) {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}

func (c CapacidadConsultaRRHH) DecisionRef() string                  { return c.decisionRef }
func (c CapacidadConsultaRRHH) CorrelacionRef() string               { return c.correlacionRef }
func (c CapacidadConsultaRRHH) MotivoRef() string                    { return c.motivoRef }
func (c CapacidadConsultaRRHH) OrganizacionRef() string              { return c.organizacionRef }
func (c CapacidadConsultaRRHH) ClaseAmbito() ClaseAmbitoConsultaRRHH { return c.claseAmbito }
func (c CapacidadConsultaRRHH) AmbitoRef() string                    { return c.ambitoRef }
func (c CapacidadConsultaRRHH) Accion() string                       { return c.accion }
func (c CapacidadConsultaRRHH) Finalidad() string                    { return c.finalidad }
func (c CapacidadConsultaRRHH) ExpedienteRef() string                { return c.expedienteRef }
func (c CapacidadConsultaRRHH) ValidaDesde() time.Time               { return c.validaDesde }
func (c CapacidadConsultaRRHH) ValidaHasta() time.Time               { return c.validaHasta }
func (CapacidadConsultaRRHH) String() string                         { return "[capacidad-consulta-rrhh-redactada]" }
func (CapacidadConsultaRRHH) GoString() string                       { return "[capacidad-consulta-rrhh-redactada]" }
func (CapacidadConsultaRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

type OrdenConsultaCuadroRRHH struct {
	contexto  ContextoConsultaRRHH
	capacidad CapacidadConsultaRRHH
	solicitud SolicitudCuadroRRHH
	instante  time.Time
}

func NuevaOrdenConsultaCuadroRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) (OrdenConsultaCuadroRRHH, error) {
	if solicitud.validar() != nil ||
		capacidad.validaPara(
			contexto, AccionConsultarCuadroRRHH,
			FinalidadConsultarCuadroRRHH, "", instante,
		) != nil {
		return OrdenConsultaCuadroRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return OrdenConsultaCuadroRRHH{
		contexto: contexto, capacidad: capacidad,
		solicitud: solicitud, instante: instante,
	}, nil
}

func (o OrdenConsultaCuadroRRHH) Contexto() ContextoConsultaRRHH   { return o.contexto }
func (o OrdenConsultaCuadroRRHH) Capacidad() CapacidadConsultaRRHH { return o.capacidad }
func (o OrdenConsultaCuadroRRHH) Solicitud() SolicitudCuadroRRHH   { return o.solicitud }
func (o OrdenConsultaCuadroRRHH) Instante() time.Time              { return o.instante }
func (OrdenConsultaCuadroRRHH) String() string                     { return "[orden-consulta-cuadro-rrhh-redactada]" }
func (OrdenConsultaCuadroRRHH) GoString() string                   { return "[orden-consulta-cuadro-rrhh-redactada]" }
func (OrdenConsultaCuadroRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

type OrdenConsultaDetalleRRHH struct {
	contexto  ContextoConsultaRRHH
	capacidad CapacidadConsultaRRHH
	solicitud SolicitudDetalleRRHH
	instante  time.Time
}

func NuevaOrdenConsultaDetalleRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	instante time.Time,
) (OrdenConsultaDetalleRRHH, error) {
	if solicitud.validar() != nil ||
		capacidad.validaPara(
			contexto, AccionConsultarDetalleRRHH,
			FinalidadConsultarDetalleRRHH, solicitud.expedienteRef, instante,
		) != nil {
		return OrdenConsultaDetalleRRHH{}, ErrOrdenConsultaRRHHInvalida
	}
	return OrdenConsultaDetalleRRHH{
		contexto: contexto, capacidad: capacidad,
		solicitud: solicitud, instante: instante,
	}, nil
}

func (o OrdenConsultaDetalleRRHH) Contexto() ContextoConsultaRRHH   { return o.contexto }
func (o OrdenConsultaDetalleRRHH) Capacidad() CapacidadConsultaRRHH { return o.capacidad }
func (o OrdenConsultaDetalleRRHH) Solicitud() SolicitudDetalleRRHH  { return o.solicitud }
func (o OrdenConsultaDetalleRRHH) Instante() time.Time              { return o.instante }
func (OrdenConsultaDetalleRRHH) String() string                     { return "[orden-consulta-detalle-rrhh-redactada]" }
func (OrdenConsultaDetalleRRHH) GoString() string                   { return "[orden-consulta-detalle-rrhh-redactada]" }
func (OrdenConsultaDetalleRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

type AutoridadContextoConsultaRRHH interface {
	ResolverContextoConsultaRRHH(context.Context) (ContextoConsultaRRHH, error)
}

type AutorizadorConsultaRRHH interface {
	AutorizarCuadroRRHH(
		context.Context,
		ContextoConsultaRRHH,
		SolicitudCuadroRRHH,
		time.Time,
	) (CapacidadConsultaRRHH, error)
	AutorizarDetalleRRHH(
		context.Context,
		ContextoConsultaRRHH,
		SolicitudDetalleRRHH,
		time.Time,
	) (CapacidadConsultaRRHH, error)
}

// SesionConsultaRRHH mantiene lectura, revalidación de capacidad y registro
// durable de acceso en una sola operación. Nunca debe devolver datos si no
// confirmó antes el recibo de lectura en la misma transacción.
type SesionConsultaRRHH interface {
	ConsultarCuadroYRegistrar(
		context.Context,
		OrdenConsultaCuadroRRHH,
	) (PaginaCuadroRRHH, error)
	ConsultarDetalleYRegistrar(
		context.Context,
		OrdenConsultaDetalleRRHH,
	) (DetalleExpedienteRRHH, error)
}

var (
	_ json.Marshaler = ContextoConsultaRRHH{}
	_ json.Marshaler = SolicitudCuadroRRHH{}
	_ json.Marshaler = SolicitudDetalleRRHH{}
	_ json.Marshaler = CapacidadConsultaRRHH{}
	_ json.Marshaler = OrdenConsultaCuadroRRHH{}
	_ json.Marshaler = OrdenConsultaDetalleRRHH{}
	_ fmt.Stringer   = ContextoConsultaRRHH{}
	_ fmt.Stringer   = SolicitudCuadroRRHH{}
	_ fmt.Stringer   = SolicitudDetalleRRHH{}
	_ fmt.Stringer   = CapacidadConsultaRRHH{}
)
