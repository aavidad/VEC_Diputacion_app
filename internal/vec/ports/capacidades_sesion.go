package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrSeleccionSesionNoDisponible = errors.New("vec: seleccion de sesion no disponible")

// ExposicionSesion describe una operación efectivamente montada en una superficie.
// La composición la declara desde sus rutas reales; el manifiesto no concede acceso.
type ExposicionSesion struct {
	Superficie  string
	ModuloID    string
	TipoRecurso string
	Accion      string
}

// SeleccionSesion procede de identidad y composición confiables, nunca de la
// petición del navegador. VigenteHasta incluye certificado, sesión y CRL.
type SeleccionSesion struct {
	PerfilActivoRef string
	Superficie      string
	VigenteHasta    time.Time
	Exposiciones    []ExposicionSesion
}

type SelectorSesion interface {
	SeleccionarSesion(context.Context, domain.Principal) (SeleccionSesion, error)
}
