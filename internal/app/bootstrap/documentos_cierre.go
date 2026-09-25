package bootstrap

import (
	"context"
	"sync"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

// cerrarDocumentos vacía el emisor antes de cerrar los recursos dependientes.
// La lista sigue el orden de apertura; se cierra en orden inverso.
func cerrarDocumentos(ctx context.Context, cerrarEmisor vecports.CierreEmisionIncidencias, cierres []func()) error {
	var err error
	if cerrarEmisor != nil {
		err = cerrarEmisor.Cerrar(ctx)
	}
	for i := len(cierres) - 1; i >= 0; i-- {
		cierres[i]()
	}
	return err
}

func nuevoCierreDocumentos(cerrar func(context.Context) error) func(context.Context) error {
	var unaVez sync.Once
	var resultado error
	return func(ctx context.Context) error {
		unaVez.Do(func() { resultado = cerrar(ctx) })
		return resultado
	}
}
