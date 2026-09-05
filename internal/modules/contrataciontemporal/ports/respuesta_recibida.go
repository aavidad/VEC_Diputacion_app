package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrSolicitudRespuestaRecibidaInvalida    = errors.New("ct_respuesta_recibida_solicitud_invalida")
	ErrClaveRespuestaRecibidaUsada           = errors.New("ct_respuesta_recibida_clave_usada")
	ErrVersionRespuestaRecibidaEnConflicto   = errors.New("ct_respuesta_recibida_version_en_conflicto")
	ErrOperacionRespuestaRecibidaDenegada    = errors.New("ct_respuesta_recibida_denegada")
	ErrRespuestaRecibidaNoDisponible         = errors.New("ct_respuesta_recibida_no_disponible")
	ErrResultadoRespuestaRecibidaNoConfiable = errors.New("ct_respuesta_recibida_resultado_no_confiable")
)

const (
	EstadoRespuestaRecibidaRegistrada = "registrada_por_rrhh"
	EstadoRespuestaRecibidaReplay     = "replay_registrada_por_rrhh"
)

// SolicitudRegistrarRespuestaRecibida recoge lo declarado por RRHH sobre un
// correo recibido. No acredita origen, firma, custodia ni entrega del aviso.
// Respuesta describe el correo, no una resolución terminal del llamamiento.
// El material autorizado es json.Marshal de esta solicitud directa: conservar
// estos diez campos, su orden y sus nombres sin etiquetas JSON ni envoltorio.
// La identidad del actor procede del contexto confiable, nunca de la solicitud.
type SolicitudRegistrarRespuestaRecibida struct {
	ClaveIdempotencia           string
	OrganizacionRef             string
	ExpedienteRef               string
	LlamamientoRef              string
	ComunicacionRef             string
	VersionComunicacionEsperada uint64
	Respuesta                   RespuestaLlamamiento
	CorreoRef                   string
	CorreoSHA256                string
	RecibidaEn                  time.Time
}

// Validar comprueba representación, no consulta un reloj ni verifica el correo.
// La persistencia rechaza fechas futuras con su reloj transaccional.
func (s SolicitudRegistrarRespuestaRecibida) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ComunicacionRef) ||
		s.VersionComunicacionEsperada != 2 ||
		(s.Respuesta != RespuestaLlamamientoAceptada && s.Respuesta != RespuestaLlamamientoRenunciada) ||
		!domain.ReferenciaOpacaValida(s.CorreoRef) ||
		!huellaSHA256BolsaValida(s.CorreoSHA256) ||
		!domain.InstanteUTCCanonico(s.RecibidaEn) {
		return ErrSolicitudRespuestaRecibidaInvalida
	}
	return nil
}

// RespuestaRecibidaRegistrada acredita exclusivamente el registro de la
// declaración, con referencias al justificante, recibo y auditoría persistidos.
// No avanza versiones de comunicación o expediente ni modifica Bolsa.
type RespuestaRecibidaRegistrada struct {
	Solicitud       SolicitudRegistrarRespuestaRecibida
	JustificanteRef string
	ReciboRef       string
	AuditoriaRef    string
	RegistradaEn    time.Time
	Estado          string
}

func (r RespuestaRecibidaRegistrada) ValidarPara(solicitud SolicitudRegistrarRespuestaRecibida) error {
	if solicitud.Validar() != nil || r.Solicitud != solicitud ||
		!domain.ReferenciaOpacaValida(r.JustificanteRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.InstanteUTCCanonico(r.RegistradaEn) ||
		solicitud.RecibidaEn.After(r.RegistradaEn) ||
		(r.Estado != EstadoRespuestaRecibidaRegistrada && r.Estado != EstadoRespuestaRecibidaReplay) {
		return ErrResultadoRespuestaRecibidaNoConfiable
	}
	return nil
}

// RegistroRespuestasRecibidas une autorización vigente, actor confiable,
// declaración, recibo y auditoría en una transacción durable. Conserva una
// respuesta por organización/comunicación y por organización/clave.
// El replay exige autorización fresca y coincidencia de todo el material;
// devuelve las referencias y fecha originales, sin nuevos efectos de negocio.
// Un error no entrega un resultado utilizable ni acredita ausencia de commit.
type RegistroRespuestasRecibidas interface {
	RegistrarRespuestaRecibida(context.Context, SolicitudRegistrarRespuestaRecibida) (RespuestaRecibidaRegistrada, error)
}
