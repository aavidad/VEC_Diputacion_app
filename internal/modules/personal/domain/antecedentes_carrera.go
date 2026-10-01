package domain

import (
	"errors"
	"strconv"
)

var ErrAntecedentesCarreraInvalidos = errors.New("personal.antecedentes_carrera.invalidos")

const EsquemaAntecedentesCarrera = "vec.personal.preparacion-antecedentes-carrera.v1"

// PreparacionAntecedentesCarrera es una proyección de la ficha RRHH ya
// autorizada. No habilita la lectura desde Carrera ni acredita eficacia del
// acto, firma, grado o antigüedad. Las ocupaciones B2 no sustituyen la fuente M.
type PreparacionAntecedentesCarrera struct {
	Esquema     string                        `json:"esquema"`
	Alcance     string                        `json:"alcance"`
	EmpleadoRef string                        `json:"empleado_ref"`
	Version     int64                         `json:"version"`
	Corte       CorteEmpleadoB2               `json:"corte"`
	Pendientes  []string                      `json:"pendientes"`
	Relaciones  []RelacionAntecedentesCarrera `json:"relaciones"`
}

type EstadoRelacionAntecedentesCarrera struct {
	Estado     string          `json:"estado"`
	RegimenRef string          `json:"regimen_ref"`
	Regimen    string          `json:"regimen"`
	Traza      TrazaEmpleadoB2 `json:"traza"`
}

type RelacionAntecedentesCarrera struct {
	Historia    []EstadoRelacionAntecedentesCarrera `json:"historia"`
	RelacionRef string                              `json:"relacion_ref"`
	Estado      string                              `json:"estado"`
	RegimenRef  string                              `json:"regimen_ref"`
	Regimen     string                              `json:"regimen"`
	Traza       TrazaEmpleadoB2                     `json:"traza"`
	Servicios   []ServicioAntecedentesCarrera       `json:"servicios"`
	Situaciones []SituacionAntecedentesCarrera      `json:"situaciones"`
	Pendientes  []string                            `json:"pendientes"`
}

type ServicioAntecedentesCarrera struct {
	UltimaRevisionConocida bool            `json:"ultima_revision_conocida"`
	ServicioRef            string          `json:"servicio_ref"`
	Estado                 string          `json:"estado"`
	PeriodoDesde           FechaCivil      `json:"periodo_desde"`
	PeriodoHasta           FechaCivil      `json:"periodo_hasta"`
	DiasReconocidos        int64           `json:"dias_reconocidos"`
	Traza                  TrazaEmpleadoB2 `json:"traza"`
}

type SituacionAntecedentesCarrera struct {
	SituacionRef string          `json:"situacion_ref"`
	Estado       string          `json:"estado"`
	CodigoRef    string          `json:"codigo_ref"`
	Traza        TrazaEmpleadoB2 `json:"traza"`
}

