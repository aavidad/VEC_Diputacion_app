package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	// ErrInformeNuevoPendiente: el catálogo exige un informe jurídico nuevo
	// tras subsanar y todavía no se ha emitido. El expediente no está listo
	// para fiscalizarse; no es una denegación de permisos.
	ErrInformeNuevoPendiente = errors.New("contratacion temporal: falta el informe juridico nuevo tras la subsanacion")
	// ErrInformeNuevoNoPrevisto: el catálogo no prevé informe nuevo tras
	// subsanar, o el expediente no está en ese punto.
	ErrInformeNuevoNoPrevisto = errors.New("contratacion temporal: informe juridico nuevo no previsto")
)

// FuenteInformeTrasSubsanacion entrega la política vigente. Un error nunca
// equivale a no exigir el informe nuevo.
type FuenteInformeTrasSubsanacion interface {
	InformeTrasSubsanacion(context.Context) (domain.PoliticaInformeTrasSubsanacion, error)
}

// FuenteRondaFirmaInforme devuelve la versión del expediente en la que se
// emitió el informe nuevo vigente (0 si no hay informe nuevo). Las firmas
// registradas antes de esa versión son de la ronda anterior.
type FuenteRondaFirmaInforme interface {
	InicioRondaInformeNuevo(ctx context.Context, organizacionRef, expedienteRef string) (uint64, error)
}
