package application

import (
	"context"
	"errors"
	"reflect"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// ErrCatalogoAdmisionNoDisponible indica que el catálogo pedido no se pudo
// obtener. Sin catálogo no hay motivos válidos y la lista no se compone.
var ErrCatalogoAdmisionNoDisponible = errors.New("seleccion.lista_admision.catalogo_no_disponible")

// PrepararListaAdmisionProvisional compone el borrador de la lista provisional
// de admitidos y excluidos. Exige una decisión por cada revisión S4 aportada,
// todas de las mismas bases, y comprueba cada antecedente recalculando su
// huella. No aprueba, publica, persiste ni calcula el vencimiento del plazo.
func PrepararListaAdmisionProvisional(ctx context.Context, m ports.MaterialListaAdmision, catalogos ports.CatalogoAdmision) (domain.ListaAdmisionProvisional, error) {
	cero := domain.ListaAdmisionProvisional{}
	if ctx == nil || catalogos == nil || reflect.ValueOf(catalogos).Kind() == reflect.Pointer && reflect.ValueOf(catalogos).IsNil() ||
		m.Alcance != domain.AlcanceAdmisionPreparacion || !meritos.ReferenciaValida(m.CatalogoRef) || !meritos.ReferenciaValida(m.CatalogoVersion) ||
		len(m.RevisionesS4) == 0 || len(m.RevisionesS4) > domain.MaximoSolicitudesLista || len(m.Decisiones) != len(m.RevisionesS4) {
		return cero, domain.ErrListaAdmision
	}
	antecedentes := make(map[string]domain.AntecedenteAdmision, len(m.RevisionesS4))
	for _, material := range m.RevisionesS4 {
		revision, err := PrepararAdmision(ctx, material)
		if err != nil {
			if ctx.Err() != nil {
				return cero, ctx.Err()
			}
			return cero, domain.ErrListaAdmision
		}
		if revision.Bases != m.Bases {
			return cero, domain.ErrListaAdmision
		}
		if _, repetida := antecedentes[material.PreparacionRef]; repetida {
			return cero, domain.ErrListaAdmision
		}
		antecedente, err := identificarMaterialAdmision(material)
		if err != nil {
			return cero, domain.ErrListaAdmision
		}
		antecedentes[material.PreparacionRef] = antecedente
	}
	// Cada decisión debe nombrar exactamente una revisión S4 aportada; como
	// hay tantas decisiones como revisiones y no se repiten, el universo queda
	// cubierto sin solicitudes olvidadas.
	for _, d := range m.Decisiones {
		if esperado, ok := antecedentes[d.Antecedente.PreparacionRef]; !ok || esperado != d.Antecedente {
			return cero, domain.ErrListaAdmision
		}
	}
	catalogo, err := catalogos.CatalogoAdmision(ctx, m.CatalogoRef, m.CatalogoVersion)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ErrCatalogoAdmisionNoDisponible
	}
	if catalogo.Referencia != m.CatalogoRef || catalogo.Version != m.CatalogoVersion {
		return cero, ErrCatalogoAdmisionNoDisponible
	}
	lista, err := domain.ComponerListaProvisional(m.ListaRef, m.Revision, m.Bases, catalogo, m.Decisiones)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return lista, nil
}
