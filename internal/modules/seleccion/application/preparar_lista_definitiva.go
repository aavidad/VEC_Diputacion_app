package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// IdentificarListaProvisional recompone la provisional y devuelve la huella de
// su serialización. RRHH la copia en el material de la definitiva para fijar
// qué provisional resuelve; no es aprobación ni firma.
func IdentificarListaProvisional(ctx context.Context, m ports.MaterialListaAdmision, catalogos ports.CatalogoAdmision) (domain.AntecedenteLista, error) {
	lista, err := PrepararListaAdmisionProvisional(ctx, m, catalogos)
	if err != nil {
		return domain.AntecedenteLista{}, err
	}
	return identificarLista(lista)
}

func identificarLista(l domain.ListaAdmisionProvisional) (domain.AntecedenteLista, error) {
	serializada, err := json.Marshal(struct {
		Esquema string                          `json:"esquema"`
		Lista   domain.ListaAdmisionProvisional `json:"lista"`
	}{domain.EsquemaAntecedenteLista, l})
	if err != nil {
		return domain.AntecedenteLista{}, domain.ErrListaDefinitiva
	}
	huella := sha256.Sum256(serializada)
	return domain.AntecedenteLista{Esquema: domain.EsquemaAntecedenteLista, ListaRef: l.ListaRef, Revision: l.Revision,
		HuellaSHA256: hex.EncodeToString(huella[:])}, nil
}

// PrepararListaAdmisionDefinitiva recompone la provisional con su catálogo,
// comprueba que su huella es la declarada y aplica las resoluciones. No
// aprueba, publica ni persiste, y no registra los escritos de subsanación.
func PrepararListaAdmisionDefinitiva(ctx context.Context, m ports.MaterialListaDefinitiva, catalogos ports.CatalogoAdmision) (domain.ListaAdmisionDefinitiva, error) {
	cero := domain.ListaAdmisionDefinitiva{}
	if ctx == nil || m.Alcance != domain.AlcanceAdmisionPreparacion || len(m.Resoluciones) > domain.MaximoSolicitudesLista {
		return cero, domain.ErrListaDefinitiva
	}
	provisional, err := PrepararListaAdmisionProvisional(ctx, m.Provisional, catalogos)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ErrCatalogoAdmisionNoDisponible) {
			return cero, err
		}
		return cero, domain.ErrListaDefinitiva
	}
	antecedente, err := identificarLista(provisional)
	if err != nil || antecedente != m.Antecedente {
		return cero, domain.ErrListaDefinitiva
	}
	definitiva, err := domain.ComponerListaDefinitiva(m.ListaRef, m.Revision, provisional, antecedente, m.Resoluciones)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return definitiva, nil
}
