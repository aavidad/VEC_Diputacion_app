package application

import (
	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

type Servicio struct{}

func (Servicio) Preparar(e domain.Escenario) (domain.Preparacion, error) { return domain.Preparar(e) }
func (s Servicio) Ejecutar(entrada ports.Entrada, salida ports.Salida) error {
	e, err := entrada.Leer()
	if err != nil {
		return err
	}
	p, err := s.Preparar(e)
	if err != nil {
		return err
	}
	return salida.Escribir(p)
}
