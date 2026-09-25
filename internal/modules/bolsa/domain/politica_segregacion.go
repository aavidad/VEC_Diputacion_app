package domain

import (
	"errors"
	"slices"
)

var ErrPoliticaSegregacionInvalida = errors.New("bolsa: politica de segregacion invalida")

// PoliticaSegregacion enumera las operaciones de B8 en las que quien valida
// no puede ser quien registra. Procede de un catálogo configurable (duda 6
// de RRHH), pero nunca relaja el mínimo fijo de ExigeValidadorDistinto: un
// catálogo solo puede añadir operaciones.
type PoliticaSegregacion struct {
	operaciones []string
}

// PoliticaSegregacionMinima es la conducta sin catálogo: solo el mínimo fijo.
func PoliticaSegregacionMinima() PoliticaSegregacion {
	minima := make([]string, 0, 1)
	for _, operacion := range OperacionesSituacionParticipacion() {
		if ExigeValidadorDistinto(operacion) {
			minima = append(minima, operacion)
		}
	}
	return PoliticaSegregacion{operaciones: minima}
}

// NuevaPoliticaSegregacion valida una lista configurada. Rechaza operaciones
// desconocidas, repetidas o vacías y cualquier lista que omita el mínimo fijo.
func NuevaPoliticaSegregacion(operaciones []string) (PoliticaSegregacion, error) {
	conocidas := OperacionesSituacionParticipacion()
	vistas := make(map[string]struct{}, len(operaciones))
	for _, operacion := range operaciones {
		if _, repetida := vistas[operacion]; repetida || !slices.Contains(conocidas, operacion) {
			return PoliticaSegregacion{}, ErrPoliticaSegregacionInvalida
		}
		vistas[operacion] = struct{}{}
	}
	canonica := make([]string, 0, len(vistas))
	for _, operacion := range conocidas {
		_, incluida := vistas[operacion]
		if ExigeValidadorDistinto(operacion) && !incluida {
			return PoliticaSegregacion{}, ErrPoliticaSegregacionInvalida
		}
		if incluida {
			canonica = append(canonica, operacion)
		}
	}
	return PoliticaSegregacion{operaciones: canonica}, nil
}

// Operaciones devuelve una copia en el orden estable del catálogo de B8.
func (p PoliticaSegregacion) Operaciones() []string {
	if len(p.operaciones) == 0 {
		return PoliticaSegregacionMinima().Operaciones()
	}
	return slices.Clone(p.operaciones)
}

// ExigeSegundaPersona combina el mínimo fijo con la lista configurada. El
// valor cero de la política equivale a la mínima.
func (p PoliticaSegregacion) ExigeSegundaPersona(operacion string) bool {
	return ExigeValidadorDistinto(operacion) || slices.Contains(p.operaciones, operacion)
}
