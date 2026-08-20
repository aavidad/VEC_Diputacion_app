package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	EsquemaAltaPersonalRPT         = "vec.contratacion-temporal.personal-rpt.alta.v1"
	VersionContratoAltaPersonalRPT = uint64(1)
	maximoEnteroSeguroPersonalRPT  = uint64(9_007_199_254_740_991)
)

var (
	ErrSolicitudAltaPersonalRPTInvalida = errors.New(
		"contratacion temporal: solicitud de alta en personal invalida",
	)
	ErrResultadoAltaPersonalRPTInvalido = errors.New(
		"contratacion temporal: resultado de alta en personal invalido",
	)
)

// ReferenciaVersionadaPersonalRPT liga una fuente gobernada a su versión y
// huella. Solo transporta identidad técnica; no copia agregados de Personal.
type ReferenciaVersionadaPersonalRPT struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaVersionadaPersonalRPT) Validar() error {
	if r.Version > maximoEnteroSeguroPersonalRPT ||
		(domain.ReferenciaFlujo{
			DefinicionRef: r.Referencia,
			Version:       r.Version,
			HuellaSHA256:  r.HuellaSHA256,
		}).Validar() != nil {
		return ErrSolicitudAltaPersonalRPTInvalida
	}
	return nil
}

// SolicitudAltaPersonalRPT es el contrato nominal de coordinación. CapacidadRef
// debe proceder de una frontera confiable; su sintaxis nunca concede autoridad.
type SolicitudAltaPersonalRPT struct {
	Esquema           string                          `json:"esquema"`
	ContratoVersion   uint64                          `json:"contrato_version"`
	SolicitudRef      string                          `json:"solicitud_ref"`
	ExpedienteRef     string                          `json:"expediente_ref"`
	VersionExpediente uint64                          `json:"version_expediente"`
	CapacidadRef      string                          `json:"capacidad_ref"`
	CorrelacionRef    string                          `json:"correlacion_ref"`
	IdempotenciaRef   string                          `json:"idempotencia_ref"`
	FuenteRPT         ReferenciaVersionadaPersonalRPT `json:"fuente_rpt"`
	PuestoRef         string                          `json:"puesto_ref"`
	PlazaRef          string                          `json:"plaza_ref"`
}

func (s SolicitudAltaPersonalRPT) Validar() error {
	if s.Esquema != EsquemaAltaPersonalRPT ||
		s.ContratoVersion != VersionContratoAltaPersonalRPT ||
		!domain.ReferenciaOpacaValida(s.SolicitudRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionExpediente == 0 ||
		s.VersionExpediente > maximoEnteroSeguroPersonalRPT ||
		!domain.ReferenciaOpacaValida(s.CapacidadRef) ||
		!domain.ReferenciaOpacaValida(s.CorrelacionRef) ||
		!domain.ReferenciaOpacaValida(s.IdempotenciaRef) ||
		s.FuenteRPT.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.PuestoRef) ||
		!domain.ReferenciaOpacaValida(s.PlazaRef) {
		return ErrSolicitudAltaPersonalRPTInvalida
	}
	return nil
}

// MaterialCanonico devuelve una copia determinista ligada al esquema. Usa
// longitudes decimales para impedir ambigüedades entre campos adyacentes.
func (s SolicitudAltaPersonalRPT) MaterialCanonico() ([]byte, error) {
	if err := s.Validar(); err != nil {
		return nil, err
	}
	constructor := constructorCanonicoPersonalRPT{}
	constructor.campo("esquema", s.Esquema)
	constructor.entero("contrato_version", s.ContratoVersion)
	constructor.campo("solicitud_ref", s.SolicitudRef)
	constructor.campo("expediente_ref", s.ExpedienteRef)
	constructor.entero("version_expediente", s.VersionExpediente)
	constructor.campo("capacidad_ref", s.CapacidadRef)
	constructor.campo("correlacion_ref", s.CorrelacionRef)
	constructor.campo("idempotencia_ref", s.IdempotenciaRef)
	constructor.campo("rpt_ref", s.FuenteRPT.Referencia)
	constructor.entero("rpt_version", s.FuenteRPT.Version)
	constructor.campo("rpt_huella_sha256", s.FuenteRPT.HuellaSHA256)
	constructor.campo("puesto_ref", s.PuestoRef)
	constructor.campo("plaza_ref", s.PlazaRef)
	return append([]byte(nil), constructor.Bytes()...), nil
}

