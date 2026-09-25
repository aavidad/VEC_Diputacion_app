package reglas

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// CatalogoVigente devuelve la versión vigente del catálogo, su huella y el
// instante con el que se eligió (para filtrar entradas vigentes), sin
// exigir el contrato de atributos de las reglas. Lo usan los catálogos de
// datos con formato propio (por ejemplo, las retribuciones de referencia) que
// comparten la selección de versión y la marca de paquete de ejemplo.
func (r *Resolutor) CatalogoVigente(ctx context.Context) (domain.CatalogoConfigurable, string, time.Time, error) {
	var vacio domain.CatalogoConfigurable
	if r == nil {
		return vacio, "", time.Time{}, ErrReglasNoConfiguradas
	}
	if ctx == nil {
		return vacio, "", time.Time{}, ErrReglasNoDisponibles
	}
	catalogo, instante, err := r.catalogoVigente(ctx)
	if err != nil {
		return vacio, "", time.Time{}, err
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return vacio, "", time.Time{}, ErrReglasNoDisponibles
	}
	return catalogo, huella, instante, nil
}

// CatalogoRetribucionesCT es el catálogo de retribuciones de referencia con el
// que Contratación temporal estima el coste de un nombramiento.
const CatalogoRetribucionesCT = "ct.retribuciones"
