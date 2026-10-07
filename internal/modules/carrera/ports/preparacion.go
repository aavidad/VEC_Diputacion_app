package ports

import "vec-diputacion-granada/internal/modules/carrera/domain"

type Entrada interface {
	Leer() (domain.Escenario, error)
}
type Salida interface {
	Escribir(domain.Preparacion) error
}
