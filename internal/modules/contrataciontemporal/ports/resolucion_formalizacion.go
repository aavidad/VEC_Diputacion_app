package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrSolicitudResolucionFormalizacionInvalida    = errors.New("contratacion temporal: solicitud de resolucion de formalizacion invalida")
	ErrResolucionFormalizacionEnConflicto          = errors.New("contratacion temporal: resolucion de formalizacion en conflicto")
	ErrClaveResolucionFormalizacionUsada           = errors.New("contratacion temporal: clave de resolucion de formalizacion usada")
	ErrResolucionFormalizacionDenegada             = errors.New("contratacion temporal: resolucion de formalizacion denegada")
	ErrResolucionFormalizacionNoDisponible         = errors.New("contratacion temporal: resolucion de formalizacion no disponible")
	ErrResultadoResolucionFormalizacionNoConfiable = errors.New("contratacion temporal: resultado de resolucion no confiable")
)

// No incluye identidad, autoridad ni material documental recibido por HTTP.
type SolicitudResolucionFormalizacion struct {
	ExpedienteRef                                                              string
	VersionEsperada                                                            uint64
	PropuestaRef, ClaveIdempotencia, NumeroResolucion, FechaResolucion, Motivo string
	ConfirmaRevisionPropuesta, ConfirmaEjercicioManual                         bool
}

func textoResolucionValido(s string, max int) bool {
	return utf8.ValidString(s) && len(s) > 0 && len(s) <= max && strings.TrimSpace(s) == s &&
		!strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) })
}
func (s SolicitudResolucionFormalizacion) Validar() error {
	if !domain.ReferenciaOpacaValida(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.PropuestaRef) ||
		!ClaveIdempotenciaValida(s.ClaveIdempotencia) || !textoResolucionValido(s.NumeroResolucion, 80) ||
		!textoResolucionValido(s.Motivo, 2000) || s.VersionEsperada != 7 ||
		!s.ConfirmaRevisionPropuesta || !s.ConfirmaEjercicioManual {
		return ErrSolicitudResolucionFormalizacionInvalida
	}
	fecha, e := time.Parse("2006-01-02", s.FechaResolucion)
	if e != nil || fecha.Year() < 1 || fecha.Format("2006-01-02") != s.FechaResolucion {
		return ErrSolicitudResolucionFormalizacionInvalida
	}
	return nil
}

// Material resuelto y autorizado por servidor. SQL coteja recibo y propuesta
// contra el original v7. El SHA corresponde al PDF existente, no a una firma.
type MaterialResolucionFormalizacion struct {
	Solicitud                     SolicitudResolucionFormalizacion
	OrganizacionRef, DocumentoRef string
	PropuestaConfirmadaEn         time.Time
	DocumentoVersion              uint64
	DocumentoSHA256               string
}

func ReferenciaDocumentoResolucion(propuesta string) string {
	h := sha256.Sum256([]byte(propuesta))
	return "documento-resolucion:" + hex.EncodeToString(h[:])
}
func (m MaterialResolucionFormalizacion) Validar() error {
	if m.Solicitud.Validar() != nil || !domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.InstanteUTCCanonico(m.PropuestaConfirmadaEn) ||
		m.DocumentoRef != ReferenciaDocumentoResolucion(m.Solicitud.PropuestaRef) || m.DocumentoVersion != 7 ||
		!huellaSHA256BolsaValida(m.DocumentoSHA256) {
		return ErrSolicitudResolucionFormalizacionInvalida
	}
	return nil
}

type ResultadoResolucionFormalizacion struct {
	Solicitud                                                                                              SolicitudResolucionFormalizacion
	Estado, ResolucionRef, DocumentoRef, DocumentoSHA256, ActuacionRef, AuditoriaRef, OutboxRef, ReciboRef string
	DocumentoVersion, VersionResultante                                                                    uint64
	RegistradaEn                                                                                           time.Time
}

func (r ResultadoResolucionFormalizacion) ValidarPara(s SolicitudResolucionFormalizacion) error {
	if s.Validar() != nil || r.Solicitud != s || (r.Estado != "registrada" && r.Estado != "replay_registrada") ||
		r.VersionResultante != 8 || r.DocumentoRef != ReferenciaDocumentoResolucion(s.PropuestaRef) ||
		r.DocumentoVersion != 7 || !huellaSHA256BolsaValida(r.DocumentoSHA256) ||
		!domain.InstanteUTCCanonico(r.RegistradaEn) {
		return ErrResultadoResolucionFormalizacionNoConfiable
	}
	for _, ref := range []string{r.ResolucionRef, r.ActuacionRef, r.AuditoriaRef, r.OutboxRef, r.ReciboRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrResultadoResolucionFormalizacionNoConfiable
		}
	}
	return nil
}

type TransaccionResolucionFormalizacion interface {
	RegistrarResolucionFormalizacion(context.Context, SolicitudResolucionFormalizacion) (ResultadoResolucionFormalizacion, error)
}

// Preparación de solo lectura: el recibo procede de la actuación confirmada,
// nunca de volver a ejecutar el registro ni de estado guardado por el cliente.
type PreparacionResolucionFormalizacion struct {
	ExpedienteRef, PropuestaRef    string
	VersionEsperada, VersionActual uint64
	Recibo                         *ResultadoResolucionFormalizacion
}

func (p PreparacionResolucionFormalizacion) ValidarPara(expediente string) error {
	if !domain.ReferenciaOpacaValida(expediente) || p.ExpedienteRef != expediente ||
		!domain.ReferenciaOpacaValida(p.PropuestaRef) || p.VersionEsperada != 7 ||
		(p.VersionActual != 7 && p.VersionActual != 8) {
		return ErrResultadoResolucionFormalizacionNoConfiable
	}
	if p.VersionActual == 7 {
		if p.Recibo != nil {
			return ErrResultadoResolucionFormalizacionNoConfiable
		}
	} else if p.Recibo == nil || p.Recibo.ValidarPara(p.Recibo.Solicitud) != nil ||
		p.Recibo.Estado != "registrada" || p.Recibo.Solicitud.ExpedienteRef != expediente ||
		p.Recibo.Solicitud.PropuestaRef != p.PropuestaRef {
		return ErrResultadoResolucionFormalizacionNoConfiable
	}
	return nil
}

func (p PreparacionResolucionFormalizacion) Clonar() PreparacionResolucionFormalizacion {
	if p.Recibo != nil {
		r := *p.Recibo
		p.Recibo = &r
	}
	return p
}

type ConsultorPreparacionResolucionFormalizacion interface {
	ConsultarPreparacionResolucionFormalizacion(context.Context, string) (PreparacionResolucionFormalizacion, error)
}

type SesionPreparacionResolucionFormalizacion interface {
	ConsultarPreparacionYRegistrarAcceso(context.Context, OrdenConsultaDetalleRRHH) (DetalleExpedienteRRHH, PreparacionResolucionFormalizacion, error)
}
