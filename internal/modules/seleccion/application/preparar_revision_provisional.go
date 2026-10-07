package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// PrepararRevisionProvisional compone la nueva revisión de la provisional desde
// su material completo, comprueba que conserva la anterior y solo añade
// solicitudes omitidas, y la enlaza con la huella de la anterior. No aprueba,
// publica ni persiste.
func PrepararRevisionProvisional(ctx context.Context, m ports.MaterialRevisionProvisional, catalogos ports.CatalogoAdmision) (domain.RevisionListaProvisional, error) {
	cero := domain.RevisionListaProvisional{}
	if ctx == nil || m.Antecedente.Validar() != nil {
		return cero, domain.ErrRevisionLista
	}
	nueva, err := PrepararListaAdmisionProvisional(ctx, m.Material, catalogos)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ErrCatalogoAdmisionNoDisponible) {
			return cero, err
		}
		return cero, domain.ErrRevisionLista
	}
	incorporadas, err := domain.ComprobarRevisionProvisional(m.Anterior, nueva)
	if err != nil {
		return cero, err
	}
	// La anterior debe ser exactamente la declarada: un archivo recortado o con
	// una decisión cambiada, aunque sea coherente por dentro, no coincide.
	anterior, err := identificarLista(m.Anterior)
	if err != nil || anterior != m.Antecedente {
		return cero, domain.ErrRevisionLista
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return domain.RevisionListaProvisional{Esquema: domain.EsquemaRevisionListaProvisional, Lista: nueva, Anterior: anterior,
		Incorporadas: incorporadas, Pendientes: []string{"seleccion.revision_lista.pendiente.plazo_incorporadas"}}, nil
}
