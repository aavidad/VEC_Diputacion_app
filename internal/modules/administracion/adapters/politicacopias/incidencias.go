package politicacopias

import (
	"context"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
	common "vec-diputacion-granada/internal/vec/domain"
	commonports "vec-diputacion-granada/internal/vec/ports"
)

// AvisadorComun reuses the shared, non-blocking technical incident emitter. The
// composition selects an existing appropriate closed-catalogue category; this
// adapter neither adds a catalogue nor forwards operational/person references.
// Successful invocation does not assert that an incident was retained/delivered.
type AvisadorComun struct {
	Emisor        commonports.EmisorIncidenciasTecnicas
	Clasificacion common.SolicitudIncidenciaTecnica
}

func (a AvisadorComun) NotificarFallo(_ context.Context, _ p.FalloAgenda) error {
	if a.Emisor == nil {
		return d.ErrAviso
	}
	definition, ok := common.DefinicionIncidenciaTecnicaDe(a.Clasificacion.Codigo)
	if !ok {
		return d.ErrAviso
	}
	component, stage := false, false
	for _, c := range definition.Componentes {
		if c == a.Clasificacion.Componente {
			component = true
		}
	}
	for _, e := range definition.Etapas {
		if e == a.Clasificacion.Etapa {
			stage = true
		}
	}
	if !component || !stage {
		return d.ErrAviso
	}
	a.Emisor.Emitir(a.Clasificacion)
	return nil
}
