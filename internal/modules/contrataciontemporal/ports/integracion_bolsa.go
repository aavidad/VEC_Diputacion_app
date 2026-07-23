package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	// VersionContratoIntegracionBolsa identifica la forma de los mensajes de
	// este puerto. Una versión incompatible exige otro contrato/adaptador.
	VersionContratoIntegracionBolsa uint64 = 1

	// MaximoElementosIntegracionBolsa es un límite técnico de protección. La
	// política funcional puede imponer uno menor sin recompilar el núcleo.
	MaximoElementosIntegracionBolsa uint32 = 250_000

	// VigenciaMaximaPeticionIntegracionBolsa limita la reutilización de una
	// petición capturada. Un adaptador puede aplicar un plazo menor.
	VigenciaMaximaPeticionIntegracionBolsa = 15 * time.Minute
)

var (
	ErrPeticionIntegracionBolsaInvalida = errors.New("contratacion temporal: peticion de integracion con bolsa invalida")
	ErrRespuestaBolsaNoConfiable        = errors.New("contratacion temporal: respuesta de bolsa no confiable")
	ErrIntegracionBolsaNoDisponible     = errors.New("contratacion temporal: integracion con bolsa no disponible")
	ErrLimiteIntegracionBolsaExcedido   = errors.New("contratacion temporal: limite de integracion con bolsa excedido")
	ErrEventoBolsaInvalido              = errors.New("contratacion temporal: evento de bolsa invalido")
	ErrAcuseEventoBolsaNoConfiable      = errors.New("contratacion temporal: acuse de evento de bolsa no confiable")
)