// PrepararAntecedentesCarrera conserva los periodos y días de cada acto, sin
// sumarlos. Una ficha vacía nunca se presenta como una cobertura completa.
// El llamante debe validar y consumir su autorización B2 antes de proyectarla.
func PrepararAntecedentesCarrera(f FichaEmpleadoB2) (PreparacionAntecedentesCarrera, error) {
	var vacio PreparacionAntecedentesCarrera
	if !ReferenciaEmpleadoValida(f.EmpleadoRef) || !ReferenciaPersonaValida(f.PersonaRef) || !patronReferenciaB2.MatchString(f.OrganismoRef) || f.Version < 1 || f.Corte.Validar() != nil || f.EficaciaAdministrativa || f.FirmaOficial || len(f.Relaciones) > LimiteFilasFichaPropia || len(f.Servicios) > LimiteFilasFichaPropia || len(f.Situaciones) > LimiteFilasFichaPropia {
		return vacio, ErrAntecedentesCarreraInvalidos
	}
	p := PreparacionAntecedentesCarrera{
		Esquema: EsquemaAntecedentesCarrera, Alcance: "preparacion", EmpleadoRef: f.EmpleadoRef,
		Version: f.Version, Corte: f.Corte, Relaciones: make([]RelacionAntecedentesCarrera, 0, len(f.Relaciones)),
		Pendientes: []string{"fuente_institucional", "cobertura_antecedentes", "politica_carrera"},
	}
	indices := make(map[string]int, len(f.Relaciones))
	versiones := make(map[string]struct{}, len(f.Relaciones))
	for _, r := range f.Relaciones {
		if !ReferenciaRelacionValida(r.RelacionRef) || r.OrganismoRef != f.OrganismoRef || !estadoRelacionB2Valido(r.Estado) || !patronReferenciaB2.MatchString(r.RegimenRef) || !trazaAntecedenteCarreraValida(r.Traza, f.Corte) {
			return vacio, ErrAntecedentesCarreraInvalidos
		}
		if repetidoB2(versiones, r.RelacionRef+":"+strconv.FormatInt(r.Traza.Version, 10)) {
			return vacio, ErrAntecedentesCarreraInvalidos
		}
		nombre := ""
		pendientes := []string{"grupo_subgrupo", "puesto_nivel_m", "grado_personal_h"}
		if c := r.CatalogoSnapshot.Regimen; c != nil {
			if !c.validar("regimen", f.OrganismoRef, r.RegimenRef, r.Traza.Desde) {
				return vacio, ErrAntecedentesCarreraInvalidos
			}
			nombre = c.Denominacion
		} else {
			pendientes = append(pendientes, "regimen_sin_catalogo")
		}
		h := EstadoRelacionAntecedentesCarrera{r.Estado, r.RegimenRef, nombre, r.Traza}
		if i, existe := indices[r.RelacionRef]; existe {
			actual := &p.Relaciones[i]
			actual.Historia = append(actual.Historia, h)
			for _, clave := range pendientes {
				actual.Pendientes = agregarPendienteCarrera(actual.Pendientes, clave)
			}
			if trazaCarreraPosterior(r.Traza, actual.Traza) {
				actual.Estado = r.Estado
				actual.RegimenRef = r.RegimenRef
				actual.Regimen = nombre
				actual.Traza = r.Traza
			}
			if f.Corte.VigenteEn.AntesDe(r.Traza.Desde) {
				actual.Pendientes = agregarPendienteCarrera(actual.Pendientes, "hechos_fuera_corte")
			}
			continue
		}
		if f.Corte.VigenteEn.AntesDe(r.Traza.Desde) {
			pendientes = append(pendientes, "hechos_fuera_corte")
		}
		indices[r.RelacionRef] = len(p.Relaciones)
		p.Relaciones = append(p.Relaciones, RelacionAntecedentesCarrera{
			Historia: []EstadoRelacionAntecedentesCarrera{h}, RelacionRef: r.RelacionRef, Estado: r.Estado, RegimenRef: r.RegimenRef, Regimen: nombre,
			Traza: r.Traza, Servicios: []ServicioAntecedentesCarrera{}, Situaciones: []SituacionAntecedentesCarrera{}, Pendientes: pendientes,
		})
	}
	ids := make(map[string]struct{}, len(f.Servicios)+len(f.Situaciones))
	for _, s := range f.Servicios {
		i, existe := indices[s.RelacionRef]
		if !existe || !patronReferenciaB2.MatchString(s.ServicioRef) || repetidoB2(ids, s.ServicioRef+":"+strconv.FormatInt(s.Traza.Version, 10)) || (s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") || s.PeriodoDesde.Validar() != nil || s.PeriodoHasta.Validar() != nil || s.PeriodoHasta.AntesDe(s.PeriodoDesde) || s.DiasReconocidos < 0 || !trazaAntecedenteCarreraValida(s.Traza, f.Corte) {
			return vacio, ErrAntecedentesCarreraInvalidos
		}
		r := &p.Relaciones[i]
		if f.Corte.VigenteEn.AntesDe(s.Traza.Desde) {
			r.Pendientes = agregarPendienteCarrera(r.Pendientes, "hechos_fuera_corte")
		}
		if f.Corte.VigenteEn.AntesDe(s.PeriodoHasta) {
			r.Pendientes = agregarPendienteCarrera(r.Pendientes, "servicios_fuera_corte")
		}
		r.Servicios = append(r.Servicios, ServicioAntecedentesCarrera{ServicioRef: s.ServicioRef, Estado: s.Estado, PeriodoDesde: s.PeriodoDesde, PeriodoHasta: s.PeriodoHasta, DiasReconocidos: s.DiasReconocidos, Traza: s.Traza})
	}
	for _, s := range f.Situaciones {
		i, existe := indices[s.RelacionRef]
		if !existe || !patronReferenciaB2.MatchString(s.SituacionRef) || repetidoB2(ids, s.SituacionRef+":"+strconv.FormatInt(s.Traza.Version, 10)) || !patronReferenciaB2.MatchString(s.CodigoRef) || (s.Estado != "vigente" && s.Estado != "finalizada" && s.Estado != "rectificada") || !trazaAntecedenteCarreraValida(s.Traza, f.Corte) {
			return vacio, ErrAntecedentesCarreraInvalidos
		}
		if f.Corte.VigenteEn.AntesDe(s.Traza.Desde) {
			p.Relaciones[i].Pendientes = agregarPendienteCarrera(p.Relaciones[i].Pendientes, "hechos_fuera_corte")
		}
		p.Relaciones[i].Situaciones = append(p.Relaciones[i].Situaciones, SituacionAntecedentesCarrera{s.SituacionRef, s.Estado, s.CodigoRef, s.Traza})
	}
	if len(p.Relaciones) == 0 {
		p.Pendientes = append(p.Pendientes, "sin_relacion")
	}
	for i := range p.Relaciones {
		r := &p.Relaciones[i]
		if len(r.Servicios) == 0 {
			r.Pendientes = append(r.Pendientes, "sin_servicios")
		}
		if len(r.Situaciones) == 0 {
			r.Pendientes = append(r.Pendientes, "sin_situaciones")
		}
		actuales := ultimosServiciosCarrera(r.Servicios)
		versionesServicio := make(map[string]int64, len(actuales))
		for _, actual := range actuales {
			versionesServicio[actual.ServicioRef] = actual.Traza.Version
		}
		for j := range r.Servicios {
			r.Servicios[j].UltimaRevisionConocida = r.Servicios[j].Traza.Version == versionesServicio[r.Servicios[j].ServicioRef]
		}
		for _, servicio := range actuales {
			if servicio.Estado != "reconocido" {
				r.Pendientes = agregarPendienteCarrera(r.Pendientes, "servicios_no_reconocidos")
			}
		}
		if serviciosCarreraSolapados(actuales) {
			r.Pendientes = append(r.Pendientes, "periodos_solapados")
		}
	}
	return p, nil
}

