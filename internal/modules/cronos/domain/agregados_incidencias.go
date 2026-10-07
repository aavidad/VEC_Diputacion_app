package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrAgregadosIncidenciasInvalidos = errors.New("cronos_agregados_incidencias_invalidos")

type CodigoIncidenciaAgregada string
type EstadoIncidenciaAgregada string
type CoberturaIncidencias string

const (
	RegistroIncompleto              CodigoIncidenciaAgregada = "registro_incompleto"
	IncidenciaPendiente             EstadoIncidenciaAgregada = "pendiente"
	IncidenciaResuelta              EstadoIncidenciaAgregada = "resuelta"
	CoberturaIncidenciasCompleta    CoberturaIncidencias     = "completa"
	CoberturaIncidenciasIncompleta  CoberturaIncidencias     = "incompleta"
	CoberturaIncidenciasDesconocida CoberturaIncidencias     = "desconocida"
)

func ReferenciaDemoAgregadosValida(ref string) bool {
	return strings.HasPrefix(ref, "demo:") && ReferenciaEfectosValida(ref)
}

// The digest binds the closed catalogue's exact values, not a labour policy.
// Canonical bytes: reference, decimal version, code, pending state, resolved
// state; each followed by LF, in that order.
func HuellaCatalogoAgregados(ref string, version int64, codigos []CodigoIncidenciaAgregada, estados []EstadoIncidenciaAgregada) (string, error) {
	if !ReferenciaDemoAgregadosValida(ref) || version < 1 || len(codigos) != 1 || codigos[0] != RegistroIncompleto || len(estados) != 2 || estados[0] != IncidenciaPendiente || estados[1] != IncidenciaResuelta {
		return "", ErrAgregadosIncidenciasInvalidos
	}
	b := ref + "\n" + strconv.FormatInt(version, 10) + "\n" + string(codigos[0]) + "\n" + string(estados[0]) + "\n" + string(estados[1]) + "\n"
	h := sha256.Sum256([]byte(b))
	return hex.EncodeToString(h[:]), nil
}

// The period consists of closed calendar days; the start is inclusive and the
// end is exclusive. A cutoff before the end cannot attest this period.
func DiasPeriodoAgregados(desde, hasta, zona string, corte time.Time) ([]string, error) {
	if zona == "" || zona == "Local" || len(zona) > 128 || strings.TrimSpace(zona) != zona || corte.IsZero() || corte.Location() != time.UTC || corte.Nanosecond()%1000 != 0 {
		return nil, ErrAgregadosIncidenciasInvalidos
	}
	loc, err := time.LoadLocation(zona)
	if err != nil {
		return nil, ErrAgregadosIncidenciasInvalidos
	}
	a, e1 := time.ParseInLocation("2006-01-02", desde, loc)
	b, e2 := time.ParseInLocation("2006-01-02", hasta, loc)
	if e1 != nil || e2 != nil || a.Format("2006-01-02") != desde || b.Format("2006-01-02") != hasta || !b.After(a) || corte.Before(b) {
		return nil, ErrAgregadosIncidenciasInvalidos
	}
	dias := []string{}
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		if len(dias) == 366 {
			return nil, ErrAgregadosIncidenciasInvalidos
		}
		dias = append(dias, d.Format("2006-01-02"))
	}
	return dias, nil
}

type HechoIncidenciaAgregada struct {
	Referencia string
	PersonaRef string
	Fecha      string
	Codigo     CodigoIncidenciaAgregada
	Estado     EstadoIncidenciaAgregada
}

type UnidadCoberturaIncidencias struct {
	PersonaRef string
	Fecha      string
	Estado     CoberturaIncidencias
}

type GrupoIncidenciasAgregadas struct {
	Codigo                          CodigoIncidenciaAgregada `json:"codigo"`
	Estado                          EstadoIncidenciaAgregada `json:"estado"`
	IncidenciasObservadas           int                      `json:"incidencias_observadas"`
	PersonasAfectadasObservadas     int                      `json:"personas_afectadas_observadas"`
	DiasAfectadosObservados         int                      `json:"dias_afectados_observados"`
	PersonasDiasAfectadosObservados int                      `json:"personas_dias_afectados_observados"`
	TotalIncidencias                *int                     `json:"total_incidencias"`
}

type AgregadoIncidencias struct {
	Estado                          string                      `json:"estado"`
	PersonasSeleccionadas           int                         `json:"personas_seleccionadas"`
	DiasPeriodo                     int                         `json:"dias_periodo"`
	PersonasDiasEsperados           int                         `json:"personas_dias_esperados"`
	PersonasDiasCompletos           int                         `json:"personas_dias_completos"`
	PersonasDiasIncompletos         int                         `json:"personas_dias_incompletos"`
	PersonasDiasDesconocidos        int                         `json:"personas_dias_desconocidos"`
	IncidenciasObservadas           int                         `json:"incidencias_observadas"`
	PersonasAfectadasObservadas     int                         `json:"personas_afectadas_observadas"`
	DiasAfectadosObservados         int                         `json:"dias_afectados_observados"`
	PersonasDiasAfectadosObservados int                         `json:"personas_dias_afectados_observados"`
	TotalIncidencias                *int                        `json:"total_incidencias"`
	Grupos                          []GrupoIncidenciasAgregadas `json:"grupos"`
}

