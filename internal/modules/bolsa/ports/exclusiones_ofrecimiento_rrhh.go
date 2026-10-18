package ports

import (
	"context"
	"errors"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrExclusionesOfrecimientoRRHHNoDisponibles = errors.New("bolsa: motivos de exclusion del llamamiento no disponibles")

const LimitePaginaExclusionesOfrecimientoRRHH = 1000

// ConsultaExclusionesOfrecimientoRRHH conserva la sesión nominal y la
// autorización puntual del llamamiento exacto. El cursor es el orden del acta
// ya entregado; cero pide la primera página.
type ConsultaExclusionesOfrecimientoRRHH struct {
	BolsaRef, LlamamientoRef string
	DesdeOrdenActa           int64
	Limite                   int
	Vinculo                  core.VinculoAutenticacionActorV2
	ResultadoContexto        core.ResultadoContextoActorRegistradoV2
	Captura                  vecports.CapturaLecturaNominalBolsa
}

type ExclusionOfrecimientoRRHH struct {
	ParticipacionRef string                          `json:"participacion_ref"`
	OrdenActa        int64                           `json:"orden_acta"`
	SituacionOrigen  string                          `json:"situacion_en_origen"`
	MotivoClave      string                          `json:"motivo_clave"`
	EventosRelacion  []string                        `json:"eventos_relacion"`
	HechosRelacion   []HechoRelacionOfrecimientoRRHH `json:"hechos_relacion"`
}

type HechoRelacionOfrecimientoRRHH struct {
	EventoRef  string    `json:"evento_ref"`
	Tipo       string    `json:"tipo"`
	OcurridoEn time.Time `json:"ocurrido_en"`
}

type PaginaExclusionesOfrecimientoRRHH struct {
	CriteriosOfrecimiento *CriteriosOfrecimientoAplicados `json:"criterios_ofrecimiento"`
	BolsaRef              string                          `json:"bolsa_ref"`
	LlamamientoRef        string                          `json:"llamamiento_ref"`
	Exclusiones           []ExclusionOfrecimientoRRHH     `json:"exclusiones"`
	TotalExclusiones      int                             `json:"total_exclusiones"`
	HayMas                bool                            `json:"hay_mas"`
	SiguienteOrdenActa    *int64                          `json:"siguiente_orden_acta"`
	AuditoriaRef          string                          `json:"auditoria_ref"`
	CorrelacionRef        string                          `json:"correlacion_ref"`
	ConsultadaEn          time.Time                       `json:"consultada_en"`
}

type LectorExclusionesOfrecimientoRRHH interface {
	ConsultarExclusionesOfrecimientoRRHH(context.Context, ConsultaExclusionesOfrecimientoRRHH) (PaginaExclusionesOfrecimientoRRHH, error)
}

func (r PaginaExclusionesOfrecimientoRRHH) ValidaPara(q ConsultaExclusionesOfrecimientoRRHH) bool {
	if r.CriteriosOfrecimiento.Validar() != nil ||
		r.BolsaRef != q.BolsaRef || r.LlamamientoRef != q.LlamamientoRef ||
		r.Exclusiones == nil || len(r.Exclusiones) > q.Limite ||
		r.TotalExclusiones < len(r.Exclusiones) || r.TotalExclusiones < 0 ||
		r.AuditoriaRef == "" || r.CorrelacionRef == "" || r.ConsultadaEn.IsZero() {
		return false
	}
	ultimo := q.DesdeOrdenActa
	for _, e := range r.Exclusiones {
		if e.ParticipacionRef == "" || e.OrdenActa <= ultimo || e.SituacionOrigen == "" || e.MotivoClave == "" ||
			e.EventosRelacion == nil || e.HechosRelacion == nil || len(e.EventosRelacion) > 100 || len(e.HechosRelacion) > 100 {
			return false
		}
		referencias := make(map[string]bool, len(e.EventosRelacion))
		for _, ref := range e.EventosRelacion {
			if ref == "" || referencias[ref] {
				return false
			}
			referencias[ref] = true
		}
		vistos := make(map[string]bool, len(e.HechosRelacion))
		for _, hecho := range e.HechosRelacion {
			if hecho.EventoRef == "" || !referencias[hecho.EventoRef] || vistos[hecho.EventoRef] ||
				(hecho.Tipo != "incorporacion" && hecho.Tipo != "cese") || hecho.OcurridoEn.IsZero() ||
				hecho.OcurridoEn.Location() != time.UTC {
				return false
			}
			vistos[hecho.EventoRef] = true
		}
		ultimo = e.OrdenActa
	}
	if r.HayMas {
		return len(r.Exclusiones) > 0 && r.SiguienteOrdenActa != nil && *r.SiguienteOrdenActa == ultimo
	}
	return r.SiguienteOrdenActa == nil
}
