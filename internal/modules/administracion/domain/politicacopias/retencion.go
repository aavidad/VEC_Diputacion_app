package politicacopias

import (
	"sort"
	"time"
)

// Copia metadata must come from a trusted authenticated catalogue. This DTO does
// not turn a caller's Verificada assertion into restoration evidence.
type Copia struct {
	Referencia            string    `json:"referencia"`
	Destino               string    `json:"destino"`
	Fecha                 time.Time `json:"fecha"`
	Verificada            bool      `json:"verificada"`
	Protegida             bool      `json:"protegida"`
	PreviaActiva          bool      `json:"previa_activa"`
	PendienteConciliacion bool      `json:"pendiente_conciliacion"`
	DependenciaNecesaria  bool      `json:"dependencia_necesaria"`
}
type DecisionRetencion struct {
	Referencia string `json:"referencia"`
	Candidata  bool   `json:"candidata"`
	Motivo     string `json:"motivo"`
}
type PlanRetencion struct {
	Politica                   string              `json:"politica"`
	PoliticaSHA256             string              `json:"politica_sha256"`
	DobleControl               bool                `json:"doble_control"`
	RequiereAutorizacionActual bool                `json:"requiere_autorizacion_actual"`
	Decisiones                 []DecisionRetencion `json:"decisiones"`
}

// PlanificarRetencion produces candidates only. Simultaneously applying the
// candidates keeps ConservarMinimo verified backups; protected dependencies and
// the last verified backup are preserved independently of age and permissions.
func (p Politica) PlanificarRetencion(copias []Copia, ahora time.Time) (PlanRetencion, error) {
	if p.Validar() != nil || ahora.IsZero() || len(copias) > 10000 {
		return PlanRetencion{}, ErrEntrada
	}
	list := append([]Copia(nil), copias...)
	refs := map[string]bool{}
	for _, c := range list {
		if !Referencia(c.Referencia) || !Referencia(c.Destino) || c.Fecha.IsZero() || c.Fecha.After(ahora) || refs[c.Referencia] {
			return PlanRetencion{}, ErrEntrada
		}
		refs[c.Referencia] = true
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Fecha.Equal(list[j].Fecha) {
			return list[i].Referencia < list[j].Referencia
		}
		return list[i].Fecha.After(list[j].Fecha)
	})
	protegidas := map[string]bool{}
	for _, r := range p.Retencion.Protegidas {
		protegidas[r] = true
	}
	retained := 0
	plan := PlanRetencion{Politica: p.Referencia, PoliticaSHA256: p.SHA256(), DobleControl: p.Retencion.ExigeDobleControl(), RequiereAutorizacionActual: true, Decisiones: make([]DecisionRetencion, 0, len(list))}
	loc, _ := time.LoadLocation(p.ZonaHoraria)
	cutoff := ahora.In(loc).AddDate(0, 0, -p.Retencion.EdadMaximaDias)
	for _, c := range list {
		reason := "fuera_destino"
		if c.Destino == p.Destino {
			switch {
			case c.PreviaActiva:
				reason = "previa_activa"
			case c.PendienteConciliacion:
				reason = "pendiente_conciliacion"
			case c.DependenciaNecesaria:
				reason = "dependencia_necesaria"
			case c.Protegida || protegidas[c.Referencia]:
				reason = "protegida"
			case !c.Verificada:
				reason = "no_verificada"
			case retained < p.Retencion.ConservarMinimo:
				reason = "minimo_verificadas"
			case !c.Fecha.Before(cutoff):
				reason = "dentro_retencion"
			case !p.Retencion.BorradoPermitido:
				reason = "borrado_no_permitido"
			default:
				reason = "candidata_revision"
			}
			if c.Verificada && reason != "candidata_revision" {
				retained++
			}
		}
		plan.Decisiones = append(plan.Decisiones, DecisionRetencion{c.Referencia, reason == "candidata_revision", reason})
	}
	return plan, nil
}
