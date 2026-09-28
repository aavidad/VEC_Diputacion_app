package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrConsultaReciboRespuestaInvalida = errors.New("ct_consulta_recibo_respuesta_invalida")
	ErrReciboRespuestaNoEncontrado     = errors.New("ct_recibo_respuesta_no_encontrado")
	ErrConsultaReciboRespuestaDenegada = errors.New("ct_consulta_recibo_respuesta_denegada")
	ErrConsultaReciboRespuestaFallo    = errors.New("ct_consulta_recibo_respuesta_fallo")
	ErrReciboRespuestaNoConfiable      = errors.New("ct_recibo_respuesta_no_confiable")
)

type SolicitudConsultaReciboRespuesta struct {
	OrganizacionRef string
	ExpedienteRef   string
	ComunicacionRef string
}

func (s SolicitudConsultaReciboRespuesta) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.ComunicacionRef) {
		return ErrConsultaReciboRespuestaInvalida
	}
	return nil
}

// Vista mínima del registro original. La lectura no confirma el correo ni resuelve el llamamiento.
type ReciboRespuestaConsultado struct {
	OrganizacionRef string
	ExpedienteRef   string
	ComunicacionRef string
	Respuesta       RespuestaLlamamiento
	JustificanteRef string
	ReciboRef       string
	AuditoriaRef    string
	RegistradaEn    time.Time
	Estado          string
}

func (r ReciboRespuestaConsultado) ValidarPara(s SolicitudConsultaReciboRespuesta) error {
	if s.Validar() != nil || r.OrganizacionRef != s.OrganizacionRef || r.ExpedienteRef != s.ExpedienteRef || r.ComunicacionRef != s.ComunicacionRef ||
		(r.Respuesta != RespuestaLlamamientoAceptada && r.Respuesta != RespuestaLlamamientoRenunciada) ||
		!domain.ReferenciaOpacaValida(r.JustificanteRef) || !domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) || !domain.InstanteUTCCanonico(r.RegistradaEn) ||
		r.Estado != EstadoRespuestaRecibidaRegistrada {
		return ErrReciboRespuestaNoConfiable
	}
	return nil
}

type LectorReciboRespuesta interface {
	ConsultarReciboRespuesta(context.Context, SolicitudConsultaReciboRespuesta) (ReciboRespuestaConsultado, error)
}