// ReferenciaVersionadaIntegracionBolsa enlaza un recurso opaco con la versión
// exacta y la huella de sus bytes canónicos. No contiene nombres, documentos
// de identidad, datos de contacto ni otros atributos personales.
type ReferenciaVersionadaIntegracionBolsa struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaVersionadaIntegracionBolsa) Validar() error {
	if !domain.ReferenciaOpacaValida(r.Referencia) || r.Version == 0 ||
		!huellaIntegracionBolsaValida(r.HuellaSHA256) {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// ContextoPeticionIntegracionBolsa liga una operación a un expediente, una
// finalidad gobernada y una vigencia. HuellaPeticionHMAC es un índice
// derivado con una clave externa; nunca se transmite la clave idempotente ni
// material personal en claro.
type ContextoPeticionIntegracionBolsa struct {
	OperacionRef       string                               `json:"operacion_ref"`
	OrganizacionRef    string                               `json:"organizacion_ref"`
	ExpedienteRef      string                               `json:"expediente_ref"`
	VersionExpediente  uint64                               `json:"version_expediente"`
	CorrelacionRef     string                               `json:"correlacion_ref"`
	ContratoVersion    uint64                               `json:"contrato_version"`
	Finalidad          ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	HuellaPeticionHMAC string                               `json:"huella_peticion_hmac"`
	SolicitadaEn       time.Time                            `json:"solicitada_en"`
	ValidaHasta        time.Time                            `json:"valida_hasta"`
}

func (c ContextoPeticionIntegracionBolsa) ValidarEn(instante time.Time) error {
	if !domain.ReferenciaOpacaValida(c.OperacionRef) ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ExpedienteRef) || c.VersionExpediente == 0 ||
		!domain.ReferenciaOpacaValida(c.CorrelacionRef) ||
		c.ContratoVersion != VersionContratoIntegracionBolsa ||
		c.Finalidad.Validar() != nil ||
		!huellaIntegracionBolsaValida(c.HuellaPeticionHMAC) ||
		!domain.InstanteUTCCanonico(c.SolicitadaEn) ||
		!domain.InstanteUTCCanonico(c.ValidaHasta) ||
		!c.ValidaHasta.After(c.SolicitadaEn) ||
		c.ValidaHasta.Sub(c.SolicitadaEn) > VigenciaMaximaPeticionIntegracionBolsa ||
		!domain.InstanteUTCCanonico(instante) ||
		instante.Before(c.SolicitadaEn) || !instante.Before(c.ValidaHasta) {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// ProcedenciaIntegracionBolsa demuestra qué autoridad y fotografía de datos
// produjo una respuesta. No autentica por sí sola: el adaptador debe verificar
// el canal y la evidencia antes de construir un resultado aceptable.
type ProcedenciaIntegracionBolsa struct {
	AutoridadRef    string                               `json:"autoridad_ref"`
	RespuestaRef    string                               `json:"respuesta_ref"`
	ContratoVersion uint64                               `json:"contrato_version"`
	Fuente          ReferenciaVersionadaIntegracionBolsa `json:"fuente"`
	EvidenciaRef    string                               `json:"evidencia_ref"`
	EmitidaEn       time.Time                            `json:"emitida_en"`
}

func (p ProcedenciaIntegracionBolsa) Validar() error {
	if !domain.ReferenciaOpacaValida(p.AutoridadRef) ||
		!domain.ReferenciaOpacaValida(p.RespuestaRef) ||
		p.ContratoVersion != VersionContratoIntegracionBolsa ||
		p.Fuente.Validar() != nil ||
		!domain.ReferenciaOpacaValida(p.EvidenciaRef) ||
		!domain.InstanteUTCCanonico(p.EmitidaEn) {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// SolicitudDisponibilidadBolsa pregunta por una necesidad ya identificada. La
// respuesta está acotada y no puede incluir el listado ni sus integrantes.
type SolicitudDisponibilidadBolsa struct {
	Contexto         ContextoPeticionIntegracionBolsa     `json:"contexto"`
	Necesidad        ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	CategoriaRef     string                               `json:"categoria_ref"`
	MaximoResultados uint32                               `json:"maximo_resultados"`
}

func (s SolicitudDisponibilidadBolsa) ValidarEn(instante time.Time) error {
	if s.Contexto.ValidarEn(instante) != nil || s.Necesidad.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) || s.MaximoResultados == 0 {
		return ErrPeticionIntegracionBolsaInvalida
	}
	if s.MaximoResultados > MaximoElementosIntegracionBolsa {
		return ErrLimiteIntegracionBolsaExcedido
	}
	return nil
}

// ResultadoDisponibilidadBolsa distingue una ausencia autoritativa de un
// fallo técnico. En caso de error o respuesta inválida, Disponible jamás debe
// interpretarse como confirmado.
type ResultadoDisponibilidadBolsa struct {
	OperacionRef       string                               `json:"operacion_ref"`
	OrganizacionRef    string                               `json:"organizacion_ref"`
	ExpedienteRef      string                               `json:"expediente_ref"`
	VersionExpediente  uint64                               `json:"version_expediente"`
	CorrelacionRef     string                               `json:"correlacion_ref"`
	Necesidad          ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	CategoriaRef       string                               `json:"categoria_ref"`
	Resultado          ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	BolsaEncontrada    bool                                 `json:"bolsa_encontrada"`
	Bolsa              ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Disponible         bool                                 `json:"disponible"`
	CantidadDisponible uint32                               `json:"cantidad_disponible"`
	CantidadExacta     bool                                 `json:"cantidad_exacta"`
	Procedencia        ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ResultadoDisponibilidadBolsa) ValidarPara(
	solicitud SolicitudDisponibilidadBolsa,
) error {
	if validarVinculoRespuestaBolsa(
		r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente, r.CorrelacionRef,
		r.Necesidad, r.Resultado, r.Procedencia, solicitud.Contexto, solicitud.Necesidad,
	) != nil || r.CategoriaRef != solicitud.CategoriaRef ||
		r.CantidadDisponible > solicitud.MaximoResultados {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.BolsaEncontrada {
		if r.Bolsa != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.Disponible || r.CantidadDisponible != 0 || !r.CantidadExacta {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if r.Bolsa.Validar() != nil ||
		r.Disponible != (r.CantidadDisponible > 0) ||
		(!r.Disponible && !r.CantidadExacta) {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// ConsultaDisponibilidadBolsa es una frontera saliente. Cancelación, timeout,
// ambigüedad, resultado cero no validable o fallo del conector son errores y
// nunca una confirmación negativa fabricada por contratación temporal.
type ConsultaDisponibilidadBolsa interface {
	ConsultarDisponibilidad(
		context.Context,
		SolicitudDisponibilidadBolsa,
	) (ResultadoDisponibilidadBolsa, error)
}

// SolicitudOrdenBolsa solicita únicamente una instantánea probatoria del
// orden. Las posiciones y las referencias de participantes permanecen en
// Bolsa y no cruzan este contrato.
type SolicitudOrdenBolsa struct {
	Contexto         ContextoPeticionIntegracionBolsa     `json:"contexto"`
	Necesidad        ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa            ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Politica         ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	MaximoPosiciones uint32                               `json:"maximo_posiciones"`
}

func (s SolicitudOrdenBolsa) ValidarEn(instante time.Time) error {
	if s.Contexto.ValidarEn(instante) != nil || s.Necesidad.Validar() != nil ||
		s.Bolsa.Validar() != nil || s.Politica.Validar() != nil ||
		s.MaximoPosiciones == 0 {
		return ErrPeticionIntegracionBolsaInvalida
	}
	if s.MaximoPosiciones > MaximoElementosIntegracionBolsa {
		return ErrLimiteIntegracionBolsaExcedido
	}
	return nil
}

// ResultadoOrdenBolsa omite el contenido de la instantánea deliberadamente.
// OrdenGenerado=false es un resultado funcional explícito y gobernado; una
// respuesta incompleta, ambigua o no verificable sigue siendo un error.
type ResultadoOrdenBolsa struct {
	OperacionRef      string                               `json:"operacion_ref"`
	OrganizacionRef   string                               `json:"organizacion_ref"`
	ExpedienteRef     string                               `json:"expediente_ref"`
	VersionExpediente uint64                               `json:"version_expediente"`
	CorrelacionRef    string                               `json:"correlacion_ref"`
	Necesidad         ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa             ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Politica          ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Resultado         ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	OrdenGenerado     bool                                 `json:"orden_generado"`
	Orden             ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	TotalPosiciones   uint32                               `json:"total_posiciones"`
	Procedencia       ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ResultadoOrdenBolsa) ValidarPara(solicitud SolicitudOrdenBolsa) error {
	if validarVinculoRespuestaBolsa(
		r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente, r.CorrelacionRef,
		r.Necesidad, r.Resultado, r.Procedencia, solicitud.Contexto, solicitud.Necesidad,
	) != nil || r.Bolsa != solicitud.Bolsa || r.Politica != solicitud.Politica ||
		r.TotalPosiciones > solicitud.MaximoPosiciones {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.OrdenGenerado {
		if r.Orden != (ReferenciaVersionadaIntegracionBolsa{}) || r.TotalPosiciones != 0 {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if r.Orden.Validar() != nil || r.TotalPosiciones == 0 {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

type ConsultaOrdenBolsa interface {
	ConsultarOrden(context.Context, SolicitudOrdenBolsa) (ResultadoOrdenBolsa, error)
}
