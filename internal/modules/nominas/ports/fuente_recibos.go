package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/nominas/domain"
)

// FuenteRecibos es una dependencia de lectura inyectada por el ensamblaje.
// El consumidor resuelve relación y autorización, y registra la lectura antes
// de revelar datos. El adaptador conserva las referencias originales, mantiene
// estable versión/cobertura/total durante la paginación y valida cada página.
// Una implementación de proveedor requiere fuente y tratamiento admitidos.
type FuenteRecibos interface {
	ConsultarPaginaRecibos(context.Context, domain.ConsultaFuenteRecibos) (domain.PaginaFuenteRecibos, error)
}
