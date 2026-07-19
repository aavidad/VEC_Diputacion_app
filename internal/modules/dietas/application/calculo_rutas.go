// Package application contiene los casos de uso del modulo de Dietas. Sus
// contratos no conocen HTTP, OSRM ni la forma de despliegue.
package application

import (
	"context"
	"errors"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

var ErrComposicionCalculoRutasInvalida = errors.New("dietas: composicion del calculo de rutas invalida")

// CasoUsoCalculoRutas es el puerto de entrada que consumen las superficies
// HTTP productiva y de presentacion.
type CasoUsoCalculoRutas interface {
	Calcular(context.Context, dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error)
}

// ServicioCalculoRutas media entre los puertos de entrada y el motor
// intercambiable. Es el lugar estable para incorporar reglas de Dietas sin
// acoplarlas al proveedor cartografico.
type ServicioCalculoRutas struct {
	motor dietasports.CalculadorRutas
}

func NuevoServicioCalculoRutas(motor dietasports.CalculadorRutas) (*ServicioCalculoRutas, error) {
	if motor == nil {
		return nil, ErrComposicionCalculoRutasInvalida
	}
	return &ServicioCalculoRutas{motor: motor}, nil
}

func (s *ServicioCalculoRutas) Calcular(ctx context.Context, solicitud dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error) {
	if s == nil || s.motor == nil {
		return dietasports.ResultadoCalculoRuta{}, ErrComposicionCalculoRutasInvalida
	}
	return s.motor.Calcular(ctx, solicitud)
}

var _ CasoUsoCalculoRutas = (*ServicioCalculoRutas)(nil)
