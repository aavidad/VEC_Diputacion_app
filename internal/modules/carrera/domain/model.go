package domain

import "errors"

var ErrEscenario = errors.New("carrera.error.escenario_invalido")

const AlcanceSintetico = "preparacion_sintetica"

type Fuente struct{ Referencia, Version string }
type Evidencia struct{ Referencia, Fuente, Version string }
type Periodo struct {
	Inicio, Fin string
	Evidencia   Evidencia
}
type Politica struct {
	Referencia, Version, Fuente, AprobacionReferencia string
	Regimenes                                         []string
}
type Caso struct {
	Referencia, Via, PersonaNombre, Regimen string
	NivelPuesto, GradoPersonal              *int
	GrupoSubgrupo, GrupoProfesional         string
	Fuentes                                 []Fuente
	Periodos                                []Periodo
	Evidencias                              []Evidencia
	Politica                                Politica
	Convenio, Curso, Prueba, Convocatoria   Evidencia
	ConvenioConsolidado                     bool
	CursoAprobacion, PruebaAprobacion       string
}
type Escenario struct {
	Alcance, Version string
	Fuente           Fuente
	Casos            []Caso
}
type Comprobacion struct{ Clave, Estado, MotivoClave string }
type Antecedentes struct {
	GrupoSubgrupo, GrupoProfesional       string
	Periodos                              []Periodo
	Evidencias                            []Evidencia
	Politica                              Politica
	Convenio, Curso, Prueba, Convocatoria Evidencia
	ConvenioConsolidado                   bool
	CursoAprobacion, PruebaAprobacion     string
}
type PreparacionCaso struct {
	Referencia, Via, PersonaNombre, Regimen, EstadoGlobal string
	NivelPuesto, GradoPersonal                            *int
	Fuentes                                               []Fuente
	Antecedentes                                          Antecedentes
	Comprobaciones                                        []Comprobacion
	Pendientes                                            []string
}
type Preparacion struct {
	Alcance, Version string
	Fuente           Fuente
	Casos            []PreparacionCaso
	Pendientes       []string
}
