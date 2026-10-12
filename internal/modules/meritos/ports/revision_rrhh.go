package ports

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionRevisionRRHH    = "meritos.hecho.revision_rrhh.consultar"
	FinalidadRevisionRRHH = "revision_meritos_inscripcion_rrhh"
	EsquemaRevisionRRHH   = "vec.meritos.revision-rrhh.v1"
)

var (
	ErrRevisionRRHHSolicitud    = errors.New("meritos.error.revision_rrhh_solicitud_invalida")
	ErrRevisionRRHHDenegada     = errors.New("meritos.error.revision_rrhh_denegada")
	ErrRevisionRRHHNoDisponible = errors.New("meritos.error.revision_rrhh_no_disponible")
	patronSolicitudRevisionRRHH = regexp.MustCompile(`^solicitud_inscripcion_[0-9a-f]{64}$`)
	patronPersonaRevisionRRHH   = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

// PersonaRef procede exclusivamente del detalle RRHH autorizado de Bolsa.
// El HTTP sólo aporta SolicitudRef, Limite y Cursor.
type SolicitudRevisionRRHH struct {
	Contexto                vec.ResultadoContextoActorRegistradoV2
	Vinculo                 vec.VinculoAutenticacionActorV2
	Correlacion             vec.ReferenciaCorrelacionAutorizacionV2
	SolicitudRef            string
	PersonaRef              string
	RevisionPermisos        uint64
	HuellaInstantaneaSHA256 string
	CapturaDetalleBolsa     []byte
	Limite                  int
	Cursor                  string
}

func (s SolicitudRevisionRRHH) Validar() error {
	if s.Contexto.Validar() != nil || s.Vinculo.ValidarPara(s.Contexto) != nil ||
		s.Correlacion.Validar() != nil || !patronSolicitudRevisionRRHH.MatchString(s.SolicitudRef) ||
		!patronPersonaRevisionRRHH.MatchString(s.PersonaRef) || s.RevisionPermisos == 0 ||
		!huellaSHA256CarteraValida(s.HuellaInstantaneaSHA256) ||
		len(s.CapturaDetalleBolsa) == 0 || len(s.CapturaDetalleBolsa) > 16384 ||
		!json.Valid(s.CapturaDetalleBolsa) ||
		s.Limite < 1 || s.Limite > 25 ||
		(s.Cursor != "" && !domain.ReferenciaValida(s.Cursor)) {
		return ErrRevisionRRHHSolicitud
	}
	d, err := s.Vinculo.Datos()
	if err != nil || d.Superficie != vec.SuperficieAutenticacionInternaCorporativaV1 ||
		d.GarantiaObservada != vec.AuthAssuranceHigh || d.CuentaPrivilegiada ||
		d.PrincipalID != s.Contexto.Contexto.PersonaRef ||
		d.PerfilActivoRef != s.Contexto.Contexto.PerfilActivoRef ||
		!d.SesionValidaHasta.After(time.Now().UTC()) {
		return ErrRevisionRRHHDenegada
	}
	return nil
}

type HechoRevisionRRHH struct {
	HechoRef     string                    `json:"hecho_ref"`
	Version      int                       `json:"version"`
	Tipo         string                    `json:"tipo"`
	Denominacion string                    `json:"denominacion"`
	Estado       domain.Estado             `json:"estado"`
	Vigencia     domain.Vigencia           `json:"vigencia"`
	Horas        *int                      `json:"horas,omitempty"`
	Evidencias   []vec.ReferenciaDocumento `json:"evidencias"`
	Revision     *domain.Revision          `json:"revision,omitempty"`
}

type ReciboRevisionRRHH struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      string    `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CorrelacionRef string    `json:"correlacion_ref"`
	RegistradaEn   time.Time `json:"registrada_en"`
}

type PaginaRevisionRRHH struct {
	Esquema         string              `json:"esquema"`
	Hechos          []HechoRevisionRRHH `json:"hechos"`
	Total           int                 `json:"total"`
	CursorSiguiente string              `json:"cursor_siguiente"`
	ReciboAcceso    ReciboRevisionRRHH  `json:"recibo_acceso"`
}

type LectorRevisionRRHH interface {
	ConsultarRevisionRRHH(context.Context, SolicitudRevisionRRHH) (PaginaRevisionRRHH, error)
}
