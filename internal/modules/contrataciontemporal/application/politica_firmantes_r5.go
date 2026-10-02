package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PasoLogicoFirmaR5 solo equipara pasos en la misma posición documental y
// bajo la misma huella de catálogo. Una huella distinta es ambigua y cuenta
// como otro paso para la política default NO.
type PasoLogicoFirmaR5 struct {
	Documento      string
	PasoOrden      int
	CatalogoHuella string
}

func (p PasoLogicoFirmaR5) Valido() bool {
	return domain.ClaveDocumentoFirmaValida(p.Documento) &&
		p.PasoOrden >= 1 && p.PasoOrden <= domain.MaximoPasosCircuitoFirma &&
		domain.HuellaSHA256FirmaValida(p.CatalogoHuella)
}

func (p PasoLogicoFirmaR5) MismoPaso(otro PasoLogicoFirmaR5) bool {
	return p.Valido() && otro.Valido() && p == otro
}

// EvaluarCoincidenciaPersonaR5 recibe solo indicadores minimizados de la
// historia global. Con default NO, una fila de otro paso sin identidad
// acreditada tampoco permite concluir que la persona es distinta.
func EvaluarCoincidenciaPersonaR5(permite, coincideOtroPaso, desconocidoOtroPaso bool) error {
	if !permite && (coincideOtroPaso || desconocidoOtroPaso) {
		return ports.ErrMismaPersonaEnOtroPasoR5
	}
	return nil
}

// ResolverPoliticaMismaPersonaEnPasos liga el valor a una versión publicada.
// El alcance de aplicación entre documentos/rondas no se deduce aquí: queda
// cerrado hasta la decisión de RRHH. Sin fuente la bandera es false.
func ResolverPoliticaMismaPersonaEnPasos(
	ctx context.Context, fuente ports.FuentePoliticaMismaPersonaEnPasos, catalogoRef, catalogoHuella string,
) (bool, error) {
	if ctx == nil {
		return false, ErrCircuitoFirmaNoDisponible
	}
	if nula(fuente) {
		return false, nil
	}
	p, err := fuente.PoliticaMismaPersonaEnPasos(ctx, catalogoRef, catalogoHuella)
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, ErrCircuitoFirmaNoDisponible
	}
	if p.ValidarContra(catalogoRef, catalogoHuella) != nil {
		return false, ErrCircuitoFirmaNoDisponible
	}
	return p.Permite, nil
}
