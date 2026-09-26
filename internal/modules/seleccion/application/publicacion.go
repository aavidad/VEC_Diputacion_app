package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// PublicacionConvocatoria resume una publicación al arrancar.
type PublicacionConvocatoria struct {
	Ref     string
	Version int
	Nueva   bool
}

// PublicarConvocatorias publica cada convocatoria del catálogo. Es
// idempotente: el mismo catálogo reutiliza las versiones vigentes, de modo
// que un segundo arranque no crea versiones ni falla.
func PublicarConvocatorias(ctx context.Context, fuente ports.FuenteConvocatorias, registro ports.RegistroConvocatorias) ([]PublicacionConvocatoria, error) {
	if ctx == nil || nula(fuente) || nula(registro) {
		return nil, ports.ErrNoDisponible
	}
	convocatorias, err := fuente.Convocatorias(ctx)
	if err != nil {
		return nil, errors.Join(ports.ErrNoDisponible, err)
	}
	if len(convocatorias) == 0 {
		return nil, ports.ErrConvocatoriaNoDisponible
	}
	resultado := make([]PublicacionConvocatoria, 0, len(convocatorias))
	vistas := map[string]bool{}
	for _, c := range convocatorias {
		if err := c.Validar(); err != nil || vistas[c.Ref] {
			return nil, errors.Join(ports.ErrNoDisponible, err)
		}
		vistas[c.Ref] = true
		version, nueva, err := registro.PublicarConvocatoria(ctx, c)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, PublicacionConvocatoria{Ref: c.Ref, Version: version, Nueva: nueva})
	}
	return resultado, nil
}
