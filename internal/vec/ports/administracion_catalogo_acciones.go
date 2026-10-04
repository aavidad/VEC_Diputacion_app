package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// FuenteCatalogoAccionesAdministracionV1 solo lee la instantánea exacta
// publicada por una fuente confiable. No acepta catálogos enviados por el
// cliente como autoridad ni publica perfiles o concesiones.
// El adaptador debe devolver un error si faltan datos de módulos o perfiles:
// una omisión podría ocultar la condición fija de una semilla de sistema.
type FuenteCatalogoAccionesAdministracionV1 interface {
	ObtenerCatalogoAccionesAdministracionV1(context.Context, string, int, string) (domain.CatalogoAccionesAdministracionV1, error)
}
