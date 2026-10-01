package json

import "vec-diputacion-granada/internal/modules/carrera/domain"

type fuente struct {
	Referencia string `json:"referencia"`
	Version    string `json:"version"`
}
type evidencia struct {
	Referencia string `json:"referencia"`
	Fuente     string `json:"fuente"`
	Version    string `json:"version"`
}
type periodo struct {
	Inicio    string    `json:"inicio"`
	Fin       string    `json:"fin"`
	Evidencia evidencia `json:"evidencia"`
}
type politica struct {
	Referencia           string   `json:"referencia"`
	Version              string   `json:"version"`
	Fuente               string   `json:"fuente"`
	AprobacionReferencia string   `json:"aprobacion_referencia"`
	Regimenes            []string `json:"regimenes"`
}
type caso struct {
	Referencia          string      `json:"referencia"`
	Via                 string      `json:"via"`
	PersonaNombre       string      `json:"persona_nombre"`
	Regimen             string      `json:"regimen"`
	NivelPuesto         *int        `json:"nivel_puesto"`
	GradoPersonal       *int        `json:"grado_personal"`
	GrupoSubgrupo       string      `json:"grupo_subgrupo"`
	GrupoProfesional    string      `json:"grupo_profesional"`
	Fuentes             []fuente    `json:"fuentes"`
	Periodos            []periodo   `json:"periodos"`
	Evidencias          []evidencia `json:"evidencias"`
	Politica            politica    `json:"politica"`
	Convenio            evidencia   `json:"convenio"`
	ConvenioConsolidado bool        `json:"convenio_consolidado"`
	Curso               evidencia   `json:"curso"`
	Prueba              evidencia   `json:"prueba"`
	Convocatoria        evidencia   `json:"convocatoria"`
	CursoAprobacion     string      `json:"curso_aprobacion"`
	PruebaAprobacion    string      `json:"prueba_aprobacion"`
}
type escenario struct {
	Alcance string `json:"alcance"`
	Version string `json:"version"`
	Fuente  fuente `json:"fuente"`
	Casos   []caso `json:"casos"`
}

func (f fuente) domain() domain.Fuente {
	return domain.Fuente{Referencia: f.Referencia, Version: f.Version}
}
func (e evidencia) domain() domain.Evidencia {
	return domain.Evidencia{Referencia: e.Referencia, Fuente: e.Fuente, Version: e.Version}
}
func (c caso) domain() domain.Caso {
	d := domain.Caso{
		Referencia: c.Referencia, Via: c.Via, PersonaNombre: c.PersonaNombre,
		Regimen: c.Regimen, NivelPuesto: c.NivelPuesto, GradoPersonal: c.GradoPersonal,
		GrupoSubgrupo: c.GrupoSubgrupo, GrupoProfesional: c.GrupoProfesional,
		Convenio: c.Convenio.domain(), ConvenioConsolidado: c.ConvenioConsolidado,
		Curso: c.Curso.domain(), Prueba: c.Prueba.domain(), Convocatoria: c.Convocatoria.domain(),
		CursoAprobacion: c.CursoAprobacion, PruebaAprobacion: c.PruebaAprobacion,
		Politica: domain.Politica{
			Referencia: c.Politica.Referencia, Version: c.Politica.Version,
			Fuente: c.Politica.Fuente, AprobacionReferencia: c.Politica.AprobacionReferencia,
			Regimenes: c.Politica.Regimenes,
		},
	}
	for _, f := range c.Fuentes {
		d.Fuentes = append(d.Fuentes, f.domain())
	}
	for _, p := range c.Periodos {
		d.Periodos = append(d.Periodos, domain.Periodo{Inicio: p.Inicio, Fin: p.Fin, Evidencia: p.Evidencia.domain()})
	}
	for _, e := range c.Evidencias {
		d.Evidencias = append(d.Evidencias, e.domain())
	}
	return d
}

