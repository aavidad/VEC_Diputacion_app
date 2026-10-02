package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrMismaPersonaEnOtroPasoR5 = errors.New("contratacion temporal: misma persona en otro paso del expediente")

// La bandera se publica dentro de la versión del catálogo de circuito.
// Ausencia o caída de la fuente nunca equivale a permitir dos pasos por la
// misma persona. El alcance entre documentos/rondas queda pendiente de RRHH.
type PoliticaMismaPersonaEnPasos struct {
	CatalogoRef    string
	CatalogoHuella string
	Permite        bool
}

func (p PoliticaMismaPersonaEnPasos) ValidarContra(ref, huella string) error {
	if !domain.ReferenciaOpacaValida(ref) || !domain.HuellaSHA256FirmaValida(huella) ||
		p.CatalogoRef != ref || p.CatalogoHuella != huella {
		return domain.ErrCircuitoFirmaIncoherente
	}
	return nil
}

type FuentePoliticaMismaPersonaEnPasos interface {
	PoliticaMismaPersonaEnPasos(context.Context, string, string) (PoliticaMismaPersonaEnPasos, error)
}
