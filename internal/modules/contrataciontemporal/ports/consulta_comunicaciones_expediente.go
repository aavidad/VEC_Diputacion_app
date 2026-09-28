package ports

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrConsultaComunicacionesExpedienteInvalida     = errors.New("contratacion temporal: consulta de comunicaciones invalida")
	ErrConsultaComunicacionesExpedienteDenegada     = errors.New("contratacion temporal: consulta de comunicaciones denegada")
	ErrConsultaComunicacionesExpedienteNoEncontrado = errors.New("contratacion temporal: expediente no encontrado")
	ErrConsultaComunicacionesExpedienteNoDisponible = errors.New("contratacion temporal: consulta de comunicaciones no disponible")
	ErrResultadoComunicacionesExpedienteNoConfiable = errors.New("contratacion temporal: resultado de comunicaciones no confiable")
)

const LimiteMaximoComunicacionesExpediente = 20

var cursorComunicacionesExpediente = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}#[0-9a-f]{64}$`)

// El cliente identifica el expediente; organización e identidad proceden de
// la autoridad de canal, nunca de parámetros HTTP.
type ConsultaComunicacionesExpediente struct {
	ExpedienteRef string
	Limite        int
	Cursor        string
}

func (c ConsultaComunicacionesExpediente) Validar() error {
	if !domain.ReferenciaOpacaValida(c.ExpedienteRef) ||
		c.Limite < 1 || c.Limite > LimiteMaximoComunicacionesExpediente ||
		(c.Cursor != "" && !cursorComunicacionesExpediente.MatchString(c.Cursor)) {
		return ErrConsultaComunicacionesExpedienteInvalida
	}
	return nil
}

type ComunicacionExpediente struct {
	OrganizacionRef       string    `json:"organizacion_ref"`
	ExpedienteRef         string    `json:"expediente_ref"`
	LlamamientoRef        string    `json:"llamamiento_ref"`
	ComunicacionRef       string    `json:"comunicacion_ref"`
	Version               uint64    `json:"version"`
	Estado                string    `json:"estado"`
	RegistradaEn          time.Time `json:"registrada_en"`
	ReciboComunicacionRef string    `json:"recibo_comunicacion_ref"`
	AntecedenteTipo       string    `json:"antecedente_tipo"`
	ReciboAntecedenteRef  string    `json:"recibo_antecedente_ref"`
}

type PaginaComunicacionesExpediente struct {
	ExpedienteRef   string                   `json:"expediente_ref"`
	Comunicaciones  []ComunicacionExpediente `json:"comunicaciones"`
	SiguienteCursor string                   `json:"siguiente_cursor,omitempty"`
}

func (p PaginaComunicacionesExpediente) ValidarPara(c ConsultaComunicacionesExpediente) error {
	if c.Validar() != nil || p.ExpedienteRef != c.ExpedienteRef ||
		p.Comunicaciones == nil || len(p.Comunicaciones) > c.Limite ||
		(p.SiguienteCursor != "" && (len(p.Comunicaciones) != c.Limite ||
			!cursorComunicacionesExpediente.MatchString(p.SiguienteCursor) ||
			!strings.HasPrefix(p.SiguienteCursor,
				p.Comunicaciones[len(p.Comunicaciones)-1].ComunicacionRef+"#"))) {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	var anterior time.Time
	var refAnterior, organizacion string
	for i, e := range p.Comunicaciones {
		if !domain.ReferenciaOpacaValida(e.OrganizacionRef) ||
			e.ExpedienteRef != c.ExpedienteRef ||
			!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
			!domain.ReferenciaOpacaValida(e.ComunicacionRef) ||
			!domain.ReferenciaOpacaValida(e.ReciboComunicacionRef) ||
			!domain.ReferenciaOpacaValida(e.ReciboAntecedenteRef) ||
			(e.AntecedenteTipo != "seleccion_confirmada" && e.AntecedenteTipo != "continuacion_confirmada") ||
			e.Version != 2 || e.Estado != "registrada_localmente" ||
			!domain.InstanteUTCCanonico(e.RegistradaEn) ||
			(i > 0 && (e.RegistradaEn.Before(anterior) ||
				(e.RegistradaEn.Equal(anterior) && e.ComunicacionRef <= refAnterior))) ||
			(organizacion != "" && e.OrganizacionRef != organizacion) {
			return ErrResultadoComunicacionesExpedienteNoConfiable
		}
		organizacion, anterior, refAnterior = e.OrganizacionRef, e.RegistradaEn, e.ComunicacionRef
	}
	return nil
}

type LectorComunicacionesExpediente interface {
	ConsultarComunicacionesExpediente(context.Context, ConsultaComunicacionesExpediente) (PaginaComunicacionesExpediente, error)
}
