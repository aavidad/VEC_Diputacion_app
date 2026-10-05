package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrPreservacionAuditoriaInvalida = errors.New("preservacion_auditoria_invalida")
var ErrPreservacionAuditoriaNoDisponible = errors.New("preservacion_auditoria_no_disponible")

const RecursoPreservacionAuditoria = "auditoria_periodica:comun_interna"
const MaxVersionPreservacionAuditoria uint64 = 999999999

var refPublicacionPreservacion = regexp.MustCompile(`^preservacion_[0-9a-f]{32}$`)
var refDecisionPreservacion = regexp.MustCompile(`^decision_tecnica_[0-9a-f]{32}$`)
var refAcusePreservacion = regexp.MustCompile(`^aud_v3_pre_[0-9a-f]{32}$`)
var refCorrelacionPreservacion = regexp.MustCompile(`^correlacion_[0-9a-f]{32}$`)

// SolicitudPreservacionAuditoria configura únicamente una medida técnica.
// La referencia de decisión no representa una resolución legal del Archivo.
type SolicitudPreservacionAuditoria struct {
	PublicacionRef     string `json:"publicacion_ref"`
	Version            uint64 `json:"version"`
	PreimagenSHA256    string `json:"preimagen_sha256"`
	DecisionTecnicaRef string `json:"decision_tecnica_ref"`
	Estado             string `json:"estado"`
	Medida             string `json:"medida"`
}

func (s SolicitudPreservacionAuditoria) Validar() error {
	if !refPublicacionPreservacion.MatchString(s.PublicacionRef) || s.Version == 0 || s.Version > MaxVersionPreservacionAuditoria ||
		!SHA256CheckpointValido(s.PreimagenSHA256) || !refDecisionPreservacion.MatchString(s.DecisionTecnicaRef) ||
		s.Estado != "provisional" || s.Medida != "conservar_todo_sin_expurgo" {
		return ErrPreservacionAuditoriaInvalida
	}
	return nil
}

type AcusePreservacionAuditoria struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      uint64    `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	RegistradaEn   time.Time `json:"registrada_en"`
	CorrelacionRef string    `json:"correlacion_ref"`
}

func (a AcusePreservacionAuditoria) Validar() error {
	if !refAcusePreservacion.MatchString(a.AuditoriaRef) || !refCorrelacionPreservacion.MatchString(a.CorrelacionRef) ||
		a.Secuencia == 0 || a.Secuencia > 9007199254740991 || !SHA256CheckpointValido(a.HuellaSHA256) ||
		a.HuellaSHA256 == strings.Repeat("0", 64) || !instantePreservacionValido(a.RegistradaEn) {
		return ErrPreservacionAuditoriaInvalida
	}
	return nil
}

type ResultadoPreservacionAuditoria struct {
	Estado              string                         `json:"estado"`
	Configuracion       SolicitudPreservacionAuditoria `json:"configuracion"`
	ConfiguracionSHA256 string                         `json:"configuracion_sha256"`
	RegistradaEn        time.Time                      `json:"registrada_en"`
	AcuseOriginal       AcusePreservacionAuditoria     `json:"acuse_original"`
	AcuseAcceso         AcusePreservacionAuditoria     `json:"acuse_acceso"`
}

func (r ResultadoPreservacionAuditoria) Validar() error {
	if r.AcuseAcceso.Validar() != nil {
		return ErrPreservacionAuditoriaInvalida
	}
	if r.Estado == "no_publicada" {
		if r.Configuracion != (SolicitudPreservacionAuditoria{}) || r.ConfiguracionSHA256 != "" || !r.RegistradaEn.IsZero() || r.AcuseOriginal != (AcusePreservacionAuditoria{}) {
			return ErrPreservacionAuditoriaInvalida
		}
		return nil
	}
	if (r.Estado != "publicada" && r.Estado != "replay" && r.Estado != "consultada") || r.Configuracion.Validar() != nil ||
		!SHA256CheckpointValido(r.ConfiguracionSHA256) || r.ConfiguracionSHA256 == strings.Repeat("0", 64) ||
		r.AcuseOriginal.Validar() != nil || !r.RegistradaEn.Equal(r.AcuseOriginal.RegistradaEn) || !instantePreservacionValido(r.RegistradaEn) ||
		r.AcuseAcceso.Secuencia < r.AcuseOriginal.Secuencia {
		return ErrPreservacionAuditoriaInvalida
	}
	if r.Estado == "publicada" && r.AcuseAcceso != r.AcuseOriginal ||
		r.Estado != "publicada" && r.AcuseAcceso.Secuencia <= r.AcuseOriginal.Secuencia {
		return ErrPreservacionAuditoriaInvalida
	}
	return nil
}

func instantePreservacionValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Year() >= 1 && t.Year() <= 9999 && t.Nanosecond()%1000 == 0
}
func (SolicitudPreservacionAuditoria) String() string { return "[SOLICITUD-PRESERVACION-TECNICA]" }
func (SolicitudPreservacionAuditoria) GoString() string {
	return "domain.SolicitudPreservacionAuditoria{[OPACA]}"
}
func (ResultadoPreservacionAuditoria) String() string { return "[RESULTADO-PRESERVACION-TECNICA]" }
func (ResultadoPreservacionAuditoria) GoString() string {
	return "domain.ResultadoPreservacionAuditoria{[OPACO]}"
}
