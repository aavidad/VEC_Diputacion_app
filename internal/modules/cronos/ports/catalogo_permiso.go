package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

// La lectura devuelve una version durable y los hechos del mismo empleado,
// permiso y año. Ninguna ausencia se interpreta como cuota cero o concesion.
type FuenteCatalogoPermisos interface {
	CatalogoVigente(context.Context, string) (domain.CatalogoPermisoVersion, error)
	CatalogoPorVersion(context.Context, string) (domain.CatalogoPermisoVersion, error)
	// Todos los hechos del permiso durante el año, con su version original.
	HechosAnuales(context.Context, OrdenPermiso, string, string, int) ([]domain.HechoResumenPermiso, error)
}

// Autoridad de calendario: aplica la version de politica y calendario comun
// a fechas civiles inclusivas. Para horas devuelve minutos exactos.
type CalculadorCantidadPermiso interface {
	CalcularCantidad(context.Context, domain.CatalogoPermisoVersion, string, string, string) (int64, error)
}
