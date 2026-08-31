package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrSolicitudComunicacionLlamamientoInvalida = errors.New(
		"contratacion temporal: solicitud de comunicacion de llamamiento invalida",
	)
	ErrResultadoComunicacionLlamamientoNoConfiable = errors.New(
		"contratacion temporal: resultado de comunicacion de llamamiento no confiable",
	)
	ErrOperacionComunicacionLlamamientoDenegada = errors.New(
		"contratacion temporal: operacion de comunicacion de llamamiento denegada",
	)
	ErrVersionComunicacionLlamamientoEnConflicto = errors.New(
		"contratacion temporal: version de comunicacion de llamamiento en conflicto",
	)
	ErrClaveComunicacionLlamamientoUsada = errors.New(
		"contratacion temporal: clave de comunicacion de llamamiento usada con otros datos",
	)
)

// ReferenciaGobernadaComunicacionLlamamiento identifica una politica o canal
// publicados. Su contenido funcional permanece en la autoridad catalogal.
type ReferenciaGobernadaComunicacionLlamamiento struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

func (r ReferenciaGobernadaComunicacionLlamamiento) Validar() error {
	if !domain.ReferenciaOpacaValida(r.Referencia) ||
		!enteroSeguroBolsa(r.Version) || !huellaSHA256BolsaValida(r.HuellaSHA256) {
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return nil
}

type EstadoResultadoComunicacionLlamamiento string

const (
	ResultadoComunicacionLlamamientoConfirmado EstadoResultadoComunicacionLlamamiento = "confirmado"
	ResultadoComunicacionLlamamientoReplay     EstadoResultadoComunicacionLlamamiento = "replay_confirmado"
)

func (e EstadoResultadoComunicacionLlamamiento) valida() bool {
	return e == ResultadoComunicacionLlamamientoConfirmado ||
		e == ResultadoComunicacionLlamamientoReplay
}

// SolicitudRegistrarComunicacionLlamamiento solo transporta intencion y una
// referencia probatoria. Canal, politica, vencimiento e identidad se resuelven
// en la frontera confiable, no desde el solicitante.
type SolicitudRegistrarComunicacionLlamamiento struct {
	ClaveIdempotencia string
	OrganizacionRef   string
	ExpedienteRef     string
	LlamamientoRef    string
	VersionEsperada   uint64
	PruebaEntregaRef  string
}

func (s SolicitudRegistrarComunicacionLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!enteroSeguroBolsa(s.VersionEsperada) ||
		s.VersionEsperada == MaximoEnteroSeguroIntegracionBolsa ||
		!domain.ReferenciaOpacaValida(s.PruebaEntregaRef) {
		return ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

// ComunicacionProbatoria es el recibo minimizado del registro. No contiene
// destino, direccion de contacto ni otra referencia directa de persona.
type ComunicacionProbatoria struct {
	Solicitud         SolicitudRegistrarComunicacionLlamamiento
	ComunicacionRef   string
	Canal             ReferenciaGobernadaComunicacionLlamamiento
	Politica          ReferenciaGobernadaComunicacionLlamamiento
	ReciboRef         string
	VersionResultante uint64
	EntregadaEn       time.Time
	RespuestaHasta    time.Time
	Estado            EstadoResultadoComunicacionLlamamiento
}

func (c ComunicacionProbatoria) ValidarPara(
	solicitud SolicitudRegistrarComunicacionLlamamiento,
) error {
	if solicitud.Validar() != nil || c.Solicitud != solicitud ||
		!domain.ReferenciaOpacaValida(c.ComunicacionRef) ||
		c.Canal.Validar() != nil || c.Politica.Validar() != nil ||
		!domain.ReferenciaOpacaValida(c.ReciboRef) ||
		c.VersionResultante != solicitud.VersionEsperada+1 ||
		!domain.InstanteUTCCanonico(c.EntregadaEn) ||
		!domain.InstanteUTCCanonico(c.RespuestaHasta) ||
		!c.RespuestaHasta.After(c.EntregadaEn) || !c.Estado.valida() {
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return nil
}

func (c ComunicacionProbatoria) EsReplayConfirmado() bool {
	return c.Estado == ResultadoComunicacionLlamamientoReplay
}

type RespuestaLlamamiento string

const (
	RespuestaLlamamientoAceptada   RespuestaLlamamiento = "aceptacion"
	RespuestaLlamamientoRenunciada RespuestaLlamamiento = "renuncia"
	RespuestaLlamamientoExpirada   RespuestaLlamamiento = "expiracion_gobernada"
)

func (r RespuestaLlamamiento) Valida() bool {
	return r == RespuestaLlamamientoAceptada ||
		r == RespuestaLlamamientoRenunciada ||
		r == RespuestaLlamamientoExpirada
}

// SolicitudResolverLlamamiento no declara plazos ni candidatos. La expiracion
// se solicita sin una falsa respuesta personal y la evalua la politica vigente.
type SolicitudResolverLlamamiento struct {
	ClaveIdempotencia  string
	OrganizacionRef    string
	ExpedienteRef      string
	LlamamientoRef     string
	ComunicacionRef    string
	VersionEsperada    uint64
	Respuesta          RespuestaLlamamiento
	PruebaRespuestaRef string
}

func (s SolicitudResolverLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ComunicacionRef) ||
		!enteroSeguroBolsa(s.VersionEsperada) ||
		s.VersionEsperada == MaximoEnteroSeguroIntegracionBolsa ||
		!s.Respuesta.Valida() {
		return ErrSolicitudComunicacionLlamamientoInvalida
	}
	if s.Respuesta == RespuestaLlamamientoExpirada {
		if s.PruebaRespuestaRef != "" {
			return ErrSolicitudComunicacionLlamamientoInvalida
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(s.PruebaRespuestaRef) {
		return ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

type EstadoPlazoLlamamiento string

const (
	PlazoLlamamientoVigente  EstadoPlazoLlamamiento = "vigente"
	PlazoLlamamientoExpirado EstadoPlazoLlamamiento = "expirado"
)

type EstadoOutboxSiguienteCandidato string

const (
	OutboxSiguienteCandidatoPendiente     EstadoOutboxSiguienteCandidato = "pendiente"
	OutboxSiguienteCandidatoDespachada    EstadoOutboxSiguienteCandidato = "despachada"
	OutboxSiguienteCandidatoIndeterminada EstadoOutboxSiguienteCandidato = "indeterminada"
)

func (e EstadoOutboxSiguienteCandidato) valida() bool {
	return e == OutboxSiguienteCandidatoPendiente ||
		e == OutboxSiguienteCandidatoDespachada ||
		e == OutboxSiguienteCandidatoIndeterminada
}

// IntencionOutboxSiguienteCandidato solo publica referencias locales. El
// comando, las reglas, el orden y cualquier seleccion permanecen opacos hasta
// que un despachador posterior los abra ante el puerto autorizado de Bolsa.
type IntencionOutboxSiguienteCandidato struct {
	IntencionRef    string
	ComandoOpacoRef string
	Estado          EstadoOutboxSiguienteCandidato
	ActualizadaEn   time.Time
}

func (i IntencionOutboxSiguienteCandidato) Validar() error {
	if !domain.ReferenciaOpacaValida(i.IntencionRef) ||
		!domain.ReferenciaOpacaValida(i.ComandoOpacoRef) ||
		!i.Estado.valida() || !domain.InstanteUTCCanonico(i.ActualizadaEn) {
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return nil
}

func (i IntencionOutboxSiguienteCandidato) vacia() bool {
	return i == (IntencionOutboxSiguienteCandidato{})
}

// ResultadoResolucionLlamamiento acredita solo el commit local de la
// respuesta. Una intencion outbox no acredita que Bolsa haya sido invocada.
type ResultadoResolucionLlamamiento struct {
	Solicitud          SolicitudResolverLlamamiento
	Politica           ReferenciaGobernadaComunicacionLlamamiento
	EvaluacionPlazoRef string
	EstadoPlazo        EstadoPlazoLlamamiento
	ResolucionRef      string
	ReciboLocalRef     string
	IntencionSiguiente IntencionOutboxSiguienteCandidato
	VersionResultante  uint64
	ResueltaEn         time.Time
	Estado             EstadoResultadoComunicacionLlamamiento
}

func (r ResultadoResolucionLlamamiento) ValidarPara(
	solicitud SolicitudResolverLlamamiento,
) error {
	if solicitud.Validar() != nil || r.Solicitud != solicitud ||
		r.Politica.Validar() != nil ||
		!domain.ReferenciaOpacaValida(r.EvaluacionPlazoRef) ||
		!domain.ReferenciaOpacaValida(r.ResolucionRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboLocalRef) ||
		r.VersionResultante != solicitud.VersionEsperada+1 ||
		!domain.InstanteUTCCanonico(r.ResueltaEn) || !r.Estado.valida() ||
		(!r.IntencionSiguiente.vacia() &&
			r.IntencionSiguiente.ActualizadaEn.Before(r.ResueltaEn)) ||
		(r.Estado == ResultadoComunicacionLlamamientoConfirmado &&
			!r.IntencionSiguiente.vacia() &&
			r.IntencionSiguiente.Estado != OutboxSiguienteCandidatoPendiente) {
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	switch solicitud.Respuesta {
	case RespuestaLlamamientoAceptada:
		if r.EstadoPlazo != PlazoLlamamientoVigente || !r.IntencionSiguiente.vacia() {
			return ErrResultadoComunicacionLlamamientoNoConfiable
		}
	case RespuestaLlamamientoRenunciada:
		if r.EstadoPlazo != PlazoLlamamientoVigente ||
			r.IntencionSiguiente.Validar() != nil {
			return ErrResultadoComunicacionLlamamientoNoConfiable
		}
	case RespuestaLlamamientoExpirada:
		if r.EstadoPlazo != PlazoLlamamientoExpirado ||
			(!r.IntencionSiguiente.vacia() && r.IntencionSiguiente.Validar() != nil) {
			return ErrResultadoComunicacionLlamamientoNoConfiable
		}
	default:
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return nil
}

func (r ResultadoResolucionLlamamiento) EsReplayConfirmado() bool {
	return r.Estado == ResultadoComunicacionLlamamientoReplay
}

// TransaccionComunicacionLlamamiento es una frontera exclusivamente local.
// Registrar resuelve autorizacion, reacredita la prueba de entrega, obtiene
// canal, politica y vencimiento gobernados y confirma estado, auditoria,
// recibo y outbox local en un commit con OCC e idempotencia. Resolver confirma
// la respuesta local y, solo tras renuncia o expiracion gobernada, persiste
// junto a ella una intencion outbox pendiente con referencia a un comando
// opaco. No invoca Bolsa. Un replay puede devolver la intencion ya despachada
// o indeterminada por un proceso posterior.
// Cualquier error previo al commit revierte todos los cambios locales.
type TransaccionComunicacionLlamamiento interface {
	RegistrarComunicacion(
		context.Context,
		SolicitudRegistrarComunicacionLlamamiento,
	) (ComunicacionProbatoria, error)
	ResolverLlamamiento(
		context.Context,
		SolicitudResolverLlamamiento,
	) (ResultadoResolucionLlamamiento, error)
}
