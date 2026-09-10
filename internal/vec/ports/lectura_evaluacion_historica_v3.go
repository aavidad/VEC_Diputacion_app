package ports

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"vec-diputacion-granada/internal/vec/domain"
)

var ErrLecturaEvaluacionOriginalV3 = errors.New("vec: evaluacion original no recuperable")
var referenciaEvaluacionOriginalV3 = regexp.MustCompile(`^[!-~]{1,512}$`)
var huellaEvaluacionOriginalV3 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type SolicitudLecturaEvaluacionOriginalV3 struct {
	DecisionRef, DecisionSHA256, SolicitudSHA256 string
}

func (s SolicitudLecturaEvaluacionOriginalV3) Validar() error {
	if !referenciaEvaluacionOriginalV3.MatchString(s.DecisionRef) || strings.Contains(s.DecisionRef, "*") || !huellaEvaluacionOriginalV3.MatchString(s.DecisionSHA256) || !huellaEvaluacionOriginalV3.MatchString(s.SolicitudSHA256) {
		return ErrLecturaEvaluacionOriginalV3
	}
	return nil
}

// El propietario garantiza que los seis documentos proceden de la decisión
// exacta. El DTO no prueba esa procedencia ni concede autoridad actual. La
// restauración vuelve a evaluar con fechas originales y coteja bytes de decisión.
type LectorEvaluacionOriginalV3 interface {
	LeerEvaluacionOriginalV3(context.Context, SolicitudLecturaEvaluacionOriginalV3) (domain.InstantaneaAutorizacion, error)
}

// Copia sólo datos públicos. No construye decisiones/concesiones nominales.
// Validar aplica las cotas de cardinalidad/campos del dominio antes de clonar.
func CopiarInstantaneaEvaluacionOriginalV3(i domain.InstantaneaAutorizacion) (domain.InstantaneaAutorizacion, error) {
	if i.Validar() != nil {
		return domain.InstantaneaAutorizacion{}, ErrLecturaEvaluacionOriginalV3
	}
	c := i
	c.AsignacionPerfil.Ambitos = slices.Clone(i.AsignacionPerfil.Ambitos)
	for n := range c.AsignacionPerfil.Ambitos {
		c.AsignacionPerfil.Ambitos[n].Valores = slices.Clone(i.AsignacionPerfil.Ambitos[n].Valores)
	}
	c.VersionRol.Concesiones = slices.Clone(i.VersionRol.Concesiones)
	for n := range c.VersionRol.Concesiones {
		a, b := &c.VersionRol.Concesiones[n], i.VersionRol.Concesiones[n]
		a.Finalidades = slices.Clone(b.Finalidades)
		a.CamposPermitidos = slices.Clone(b.CamposPermitidos)
		a.Obligaciones = slices.Clone(b.Obligaciones)
	}
	c.Politicas = slices.Clone(i.Politicas)
	for n := range c.Politicas {
		a, b := &c.Politicas[n], i.Politicas[n]
		a.Acciones = slices.Clone(b.Acciones)
		a.Modulos = slices.Clone(b.Modulos)
		a.TiposRecurso = slices.Clone(b.TiposRecurso)
		a.FinalidadesPermitidas = slices.Clone(b.FinalidadesPermitidas)
		a.CamposPermitidos = slices.Clone(b.CamposPermitidos)
		a.Obligaciones = slices.Clone(b.Obligaciones)
		a.Restricciones = slices.Clone(b.Restricciones)
		for k := range a.Restricciones {
			a.Restricciones[k].ValoresPermitidos = slices.Clone(b.Restricciones[k].ValoresPermitidos)
		}
	}
	return c, nil
}