// AgregarIncidenciasRegistradas counts explicit facts; it does not inspect
// punches, infer absence, resolve a request, or change any incidence state.
// References are only used internally to deduplicate; none escape the aggregate.
func AgregarIncidenciasRegistradas(personas, dias []string, cobertura []UnidadCoberturaIncidencias, hechos []HechoIncidenciaAgregada) (AgregadoIncidencias, error) {
	vacio := AgregadoIncidencias{}
	if len(personas) < 1 || len(personas) > 100 || len(dias) < 1 || len(dias) > 366 || len(cobertura) != len(personas)*len(dias) || hechos == nil || len(hechos) > 10000 {
		return vacio, ErrAgregadosIncidenciasInvalidos
	}
	ps, ds := map[string]bool{}, map[string]bool{}
	for _, p := range personas {
		if !ReferenciaDemoAgregadosValida(p) || ps[p] {
			return vacio, ErrAgregadosIncidenciasInvalidos
		}
		ps[p] = true
	}
	for _, d := range dias {
		if _, err := fechaPermiso(d); err != nil || ds[d] {
			return vacio, ErrAgregadosIncidenciasInvalidos
		}
		ds[d] = true
	}
	r := AgregadoIncidencias{Estado: "completo", PersonasSeleccionadas: len(personas), DiasPeriodo: len(dias), PersonasDiasEsperados: len(cobertura), Grupos: []GrupoIncidenciasAgregadas{{Codigo: RegistroIncompleto, Estado: IncidenciaPendiente}, {Codigo: RegistroIncompleto, Estado: IncidenciaResuelta}}}
	unidades := map[[2]string]bool{}
	for _, u := range cobertura {
		k := [2]string{u.PersonaRef, u.Fecha}
		if !ps[u.PersonaRef] || !ds[u.Fecha] || unidades[k] {
			return vacio, ErrAgregadosIncidenciasInvalidos
		}
		unidades[k] = true
		switch u.Estado {
		case CoberturaIncidenciasCompleta:
			r.PersonasDiasCompletos++
		case CoberturaIncidenciasIncompleta:
			r.PersonasDiasIncompletos++
		case CoberturaIncidenciasDesconocida:
			r.PersonasDiasDesconocidos++
		default:
			return vacio, ErrAgregadosIncidenciasInvalidos
		}
	}
	if r.PersonasDiasCompletos != r.PersonasDiasEsperados {
		r.Estado = "incompleto"
	}
	if r.PersonasDiasDesconocidos == r.PersonasDiasEsperados {
		r.Estado = "desconocido"
	}
	type afectados struct {
		personas, dias map[string]bool
		unidades       map[[2]string]bool
	}
	nuevo := func() afectados { return afectados{map[string]bool{}, map[string]bool{}, map[[2]string]bool{}} }
	total, grupos := nuevo(), []afectados{nuevo(), nuevo()}
	refs := map[string]bool{}
	for _, h := range hechos {
		if !ReferenciaDemoAgregadosValida(h.Referencia) || refs[h.Referencia] || !ps[h.PersonaRef] || !ds[h.Fecha] || h.Codigo != RegistroIncompleto || (h.Estado != IncidenciaPendiente && h.Estado != IncidenciaResuelta) {
			return vacio, ErrAgregadosIncidenciasInvalidos
		}
		refs[h.Referencia] = true
		i := 0
		if h.Estado == IncidenciaResuelta {
			i = 1
		}
		r.Grupos[i].IncidenciasObservadas++
		for _, a := range []afectados{total, grupos[i]} {
			a.personas[h.PersonaRef], a.dias[h.Fecha], a.unidades[[2]string{h.PersonaRef, h.Fecha}] = true, true, true
		}
	}
	r.IncidenciasObservadas, r.PersonasAfectadasObservadas, r.DiasAfectadosObservados, r.PersonasDiasAfectadosObservados = len(hechos), len(total.personas), len(total.dias), len(total.unidades)
	for i := range r.Grupos {
		g, a := &r.Grupos[i], grupos[i]
		g.PersonasAfectadasObservadas, g.DiasAfectadosObservados, g.PersonasDiasAfectadosObservados = len(a.personas), len(a.dias), len(a.unidades)
		if r.Estado == "completo" {
			n := g.IncidenciasObservadas
			g.TotalIncidencias = &n
		}
	}
	if r.Estado == "completo" {
		n := len(hechos)
		r.TotalIncidencias = &n
	}
	return r, nil
}