func trazaAntecedenteCarreraValida(t TrazaEmpleadoB2, c CorteEmpleadoB2) bool {
	return t.ValidarEn(c) == nil
}

func agregarPendienteCarrera(p []string, clave string) []string {
	for _, actual := range p {
		if actual == clave {
			return p
		}
	}
	return append(p, clave)
}

func serviciosCarreraSolapados(servicios []ServicioAntecedentesCarrera) bool {
	for i, a := range servicios {
		for _, b := range servicios[i+1:] {
			if a.ServicioRef != b.ServicioRef && !a.PeriodoHasta.AntesDe(b.PeriodoDesde) && !b.PeriodoHasta.AntesDe(a.PeriodoDesde) {
				return true
			}
		}
	}
	return false
}

func ultimosServiciosCarrera(servicios []ServicioAntecedentesCarrera) []ServicioAntecedentesCarrera {
	actuales := make([]ServicioAntecedentesCarrera, 0, len(servicios))
	indices := make(map[string]int, len(servicios))
	for _, s := range servicios {
		if i, existe := indices[s.ServicioRef]; existe {
			if trazaCarreraPosterior(s.Traza, actuales[i].Traza) {
				actuales[i] = s
			}
			continue
		}
		indices[s.ServicioRef] = len(actuales)
		actuales = append(actuales, s)
	}
	return actuales
}

// La fecha de conocimiento precede a la versión, como en las consultas B2.
func trazaCarreraPosterior(a, b TrazaEmpleadoB2) bool {
	return a.RegistradaEn.After(b.RegistradaEn) || (a.RegistradaEn.Equal(b.RegistradaEn) && a.Version > b.Version)
}
