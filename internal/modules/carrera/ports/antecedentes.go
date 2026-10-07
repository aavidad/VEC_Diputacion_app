package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/carrera/domain"
)

// DTO propuesto para preparar el consumidor con datos sintéticos; B no ha
// aprobado un contrato nominal. Su lector y la autorización H08 siguen pendientes.
type ConsultaAntecedentesSinteticos struct {
	CasoRef, VersionEsperada string
}

type OcupacionAntecedente struct {
	Referencia string
	Periodo    domain.Periodo
	Nivel      *int
}

type GradoAntecedente struct {
	Valor     int
	Evidencia domain.Evidencia
}

type ServicioAntecedente struct {
	Referencia, Estado string
	Periodo            domain.Periodo
}

type InstantaneaAntecedentesSinteticos struct {
	Alcance, CasoRef, Version                   string
	PersonaRef, EmpleadoRef, RelacionRef        string
	CorteEfectivo, CorteConocimiento, Cobertura string
	Regimen, GrupoSubgrupo, GrupoProfesional    string
	Fuentes                                     []domain.Fuente
	Ocupaciones                                 []OcupacionAntecedente
	Grado                                       *GradoAntecedente
	Servicios                                   []ServicioAntecedente
}

type LectorAntecedentesSinteticos interface {
	ConsultarAntecedentesSinteticos(context.Context, ConsultaAntecedentesSinteticos) (InstantaneaAntecedentesSinteticos, error)
}
