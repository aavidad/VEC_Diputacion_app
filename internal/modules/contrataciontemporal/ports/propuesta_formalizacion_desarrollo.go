package ports

import (
	"reflect"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// MaterialPropuestaFormalizacion liga cada etapa a la solicitud canónica.
// La evidencia de Bolsa solo la aporta composición confiable, nunca el navegador.
type MaterialPropuestaFormalizacion struct {
	Etapa                 string
	Solicitud             SolicitudPropuestaFormalizacion
	AceptacionBolsa       *EvidenciaAceptacionBolsaPropuesta `json:",omitempty"`
	AceptacionBolsaNativa *EvidenciaAceptacionBolsaNativa    `json:",omitempty"`
	AceptacionDerivada    *EvidenciaAceptacionDerivada       `json:",omitempty"`
}

func (m MaterialPropuestaFormalizacion) Validar() error {
	// La versión OCC pertenece al expediente fiscalizado. La resolución de
	// llamamiento y la apertura de Bolsa conservan sus propias versiones
	// históricas y no pueden sustituirla.
	if m.Solicitud.Validar() != nil || m.Solicitud.VersionEsperada < 6 || len(m.Solicitud.Anexos) != 0 {
		return ErrSolicitudPropuestaFormalizacionInvalida
	}
	switch m.Etapa {
	case "consulta":
		if m.AceptacionBolsa != nil || m.AceptacionBolsaNativa != nil || m.AceptacionDerivada != nil {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
	case "confirmacion":
		origenes := 0
		for _, presente := range []bool{m.AceptacionBolsa != nil, m.AceptacionBolsaNativa != nil, m.AceptacionDerivada != nil} {
			if presente {
				origenes++
			}
		}
		if origenes != 1 || (m.Solicitud.DerivacionRef != "") != (m.AceptacionDerivada != nil) {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
		if m.AceptacionBolsa != nil && (m.AceptacionBolsa.Validar() != nil ||
			m.AceptacionBolsa.LlamamientoRef != m.Solicitud.LlamamientoRef) {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
		if m.AceptacionBolsaNativa != nil && (m.AceptacionBolsaNativa.Validar() != nil ||
			m.AceptacionBolsaNativa.Selector.LlamamientoRef != m.Solicitud.LlamamientoRef) {
			return ErrSolicitudPropuestaFormalizacionInvalida
		}
		if m.AceptacionDerivada != nil && m.AceptacionDerivada.ValidarPara(m.Solicitud) != nil {
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
	if m.AceptacionBolsaNativa != nil {
		copia := *m.AceptacionBolsaNativa
		m.AceptacionBolsaNativa = &copia
	}
	if m.AceptacionDerivada != nil {
		copia := *m.AceptacionDerivada
		m.AceptacionDerivada = &copia
	}
	return m
}

var patronOfertaAceptacionDerivada = regexp.MustCompile(`^oferta:[0-9a-f]{64}$`)

// EvidenciaAceptacionDerivada identifica la aceptación real de la plaza del
// expediente hijo. CT211 relee CT234; estas referencias no son autoridad por sí solas.
type EvidenciaAceptacionDerivada struct {
	DerivacionRef       string
	PadreRef            string
	HijoRef             string
	OfertaRef           string
	NumeroPlaza         uint32
	AceptacionRef       string
	ReciboDerivacionRef string
}

func (e EvidenciaAceptacionDerivada) Validar() error {
	if !domain.ReferenciaOpacaValida(e.DerivacionRef) || !domain.ReferenciaOpacaValida(e.PadreRef) ||
		!domain.ReferenciaOpacaValida(e.HijoRef) || !patronOfertaAceptacionDerivada.MatchString(e.OfertaRef) ||
		e.NumeroPlaza < 1 || e.NumeroPlaza > 100 ||
		!domain.ReferenciaOpacaValida(e.AceptacionRef) || !domain.ReferenciaOpacaValida(e.ReciboDerivacionRef) {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}

func (e EvidenciaAceptacionDerivada) ValidarPara(s SolicitudPropuestaFormalizacion) error {
	if e.Validar() != nil || s.Validar() != nil || s.DerivacionRef == "" ||
		e.DerivacionRef != s.DerivacionRef || e.HijoRef != s.ExpedienteRef ||
		e.HijoRef != HijoRefDerivacionNombramiento(s.OrganizacionRef, e.PadreRef, e.NumeroPlaza) {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}

// EvidenciaAceptacionBolsaNativa describe B110/B7/B30 sin reaprovechar
// OperacionRef, AperturaOperacionRef ni PropuestaRef del ledger antiguo.
// CT211 revalida estos hechos en Bolsa tras consumir la autorización CT.
type EvidenciaAceptacionBolsaNativa struct {
	Selector     domain.SelectorBolsaPlanB2Nativo
	RespuestaRef string
	RespondidaEn time.Time
}

func (e EvidenciaAceptacionBolsaNativa) Validar() error {
	if !e.Selector.Valido() || !domain.ReferenciaOpacaValida(e.RespuestaRef) ||
		!domain.InstanteUTCCanonico(e.RespondidaEn) || e.RespondidaEn.Nanosecond()%1000 != 0 {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}

func (e EvidenciaAceptacionBolsaNativa) ValidarPara(a AntecedentePropuestaFormalizacion) error {
	r := a.Resolucion
	j := a.Justificante
	if e.Validar() != nil || r.ValidarPara(r.Solicitud) != nil ||
		r.Solicitud.Respuesta != RespuestaLlamamientoAceptada || j.ValidarPara(r.Solicitud) != nil ||
		j.Continuacion != nil || !j.Seleccion.Procedencia.SinRetencionDeclarada ||
		e.Selector.ClaveCT != a.SeleccionClave ||
		e.Selector.OrganizacionRef != r.Solicitud.OrganizacionRef ||
		e.Selector.ExpedienteRef != r.Solicitud.ExpedienteRef ||
		e.Selector.LlamamientoRef != r.Solicitud.LlamamientoRef ||
		e.Selector.NecesidadRef != j.Seleccion.Necesidad.Referencia ||
		e.Selector.BolsaRef != j.Seleccion.Bolsa.Referencia ||
		e.Selector.ReciboEmisionRef != j.Seleccion.ReciboRef ||
		e.RespondidaEn.Before(j.Seleccion.ConfirmadaEn) || e.RespondidaEn.After(r.ResueltaEn) {
		return ErrResolucionLlamamientoNoAceptada
	}
	return nil
}

// AntecedentePropuestaFormalizacion conserva los recibos originales CT; no
// acredita por sí mismo el terminal de Bolsa ni firma o plazo legal alguno.
type AntecedentePropuestaFormalizacion struct {
	Resolucion     ResultadoResolucionLlamamiento
	Justificante   JustificanteRespuestaRecibida
	SeleccionClave string
	Derivacion     *EvidenciaAceptacionDerivada `json:",omitempty"`
}

func (a AntecedentePropuestaFormalizacion) ValidarPara(s SolicitudPropuestaFormalizacion) error {
	if s.DerivacionRef != "" {
		if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil ||
			a.Derivacion == nil || a.Derivacion.ValidarPara(s) != nil || a.SeleccionClave != "" ||
			!reflect.DeepEqual(a.Resolucion, ResultadoResolucionLlamamiento{}) ||
			!reflect.DeepEqual(a.Justificante, JustificanteRespuestaRecibida{}) {
			return ErrResultadoPropuestaFormalizacionNoConfiable
		}
		return nil
	}
	r := a.Resolucion
	if a.Derivacion != nil || (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil ||
		r.ValidarPara(r.Solicitud) != nil || r.Solicitud.Respuesta != RespuestaLlamamientoAceptada ||
		!r.Solicitud.RevisionManualConfirmada() || r.Solicitud.CriterioValidacionRef != r.Politica.Referencia ||
		r.Estado != ResultadoComunicacionLlamamientoConfirmado || r.Solicitud.VersionEsperada != 2 ||
		r.Solicitud.OrganizacionRef != s.OrganizacionRef || r.Solicitud.ExpedienteRef != s.ExpedienteRef ||
		r.Solicitud.LlamamientoRef != s.LlamamientoRef || r.ResolucionRef != s.ResolucionLlamamientoAceptadaRef ||
		r.ReciboLocalRef != s.ReciboResolucionAceptadaRef || a.Justificante.ValidarPara(r.Solicitud) != nil ||
		!versionSeleccionPropuestaValida(a.Justificante, s.VersionEsperada) ||
		r.ResueltaEn.Before(a.Justificante.Respuesta.RegistradaEn) || !ClaveIdempotenciaValida(a.SeleccionClave) {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return nil
}

// versionSeleccionPropuestaValida: sin continuación la propuesta parte de la
// versión fiscalizada en la que se seleccionó. Con continuación (renuncia,
// expiración o no incorporación, CT128) la selección es la original y el
// expediente puede haber avanzado después (tras una no incorporación vuelve
// a la fiscalización en una versión posterior); PostgreSQL fija la versión
// exacta que admite cada antecedente.
func versionSeleccionPropuestaValida(j JustificanteRespuestaRecibida, esperada uint64) bool {
	if j.Continuacion == nil {
		return j.Seleccion.VersionExpediente == esperada
	}
	return j.Seleccion.VersionExpediente <= esperada
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
