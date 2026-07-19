// Package ports declara los contratos que el dominio de Dietas exige a un
// motor cartografico. Ningun tipo de este fichero depende de OSRM ni de HTTP.
package ports

import (
	"context"
	"errors"
)

const (
	MaximoCoordenadasRuta  = 25
	MaximoAlternativasRuta = 3
)

var (
	ErrSolicitudRutaInvalida       = errors.New("dietas: solicitud de ruta invalida")
	ErrMotorRutasNoDisponible      = errors.New("dietas: motor de rutas no disponible")
	ErrRespuestaMotorRutasInvalida = errors.New("dietas: respuesta del motor de rutas invalida")
)

// CoordenadaRuta es un punto autorizado del catalogo territorial. Nombre es
// descriptivo y nunca interviene en la consulta al motor.
type CoordenadaRuta struct {
	Latitud  float64
	Longitud float64
	Nombre   string
}

// SolicitudCalculoRuta contiene exclusivamente los datos necesarios para
// calcular un itinerario. Cero alternativas conserva el minimo de una ruta.
type SolicitudCalculoRuta struct {
	Coordenadas  []CoordenadaRuta
	Alternativas int
}

// PuntoGeometriaRuta usa el orden GeoJSON: longitud y despues latitud.
type PuntoGeometriaRuta struct {
	Longitud float64
	Latitud  float64
}

type GeometriaRuta struct {
	Tipo        string
	Coordenadas []PuntoGeometriaRuta
}

type TramoRuta struct {
	DistanciaMetros  float64
	DuracionSegundos float64
}

type AlternativaRuta struct {
	DistanciaMetros  float64
	DuracionSegundos float64
	Tramos           []TramoRuta
	Geometria        GeometriaRuta
}

// ResultadoCalculoRuta es la proyeccion minima y neutral que necesita VEC. No
// reexpone waypoints, pesos internos ni campos accidentales del proveedor.
type ResultadoCalculoRuta struct {
	Alternativas []AlternativaRuta
	VersionGrafo string
	Motor        string
	Ambito       string
}

// CalculadorRutas es el puerto de salida compartido por la API productiva y
// por la superficie cartografica no autoritativa de presentacion.
type CalculadorRutas interface {
	Calcular(context.Context, SolicitudCalculoRuta) (ResultadoCalculoRuta, error)
}
