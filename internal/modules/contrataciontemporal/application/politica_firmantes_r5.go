package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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
