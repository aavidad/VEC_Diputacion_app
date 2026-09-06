package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// MaterialPropuestaFormalizacion liga cada etapa a la solicitud canónica.
// La evidencia de Bolsa solo la aporta composición confiable, nunca el navegador.
type MaterialPropuestaFormalizacion struct {
	Etapa           string
	Solicitud       SolicitudPropuestaFormalizacion
	AceptacionBolsa *EvidenciaAceptacionBolsaPropuesta `json:",omitempty"`
}

func (m MaterialPropuestaFormalizacion) Validar() error {
	if m.Solicitud.Validar() != nil || m.Solicitud.VersionEsperada != 6 || len(m.Solicitud.Anexos) != 0 {
		return ErrSolicitudPropuestaFormalizacionInvalida
	}
	switch m.Etapa {
	case "consulta":
		if m.AceptacionBolsa != nil {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
	case "confirmacion":
		if m.AceptacionBolsa == nil || m.AceptacionBolsa.Validar() != nil ||
			m.AceptacionBolsa.LlamamientoRef != m.Solicitud.LlamamientoRef {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
	default:
		return ErrSolicitudPropuestaFormalizacionInvalida
	}
	return nil
}

func (m MaterialPropuestaFormalizacion) Clonar() MaterialPropuestaFormalizacion {
	m.Solicitud = m.Solicitud.Clonar()
	if m.AceptacionBolsa != nil {
		copia := *m.AceptacionBolsa
		m.AceptacionBolsa = &copia
	}
	return m
}

// AntecedentePropuestaFormalizacion conserva los recibos originales CT; no
// acredita por sí mismo el terminal de Bolsa ni firma o plazo legal alguno.
type AntecedentePropuestaFormalizacion struct {
	Resolucion     ResultadoResolucionLlamamiento
	Justificante   JustificanteRespuestaRecibida
	SeleccionClave string
}

func (a AntecedentePropuestaFormalizacion) ValidarPara(s SolicitudPropuestaFormalizacion) error {
	r := a.Resolucion
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil ||
		r.ValidarPara(r.Solicitud) != nil || r.Solicitud.Respuesta != RespuestaLlamamientoAceptada ||
		!r.Solicitud.RevisionManualConfirmada() || r.Solicitud.CriterioValidacionRef != r.Politica.Referencia ||
		r.Estado != ResultadoComunicacionLlamamientoConfirmado || r.Solicitud.VersionEsperada != 2 ||
		r.Solicitud.OrganizacionRef != s.OrganizacionRef || r.Solicitud.ExpedienteRef != s.ExpedienteRef ||
		r.Solicitud.LlamamientoRef != s.LlamamientoRef || r.ResolucionRef != s.ResolucionLlamamientoAceptadaRef ||
		r.ReciboLocalRef != s.ReciboResolucionAceptadaRef || a.Justificante.ValidarPara(r.Solicitud) != nil ||
		r.ResueltaEn.Before(a.Justificante.Respuesta.RegistradaEn) || !ClaveIdempotenciaValida(a.SeleccionClave) {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return nil
}

// EvidenciaAceptacionBolsaPropuesta es una proyección del terminal recuperado
// por el repositorio propietario de Bolsa y ligado a la aceptación CT.
// No incorpora identidad de candidato ni concede autoridad documental.
type EvidenciaAceptacionBolsaPropuesta struct {
	OperacionRef         string
	AperturaOperacionRef string
	LlamamientoRef       string
	JustificanteRef      string
	EvaluacionPlazoRef   string
	Politica             SnapshotGobernadoFormalizacion
	RegistroSHA256       string
	ResueltaEn           time.Time
}

func (e EvidenciaAceptacionBolsaPropuesta) Validar() error {
	for _, r := range []string{e.OperacionRef, e.AperturaOperacionRef, e.LlamamientoRef, e.JustificanteRef, e.EvaluacionPlazoRef} {
		if !domain.ReferenciaOpacaValida(r) {
			return ErrResolucionLlamamientoNoAceptada
		}
	}
	if e.OperacionRef == e.AperturaOperacionRef || !e.Politica.valido() ||
		!huellaSHA256BolsaValida(e.RegistroSHA256) || !domain.InstanteUTCCanonico(e.ResueltaEn) ||
		e.ResueltaEn.Nanosecond()%1000 != 0 {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}

func (e EvidenciaAceptacionBolsaPropuesta) ValidarPara(a AntecedentePropuestaFormalizacion) error {
	r := a.Resolucion
	apertura := a.Justificante.Seleccion.OperacionRef
	if c := a.Justificante.Continuacion; c != nil {
		apertura = c.ReciboBolsa.OperacionRef
	}
	if e.Validar() != nil || r.ValidarPara(r.Solicitud) != nil ||
		r.Solicitud.Respuesta != RespuestaLlamamientoAceptada || a.Justificante.ValidarPara(r.Solicitud) != nil ||
		e.AperturaOperacionRef != apertura ||
		e.LlamamientoRef != r.Solicitud.LlamamientoRef || e.JustificanteRef != r.Solicitud.PruebaRespuestaRef ||
		e.EvaluacionPlazoRef != r.EvaluacionPlazoRef ||
		e.Politica != (SnapshotGobernadoFormalizacion{Referencia: r.Politica.Referencia, Version: r.Politica.Version, HuellaSHA256: r.Politica.HuellaSHA256}) ||
		e.ResueltaEn.Before(r.ResueltaEn) {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}
