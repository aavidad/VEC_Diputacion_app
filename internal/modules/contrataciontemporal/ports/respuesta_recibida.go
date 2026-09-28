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
// ClaveIdempotencia es opcional en la entrada v2. Si llega de un cliente v1,
// se valida pero no decide la identidad de la operación. SQL deriva la clave
// durable del antecedente CT y del contenido. El material autorizado sigue
// siendo json.Marshal de estos diez campos, con su orden y nombres originales.
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
	if (s.ClaveIdempotencia != "" && !ClaveIdempotenciaValida(s.ClaveIdempotencia)) ||
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
	// CT138 devuelve el recibo original de CT56 y conserva su clave histórica.
	// La única diferencia permitida es esa clave: todo el contenido de la
	// declaración, incluidas comunicación y fecha, debe ser idéntico.
	original := r.Solicitud
	original.ClaveIdempotencia = solicitud.ClaveIdempotencia
	if solicitud.Validar() != nil || r.Solicitud.Validar() != nil ||
		!ClaveIdempotenciaValida(r.Solicitud.ClaveIdempotencia) || original != solicitud ||
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
// respuesta por organización, llamamiento y selección seudonimizada.
// El replay exige autorización fresca, mismo actor/perfil creador y
// coincidencia de la declaración; devuelve las referencias y fecha originales,
// sin nuevos efectos de negocio.
// Un error no entrega un resultado utilizable ni acredita ausencia de commit.
type RegistroRespuestasRecibidas interface {
	RegistrarRespuestaRecibida(context.Context, SolicitudRegistrarRespuestaRecibida) (RespuestaRecibidaRegistrada, error)
}
