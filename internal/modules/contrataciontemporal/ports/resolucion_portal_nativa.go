package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// SolicitudResolucionPortal identifica una única respuesta firme de Mi Bolsa.
// La selección O6 y la persona se acreditan en la transacción propietaria.
type SolicitudResolucionPortal struct {
	ClaveIdempotencia     string
	OrganizacionRef       string
	ExpedienteRef         string
	BolsaRef              string
	LlamamientoRef        string
	ReciboEmisionRef      string
	ParticipacionRef      string
	ReciboRespuestaRef    string
	Respuesta             RespuestaLlamamiento
	RevisionRespuestaRRHH bool
	RevisionPlazoRRHH     bool
	CriterioValidacionRef string
}

func (s SolicitudResolucionPortal) Validar() error {
	if s.ValidarEntrada() != nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) {
		return ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

// La organización procede de la sesión del servidor, nunca del navegador.
func (s SolicitudResolucionPortal) ValidarEntrada() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.BolsaRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ReciboEmisionRef) ||
		!domain.ReferenciaOpacaValida(s.ParticipacionRef) ||
		!domain.ReferenciaOpacaValida(s.ReciboRespuestaRef) ||
		s.Respuesta != RespuestaLlamamientoAceptada ||
		!s.RevisionRespuestaRRHH || !s.RevisionPlazoRRHH ||
		!domain.ReferenciaOpacaValida(s.CriterioValidacionRef) {
		return ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

type RespuestaPortalResolucion struct {
	ParticipacionRef   string
	RespuestaRef       string
	ReciboRespuestaRef string
	RespondidaEn       time.Time
}

// ResultadoResolucionPortal acredita el asiento propio de CT. El estado del
// plazo refleja la revisión de RRHH; no afirma un vencimiento calculado.
type ResultadoResolucionPortal struct {
	Solicitud          SolicitudResolucionPortal
	Politica           ReferenciaGobernadaComunicacionLlamamiento
	FuenteRespuesta    string
	RespuestaPortal    RespuestaPortalResolucion
	EvaluacionPlazoRef string
	EstadoPlazo        string
	ResolucionRef      string
	ReciboLocalRef     string
	AuditoriaRef       string
	IntencionSiguiente IntencionOutboxSiguienteCandidato
	VersionResultante  uint64
	ResueltaEn         time.Time
	Estado             EstadoResultadoComunicacionLlamamiento
}

func (r ResultadoResolucionPortal) ValidarPara(s SolicitudResolucionPortal) error {
	if s.Validar() != nil || r.Solicitud != s || r.Politica.Validar() != nil ||
		r.Politica.Referencia != s.CriterioValidacionRef ||
		r.FuenteRespuesta != "portal" || r.EstadoPlazo != "revision_manual_portal" ||
		r.RespuestaPortal.ParticipacionRef != s.ParticipacionRef ||
		r.RespuestaPortal.ReciboRespuestaRef != s.ReciboRespuestaRef ||
		!domain.ReferenciaOpacaValida(r.RespuestaPortal.RespuestaRef) ||
		!domain.InstanteUTCCanonico(r.RespuestaPortal.RespondidaEn) ||
		!domain.ReferenciaOpacaValida(r.EvaluacionPlazoRef) ||
		!domain.ReferenciaOpacaValida(r.ResolucionRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboLocalRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		r.VersionResultante != 1 || !domain.InstanteUTCCanonico(r.ResueltaEn) ||
		r.ResueltaEn.Before(r.RespuestaPortal.RespondidaEn) || !r.Estado.valida() ||
		!r.IntencionSiguiente.vacia() {
		return ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return nil
}

func (r ResultadoResolucionPortal) EsReplayConfirmado() bool {
	return r.Estado == ResultadoComunicacionLlamamientoReplay
}
