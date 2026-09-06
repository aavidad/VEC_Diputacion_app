package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrOperacionContinuacionInvalida     = errors.New("contratacion temporal: solicitud de continuacion invalida")
	ErrOperacionContinuacionConflicto    = errors.New("contratacion temporal: continuacion en conflicto")
	ErrOperacionContinuacionDenegada     = errors.New("contratacion temporal: continuacion denegada")
	ErrOperacionContinuacionNoDisponible = errors.New("contratacion temporal: continuacion no disponible")
)

// SolicitudContinuarLlamamiento no contiene actor, candidato, orden ni permisos.
// Su JSON de cinco campos forma parte del material autorizado, sin etiquetas.
type SolicitudContinuarLlamamiento struct {
	ClaveIdempotencia string
	OrganizacionRef   string
	ExpedienteRef     string
	ResolucionRef     string
	IntencionRef      string
}

func (s SolicitudContinuarLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) || len(s.ClaveIdempotencia) != 36 ||
		s.ClaveIdempotencia[14] != '4' || !strings.ContainsRune("89ab", rune(s.ClaveIdempotencia[19])) {
		return ErrOperacionContinuacionInvalida
	}
	for _, ref := range []string{s.OrganizacionRef, s.ExpedienteRef, s.ResolucionRef, s.IntencionRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrOperacionContinuacionInvalida
		}
	}
	return nil
}

// ComandoSiguienteLlamamiento es la carga durable CT59, no una capacidad Bolsa.
// Se abre con permiso propio; la composición reacredita los antecedentes.
type ComandoSiguienteLlamamiento struct {
	Esquema         string `json:"esquema"`
	ComandoRef      string `json:"comando_ref"`
	IntencionRef    string `json:"intencion_ref"`
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
	LlamamientoRef  string `json:"llamamiento_ref"`
	JustificanteRef string `json:"justificante_ref"`
	SeleccionClave  string `json:"seleccion_clave"`
}

type AntecedenteContinuacionLlamamiento struct {
	Resolucion          ResultadoResolucionLlamamiento
	ComandoSiguienteRef string
	ComandoSiguiente    ComandoSiguienteLlamamiento
}

func (a AntecedenteContinuacionLlamamiento) ValidarPara(s SolicitudContinuarLlamamiento) error {
	r, c := a.Resolucion, a.ComandoSiguiente
	if s.Validar() != nil || r.ValidarPara(r.Solicitud) != nil ||
		r.Estado != ResultadoComunicacionLlamamientoConfirmado ||
		r.Solicitud.Respuesta != RespuestaLlamamientoRenunciada || !r.Solicitud.RevisionManualConfirmada() ||
		r.Solicitud.OrganizacionRef != s.OrganizacionRef || r.Solicitud.ExpedienteRef != s.ExpedienteRef ||
		r.ResolucionRef != s.ResolucionRef || r.IntencionSiguiente.IntencionRef != s.IntencionRef ||
		c.Esquema != "vec.contratacion-temporal.siguiente-candidato.intencion.v1" ||
		c.ComandoRef != a.ComandoSiguienteRef || c.ComandoRef != r.IntencionSiguiente.ComandoOpacoRef ||
		c.IntencionRef != s.IntencionRef || c.OrganizacionRef != s.OrganizacionRef || c.ExpedienteRef != s.ExpedienteRef ||
		c.LlamamientoRef != r.Solicitud.LlamamientoRef || c.JustificanteRef != r.Solicitud.PruebaRespuestaRef ||
		!ClaveIdempotenciaValida(c.SeleccionClave) {
		return ErrOperacionContinuacionNoDisponible
	}
	return nil
}

// ReciboBolsaContinuacion es una proyección interna de un recibo ya confirmado
// y comprobado por el puente propietario. Referencias solas no prueban efectos;
// no se acepta desde HTTP ni se usa para decidir elegibilidad o plazos.
type ReciboBolsaContinuacion struct {
	IntencionRef         string
	TerminalOperacionRef string
	OperacionRef         string
	LlamamientoRef       string
	PropuestaRef         string
	ReciboRef            string
	AuditoriaRef         string
	EventoRef            string
	RegistroSHA256       string
	ConfirmadaEn         time.Time
}

func (r ReciboBolsaContinuacion) ValidarPara(s SolicitudContinuarLlamamiento) error {
	if s.Validar() != nil || r.IntencionRef != s.IntencionRef || r.OperacionRef == r.TerminalOperacionRef ||
		!patronHuellaEfectoAlta.MatchString(r.RegistroSHA256) || r.RegistroSHA256 == strings.Repeat("0", 64) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrOperacionContinuacionNoDisponible
	}
	for _, ref := range []string{r.TerminalOperacionRef, r.OperacionRef, r.LlamamientoRef, r.PropuestaRef, r.ReciboRef, r.AuditoriaRef, r.EventoRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrOperacionContinuacionNoDisponible
		}
	}
	return nil
}

type MaterialContinuacionLlamamiento struct {
	Etapa       string
	Solicitud   SolicitudContinuarLlamamiento
	ReciboBolsa *ReciboBolsaContinuacion `json:",omitempty"`
}

func (m MaterialContinuacionLlamamiento) Validar() error {
	if m.Solicitud.Validar() != nil {
		return ErrOperacionContinuacionInvalida
	}
	switch m.Etapa {
	case "consulta":
		if m.ReciboBolsa == nil {
			return nil
		}
	case "confirmacion":
		if m.ReciboBolsa != nil {
			return m.ReciboBolsa.ValidarPara(m.Solicitud)
		}
	}
	return ErrOperacionContinuacionInvalida
}

// ResultadoContinuacionLlamamiento acredita exclusivamente la confirmación
// local del recibo Bolsa. Al recuperar solo cambia Estado; no es correo
// enviado ni cambia el recibo original de la renuncia.
type ResultadoContinuacionLlamamiento struct {
	Solicitud              SolicitudContinuarLlamamiento
	LlamamientoAnteriorRef string
	ReciboBolsa            ReciboBolsaContinuacion
	ReciboRef              string
	AuditoriaRef           string
	ConfirmadaEn           time.Time
	Estado                 string
}

func (r ResultadoContinuacionLlamamiento) ValidarPara(s SolicitudContinuarLlamamiento) error {
	if s.Validar() != nil || r.Solicitud != s || r.ReciboBolsa.ValidarPara(s) != nil ||
		!domain.ReferenciaOpacaValida(r.LlamamientoAnteriorRef) || r.LlamamientoAnteriorRef == r.ReciboBolsa.LlamamientoRef ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) || !domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) || r.ConfirmadaEn.Before(r.ReciboBolsa.ConfirmadaEn) || (r.Estado != "confirmado" && r.Estado != "replay_confirmado") {
		return ErrOperacionContinuacionNoDisponible
	}
	return nil
}

type RegistroContinuacionLlamamiento interface {
	LeerAntecedente(context.Context, SolicitudContinuarLlamamiento) (AntecedenteContinuacionLlamamiento, error)
	Confirmar(context.Context, SolicitudContinuarLlamamiento, ReciboBolsaContinuacion) (ResultadoContinuacionLlamamiento, error)
}