func (s SolicitudAltaPersonalRPT) HuellaSHA256() (string, error) {
	material, err := s.MaterialCanonico()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:]), nil
}

type EstadoResultadoAltaPersonalRPT string

const (
	AltaPersonalRPTConfirmada EstadoResultadoAltaPersonalRPT = "confirmada"
	AltaPersonalRPTRechazada  EstadoResultadoAltaPersonalRPT = "rechazada"
)

// ResultadoAltaPersonalRPT confirma referencias propiedad de Personal o
// rechaza mediante una causa gobernada. No transporta datos identificativos.
type ResultadoAltaPersonalRPT struct {
	Esquema               string                          `json:"esquema"`
	ContratoVersion       uint64                          `json:"contrato_version"`
	ResultadoRef          string                          `json:"resultado_ref"`
	ReciboRef             string                          `json:"recibo_ref"`
	SolicitudRef          string                          `json:"solicitud_ref"`
	CorrelacionRef        string                          `json:"correlacion_ref"`
	IdempotenciaRef       string                          `json:"idempotencia_ref"`
	HuellaSolicitudSHA256 string                          `json:"huella_solicitud_sha256"`
	Estado                EstadoResultadoAltaPersonalRPT  `json:"estado"`
	RelacionRef           string                          `json:"relacion_ref,omitempty"`
	OcupacionRef          string                          `json:"ocupacion_ref,omitempty"`
	MotivoRechazo         ReferenciaVersionadaPersonalRPT `json:"motivo_rechazo,omitempty"`
}

func (r ResultadoAltaPersonalRPT) ValidarPara(s SolicitudAltaPersonalRPT) error {
	huella, err := s.HuellaSHA256()
	if err != nil || r.Esquema != EsquemaAltaPersonalRPT ||
		r.ContratoVersion != VersionContratoAltaPersonalRPT ||
		!domain.ReferenciaOpacaValida(r.ResultadoRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		r.SolicitudRef != s.SolicitudRef ||
		r.CorrelacionRef != s.CorrelacionRef ||
		r.IdempotenciaRef != s.IdempotenciaRef ||
		r.HuellaSolicitudSHA256 != huella {
		return ErrResultadoAltaPersonalRPTInvalido
	}
	switch r.Estado {
	case AltaPersonalRPTConfirmada:
		if !domain.ReferenciaOpacaValida(r.RelacionRef) ||
			!domain.ReferenciaOpacaValida(r.OcupacionRef) ||
			r.MotivoRechazo != (ReferenciaVersionadaPersonalRPT{}) {
			return ErrResultadoAltaPersonalRPTInvalido
		}
	case AltaPersonalRPTRechazada:
		if r.RelacionRef != "" || r.OcupacionRef != "" ||
			r.MotivoRechazo.Validar() != nil {
			return ErrResultadoAltaPersonalRPTInvalido
		}
	default:
		return ErrResultadoAltaPersonalRPTInvalido
	}
	return nil
}

// AltaPersonalRPT declara la frontera; adaptadores y efectos pertenecen a
// minitareas posteriores y no forman parte de este corte.
type AltaPersonalRPT interface {
	SolicitarAlta(context.Context, SolicitudAltaPersonalRPT) (ResultadoAltaPersonalRPT, error)
}

type constructorCanonicoPersonalRPT struct {
	bytes.Buffer
}

func (c *constructorCanonicoPersonalRPT) campo(nombre, valor string) {
	c.WriteString(strconv.Itoa(len(nombre)))
	c.WriteByte(':')
	c.WriteString(nombre)
	c.WriteString(strconv.Itoa(len(valor)))
	c.WriteByte(':')
	c.WriteString(valor)
}

func (c *constructorCanonicoPersonalRPT) entero(nombre string, valor uint64) {
	c.campo(nombre, strconv.FormatUint(valor, 10))
}