type comprobacion struct {
	Clave       string `json:"clave"`
	Estado      string `json:"estado"`
	MotivoClave string `json:"motivo_clave"`
}
type preparacionCaso struct {
	Referencia     string         `json:"referencia"`
	Via            string         `json:"via"`
	PersonaNombre  string         `json:"persona_nombre"`
	Regimen        string         `json:"regimen"`
	EstadoGlobal   string         `json:"estado_global"`
	NivelPuesto    *int           `json:"nivel_puesto"`
	GradoPersonal  *int           `json:"grado_personal"`
	Fuentes        []fuente       `json:"fuentes"`
	Antecedentes   antecedentes   `json:"antecedentes"`
	Comprobaciones []comprobacion `json:"comprobaciones"`
	Pendientes     []string       `json:"pendientes"`
}
type antecedentes struct {
	GrupoSubgrupo       string      `json:"grupo_subgrupo"`
	GrupoProfesional    string      `json:"grupo_profesional"`
	Periodos            []periodo   `json:"periodos"`
	Evidencias          []evidencia `json:"evidencias"`
	Politica            politica    `json:"politica"`
	Convenio            evidencia   `json:"convenio"`
	ConvenioConsolidado bool        `json:"convenio_consolidado"`
	Curso               evidencia   `json:"curso"`
	Prueba              evidencia   `json:"prueba"`
	Convocatoria        evidencia   `json:"convocatoria"`
	CursoAprobacion     string      `json:"curso_aprobacion"`
	PruebaAprobacion    string      `json:"prueba_aprobacion"`
}
type preparacion struct {
	Alcance    string            `json:"alcance"`
	Version    string            `json:"version"`
	Fuente     fuente            `json:"fuente"`
	Casos      []preparacionCaso `json:"casos"`
	Pendientes []string          `json:"pendientes"`
}

func proyectar(p domain.Preparacion) preparacion {
	o := preparacion{Alcance: p.Alcance, Version: p.Version, Fuente: fuente{p.Fuente.Referencia, p.Fuente.Version}, Casos: []preparacionCaso{}, Pendientes: p.Pendientes}
	for _, c := range p.Casos {
		x := preparacionCaso{Referencia: c.Referencia, Via: c.Via, PersonaNombre: c.PersonaNombre, Regimen: c.Regimen, EstadoGlobal: c.EstadoGlobal, NivelPuesto: c.NivelPuesto, GradoPersonal: c.GradoPersonal, Fuentes: []fuente{}, Comprobaciones: []comprobacion{}, Pendientes: c.Pendientes}
		x.Antecedentes = proyectarAntecedentes(c.Antecedentes)
		for _, f := range c.Fuentes {
			x.Fuentes = append(x.Fuentes, fuente{f.Referencia, f.Version})
		}
		for _, v := range c.Comprobaciones {
			x.Comprobaciones = append(x.Comprobaciones, comprobacion{v.Clave, v.Estado, v.MotivoClave})
		}
		o.Casos = append(o.Casos, x)
	}
	return o
}
func evidenciaDTO(e domain.Evidencia) evidencia {
	return evidencia{Referencia: e.Referencia, Fuente: e.Fuente, Version: e.Version}
}
func proyectarAntecedentes(a domain.Antecedentes) antecedentes {
	o := antecedentes{
		GrupoSubgrupo: a.GrupoSubgrupo, GrupoProfesional: a.GrupoProfesional,
		Periodos: []periodo{}, Evidencias: []evidencia{},
		Politica: politica{
			Referencia: a.Politica.Referencia, Version: a.Politica.Version,
			Fuente: a.Politica.Fuente, AprobacionReferencia: a.Politica.AprobacionReferencia,
			Regimenes: append([]string{}, a.Politica.Regimenes...),
		},
		Convenio: evidenciaDTO(a.Convenio), ConvenioConsolidado: a.ConvenioConsolidado,
		Curso: evidenciaDTO(a.Curso), Prueba: evidenciaDTO(a.Prueba), Convocatoria: evidenciaDTO(a.Convocatoria),
		CursoAprobacion: a.CursoAprobacion, PruebaAprobacion: a.PruebaAprobacion,
	}
	for _, p := range a.Periodos {
		o.Periodos = append(o.Periodos, periodo{Inicio: p.Inicio, Fin: p.Fin, Evidencia: evidenciaDTO(p.Evidencia)})
	}
	for _, e := range a.Evidencias {
		o.Evidencias = append(o.Evidencias, evidenciaDTO(e))
	}
	return o
}
